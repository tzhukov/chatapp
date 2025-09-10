package models

import (
	"testing"
	"time"
)

func TestMessageStruct(t *testing.T) {
	msg := Message{
		MessageID: "123",
		UserID:    "user1",
		Content:   "Hello, world!",
		Timestamp: time.Now(),
	}

	if msg.MessageID != "123" {
		t.Errorf("Expected MessageID '123', got '%s'", msg.MessageID)
	}
	if msg.UserID != "user1" {
		t.Errorf("Expected UserID 'user1', got '%s'", msg.UserID)
	}
	if msg.Content != "Hello, world!" {
		t.Errorf("Expected Content 'Hello, world!', got '%s'", msg.Content)
	}
	if msg.Timestamp.IsZero() {
		t.Errorf("Expected non-zero Timestamp")
	}
}
