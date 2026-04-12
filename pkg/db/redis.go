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

func EventRegistrantsKey(eventID string) string {
	return fmt.Sprintf("event:%s:registrants", eventID)
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

var takeSeatScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current then
	return -2
end

current = tonumber(current)
if current <= 0 then
	return 0
end

redis.call("DECR", KEYS[1])
return 1
`)

// TryTakeEventSeat atomically checks whether the event has remaining capacity
// and decrements it by 1 only if capacity is available.
func TryTakeEventSeat(eventID string) (bool, error) {
	if redisClient == nil {
		return false, fmt.Errorf("redis not connected")
	}

	ctx := context.Background()
	key := EventHeadcountKey(eventID)

	result, err := takeSeatScript.Run(ctx, redisClient, []string{key}).Int()
	if err != nil {
		return false, err
	}

	switch result {
	case 1:
		return true, nil
	case 0:
		return false, nil
	case -2:
		return false, fmt.Errorf("missing Redis headcount for event %s", eventID)
	default:
		return false, fmt.Errorf("unexpected Redis script result: %d", result)
	}
}

// ReleaseEventSeat adds 1 back to the remaining capacity.
func ReleaseEventSeat(eventID string) error {
	if redisClient == nil {
		return fmt.Errorf("redis not connected")
	}

	ctx := context.Background()
	key := EventHeadcountKey(eventID)

	return redisClient.Incr(ctx, key).Err()
}

func IsUserRegisteredForEvent(eventID string, userID string) (bool, error) {
	if redisClient == nil {
		return false, fmt.Errorf("redis not connected")
	}
	ctx := context.Background()
	key := EventRegistrantsKey(eventID)
	isMember, err := redisClient.SIsMember(ctx, key, userID).Result()
	if err != nil {
		return false, err
	}
	return isMember, nil
}
