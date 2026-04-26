package redis

import (
	"context"
	"encoding/json"
	"time"
)

// CacheGetJSON reads a cached JSON payload into destination.
// Returns (hit, err).
func CacheGetJSON(ctx context.Context, key string, destination any) (bool, error) {
	if !IsReady() {
		return false, nil
	}
	raw, err := Client.Get(ctx, key).Bytes()
	if err != nil {
		return false, nil
	}
	if err := json.Unmarshal(raw, destination); err != nil {
		return false, err
	}
	return true, nil
}

// CacheSetJSON stores a JSON payload with TTL.
func CacheSetJSON(ctx context.Context, key string, payload any, ttl time.Duration) error {
	if !IsReady() {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return Client.Set(ctx, key, data, ttl).Err()
}

// CacheDelete removes one or more cache keys.
func CacheDelete(ctx context.Context, keys ...string) error {
	if !IsReady() || len(keys) == 0 {
		return nil
	}
	return Client.Del(ctx, keys...).Err()
}
