package kafka

import (
	"src/models"
	"testing"
)

func TestWriterAndReader(t *testing.T) {
	msg := models.Message{
		MessageID: "test-id",
		UserID:    "user1",
		Content:   "test message",
	}

	// Writer and Reader require a running Kafka broker, so here we just check function signatures
	t.Run("Writer", func(t *testing.T) {
		_ = Writer(msg) // Should not panic even if broker unreachable (will return error)
	})

	t.Run("Reader", func(t *testing.T) {
		ch := make(chan models.Message)
		go func() {
			Reader(ch) // Should not panic
		}()
	})
}
