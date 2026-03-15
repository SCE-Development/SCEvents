package db

import (
	"context"
	"os"
	"time"
	"github.com/go-redis/redis/v9"
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
		return fmt.Errorf("unable to reach redis at %s: %w", addr, err)
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


