package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	skafka "github.com/segmentio/kafka-go"

	"github.com/gorilla/websocket"
	"github.com/xeipuuv/gojsonschema"

	"src/config"
	"src/kafka"
	"src/logger"
	"src/metrics"
	"src/models"
	oidcutil "src/oidc"
	"src/store"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var broadcast = make(chan models.Message)

// Mongo handled by store package
var oidcVerifier *coreoidc.IDTokenVerifier
var clients = make(map[*websocket.Conn]bool)
var appCtx context.Context
var appCancel context.CancelFunc
var maxMsgLen = 1000

func main() {
	logger.Info("starting application")

	// Root context with cancellation for graceful shutdown (used across subsystems)
	appCtx, appCancel = context.WithCancel(context.Background())
	defer appCancel()

	// Initialize Mongo store layer
	if err := store.Init(appCtx); err != nil {
		log.Fatalf("mongo init failed: %v", err)
	}
	defer store.Close(context.Background())

	// Initialize OIDC (provider + verifier)
	_, v := oidcutil.Init(appCtx)
	oidcVerifier = v

	// Capture OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutdown signal received")
		appCancel()
		// Allow a short drain period
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	// Parse MESSAGE_MAX_LENGTH env
	if v := os.Getenv("MESSAGE_MAX_LENGTH"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxMsgLen = n
		}
	}

	// Start Kafka reader consumer -> broadcast channel
	go kafka.Reader(appCtx, broadcast)

	// HTTP routes
	http.HandleFunc("/ws", handleConnections)
	http.Handle("/messages", authMiddleware(http.HandlerFunc(handleMessages)))
	http.HandleFunc("/healthz", handleHealth)
	http.HandleFunc("/readyz", handleReady)
	http.HandleFunc("/metrics", metrics.Handler)

	// Fan-out broadcaster (Kafka -> WebSocket clients)
	go handleMessagesBroadcasting()

	logger.Info("http server listening", logger.FieldKV("port", config.ApiPort))
	if err := http.ListenAndServe(":"+config.ApiPort, nil); err != nil {
		logger.Error("http server error", err)
	}
}

