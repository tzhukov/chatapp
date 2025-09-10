package store

import (
	"context"
	"fmt"
	"time"

	"src/config"
	"src/logger"
	"src/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	client       *mongo.Client
	messagesColl *mongo.Collection
)

// Init connects to MongoDB, pings, ensures indexes and prepares collections.
func Init(ctx context.Context) error {
	var err error
	client, err = mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
	if err != nil {
		return fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return fmt.Errorf("mongo ping: %w", err)
	}
	messagesColl = client.Database("chatapp").Collection("messages")
	if err := ensureIndexes(ctx); err != nil {
		return fmt.Errorf("ensure indexes: %w", err)
	}
	logger.Info("mongo initialized", logger.FieldKV("uri", config.MongoURI))
	return nil
}

// Close disconnects the client.
func Close(ctx context.Context) error {
	if client == nil {
		return nil
	}
	return client.Disconnect(ctx)
}

// Ping health check.
func Ping(ctx context.Context) error {
	if client == nil {
		return fmt.Errorf("mongo client not initialized")
	}
	return client.Ping(ctx, readpref.Primary())
}

// InsertMessage performs idempotent insert (upsert ignoring duplicates).
func InsertMessage(ctx context.Context, msg models.Message) error {
	if messagesColl == nil {
		return fmt.Errorf("messages collection not initialized")
	}
	filter := bson.M{"message_id": msg.MessageID}
	update := bson.M{"$setOnInsert": msg}
	opts := options.Update().SetUpsert(true)
	_, err := messagesColl.UpdateOne(ctx, filter, update, opts)
	return err
}

// GetAllMessages returns all stored messages.
func GetAllMessages(ctx context.Context) ([]models.Message, error) {
	if messagesColl == nil {
		return nil, fmt.Errorf("messages collection not initialized")
	}
	cur, err := messagesColl.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.Message
	for cur.Next(ctx) {
		var m models.Message
		if err := cur.Decode(&m); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

func ensureIndexes(ctx context.Context) error {
	if messagesColl == nil {
		return fmt.Errorf("messages collection not initialized")
	}
	_, err := messagesColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "message_id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_message_id")},
		{Keys: bson.D{{Key: "timestamp", Value: 1}}, Options: options.Index().SetName("idx_timestamp")},
	})
	return err
}

// Helper for TTL or future retention policies (placeholder).
func PruneOldMessages(ctx context.Context, olderThan time.Duration) error { return nil }
