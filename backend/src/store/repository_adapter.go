package store

import (
	"context"
	"src/models"
)

// RepositoryAdapter exposes store functions as an object implementing api.Repository.
type RepositoryAdapter struct{}

func (RepositoryAdapter) InsertMessage(ctx context.Context, msg models.Message) error {
	return InsertMessage(ctx, msg)
}
func (RepositoryAdapter) GetAllMessages(ctx context.Context) ([]models.Message, error) {
	return GetAllMessages(ctx)
}
func (RepositoryAdapter) GetMessagesByChatID(ctx context.Context, id int) ([]models.Message, error) {
	return GetMessagesByChatID(ctx, id)
}
func (RepositoryAdapter) DeleteMessagesMissingChatID(ctx context.Context) (int64, error) {
	return DeleteMessagesMissingChatID(ctx)
}

// Rooms
func (RepositoryAdapter) ListRooms(ctx context.Context) ([]models.ChatRoom, error) {
	return ListRooms(ctx)
}
func (RepositoryAdapter) CreateRoom(ctx context.Context, r models.ChatRoom) error {
	return CreateRoom(ctx, r)
}
