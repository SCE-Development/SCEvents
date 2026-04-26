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

// ConnectRedis initializes the global Redis client using the provided address.
func ConnectRedis(addr string) error {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: "",
		DB:       0,
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

// DisconnectRedis closes the global Redis client if it has been initialized.
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

type redisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) RedisStore {
	return &redisStore{client: client}
}

// EventHeadcountKey returns the Redis key for an event's headcount (capacity).
func EventHeadcountKey(eventID string) string {
	return fmt.Sprintf("event:%s:headcount", eventID)
}

func EventRegistrantsKey(eventID string) string {
	return fmt.Sprintf("event:%s:registrants", eventID)
}

func (s *redisStore) SetEventHeadcount(ctx context.Context, eventID string, capacity int) error {
	if s.client == nil {
		return fmt.Errorf("redis not connected")
	}
	key := EventHeadcountKey(eventID)
	return s.client.Set(ctx, key, capacity, 0).Err()
}

func (s *redisStore) GetEventHeadcount(ctx context.Context, eventID string) (int, error) {
	if s.client == nil {
		return 0, fmt.Errorf("redis not connected")
	}
	key := EventHeadcountKey(eventID)
	return s.client.Get(ctx, key).Int()
}

func (s *redisStore) DeleteEventHeadcount(ctx context.Context, eventID string) error {
	if s.client == nil {
		return fmt.Errorf("redis not connected")
	}
	key := EventHeadcountKey(eventID)
	return s.client.Del(ctx, key).Err()
}

var takeSeatScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current then
	return -2
end

current = tonumber(current)
if current < 0 then
	return 1
end

if current == 0 then
	return 0
end

redis.call("DECR", KEYS[1])
return 1
`)

var releaseSeatScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current then
	return -2
end

current = tonumber(current)
if current < 0 then
	return current
end

return redis.call("INCR", KEYS[1])
`)

func (s *redisStore) TryTakeEventSeat(ctx context.Context, eventID string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("redis not connected")
	}
	key := EventHeadcountKey(eventID)
	result, err := takeSeatScript.Run(ctx, s.client, []string{key}).Int()
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

func (s *redisStore) ReleaseEventSeat(ctx context.Context, eventID string) error {
	if s.client == nil {
		return fmt.Errorf("redis not connected")
	}
	key := EventHeadcountKey(eventID)
	return releaseSeatScript.Run(ctx, s.client, []string{key}).Err()
}

func (s *redisStore) IsUserRegisteredForEvent(ctx context.Context, eventID string, userID string) (bool, error) {
	if s.client == nil {
		return false, fmt.Errorf("redis not connected")
	}
	key := EventRegistrantsKey(eventID)
	return s.client.SIsMember(ctx, key, userID).Result()
}

func (s *redisStore) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}
