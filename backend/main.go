package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
	"github.com/xeipuuv/gojsonschema"

	"backend/config"
	"backend/kafka"
	"backend/models"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan models.Message)
var db *sql.DB

var verifier *oidc.IDTokenVerifier

func main() {
	log.Println("Starting application...")

	// Open SQLite database
	var err error
	db, err = sql.Open("sqlite3", "./chat.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()
	if err := createMessagesTable(); err != nil {
		log.Fatalf("Failed to create messages table: %v", err)
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, config.DexIssuer)
	if err != nil {
		log.Fatalf("Failed to create OIDC provider: %v", err)
	}

	verifier = provider.Verifier(&oidc.Config{ClientID: config.ClientID})

	go kafka.Reader(broadcast)

	http.HandleFunc("/ws", handleConnections)
	http.Handle("/messages", authMiddleware(http.HandlerFunc(handleMessages)))

	go handleMessagesBroadcasting()

	log.Printf("http server started on :%s", config.ApiPort)
	err = http.ListenAndServe(":"+config.ApiPort, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// createMessagesTable creates the messages table if it doesn't exist
func createMessagesTable() error {
	query := `CREATE TABLE IF NOT EXISTS messages (
		message_id TEXT PRIMARY KEY,
		user_id TEXT,
		content TEXT,
		timestamp DATETIME
	)`
	_, err := db.Exec(query)
	return err
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
		var msg models.Message
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
		var msg models.Message
		err := json.NewDecoder(r.Body).Decode(&msg)
		if err != nil {
			log.Printf("error decoding message body: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Validate message against schema
		if err := validateMessageSchema(msg); err != nil {
			log.Printf("message schema validation failed: %v", err)
			http.Error(w, "Invalid message schema: "+err.Error(), http.StatusBadRequest)
			return
		}

		msg.Timestamp = time.Now()
		log.Printf("received message via POST: %s", msg.Content)

		kafka.Writer(msg)
		if err := insertMessage(msg); err != nil {
			log.Printf("failed to persist message: %v", err)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(msg)
	} else if r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		messages, err := getAllMessages()
		if err != nil {
			log.Printf("failed to fetch messages: %v", err)
			http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(messages)
	} else {
		log.Printf("received invalid method for /messages: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// insertMessage persists a message to the database
func insertMessage(msg models.Message) error {
	query := `INSERT INTO messages (message_id, user_id, content, timestamp) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, msg.MessageID, msg.UserID, msg.Content, msg.Timestamp)
	return err
}

// getAllMessages fetches all messages from the database
func getAllMessages() ([]models.Message, error) {
	rows, err := db.Query("SELECT message_id, user_id, content, timestamp FROM messages ORDER BY timestamp ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		var ts string
		if err := rows.Scan(&msg.MessageID, &msg.UserID, &msg.Content, &ts); err != nil {
			return nil, err
		}
		msg.Timestamp, _ = time.Parse(time.RFC3339, ts)
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