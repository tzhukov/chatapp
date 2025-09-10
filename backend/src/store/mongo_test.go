package store

import (
	"context"
	"testing"

	"src/models"
)

// NOTE: These tests are lightweight structural tests; full integration would require a running MongoDB.
// They focus on error paths prior to real DB interactions or after Init guard conditions.

func TestPingWithoutInit(t *testing.T) {
	ctx := context.Background()
	if err := Ping(ctx); err == nil {
		// Expect error because client not initialized
		t.Fatalf("expected error when ping before Init")
	}
}

func TestInsertMessageWithoutInit(t *testing.T) {
	ctx := context.Background()
	if err := InsertMessage(ctx, dummyMessage()); err == nil {
		t.Fatalf("expected error when inserting before Init")
	}
}

func TestGetAllMessagesWithoutInit(t *testing.T) {
	ctx := context.Background()
	if _, err := GetAllMessages(ctx); err == nil {
		t.Fatalf("expected error when listing before Init")
	}
}

// dummyMessage creates a minimal valid message
func dummyMessage() models.Message {
	return models.Message{MessageID: "test-id", UserID: "u", Content: "c"}
}
