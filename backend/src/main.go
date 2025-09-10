package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan Message)
var messagesStore = make([]Message, 0)

import "os"

var (
	kafkaBroker = getEnv("KAFKA_BROKER", "kafka:9092")
	topic       = getEnv("KAFKA_TOPIC", "chat-messages")
	dexIssuer   = getEnv("DEX_ISSUER_URL", "http://dex:5556/dex")
	clientID    = getEnv("DEX_CLIENT_ID", "backend")
	audience    = getEnv("DEX_AUDIENCE", "backend")
	apiPort     = getEnv("API_PORT", "8080")
)

type Message struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

var verifier *oidc.IDTokenVerifier

func main() {
	log.Println("Starting application...")

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, dexIssuer)
	if err != nil {
		log.Fatalf("Failed to create OIDC provider: %v", err)
	}

	verifier = provider.Verifier(&oidc.Config{ClientID: clientID})

	go kafkaReader()

	http.HandleFunc("/ws", handleConnections)
	http.Handle("/messages", authMiddleware(http.HandlerFunc(handleMessages)))

	go handleMessagesBroadcasting()

	log.Printf("http server started on :%s", apiPort)
	err = http.ListenAndServe(":"+apiPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
// getEnv returns the value of the environment variable or a default value
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
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
			log.Printf("Token verification failed: %v", err)
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
		log.Printf("Token verification failed: %v", err)
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
	log.Printf("client connected: %s", ws.RemoteAddr())

	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error reading json from %s: %v", ws.RemoteAddr(), err)
			delete(clients, ws)
			log.Printf("client disconnected: %s", ws.RemoteAddr())
			break
		}
		log.Printf("received message from %s: %s", ws.RemoteAddr(), msg.Content)
		broadcast <- msg
	}
}

func handleMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var msg Message
		err := json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			log.Printf("error decoding message body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		msg.Timestamp = time.Now()
		log.Printf("received message via POST: %s", msg.Content)

		kafkaWriter(msg)
		messagesStore = append(messagesStore, msg)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(msg)
	} else if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messagesStore)
	} else {
		log.Printf("received invalid method for /messages: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handleMessagesBroadcasting() {
	log.Println("starting message broadcasting loop")
	for {
		msg := <-broadcast
		log.Printf("broadcasting message to %d clients", len(clients))
		for client := range clients {
			err := client.WriteJSON(msg)
			if err != nil {
				log.Printf("error writing json to client %s: %v", client.RemoteAddr(), err)
				client.Close()
				delete(clients, client)
				log.Printf("client disconnected due to error: %s", client.RemoteAddr())
			}
		}
	}
}

func kafkaWriter(msg Message) {
	log.Printf("writing message to kafka topic %s", topic)
	w := &kafka.Writer{
		Addr:     kafka.TCP(kafkaBroker),
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("error marshalling message: %v", err)
		return
	}

	err = w.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(msg.MessageID),
			Value: msgBytes,
		},
	)
	if err != nil {
		log.Printf("failed to write messages to kafka: %v", err)
	} else {
		log.Printf("successfully wrote message to kafka: %s", msg.MessageID)
	}

	if err := w.Close(); err != nil {
		log.Printf("failed to close kafka writer: %v", err)
	}
}

func kafkaReader() {
	log.Printf("starting kafka reader on topic %s", topic)
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   []string{kafkaBroker},
		Topic:     topic,
		Partition: 0,
		MinBytes:  10e3, // 10KB
		MaxBytes:  10e6, // 10MB
	})

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("error reading message from kafka: %v", err)
			break
		}
		log.Printf("message read from kafka partition %d at offset %d", m.Partition, m.Offset)

		var msg Message
		err = json.Unmarshal(m.Value, &msg)
		if err != nil {
			log.Printf("error unmarshalling message from kafka: %v", err)
			continue
		}

		broadcast <- msg
	}

	if err := r.Close(); err != nil {
		log.Printf("failed to close kafka reader: %v", err)
	}
}
