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
	roomsColl    *mongo.Collection
)

// Init connects to MongoDB, pings, ensures indexes and prepares collections.
func Init(ctx context.Context) error {
	var err error
	// Retry connect/ping for up to ~15s to handle startup races
	for i := 0; i < 5; i++ {
		client, err = mongo.Connect(ctx, options.Client().ApplyURI(config.MongoURI))
		if err == nil {
			if pingErr := client.Ping(ctx, readpref.Primary()); pingErr == nil {
				break
			} else {
				err = fmt.Errorf("mongo ping: %w", pingErr)
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("mongo init canceled: %w", ctx.Err())
		case <-time.After(3 * time.Second):
		}
	}
	if err != nil {
		return err
	}
	messagesColl = client.Database("chatapp").Collection("messages")
	roomsColl = client.Database("chatapp").Collection("rooms")
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

// GetMessagesByChatID returns messages for a specific chat id.
func GetMessagesByChatID(ctx context.Context, chatID int) ([]models.Message, error) {
	if messagesColl == nil {
		return nil, fmt.Errorf("messages collection not initialized")
	}
	cur, err := messagesColl.Find(ctx, bson.M{"chat_id": chatID})
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
		{Keys: bson.D{{Key: "chat_id", Value: 1}}, Options: options.Index().SetName("idx_chat_id")},
	})
	if err != nil {
		return err
	}
	if roomsColl == nil {
		return fmt.Errorf("rooms collection not initialized")
	}
	_, err = roomsColl.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "id", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_room_id")},
		{Keys: bson.D{{Key: "name", Value: 1}}, Options: options.Index().SetUnique(true).SetName("uniq_room_name")},
	})
	return err
}

// Helper for TTL or future retention policies (placeholder).
func PruneOldMessages(ctx context.Context, olderThan time.Duration) error { return nil }

// Rooms APIs
func ListRooms(ctx context.Context) ([]models.ChatRoom, error) {
	if roomsColl == nil {
		return nil, fmt.Errorf("rooms collection not initialized")
	}
	cur, err := roomsColl.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var out []models.ChatRoom
	for cur.Next(ctx) {
		var r models.ChatRoom
		if err := cur.Decode(&r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func CreateRoom(ctx context.Context, r models.ChatRoom) error {
	if roomsColl == nil {
		return fmt.Errorf("rooms collection not initialized")
	}
	_, err := roomsColl.InsertOne(ctx, r)
	return err
}

// DeleteMessagesMissingChatID removes documents without chat_id.
func DeleteMessagesMissingChatID(ctx context.Context) (int64, error) {
	if messagesColl == nil {
		return 0, fmt.Errorf("messages collection not initialized")
	}
	res, err := messagesColl.DeleteMany(ctx, bson.M{"chat_id": bson.M{"$exists": false}})
	if err != nil {
		return 0, err
	}
	// Also handle explicit null if any
	res2, err := messagesColl.DeleteMany(ctx, bson.M{"chat_id": nil})
	if err != nil {
		return res.DeletedCount, err
	}
	return res.DeletedCount + res2.DeletedCount, nil
}
