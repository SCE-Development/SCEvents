package handlers

import (
	"context"
	"testing"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/mocks"
	"github.com/SCE-Development/SCEvents/pkg/models"
)

func newTestHandler(redis db.RedisStore) *EventHandler {
	return &EventHandler{
		stores: &db.Stores{
			Redis: redis,
		},
	}
}

func TestSyncMaxAttendeesHeadcount_FiniteToUnlimited(t *testing.T) {
	redis := &mocks.MockRedisStore{}
	h := newTestHandler(redis)

	existing := &models.Event{MaxAttendees: 10}
	fields := map[string]interface{}{"max_attendees": float64(-1)}

	err := h.syncMaxAttendeesHeadcount(context.Background(), "event-1", existing, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !redis.DeleteCalled {
		t.Fatalf("expected DeleteEventHeadcount to be called")
	}
	if redis.SetCalled {
		t.Fatalf("did not expect SetEventHeadcount to be called")
	}
}

func TestSyncMaxAttendeesHeadcount_UnlimitedToFinite(t *testing.T) {
	redis := &mocks.MockRedisStore{}
	h := newTestHandler(redis)

	existing := &models.Event{MaxAttendees: -1}
	fields := map[string]interface{}{"max_attendees": float64(25)}

	err := h.syncMaxAttendeesHeadcount(context.Background(), "event-2", existing, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !redis.SetCalled {
		t.Fatalf("expected SetEventHeadcount to be called")
	}
	if redis.SetCapacity != 25 {
		t.Fatalf("expected capacity 25, got %d", redis.SetCapacity)
	}
	if redis.DeleteCalled {
		t.Fatalf("did not expect DeleteEventHeadcount to be called")
	}
}

func TestSyncMaxAttendeesHeadcount_FiniteToFinite(t *testing.T) {
	redis := &mocks.MockRedisStore{
		GetRemaining: 7, // existing max 10 => 3 seats taken
	}
	h := newTestHandler(redis)

	existing := &models.Event{MaxAttendees: 10}
	fields := map[string]interface{}{"max_attendees": float64(12)}

	err := h.syncMaxAttendeesHeadcount(context.Background(), "event-3", existing, fields)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !redis.GetCalled {
		t.Fatalf("expected GetEventHeadcount to be called")
	}
	if !redis.SetCalled {
		t.Fatalf("expected SetEventHeadcount to be called")
	}
	if redis.SetCapacity != 9 { // new max 12 - 3 seats taken
		t.Fatalf("expected remaining capacity 9, got %d", redis.SetCapacity)
	}
}