func authMiddleware(next http.Handler) http.Handler {
	return oidcutil.AuthMiddleware(oidcVerifier)(next)
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		logger.Error("token missing", fmt.Errorf("no token in query"))
		http.Error(w, "token not provided", http.StatusUnauthorized)
		return
	}

	if _, err := oidcutil.VerifyToken(r.Context(), oidcVerifier, token); err != nil {
		logger.Error("token verification failed", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", err)
		return
	}
	clients[ws] = true
	metrics.IncWSConnections()
	logger.Info("websocket client connected", logger.FieldKV("remote_addr", ws.RemoteAddr().String()))

	for {
		var msg models.Message
		if err := ws.ReadJSON(&msg); err != nil {
			logger.Error("websocket read error", err, logger.FieldKV("remote_addr", ws.RemoteAddr().String()))
			delete(clients, ws)
			metrics.DecWSConnections()
			logger.Info("client disconnected", logger.FieldKV("remote_addr", ws.RemoteAddr().String()))
			ws.Close()
			return
		}

		// Ensure message ID & timestamp
		if msg.MessageID == "" {
			msg.MessageID = uuid.NewString()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now().UTC()
		}

		// Validate schema & size
		if len(msg.Content) > maxMsgLen {
			logger.Error("websocket message too long", fmt.Errorf("len=%d max=%d", len(msg.Content), maxMsgLen))
			continue
		}
		if err := validateMessageSchema(msg); err != nil {
			logger.Error("websocket message schema invalid", err, logger.FieldKV("message_id", msg.MessageID))
			continue
		}

		// Publish via Kafka (consumer will broadcast)
		if err := kafka.Writer(appCtx, msg); err != nil {
			logger.Error("kafka write from websocket failed", err, logger.FieldKV("message_id", msg.MessageID))
			continue
		}
		metrics.IncMsgIngested()
	}
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var msg models.Message
		err := json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			logger.Error("decode message body failed", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Size & schema validation
		if len(msg.Content) > maxMsgLen {
			logger.Error("http message too long", fmt.Errorf("len=%d max=%d", len(msg.Content), maxMsgLen))
			http.Error(w, "message too long", http.StatusBadRequest)
			return
		}
		if err := validateMessageSchema(msg); err != nil {
			logger.Error("message schema validation failed", err)
			http.Error(w, "Invalid message schema: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Ensure server-side ID and timestamp
		if msg.MessageID == "" {
			msg.MessageID = uuid.NewString()
		}
		msg.Timestamp = time.Now()
		logger.Info("received message via POST", logger.FieldKV("message_id", msg.MessageID))

		if err := kafka.Writer(appCtx, msg); err != nil {
			logger.Error("kafka write from http failed", err, logger.FieldKV("message_id", msg.MessageID))
			http.Error(w, "Failed to enqueue message", http.StatusInternalServerError)
			return
		}
		metrics.IncMsgIngested()

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{"message_id": msg.MessageID, "status": "enqueued"})
	} else if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		messages, err := getAllMessagesMongo()
		if err != nil {
			logger.Error("fetch messages failed", err)
			http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(messages)
	} else {
		logger.Error("invalid method", fmt.Errorf("method %s", r.Method))
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// insertMessageMongo persists a message to MongoDB
func insertMessageMongo(msg models.Message) error {
	return store.InsertMessage(context.Background(), msg)
}
func getAllMessagesMongo() ([]models.Message, error) {
	return store.GetAllMessages(context.Background())
}

// validateMessageSchema validates a Message struct against schema.json
func validateMessageSchema(msg models.Message) error {
	schemaLoader := gojsonschema.NewReferenceLoader("file://" + getSchemaPath())
	doc, _ := json.Marshal(msg)
	documentLoader := gojsonschema.NewBytesLoader(doc)
	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}
	if !result.Valid() {
		return fmt.Errorf("message does not match schema: %v", result.Errors())
	}
	return nil
}

// getSchemaPath returns the absolute path to schema.json
func getSchemaPath() string {
	cwd, _ := os.Getwd()
	return cwd + "/../schema.json"
}

func handleMessagesBroadcasting() {
	logger.Info("starting message broadcasting loop")
	for {
		msg := <-broadcast
		logger.Debug("broadcasting message", logger.FieldKV("client_count", len(clients)))
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				logger.Error("websocket write error", err, logger.FieldKV("remote_addr", client.RemoteAddr().String()))
				client.Close()
				delete(clients, client)
				logger.Info("client disconnected due to error", logger.FieldKV("remote_addr", client.RemoteAddr().String()))
			}
		}
		metrics.IncMsgBroadcast()
		// Persist message to MongoDB after broadcasting
		if err := insertMessageMongo(msg); err != nil {
			logger.Error("persist message failed", err, logger.FieldKV("message_id", msg.MessageID))
			_ = kafka.DLQWriter(appCtx, msg, "mongo_persist_failure")
		}
	}
}

// Health endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Readiness endpoint
func handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()
	if err := store.Ping(ctx); err != nil {
		http.Error(w, "mongo not ready", http.StatusServiceUnavailable)
		return
	}
	// Kafka readiness: attempt to create a reader and fetch 0 messages via offset check
	kr := skafka.NewReader(skafka.ReaderConfig{Brokers: []string{config.KafkaBroker}, Topic: config.Topic, Partition: 0, MinBytes: 1, MaxBytes: 1})
	defer kr.Close()
	// Use context timeout; try a single fetch of offset to validate metadata
	_, err := kr.ReadLag(ctx)
	if err != nil {
		http.Error(w, "kafka not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

// ensureMongoIndexes creates required indexes (idempotent). Unique index on message_id and TTL/index on timestamp for sorting.
// (Mongo index creation moved to store package)

// (OIDC provider/backoff logic moved to oidc/oidc.go)
