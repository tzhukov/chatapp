package models

import "time"

type Message struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
