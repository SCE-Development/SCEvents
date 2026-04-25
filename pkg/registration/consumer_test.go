package registration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/mocks"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/mongo"
)

func validKafkaMessage() []byte {
	msg := models.KafkaRegistrationMessage{
		RequestID: "req-1",
		EventID:   "event-1",
		UserID:    "user-1",
		CreatedAt: time.Now().UTC(),
	}
	b, _ := json.Marshal(msg)
	return b
}

func newTestConsumer(mongoStore db.MongoStore, redisStore db.RedisStore) *Consumer {
	return &Consumer{
		stores: &db.Stores{
			Mongo: mongoStore,
			Redis: redisStore,
		},
	}
}

func TestProcessKafkaMessage_InvalidJSON(t *testing.T) {
	c := newTestConsumer(&mocks.MockMongoStore{}, &mocks.MockRedisStore{})
	err := c.processKafkaMessage(context.Background(), []byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestProcessKafkaMessage_MissingFields(t *testing.T) {
	c := newTestConsumer(&mocks.MockMongoStore{}, &mocks.MockRedisStore{})
	msg, _ := json.Marshal(models.KafkaRegistrationMessage{RequestID: "req-1"})
	err := c.processKafkaMessage(context.Background(), msg)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProcessKafkaMessage_RegistrationNotFound(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		RegistrationErr: mongo.ErrNoDocuments,
	}
	c := newTestConsumer(mongoMock, &mocks.MockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected ErrNoDocuments, got %v", err)
	}
}

func TestProcessKafkaMessage_AlreadyProcessed(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		Registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusAccepted,
		},
	}
	c := newTestConsumer(mongoMock, &mocks.MockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (no-op) for already processed, got %v", err)
	}
}

func TestProcessKafkaMessage_EventNotFound(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		Registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		EventErr: mongo.ErrNoDocuments,
	}
	c := newTestConsumer(mongoMock, &mocks.MockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (rejected), got %v", err)
	}
	if mongoMock.RejectedReason != models.ReasonEventNotFound {
		t.Fatalf("expected reason %s, got %s", models.ReasonEventNotFound, mongoMock.RejectedReason)
	}
}

func TestProcessKafkaMessage_DuplicateUser(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		Registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		Event:       &models.Event{ID: "event-1"},
		HasAccepted: true,
	}
	c := newTestConsumer(mongoMock, &mocks.MockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (rejected), got %v", err)
	}
	if mongoMock.RejectedReason != models.ReasonDuplicateUser {
		t.Fatalf("expected reason %s, got %s", models.ReasonDuplicateUser, mongoMock.RejectedReason)
	}
}

func TestProcessKafkaMessage_CapacityFull(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		Registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		Event: &models.Event{ID: "event-1"},
	}
	redisMock := &mocks.MockRedisStore{SeatAvailable: false}
	c := newTestConsumer(mongoMock, redisMock)
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (rejected), got %v", err)
	}
	if mongoMock.RejectedReason != models.ReasonCapacityFull {
		t.Fatalf("expected reason %s, got %s", models.ReasonCapacityFull, mongoMock.RejectedReason)
	}
}

func TestProcessKafkaMessage_HappyPath(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		Registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		Event: &models.Event{ID: "event-1"},
	}
	redisMock := &mocks.MockRedisStore{SeatAvailable: true}
	c := newTestConsumer(mongoMock, redisMock)
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessKafkaMessage_AcceptFailsReleasesSeat(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		Registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		Event:           &models.Event{ID: "event-1"},
		MarkAcceptedErr: errors.New("mongo write failure"),
	}
	redisMock := &mocks.MockRedisStore{SeatAvailable: true}
	c := newTestConsumer(mongoMock, redisMock)
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err == nil {
		t.Fatal("expected error when accept fails")
	}
	if !redisMock.ReleaseCalled {
		t.Fatal("expected ReleaseEventSeat to be called when accept fails")
	}
}

func TestProcessKafkaMessage_EventClosed(t *testing.T) {
	mongoMock := &mocks.MockMongoStore{
		Registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		Event: &models.Event{
			ID:     "event-1",
			Status: models.StatusClosed,
		},
	}

	c := newTestConsumer(mongoMock, &mocks.MockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (rejected), got %v", err)
	}
	if mongoMock.RejectedReason != models.ReasonEventClosed {
		t.Fatalf("expected reason %s, got %s", models.ReasonEventClosed, mongoMock.RejectedReason)
	}
}
