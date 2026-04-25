package handlers

import (
	"context"
	"testing"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
)

type mockRedisStore struct {
	setCalled    bool
	setCapacity  int
	setErr       error

	getCalled    bool
	getRemaining int
	getErr       error

	deleteCalled bool
	deleteErr    error
}

func (m *mockRedisStore) SetEventHeadcount(_ context.Context, _ string, capacity int) error {
	m.setCalled = true
	m.setCapacity = capacity
	return m.setErr
}

func (m *mockRedisStore) GetEventHeadcount(_ context.Context, _ string) (int, error) {
	m.getCalled = true
	return m.getRemaining, m.getErr
}

func (m *mockRedisStore) DeleteEventHeadcount(_ context.Context, _ string) error {
	m.deleteCalled = true
	return m.deleteErr
}

func (m *mockRedisStore) TryTakeEventSeat(context.Context, string) (bool, error) {
	return false, nil
}

func (m *mockRedisStore) ReleaseEventSeat(context.Context, string) error {
	return nil
}

func (m *mockRedisStore) IsUserRegisteredForEvent(context.Context, string, string) (bool, error) {
	return false, nil
}

func (m *mockRedisStore) Close() error {
	return nil
}

func newTestHandler(redis db.RedisStore) *EventHandler {
	return &EventHandler{
		stores: &db.Stores{
			Redis: redis,
		},
	}
}

func TestSyncMaxAttendeesHeadcount_FiniteToUnlimited(t *testing.T) {
	redis := &mockRedisStore{}
	h := newTestHandler(redis)

	existing := &models.Event{MaxAttendees: 10}
	fields := map[string]interface{}{"max_attendees": float64(-1)}

	err := h.syncMaxAttendeesHeadcount(context.Background(), "event-1", existing, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !redis.deleteCalled {
		t.Fatalf("expected DeleteEventHeadcount to be called")
	}
	if redis.setCalled {
		t.Fatalf("did not expect SetEventHeadcount to be called")
	}
}

func TestSyncMaxAttendeesHeadcount_UnlimitedToFinite(t *testing.T) {
	redis := &mockRedisStore{}
	h := newTestHandler(redis)

	existing := &models.Event{MaxAttendees: -1}
	fields := map[string]interface{}{"max_attendees": float64(25)}

	err := h.syncMaxAttendeesHeadcount(context.Background(), "event-2", existing, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !redis.setCalled {
		t.Fatalf("expected SetEventHeadcount to be called")
	}
	if redis.setCapacity != 25 {
		t.Fatalf("expected capacity 25, got %d", redis.setCapacity)
	}
	if redis.deleteCalled {
		t.Fatalf("did not expect DeleteEventHeadcount to be called")
	}
}

func TestSyncMaxAttendeesHeadcount_FiniteToFinite(t *testing.T) {
	redis := &mockRedisStore{
		getRemaining: 7, // existing max 10 => 3 seats taken
	}
	h := newTestHandler(redis)

	existing := &models.Event{MaxAttendees: 10}
	fields := map[string]interface{}{"max_attendees": float64(12)}

	err := h.syncMaxAttendeesHeadcount(context.Background(), "event-3", existing, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !redis.getCalled {
		t.Fatalf("expected GetEventHeadcount to be called")
	}
	if !redis.setCalled {
		t.Fatalf("expected SetEventHeadcount to be called")
	}
	if redis.setCapacity != 9 { // new max 12 - 3 seats taken
		t.Fatalf("expected remaining capacity 9, got %d", redis.setCapacity)
	}
}