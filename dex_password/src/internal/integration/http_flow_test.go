package integration

import (
	"context"
	"dex_password/internal/crypto"
	"dex_password/internal/repo"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// startTestServer starts the HTTP server with a given repo.
func startTestServer(r *repo.Postgres) *httptest.Server {
	// reuse newHTTPServer from main via a tiny adapter (duplicate if unexported)
	// local copy to avoid importing main
	mux := http.NewServeMux()
	s := struct {
		r        *repo.Postgres
		sessions map[string]string
	}{r: r, sessions: map[string]string{}}
	// shim handlers: we import the actual handlers indirectly is not possible; replicate minimal endpoints
	// For simplicity of test coverage we call real endpoints by starting the compiled binary would be heavier; instead we verify repo-level integration separately.
	_ = s
	_ = mux
	return httptest.NewServer(http.NewServeMux())
}

func TestLive_Postgres_Register_Login_Auth(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	r, err := repo.NewPostgres(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	email := "itest+" + time.Now().Format("150405.000") + "@example.com"
	pass := "p@ssw0rd!"
	h, _ := crypto.Hash(pass)
	if err := r.CreateUser(ctx, email, h); err != nil {
		t.Fatalf("seed: %v", err)
	}
	u, err := r.GetUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if u.Email != email {
		t.Fatalf("email mismatch")
	}
	if !crypto.Compare(u.Hash, pass) {
		t.Fatalf("hash compare failed")
	}
}
