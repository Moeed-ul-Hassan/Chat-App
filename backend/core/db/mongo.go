package db

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	Client   *mongo.Client
	Database *mongo.Database
)

func Connect() error {
	uri := os.Getenv("MONGODB_URI")
	if uri == "" {
		host := getEnv("MONGO_HOST", "localhost")
		port := getEnv("MONGO_PORT", "27017")
		uri = fmt.Sprintf("mongodb://%s:%s", host, port)
	}

	dbName := getEnv("DB_NAME", "Echo")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("connect mongodb: %w", err)
	}

	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("ping mongodb: %w", err)
	}

	Client = client
	Database = client.Database(dbName)

	fmt.Printf("connected mongodb (%s)\n", dbName)
	return nil
}

func CreateIndexes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	users := Database.Collection("users")
	_, err := users.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "username", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "username_lower", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"username_lower": bson.M{"$exists": true}}),
		},
		{
			Keys: bson.D{{Key: "clerk_id", Value: 1}},
			Options: options.Index().
				SetUnique(true).
				SetPartialFilterExpression(bson.M{"clerk_id": bson.M{"$exists": true}}),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	if err != nil {
		return fmt.Errorf("create user indexes: %w", err)
	}

	rooms := Database.Collection("rooms")
	_, err = rooms.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "room_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "owner_id", Value: 1}},
		},
	})
	if err != nil {
		return fmt.Errorf("create room indexes: %w", err)
	}

	messages := Database.Collection("messages")
	_, err = messages.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "room_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "channel_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "conversation_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
	})
	if err != nil {
		return fmt.Errorf("create message indexes: %w", err)
	}

	sessions := Database.Collection("sessions")
	_, err = sessions.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "token_hash", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "user_id", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	if err != nil {
		return fmt.Errorf("create session indexes: %w", err)
	}

	otps := Database.Collection("otp_codes")
	_, err = otps.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "email", Value: 1}, {Key: "code", Value: 1}},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	if err != nil {
		return fmt.Errorf("create otp indexes: %w", err)
	}
	convs := Database.Collection("conversations")
	_, err = convs.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "participants", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "updated_at", Value: -1}},
		},
	})
	if err != nil {
		return fmt.Errorf("create conversation indexes: %w", err)
	}

	stories := Database.Collection("stories")
	_, err = stories.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
	})
	if err != nil {
		return fmt.Errorf("create story indexes: %w", err)
	}

	fmt.Println("indexes ready")
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
