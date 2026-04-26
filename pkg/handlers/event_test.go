package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/mocks"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func newAttendanceSummaryTestRouter(mongoStore db.MongoStore) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewEventHandler(&db.Stores{Mongo: mongoStore})
	router.Use(func(c *gin.Context) {
		c.Set("userID", "user-1")
		c.Set("userRole", models.RoleMember)
		c.Next()
	})
	router.GET("/events/:id/attendance", handler.GetEventAttendanceSummary)
	return router
}

func TestGetEventAttendanceSummary(t *testing.T) {
	t.Run("returns attendee count", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{
			Event:         &models.Event{ID: "event-1", Admins: []string{"user-1"}},
			AttendeeCount: 2,
		}
		router := newAttendanceSummaryTestRouter(mongoStore)

		req := httptest.NewRequest(http.MethodGet, "/events/event-1/attendance", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var body map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("expected valid JSON, got %v", err)
		}
		if body["event_id"] != "event-1" {
			t.Fatalf("expected event_id event-1, got %v", body["event_id"])
		}
		if body["attendee_count"] != float64(2) {
			t.Fatalf("expected attendee_count 2, got %v", body["attendee_count"])
		}
		if !mongoStore.CountCalled {
			t.Fatal("expected attendee count query to be called")
		}
	})

	t.Run("returns not found for missing event", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{EventErr: mongo.ErrNoDocuments}
		router := newAttendanceSummaryTestRouter(mongoStore)

		req := httptest.NewRequest(http.MethodGet, "/events/event-1/attendance", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected status 404, got %d", w.Code)
		}
		if mongoStore.CountCalled {
			t.Fatal("expected attendee count query not to be called")
		}
	})

	t.Run("returns server error when count fails", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{
			Event:            &models.Event{ID: "event-1", Admins: []string{"user-1"}},
			AttendeeCountErr: errors.New("count failed"),
		}
		router := newAttendanceSummaryTestRouter(mongoStore)

		req := httptest.NewRequest(http.MethodGet, "/events/event-1/attendance", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Fatalf("expected status 500, got %d", w.Code)
		}
		if !mongoStore.CountCalled {
			t.Fatal("expected attendee count query to be called")
		}
	})

	t.Run("returns attendee count for non-admin", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{
			Event:         &models.Event{ID: "event-1", Admins: []string{"user-2"}},
			AttendeeCount: 1,
		}
		router := newAttendanceSummaryTestRouter(mongoStore)

		req := httptest.NewRequest(http.MethodGet, "/events/event-1/attendance", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		if !mongoStore.CountCalled {
			t.Fatal("expected attendee count query to be called")
		}
	})
}

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
