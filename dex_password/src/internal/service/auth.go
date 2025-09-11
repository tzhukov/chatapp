package service

import (
	"context"
	"dex_password/internal/crypto"
	"dex_password/internal/repo"
	"errors"
	"strings"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type UserRepo interface {
	GetUserByEmail(ctx context.Context, email string) (*repo.User, error)
}

type AuthService struct{ R UserRepo }

func NewAuthService(r UserRepo) *AuthService { return &AuthService{R: r} }

// Authenticate verifies email+password and returns user id on success.
func (s *AuthService) Authenticate(ctx context.Context, email, password string) (int64, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return 0, ErrInvalidCredentials
	}
	u, err := s.R.GetUserByEmail(ctx, email)
	if err != nil {
		return 0, ErrInvalidCredentials
	}
	if !crypto.Compare(u.Hash, password) {
		return 0, ErrInvalidCredentials
	}
	return u.ID, nil
}
