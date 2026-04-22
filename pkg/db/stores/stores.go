package stores

import (
	"context"

	"github.com/SCE-Development/SCEvents/pkg/models"
)

type RedisStore interface {
	SetEventHeadcount(ctx context.Context, eventID string, capacity int) error
	GetEventHeadcount(ctx context.Context, eventID string) (int, error)
	TryTakeEventSeat(ctx context.Context, eventID string) (bool, error)
	ReleaseEventSeat(ctx context.Context, eventID string) error
	IsUserRegisteredForEvent(ctx context.Context, eventID string, userID string) (bool, error)
	Close() error
}

type KafkaProducer interface {
	PublishRegistration(ctx context.Context, requestID string, eventID string, userID string) error
	Close() error
}

type MongoStore interface {
	GetEvents(ctx context.Context, startDate, endDate string) ([]models.Event, error)
	GetEventByID(ctx context.Context, id string) (*models.Event, error)
	CreateEvent(ctx context.Context, e models.Event) (*models.Event, error)
	DeleteEventByID(ctx context.Context, id string) error
	UpdateEventByID(ctx context.Context, id string, fields map[string]interface{}) error
	CreatePendingRegistration(ctx context.Context, r models.RegistrationRequest) (*models.RegistrationRequest, error)
	GetRegistrationByID(ctx context.Context, requestID string) (*models.RegistrationRequest, error)
	HasAcceptedRegistration(ctx context.Context, eventID, userID string) (bool, error)
	HasPendingOrAcceptedRegistration(ctx context.Context, eventID, userID string) (bool, error)
	MarkRegistrationAccepted(ctx context.Context, requestID string) error
	MarkRegistrationRejected(ctx context.Context, requestID string, reason models.DecisionReason) error
}

type Stores struct {
	Redis RedisStore
	Mongo MongoStore
	Kafka KafkaProducer
}
