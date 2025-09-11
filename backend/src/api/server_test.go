package api

import (
	"context"
	"net/http/httptest"
	"src/models"
	"testing"
)

type mockProducer struct{ called bool }

func (m *mockProducer) Publish(ctx context.Context, msg models.Message) error {
	m.called = true
	return nil
}

type mockRepo struct{}

func (m *mockRepo) InsertMessage(ctx context.Context, msg models.Message) error { return nil }
func (m *mockRepo) GetAllMessages(ctx context.Context) ([]models.Message, error) {
	return []models.Message{}, nil
}

type mockVerifier struct{ deny bool }

func (v *mockVerifier) Verify(ctx context.Context, raw string) error {
	if v.deny || raw == "" {
		return context.Canceled
	}
	return nil
}

func TestUnauthorized(t *testing.T) {
	srv := NewServer(&mockProducer{}, &mockRepo{}, &mockVerifier{deny: true}, nil, make(chan models.Message), 100)
	r := httptest.NewRequest("GET", "/messages", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, r)
	if w.Result().StatusCode != 401 {
		t.Fatalf("expected 401 got %d", w.Result().StatusCode)
	}
}
