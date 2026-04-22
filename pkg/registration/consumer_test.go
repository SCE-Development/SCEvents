package registration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db/stores"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/mongo"
)

type mockMongoStore struct {
	registration     *models.RegistrationRequest
	registrationErr  error
	event            *models.Event
	eventErr         error
	hasAccepted      bool
	hasAcceptedErr   error
	markAcceptedErr  error
	markRejectedErr  error
	rejectedReason   models.DecisionReason
}

func (m *mockMongoStore) GetEvents(_ context.Context, _, _ string) ([]models.Event, error) {
	return nil, nil
}
func (m *mockMongoStore) GetEventByID(_ context.Context, _ string) (*models.Event, error) {
	return m.event, m.eventErr
}
func (m *mockMongoStore) CreateEvent(_ context.Context, e models.Event) (*models.Event, error) {
	return &e, nil
}
func (m *mockMongoStore) DeleteEventByID(_ context.Context, _ string) error {
	return nil
}
func (m *mockMongoStore) UpdateEventByID(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockMongoStore) CreatePendingRegistration(_ context.Context, r models.RegistrationRequest) (*models.RegistrationRequest, error) {
	return &r, nil
}
func (m *mockMongoStore) GetRegistrationByID(_ context.Context, _ string) (*models.RegistrationRequest, error) {
	return m.registration, m.registrationErr
}
func (m *mockMongoStore) HasAcceptedRegistration(_ context.Context, _, _ string) (bool, error) {
	return m.hasAccepted, m.hasAcceptedErr
}
func (m *mockMongoStore) HasPendingOrAcceptedRegistration(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockMongoStore) MarkRegistrationAccepted(_ context.Context, _ string) error {
	return m.markAcceptedErr
}
func (m *mockMongoStore) MarkRegistrationRejected(_ context.Context, _ string, reason models.DecisionReason) error {
	m.rejectedReason = reason
	return m.markRejectedErr
}

type mockRedisStore struct {
	seatAvailable  bool
	seatErr        error
	releaseCalled  bool
	releaseErr     error
}

func (m *mockRedisStore) SetEventHeadcount(_ context.Context, _ string, _ int) error {
	return nil
}
func (m *mockRedisStore) GetEventHeadcount(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (m *mockRedisStore) TryTakeEventSeat(_ context.Context, _ string) (bool, error) {
	return m.seatAvailable, m.seatErr
}
func (m *mockRedisStore) ReleaseEventSeat(_ context.Context, _ string) error {
	m.releaseCalled = true
	return m.releaseErr
}
func (m *mockRedisStore) IsUserRegisteredForEvent(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *mockRedisStore) Close() error {
	return nil
}

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

func newTestConsumer(mongoStore stores.MongoStore, redisStore stores.RedisStore) *Consumer {
	return &Consumer{
		stores: &stores.Stores{
			Mongo: mongoStore,
			Redis: redisStore,
		},
	}
}

func TestProcessKafkaMessage_InvalidJSON(t *testing.T) {
	c := newTestConsumer(&mockMongoStore{}, &mockRedisStore{})
	err := c.processKafkaMessage(context.Background(), []byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestProcessKafkaMessage_MissingFields(t *testing.T) {
	c := newTestConsumer(&mockMongoStore{}, &mockRedisStore{})
	msg, _ := json.Marshal(models.KafkaRegistrationMessage{RequestID: "req-1"})
	err := c.processKafkaMessage(context.Background(), msg)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestProcessKafkaMessage_RegistrationNotFound(t *testing.T) {
	mongoMock := &mockMongoStore{
		registrationErr: mongo.ErrNoDocuments,
	}
	c := newTestConsumer(mongoMock, &mockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("expected ErrNoDocuments, got %v", err)
	}
}

func TestProcessKafkaMessage_AlreadyProcessed(t *testing.T) {
	mongoMock := &mockMongoStore{
		registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusAccepted,
		},
	}
	c := newTestConsumer(mongoMock, &mockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (no-op) for already processed, got %v", err)
	}
}

func TestProcessKafkaMessage_EventNotFound(t *testing.T) {
	mongoMock := &mockMongoStore{
		registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		eventErr: mongo.ErrNoDocuments,
	}
	c := newTestConsumer(mongoMock, &mockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (rejected), got %v", err)
	}
	if mongoMock.rejectedReason != models.ReasonEventNotFound {
		t.Fatalf("expected reason %s, got %s", models.ReasonEventNotFound, mongoMock.rejectedReason)
	}
}

func TestProcessKafkaMessage_DuplicateUser(t *testing.T) {
	mongoMock := &mockMongoStore{
		registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		event:       &models.Event{ID: "event-1"},
		hasAccepted: true,
	}
	c := newTestConsumer(mongoMock, &mockRedisStore{})
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (rejected), got %v", err)
	}
	if mongoMock.rejectedReason != models.ReasonDuplicateUser {
		t.Fatalf("expected reason %s, got %s", models.ReasonDuplicateUser, mongoMock.rejectedReason)
	}
}

func TestProcessKafkaMessage_CapacityFull(t *testing.T) {
	mongoMock := &mockMongoStore{
		registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		event: &models.Event{ID: "event-1"},
	}
	redisMock := &mockRedisStore{seatAvailable: false}
	c := newTestConsumer(mongoMock, redisMock)
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected nil (rejected), got %v", err)
	}
	if mongoMock.rejectedReason != models.ReasonCapacityFull {
		t.Fatalf("expected reason %s, got %s", models.ReasonCapacityFull, mongoMock.rejectedReason)
	}
}

func TestProcessKafkaMessage_HappyPath(t *testing.T) {
	mongoMock := &mockMongoStore{
		registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		event: &models.Event{ID: "event-1"},
	}
	redisMock := &mockRedisStore{seatAvailable: true}
	c := newTestConsumer(mongoMock, redisMock)
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessKafkaMessage_AcceptFailsReleasesSeat(t *testing.T) {
	mongoMock := &mockMongoStore{
		registration: &models.RegistrationRequest{
			RequestID: "req-1",
			Status:    models.StatusPending,
		},
		event:           &models.Event{ID: "event-1"},
		markAcceptedErr: errors.New("mongo write failure"),
	}
	redisMock := &mockRedisStore{seatAvailable: true}
	c := newTestConsumer(mongoMock, redisMock)
	err := c.processKafkaMessage(context.Background(), validKafkaMessage())
	if err == nil {
		t.Fatal("expected error when accept fails")
	}
	if !redisMock.releaseCalled {
		t.Fatal("expected ReleaseEventSeat to be called when accept fails")
	}
}
