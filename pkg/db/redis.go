package db

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
)


// Connect initializes the global Redis client using the provided address.

func ConnectRedis(addr string) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr: addr,
		Password: "", 
		DB: 0, 
	})
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Test the connection with a ping
	if err := redisClient.Ping(ctx).Err(); err != nil {
		_ = redisClient.Close()
		redisClient = nil
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	return nil
}

// Disconnect closes the global Redis client if it has been initialized.
func DisconnectRedis() error {
	if redisClient == nil {
		return nil
	}

	err := redisClient.Close()
	redisClient = nil

	return err
}

// RedisClient returns the initialized Redis client.
func RedisClient() *redis.Client {
	return redisClient
}

// EventHeadcountKey returns the Redis key for an event's headcount (capacity).
// Pattern: event:{event_id}:headcount -> value: capacity (integer).
func EventHeadcountKey(eventID string) string {
	return fmt.Sprintf("event:%s:headcount", eventID)
}

func SetEventHeadcount(eventID string, capacity int) error {
	if redisClient == nil {
		return fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	key := EventHeadcountKey(eventID)
	return redisClient.Set(ctx, key, capacity, 0).Err()
}

// GetEventHeadcount returns the current remaining headcount for an event from Redis.
func GetEventHeadcount(eventID string) (int, error) {
	if redisClient == nil {
		return 0, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	key := EventHeadcountKey(eventID)
	val, err := redisClient.Get(ctx, key).Int()
	if err != nil {
		return 0, err
	}
	return val, nil
}
