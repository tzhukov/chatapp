package kafka

import (
	"context"
	"src/models"
)

// ProducerAdapter implements api.Producer using existing Writer logic.
type ProducerAdapter struct{}

func (ProducerAdapter) Publish(ctx context.Context, msg models.Message) error {
	return Writer(ctx, msg)
}
