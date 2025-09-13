package api

import (
	"context"
	"encoding/json"
	"net/http"
	"src/logger"
	"src/metrics"
	"src/models"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Producer abstracts Kafka publishing.
type Producer interface {
	Publish(ctx context.Context, msg models.Message) error
}

// Repository abstracts message persistence & retrieval.
type Repository interface {
	InsertMessage(ctx context.Context, msg models.Message) error
	GetAllMessages(ctx context.Context) ([]models.Message, error)
}

// TokenVerifier abstracts OIDC token verification.
type TokenVerifier interface {
	Verify(ctx context.Context, raw string) error
}

type Server struct {
	mux        *http.ServeMux
	hub        *Hub
	validator  *MessageValidator
	producer   Producer
	repo       Repository
	verifier   TokenVerifier
	maxMsgLen  int
	broadcastC <-chan models.Message
}

func NewServer(p Producer, r Repository, v TokenVerifier, validator *MessageValidator, broadcast <-chan models.Message, maxLen int) *Server {
	s := &Server{mux: http.NewServeMux(), hub: NewHub(), validator: validator, producer: p, repo: r, verifier: v, maxMsgLen: maxLen, broadcastC: broadcast}
	s.routes()
	go s.broadcastLoop()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/ws", s.handleWS)
	s.mux.HandleFunc("/api/ws", s.handleWS)
	s.mux.HandleFunc("/messages", s.withAuth(s.handleMessages))
	s.mux.HandleFunc("/api/messages", s.withAuth(s.handleMessages))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

// withAuth simple bearer token extraction passed to verifier.
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || len(auth) < 8 || auth[:7] != "Bearer " {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		if err := s.verifier.Verify(r.Context(), auth[7:]); err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" || s.verifier.Verify(r.Context(), token) != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error("websocket upgrade failed", err)
		return
	}
	s.hub.Add(conn)
	metrics.IncWSConnections()
	go func() {
		defer func() { s.hub.Remove(conn); metrics.DecWSConnections() }()
		for {
			var msg models.Message
			if err := conn.ReadJSON(&msg); err != nil {
				logger.Error("ws read", err)
				return
			}
			if msg.MessageID == "" {
				msg.MessageID = uuid.NewString()
			}
			if msg.Timestamp.IsZero() {
				msg.Timestamp = time.Now().UTC()
			}
			if len(msg.Content) > s.maxMsgLen {
				continue
			}
			if s.validator != nil {
				if err := s.validator.Validate(msg); err != nil {
					continue
				}
			}
			if err := s.producer.Publish(r.Context(), msg); err != nil {
				logger.Error("publish fail", err)
				// Fallback: directly broadcast and persist so connected clients aren't blocked by Kafka
				s.hub.BroadcastExcept(msg, conn)
				if s.repo != nil {
					if perr := s.repo.InsertMessage(context.Background(), msg); perr != nil {
						logger.Error("fallback persist fail", perr)
					}
				}
				metrics.IncMsgIngested()
				continue
			}
			metrics.IncMsgIngested()
		}
	}()
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		var msg models.Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if len(msg.Content) > s.maxMsgLen {
			http.Error(w, "message too long", http.StatusBadRequest)
			return
		}
		if msg.MessageID == "" {
			msg.MessageID = uuid.NewString()
		}
		msg.Timestamp = time.Now().UTC()
		if s.validator != nil {
			if err := s.validator.Validate(msg); err != nil {
				http.Error(w, "invalid", http.StatusBadRequest)
				return
			}
		}
		if err := s.producer.Publish(r.Context(), msg); err != nil {
			// Fallback: broadcast and persist immediately if enqueue fails
			s.hub.Broadcast(msg)
			if s.repo != nil {
				_ = s.repo.InsertMessage(r.Context(), msg)
			}
			metrics.IncMsgIngested()
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(map[string]string{"message_id": msg.MessageID, "status": "broadcasted-fallback"})
			return
		}
		metrics.IncMsgIngested()
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"message_id": msg.MessageID, "status": "enqueued"})
	case http.MethodGet:
		list, err := s.repo.GetAllMessages(r.Context())
		if err != nil {
			http.Error(w, "fetch failed", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(list)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) broadcastLoop() {
	for m := range s.broadcastC {
		s.hub.Broadcast(m)
		metrics.IncMsgBroadcast()
		_ = s.repo.InsertMessage(context.Background(), m)
	}
}

// Helper to parse max length env already resolved upstream; fallback logic kept here if input <1
func ParseMaxLen(v string, fallback int) int {
	if n, err := strconv.Atoi(v); err == nil && n > 0 {
		return n
	}
	return fallback
}
