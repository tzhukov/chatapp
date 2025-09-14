package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"src/config"
	"src/logger"
	"src/models"

	"github.com/segmentio/kafka-go"
)

var (
	writerOnce   sync.Once
	sharedWriter *kafka.Writer
)

func getWriter(topic string) *kafka.Writer {
	writerOnce.Do(func() {
		sharedWriter = &kafka.Writer{
			Addr:     kafka.TCP(config.KafkaBroker),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
	})
	if sharedWriter.Topic != topic { // need a separate writer for different topic
		return &kafka.Writer{
			Addr:     kafka.TCP(config.KafkaBroker),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
		}
	}
	return sharedWriter
}

// Writer publishes a single message with retry using the provided context.
func Writer(ctx context.Context, msg models.Message) error {
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	w := getWriter(config.Topic)
	maxAttempts := 5
	baseDelay := 50 * time.Millisecond
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		perAttemptCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		lastErr = w.WriteMessages(perAttemptCtx, kafka.Message{Key: []byte(msg.MessageID), Value: msgBytes})
		cancel()
		if lastErr == nil {
			logger.Info("kafka write success", logger.FieldKV("message_id", msg.MessageID), logger.FieldKV("attempt", attempt))
			break
		}
		logger.Error("kafka write failure", lastErr, logger.FieldKV("attempt", attempt), logger.FieldKV("message_id", msg.MessageID))
		time.Sleep(baseDelay * time.Duration(attempt*attempt))
	}
	if lastErr != nil {
		return errors.New("kafka write failed after retries: " + lastErr.Error())
	}
	return nil
}

// DLQWriter sends a failed message + reason to dead-letter topic.
func DLQWriter(ctx context.Context, msg models.Message, reason string) error {
	payload := struct {
		Msg      models.Message `json:"message"`
		Reason   string         `json:"reason"`
		FailedAt time.Time      `json:"failed_at"`
	}{Msg: msg, Reason: reason, FailedAt: time.Now().UTC()}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	w := getWriter(config.DLQTopic)
	writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := w.WriteMessages(writeCtx, kafka.Message{Key: []byte(msg.MessageID), Value: b}); err != nil {
		logger.Error("dlq write failure", err, logger.FieldKV("message_id", msg.MessageID))
		return err
	}
	logger.Info("dlq write success", logger.FieldKV("message_id", msg.MessageID), logger.FieldKV("reason", reason))
	return nil
}

// Reader consumes messages until context cancellation.
func Reader(ctx context.Context, broadcast chan<- models.Message) {
	logger.Info("starting kafka reader", logger.FieldKV("topic", config.Topic))
	// Outer loop recreates the reader on terminal/transient errors like EOF or broker resets
	for {
		// Use a consumer group to subscribe to all partitions.
		// Start from the end to avoid replaying historical messages on (re)start.
		r := kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{config.KafkaBroker},
			Topic:       config.Topic,
			GroupID:     "chatapp-backend",
			StartOffset: kafka.LastOffset,
			MinBytes:    1e3,
			MaxBytes:    10e6,
		})
		// Inner loop reads until an error worth restarting occurs
		consecutiveErrs := 0
		for {
			m, err := r.ReadMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					_ = r.Close()
					logger.Info("kafka reader context canceled")
					return
				}
				// Treat EOF as an expected condition during rebalance/broker changes
				if errors.Is(err, io.EOF) || err.Error() == "EOF" {
					logger.Info("kafka reader EOF; recreating reader")
					break
				}
				consecutiveErrs++
				if consecutiveErrs >= 10 {
					logger.Error("kafka read errors threshold reached; recreating reader", err, logger.FieldKV("consecutive", consecutiveErrs))
					break
				}
				logger.Error("kafka transient read error", err, logger.FieldKV("consecutive", consecutiveErrs))
				time.Sleep(500 * time.Millisecond)
				continue
			}
			consecutiveErrs = 0
			logger.Debug("kafka message read", logger.FieldKV("offset", m.Offset), logger.FieldKV("partition", m.Partition))
			var msg models.Message
			if err := json.Unmarshal(m.Value, &msg); err != nil {
				logger.Error("kafka message unmarshal error", err)
				continue
			}
			broadcast <- msg
		}
		if err := r.Close(); err != nil {
			logger.Error("kafka reader close error", err)
		}
		// Small backoff before recreating reader
		select {
		case <-ctx.Done():
			logger.Info("kafka reader exiting on context cancel")
			return
		case <-time.After(1 * time.Second):
		}
	}
}
