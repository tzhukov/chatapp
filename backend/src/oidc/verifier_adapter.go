package oidcutil

import (
	"context"
)

// VerifierAdapter wraps the existing VerifyToken for injection.
type VerifierAdapter struct{ V TokenVerifier }

// TokenVerifier interface for adapter (subset of core verifier usage).
type TokenVerifier interface {
	Verify(ctx context.Context, rawIDToken string) (*IDToken, error)
}

// Provided by coreos; we alias minimal to avoid circular import.
type IDToken interface{}

// Simple adapter implementing api.TokenVerifier using existing logic.
type Adapter struct{ Verifier *Verifier }

// Verifier is a thin wrapper around underlying id token verifier to match method signature.
type Verifier struct {
	Fn func(ctx context.Context, raw string) error
}

func (v *Verifier) Verify(ctx context.Context, raw string) error { return v.Fn(ctx, raw) }
