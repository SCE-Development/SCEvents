package db

import (
	"context"
)

type RedisStore interface {
	SetEventHeadcount(ctx context.Context, eventID string, capacity int) error
	GetEventHeadcount(ctx context.Context, eventID string) (int, error)
	TryTakeEventSeat(ctx context.Context, eventID string) (bool, error)
	ReleaseEventSeat(ctx context.Context, eventID string) error
	IsUserRegisteredForEvent(ctx context.Context, eventID string, userID string) (bool, error)
	Close() error
}

type Stores struct {
	Redis RedisStore
	// Mongo MongoStore 
	// Kafka KafkaClient
}
