package service

import (
	"context"
	"dex_password/internal/crypto"
	"dex_password/internal/repo"
	"testing"
)

type fakeRepo struct {
	u   *repo.User
	err error
}

func (f *fakeRepo) GetUserByEmail(ctx context.Context, email string) (*repo.User, error) {
	return f.u, f.err
}

func TestAuthService_Authenticate(t *testing.T) {
	h, err := crypto.Hash("s3cret")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewAuthService(&fakeRepo{u: &repo.User{ID: 1, Email: "a@example.com", Hash: h}})
	if _, err := svc.Authenticate(context.Background(), "a@example.com", "s3cret"); err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if _, err := svc.Authenticate(context.Background(), "a@example.com", "nope"); err == nil {
		t.Fatalf("expected error on wrong password")
	}
}
