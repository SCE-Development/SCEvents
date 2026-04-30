package db

import (
	"context"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/mongo"
)

type RedisStore interface {
	SetEventHeadcount(ctx context.Context, eventID string, capacity int) error
	GetEventHeadcount(ctx context.Context, eventID string) (int, error)
	DeleteEventHeadcount(ctx context.Context, eventID string) error
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
	GetVisibleEvents(ctx context.Context, viewer models.EventViewer, startDate, endDate string) ([]models.Event, error)
	GetVisibleEventByID(ctx context.Context, viewer models.EventViewer, id string) (*models.Event, error)

	GetEventByID(ctx context.Context, id string) (*models.Event, error)
	CreateEvent(ctx context.Context, e models.Event) (*models.Event, error)
	DeleteEventByID(ctx context.Context, id string) error
	UpdateEventByID(ctx context.Context, id string, fields map[string]interface{}) error
	PublishDueEvents(ctx context.Context, now time.Time) (int64, error)

	CreatePendingRegistration(ctx context.Context, r models.RegistrationRequest) (*models.RegistrationRequest, error)
	GetRegistrationByID(ctx context.Context, requestID string) (*models.RegistrationRequest, error)
	ListRegistrationsByEventID(ctx context.Context, eventID string, limit, offset int64) ([]models.RegistrationRequest, error)
	GetRegistrationByEventAndRequestID(ctx context.Context, eventID, requestID string) (*models.RegistrationRequest, error)
	CountRegistrationsByStatusForEvent(ctx context.Context, eventID string) (map[models.Status]int64, error)
	CountAcceptedRegistrationsForEvent(ctx context.Context, eventID string) (int64, error)
	HasAcceptedRegistration(ctx context.Context, eventID, userID string) (bool, error)
	HasPendingOrAcceptedRegistration(ctx context.Context, eventID, userID string) (bool, error)
	MarkRegistrationAccepted(ctx context.Context, requestID string) error
	MarkRegistrationRejected(ctx context.Context, requestID string, reason models.DecisionReason) error
	GetRegistrationStatusesForUser(ctx context.Context, userID string, eventIDs []string) (map[string]models.Status, error)
	HasWaitlistEntry(ctx context.Context, eventID, userID string) (bool, error)
	CountWaitlistEntries(ctx context.Context, eventID string) (int64, error)
	CreateWaitlistEntry(ctx context.Context, entry models.WaitlistEntry) error
	GetWaitlistedEventIDsForUser(ctx context.Context, userID string, eventIDs []string) (map[string]bool, error)
}

type Stores struct {
	Redis RedisStore
	Mongo MongoStore
	Kafka KafkaProducer
}

type mongoStore struct {
	events        *mongo.Collection
	registrations *mongo.Collection
	waitlists     *mongo.Collection
}

// Batch user-event lookups used to enrich event responses with per-user registration state
func NewMongoStore(events, registrations, waitlists *mongo.Collection) MongoStore {
	return &mongoStore{
		events:        events,
		registrations: registrations,
		waitlists:     waitlists,
	}
}
