package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"dex_password/internal/crypto"
	"dex_password/internal/repo"

	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

// TODO: implement Dex PasswordConnector gRPC service.

func main() {
	addr := getEnv("LISTEN_ADDR", ":5557")
	httpAddr := getEnv("LISTEN_HTTP", ":8080")
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Println("warning: POSTGRES_DSN not set; connector will fail auth until configured")
	}
	// Initialize database if DSN provided
	var r *repo.Postgres
	if dsn != "" {
		var err error
		r, err = repo.NewPostgres(dsn)
		if err != nil {
			log.Fatalf("db connect: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := r.Migrate(ctx); err != nil {
			log.Fatalf("db migrate: %v", err)
		}
		// Optional seed
		if email := os.Getenv("SEED_USER_EMAIL"); email != "" {
			pass := os.Getenv("SEED_USER_PASSWORD")
			if pass == "" {
				pass = "s3cret"
			}
			h, err := crypto.Hash(pass)
			if err != nil {
				log.Fatalf("seed hash: %v", err)
			}
			if err := r.CreateUser(ctx, email, h); err != nil {
				log.Printf("seed user: %v", err)
			} else {
				log.Printf("seeded user %s", email)
			}
		}
		defer r.Close()
	}

	// Start gRPC in background (placeholder)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	gs := grpc.NewServer()
	go func() {
		log.Printf("auth-connector gRPC listening on %s", addr)
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("grpc serve: %v", err)
		}
	}()

	// Simple HTTP auth API for dev and potential authproxy use
	s := newHTTPServer(r)
	log.Printf("auth-connector HTTP listening on %s", httpAddr)
	log.Fatal(http.ListenAndServe(httpAddr, s))
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// HTTP server and handlers
type httpServer struct {
	r  *repo.Postgres
	mu sync.RWMutex
	// naive in-memory sessions; dev only
	sessions map[string]string // sid -> email
}

func newHTTPServer(r *repo.Postgres) http.Handler {
	s := &httpServer{r: r, sessions: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/register", s.register)
	mux.HandleFunc("/login", s.login)
	mux.HandleFunc("/auth", s.auth)
	return mux
}

func (s *httpServer) health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *httpServer) register(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body><h2>Register</h2><form method="post"><input type="email" name="email" placeholder="email" required><br><input type="password" name="password" placeholder="password" required><br><button type="submit">Register</button></form></body></html>`))
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		email := r.FormValue("email")
		pass := r.FormValue("password")
		if email == "" || pass == "" || s.r == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		h, err := crypto.Hash(pass)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := s.r.CreateUser(ctx, email, h); err != nil {
			w.WriteHeader(http.StatusConflict)
			return
		}
		w.WriteHeader(http.StatusCreated)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (s *httpServer) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		rd := r.URL.Query().Get("rd")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!doctype html><html><body><h2>Login</h2><form method="post">` +
			`<input type="hidden" name="rd" value="` + htmlEscape(rd) + `">` +
			`<input type="email" name="email" placeholder="email" required><br>` +
			`<input type="password" name="password" placeholder="password" required><br>` +
			`<button type="submit">Login</button></form>` +
			`<p><a href="/register">Register</a></p>` +
			`</body></html>`))
		return
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		email := r.FormValue("email")
		pass := r.FormValue("password")
		rd := r.FormValue("rd")
		if email == "" || pass == "" || s.r == nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		u, err := s.r.GetUserByEmail(ctx, email)
		if err != nil || !crypto.Compare(u.Hash, pass) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		sid := randomID(16)
		s.mu.Lock()
		s.sessions[sid] = email
		s.mu.Unlock()
		http.SetCookie(w, &http.Cookie{Name: "sid", Value: sid, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode})
		if rd != "" {
			http.Redirect(w, r, rd, http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (s *httpServer) auth(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie("sid")
	if c == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	s.mu.RLock()
	email := s.sessions[c.Value]
	s.mu.RUnlock()
	if email == "" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// Set headers that an auth proxy might forward to Dex
	w.Header().Set("X-Remote-User", email)
	w.Header().Set("X-Remote-Email", email)
	w.WriteHeader(http.StatusOK)
}

func randomID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", "\"", "&quot;")
	return r.Replace(s)
}
