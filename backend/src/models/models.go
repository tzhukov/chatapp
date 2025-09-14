package models

import "time"

type Message struct {
	MessageID string    `json:"message_id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	ChatID    int       `json:"chat_id,omitempty" bson:"chat_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
