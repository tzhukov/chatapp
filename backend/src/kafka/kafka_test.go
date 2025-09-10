package kafka

import (
	"context"
	"src/models"
	"testing"
)

func TestWriterAndReader(t *testing.T) {
	msg := models.Message{
		MessageID: "test-id",
		UserID:    "user1",
		Content:   "test message",
	}

	// Writer: context canceled immediately to force fast exit
	t.Run("Writer", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_ = Writer(ctx, msg) // Should not panic; error acceptable
	})

	// Reader: use already-canceled context so it returns immediately
	t.Run("ReaderImmediateReturn", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		ch := make(chan models.Message)
		Reader(ctx, ch) // Should return promptly
	})

	// Reader with short timeout context (may attempt dial then exit)
	t.Run("ReaderTimeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10e6) // 10ms
		defer cancel()
		ch := make(chan models.Message)
		go Reader(ctx, ch)
		<-ctx.Done()
	})
}
