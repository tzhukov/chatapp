package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	skafka "github.com/segmentio/kafka-go"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/websocket"
	"github.com/xeipuuv/gojsonschema"

	"src/config"
	"src/kafka"
	"src/logger"
	"src/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var broadcast = make(chan models.Message)
var mongoClient *mongo.Client
var mongoCollection *mongo.Collection
var verifier *oidc.IDTokenVerifier
var clients = make(map[*websocket.Conn]bool)
var appCtx context.Context
var appCancel context.CancelFunc

func main() {
	logger.Info("starting application")

	// Connect to MongoDB
	var err error
	mongoClient, err = mongo.Connect(context.Background(), options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	mongoCollection = mongoClient.Database("chatapp").Collection("messages")

	// Ping to ensure MongoDB connection is alive
	if err := mongoClient.Ping(context.Background(), readpref.Primary()); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}

	// Ensure indexes (idempotent)
	if err := ensureMongoIndexes(context.Background(), mongoCollection); err != nil {
		log.Fatalf("Failed creating Mongo indexes: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, config.DexIssuer)
	if err != nil {
		log.Fatalf("Failed to create OIDC provider: %v", err)
	}

	verifier = provider.Verifier(&oidc.Config{ClientID: config.ClientID})

	// Root context with cancellation for graceful shutdown
	appCtx, appCancel = context.WithCancel(context.Background())
	defer appCancel()

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

	go kafka.Reader(appCtx, broadcast)

	http.HandleFunc("/ws", handleConnections)
	http.Handle("/messages", authMiddleware(http.HandlerFunc(handleMessages)))
	http.HandleFunc("/healthz", handleHealth)
	http.HandleFunc("/readyz", handleReady)

	go handleMessagesBroadcasting()

	logger.Info("http server listening", logger.FieldKV("port", config.ApiPort))
	if err := http.ListenAndServe(":"+config.ApiPort, nil); err != nil {
		logger.Error("http server error", err)
	}
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if _, err := verifyToken(r.Context(), token); err != nil {
			logger.Error("token verification failed", err)
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func verifyToken(ctx context.Context, rawToken string) (*oidc.IDToken, error) {
	token, err := verifier.Verify(ctx, rawToken)
	if err != nil {
		return nil, err
	}
	// Check audience claim
	var claims struct {
		Aud string `json:"aud"`
		Exp int64  `json:"exp"`
	}
	if err := token.Claims(&claims); err != nil {
		return nil, err
	}
	if claims.Aud != config.Audience {
		return nil, fmt.Errorf("invalid audience: expected %s, got %s", config.Audience, claims.Aud)
	}
	// Check expiration
	if time.Now().Unix() > claims.Exp {
		return nil, fmt.Errorf("token expired")
	}
	return token, nil
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		log.Printf("token not provided in query params")
		http.Error(w, "token not provided", http.StatusUnauthorized)
		return
	}

	if _, err := verifyToken(r.Context(), token); err != nil {
		logger.Error("token verification failed", err)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading connection: %v", err)
		return
	}
	defer ws.Close()

	clients[ws] = true
	logger.Info("client connected", logger.FieldKV("remote_addr", ws.RemoteAddr().String()))

	for {
		var msg models.Message
		if err := ws.ReadJSON(&msg); err != nil {
			logger.Error("websocket read error", err, logger.FieldKV("remote_addr", ws.RemoteAddr().String()))
			delete(clients, ws)
			logger.Info("client disconnected", logger.FieldKV("remote_addr", ws.RemoteAddr().String()))
			break
		}

		// Assign server-side ID and timestamp if missing
		if msg.MessageID == "" {
			msg.MessageID = uuid.NewString()
		}
		if msg.Timestamp.IsZero() {
			msg.Timestamp = time.Now()
		}

		// Validate schema
		if err := validateMessageSchema(msg); err != nil {
			logger.Error("websocket message schema invalid", err)
			continue
		}

		// Always route through Kafka
		if err := kafka.Writer(appCtx, msg); err != nil {
			logger.Error("kafka write from websocket failed", err, logger.FieldKV("message_id", msg.MessageID))
			continue
		}
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

		// Validate message against schema
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
	filter := bson.M{"message_id": msg.MessageID}
	update := bson.M{"$setOnInsert": msg}
	opts := options.Update().SetUpsert(true)
	_, err := mongoCollection.UpdateOne(context.Background(), filter, update, opts)
	return err
}

// getAllMessagesMongo fetches all messages from MongoDB
func getAllMessagesMongo() ([]models.Message, error) {
	cur, err := mongoCollection.Find(context.Background(), map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(context.Background())
	var messages []models.Message
	for cur.Next(context.Background()) {
		var msg models.Message
		if err := cur.Decode(&msg); err != nil {
			return nil, err
		}
		messages = append(messages, msg)
	}
	return messages, nil
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
	if err := mongoClient.Ping(ctx, readpref.Primary()); err != nil {
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
func ensureMongoIndexes(ctx context.Context, coll *mongo.Collection) error {
	// Unique index on message_id
	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "message_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("uniq_message_id"),
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: 1}},
			Options: options.Index().SetName("idx_timestamp"),
		},
	})
	return err
}
