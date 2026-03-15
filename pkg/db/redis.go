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
	if addr == "" {
		addr = os.Getenv("REDIS_ADDR")
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: addr,
		password: "", 
		DB: 0, 
	})
	return nil
}

// Disconnect closes the global Redis client if it has been initialized.
func DisconnectRedis() error {
	if redisClient == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := redisClient.Close()
	redisClient = nil

	return err
}

// RedisClient returns the initialized Redis client.
func RedisClient() *redis.Client {
	return redisClient
}


