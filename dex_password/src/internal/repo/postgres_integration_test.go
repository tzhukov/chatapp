package repo

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestPostgresRepo_Integration(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set; skipping integration test")
	}
	r, err := NewPostgres(dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer r.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := r.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := r.CreateUser(ctx, "a@example.com", "$2a$10$N0wPrE6gI4o.7u3q1k.6pOdS5h9g2rQ0m2r2jz9YF8l1kQ/8i4fEu"); err != nil {
		t.Fatalf("create: %v", err)
	}
	u, err := r.GetUserByEmail(ctx, "a@example.com")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if u.Email != "a@example.com" {
		t.Fatalf("email mismatch")
	}
}
