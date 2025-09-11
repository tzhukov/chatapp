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
