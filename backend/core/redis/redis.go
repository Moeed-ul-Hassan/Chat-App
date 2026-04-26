package redis

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var Client *goredis.Client
var Ctx = context.Background()
var ready bool

func Connect() error {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "localhost:6379"
	}
	redisURL = strings.TrimPrefix(redisURL, "redis://")

	Client = goredis.NewClient(&goredis.Options{
		Addr:     redisURL,
		Password: "", // no password set
		DB:       0,  // use default DB
		// Keep startup failures fast/quiet when Redis is optional in local dev.
		MaxRetries:      0,
		DialTimeout:     800 * time.Millisecond,
		ReadTimeout:     800 * time.Millisecond,
		WriteTimeout:    800 * time.Millisecond,
		PoolTimeout:     800 * time.Millisecond,
		MinRetryBackoff: 0,
		MaxRetryBackoff: 0,
	})

	_, err := Client.Ping(Ctx).Result()
	if err != nil {
		ready = false
		return fmt.Errorf("failed to connect to Redis at %s: %w", redisURL, err)
	}

	ready = true
	fmt.Println("🚀 Connected to Redis successfully!")
	return nil
}

// IsReady reports whether the Redis client was initialized and reachable.
func IsReady() bool {
	return ready && Client != nil
}

// Publish is a helper to send a message to a redis channel
func Publish(ctx context.Context, channel string, message interface{}) error {
	return Client.Publish(ctx, channel, message).Err()
}

// Subscribe returns a PubSub instance for a channel
func Subscribe(ctx context.Context, channel string) *goredis.PubSub {
	return Client.Subscribe(ctx, channel)
}

// SetUserPresence sets the online status of a user in Redis
func SetUserPresence(ctx context.Context, userID string, isOnline bool) error {
	key := fmt.Sprintf("presence:%s", userID)
	if isOnline {
		// Set to "online" with a 1-hour expiration (refreshed on WS activity)
		return Client.Set(ctx, key, "online", 0).Err()
	}
	// On disconnect, we can either delete or set to "offline"
	return Client.Del(ctx, key).Err()
}

// GetUserPresence checks if a user is online
func GetUserPresence(ctx context.Context, userID string) (string, error) {
	key := fmt.Sprintf("presence:%s", userID)
	val, err := Client.Get(ctx, key).Result()
	if err == goredis.Nil {
		return "offline", nil
	}
	return val, err
}
