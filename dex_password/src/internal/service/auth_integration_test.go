package service

import (
	"context"
	"dex_password/internal/crypto"
	"dex_password/internal/repo"
	"os"
	"testing"
	"time"
)

func TestAuthService_WithPostgres(t *testing.T) {
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
	h, err := crypto.Hash(pass)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := r.CreateUser(ctx, email, h); err != nil {
		t.Fatalf("seed: %v", err)
	}

	svc := NewAuthService(r)
	id, err := svc.Authenticate(ctx, email, pass)
	if err != nil || id == 0 {
		t.Fatalf("auth ok expected, got id=%d err=%v", id, err)
	}

	if _, err := svc.Authenticate(ctx, email, "wrong"); err == nil {
		t.Fatalf("expected invalid creds for wrong password")
	}
}
