package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

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
		c.Set("accessLevel", 1)
		c.Next()
	})
	router.GET("/events/:id/attendance", handler.GetEventAttendanceSummary)
	return router
}

func TestGetEventAttendanceSummary(t *testing.T) {
	t.Run("returns attendee count for listed event admin", func(t *testing.T) {
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

	t.Run("returns attendee count for site admin on admin-less event", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		mongoStore := &mocks.MockMongoStore{
			Event:         &models.Event{ID: "event-1", Admins: []string{}},
			AttendeeCount: 4,
		}
		handler := NewEventHandler(&db.Stores{Mongo: mongoStore})

		router.Use(func(c *gin.Context) {
			c.Set("userID", "site-admin-1")
			c.Set("userRole", models.RoleAdmin)
			c.Set("accessLevel", 3)
			c.Next()
		})

		router.GET("/events/:id/attendance", handler.GetEventAttendanceSummary)

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

func newCreateEventTestRouter(mongoStore db.MongoStore, redisStore db.RedisStore, userID string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewEventHandler(&db.Stores{Mongo: mongoStore, Redis: redisStore})
	router.Use(func(c *gin.Context) {
		c.Set("userID", userID)
		c.Set("userRole", models.RoleAdmin)
		c.Next()
	})
	router.POST("/events", handler.CreateEvent)
	return router
}

func TestCreateEventForcesCreatorIntoAdmins(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{}
	redisStore := &mocks.MockRedisStore{}
	router := newCreateEventTestRouter(mongoStore, redisStore, "creator-1")

	body := []byte(`{
		"id": "event-1",
		"name": "Hack Night",
		"date": "2026-05-01",
		"time": "18:00",
		"location": "SCE",
		"description": "Build things",
		"admins": ["admin-2"],
		"registration_form": [],
		"max_attendees": -1,
		"created_at": "2026-04-28T00:00:00Z",
		"waitlist_enabled": false
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	if mongoStore.CreatedEvent == nil {
		t.Fatal("expected event to be created")
	}
	expectedAdmins := []string{"admin-2", "creator-1"}
	if !reflect.DeepEqual(mongoStore.CreatedEvent.Admins, expectedAdmins) {
		t.Fatalf("expected admins %v, got %v", expectedAdmins, mongoStore.CreatedEvent.Admins)
	}
	if redisStore.SetCalled {
		t.Fatal("did not expect Redis headcount for unlimited event")
	}
}

func TestCreateEventDedupesCreatorAdmin(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{}
	redisStore := &mocks.MockRedisStore{}
	router := newCreateEventTestRouter(mongoStore, redisStore, "creator-1")

	body := []byte(`{
		"id": "event-1",
		"name": "Hack Night",
		"date": "2026-05-01",
		"time": "18:00",
		"location": "SCE",
		"description": "Build things",
		"admins": [" creator-1 ", "admin-2", "creator-1"],
		"registration_form": [],
		"max_attendees": 20,
		"created_at": "2026-04-28T00:00:00Z",
		"waitlist_enabled": false
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
	expectedAdmins := []string{"creator-1", "admin-2"}
	if !reflect.DeepEqual(mongoStore.CreatedEvent.Admins, expectedAdmins) {
		t.Fatalf("expected admins %v, got %v", expectedAdmins, mongoStore.CreatedEvent.Admins)
	}
	if !redisStore.SetCalled || redisStore.SetCapacity != 20 {
		t.Fatalf("expected Redis headcount capacity 20, got called=%v capacity=%d", redisStore.SetCalled, redisStore.SetCapacity)
	}
}

func TestCreateEventRejectsMissingCreatorID(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{}
	redisStore := &mocks.MockRedisStore{}
	router := newCreateEventTestRouter(mongoStore, redisStore, "")

	body := []byte(`{
		"id": "event-1",
		"name": "Hack Night",
		"date": "2026-05-01",
		"time": "18:00",
		"location": "SCE",
		"description": "Build things",
		"admins": ["admin-2"],
		"registration_form": [],
		"max_attendees": -1,
		"created_at": "2026-04-28T00:00:00Z",
		"waitlist_enabled": false
	}`)

	req := httptest.NewRequest(http.MethodPost, "/events", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", w.Code, w.Body.String())
	}
	if mongoStore.CreatedEvent != nil {
		t.Fatal("did not expect event to be created")
	}
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

func newEventReadTestRouter(mongoStore db.MongoStore, userID string, accessLevel int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewEventHandler(&db.Stores{Mongo: mongoStore})

	router.Use(func(c *gin.Context) {
		if userID != "" {
			c.Set("userID", userID)
			c.Set("accessLevel", accessLevel)
		}
		c.Next()
	})

	router.GET("/events", handler.GetEvents)
	router.GET("/events/:id", handler.GetEventByID)

	return router
}

func TestGetEvents_Unauthenticated_OmitsRegistrationStatus(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Events: []models.Event{
			{ID: "event-1", Name: "Hack Night"},
		},
	}
	router := newEventReadTestRouter(mongoStore, "", 0)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if len(body) != 1 {
		t.Fatalf("expected 1 event, got %d", len(body))
	}

	if _, ok := body[0]["registration_status"]; ok {
		t.Fatalf("expected registration_status to be omitted, got %v", body[0]["registration_status"])
	}
}

func TestGetEvents_Authenticated_ReturnsRegistrationStatus(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Events: []models.Event{
			{ID: "event-1", Name: "Hack Night"},
			{ID: "event-2", Name: "Company Tour"},
			{ID: "event-3", Name: "Workshop"},
		},
		RegistrationStatuses: map[string]models.Status{
			"event-1": models.StatusAccepted,
			"event-2": models.StatusPending,
			"event-3": models.StatusRejected,
		},
		WaitlistedEventIDs: map[string]bool{},
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	got := map[string]string{}
	for _, event := range body {
		got[event["id"].(string)] = event["registration_status"].(string)
	}

	if got["event-1"] != "registered" {
		t.Fatalf("expected event-1 registered, got %s", got["event-1"])
	}
	if got["event-2"] != "pending" {
		t.Fatalf("expected event-2 pending, got %s", got["event-2"])
	}
	if got["event-3"] != "rejected" {
		t.Fatalf("expected event-3 rejected, got %s", got["event-3"])
	}
}

func TestGetEvents_Authenticated_WaitlistOverridesRejected(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Events: []models.Event{
			{ID: "event-1", Name: "Hack Night"},
		},
		RegistrationStatuses: map[string]models.Status{
			"event-1": models.StatusRejected,
		},
		WaitlistedEventIDs: map[string]bool{
			"event-1": true,
		},
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if body[0]["registration_status"] != "waitlisted" {
		t.Fatalf("expected waitlisted, got %v", body[0]["registration_status"])
	}
}

func TestGetEvents_Authenticated_WaitlistOnly(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Events: []models.Event{
			{ID: "event-1", Name: "Hack Night"},
		},
		RegistrationStatuses: map[string]models.Status{},
		WaitlistedEventIDs: map[string]bool{
			"event-1": true,
		},
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if body[0]["registration_status"] != "waitlisted" {
		t.Fatalf("expected waitlisted, got %v", body[0]["registration_status"])
	}
}

func TestGetEvents_Authenticated_NoStatus_ReturnsNone(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Events: []models.Event{
			{ID: "event-1", Name: "Hack Night"},
		},
		RegistrationStatuses: map[string]models.Status{},
		WaitlistedEventIDs:   map[string]bool{},
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body []map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if body[0]["registration_status"] != "none" {
		t.Fatalf("expected none, got %v", body[0]["registration_status"])
	}
}

func TestGetEvents_Returns500_WhenRegistrationLookupFails(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Events:                  []models.Event{{ID: "event-1", Name: "Hack Night"}},
		RegistrationStatusesErr: errors.New("boom"),
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestGetEvents_Returns500_WhenWaitlistLookupFails(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Events:                []models.Event{{ID: "event-1", Name: "Hack Night"}},
		RegistrationStatuses:  map[string]models.Status{},
		WaitlistedEventIDsErr: errors.New("boom"),
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", w.Code)
	}
}

func TestGetEventByID_Unauthenticated_OmitsRegistrationStatus(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Event: &models.Event{ID: "event-1", Name: "Hack Night"},
	}
	router := newEventReadTestRouter(mongoStore, "", 0)

	req := httptest.NewRequest(http.MethodGet, "/events/event-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if _, ok := body["registration_status"]; ok {
		t.Fatalf("expected registration_status to be omitted, got %v", body["registration_status"])
	}
}

func TestGetEventByID_Authenticated_ReturnsRegistered(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Event: &models.Event{ID: "event-1", Name: "Hack Night"},
		RegistrationStatuses: map[string]models.Status{
			"event-1": models.StatusAccepted,
		},
		WaitlistedEventIDs: map[string]bool{},
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events/event-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if body["registration_status"] != "registered" {
		t.Fatalf("expected registered, got %v", body["registration_status"])
	}
}

func TestGetEventByID_Authenticated_ReturnsWaitlisted(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		Event: &models.Event{ID: "event-1", Name: "Hack Night"},
		RegistrationStatuses: map[string]models.Status{
			"event-1": models.StatusRejected,
		},
		WaitlistedEventIDs: map[string]bool{
			"event-1": true,
		},
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events/event-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if body["registration_status"] != "waitlisted" {
		t.Fatalf("expected waitlisted, got %v", body["registration_status"])
	}
}

func TestGetEventByID_NotFound(t *testing.T) {
	mongoStore := &mocks.MockMongoStore{
		EventErr: mongo.ErrNoDocuments,
	}
	router := newEventReadTestRouter(mongoStore, "", 0)

	req := httptest.NewRequest(http.MethodGet, "/events/event-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}

func TestResolveEventRegistrationStatus(t *testing.T) {
	tests := []struct {
		name                 string
		registrationStatuses map[string]models.Status
		waitlistedEventIDs   map[string]bool
		expected             EventRegistrationStatus
	}{
		{
			name: "accepted -> registered",
			registrationStatuses: map[string]models.Status{
				"event-1": models.StatusAccepted,
			},
			waitlistedEventIDs: map[string]bool{},
			expected:           EventRegistrationStatusRegistered,
		},
		{
			name: "pending -> pending",
			registrationStatuses: map[string]models.Status{
				"event-1": models.StatusPending,
			},
			waitlistedEventIDs: map[string]bool{},
			expected:           EventRegistrationStatusPending,
		},
		{
			name: "rejected -> rejected",
			registrationStatuses: map[string]models.Status{
				"event-1": models.StatusRejected,
			},
			waitlistedEventIDs: map[string]bool{},
			expected:           EventRegistrationStatusRejected,
		},
		{
			name: "rejected + waitlisted -> waitlisted",
			registrationStatuses: map[string]models.Status{
				"event-1": models.StatusRejected,
			},
			waitlistedEventIDs: map[string]bool{
				"event-1": true,
			},
			expected: EventRegistrationStatusWaitlisted,
		},
		{
			name:                 "waitlist only -> waitlisted",
			registrationStatuses: map[string]models.Status{},
			waitlistedEventIDs: map[string]bool{
				"event-1": true,
			},
			expected: EventRegistrationStatusWaitlisted,
		},
		{
			name:                 "nothing -> none",
			registrationStatuses: map[string]models.Status{},
			waitlistedEventIDs:   map[string]bool{},
			expected:             EventRegistrationStatusNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveEventRegistrationStatus("event-1", tt.registrationStatuses, tt.waitlistedEventIDs)
			if result != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetEventByID_IncludesPublishFields(t *testing.T) {
	publishDate := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	publishedAt := time.Date(2026, 5, 1, 12, 5, 0, 0, time.UTC)

	mongoStore := &mocks.MockMongoStore{
		Event: &models.Event{
			ID:          "event-1",
			Name:        "Hack Night",
			PublishDate: &publishDate,
			PublishedAt: &publishedAt,
		},
		RegistrationStatuses: map[string]models.Status{},
		WaitlistedEventIDs:   map[string]bool{},
	}
	router := newEventReadTestRouter(mongoStore, "user-1", 1)

	req := httptest.NewRequest(http.MethodGet, "/events/event-1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if _, ok := body["publish_date"]; !ok {
		t.Fatal("expected publish_date in response")
	}
	if _, ok := body["published_at"]; !ok {
		t.Fatal("expected published_at in response")
	}
}

func TestDeleteEventByID_RequiresDraftOrClosed(t *testing.T) {
	t.Run("conflict when published", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{
			Event: &models.Event{
				ID:           "event-1",
				Status:       models.StatusPublished,
				Admins:       []string{"user-1"},
				MaxAttendees: -1,
			},
		}
		redis := &mocks.MockRedisStore{}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		handler := NewEventHandler(&db.Stores{Mongo: mongoStore, Redis: redis})
		router.Use(func(c *gin.Context) {
			c.Set("userID", "user-1")
			c.Set("userRole", models.RoleOfficer)
			c.Set("accessLevel", 2)
			c.Next()
		})
		router.DELETE("/events/:id", handler.DeleteEventByID)

		req := httptest.NewRequest(http.MethodDelete, "/events/event-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected status 409, got %d: %s", w.Code, w.Body.String())
		}
		if mongoStore.DeletedEventID != "" {
			t.Fatal("expected mongo delete not to be called")
		}
	})

	t.Run("ok when draft", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{
			Event: &models.Event{
				ID:           "event-1",
				Status:       models.StatusDraft,
				Admins:       []string{"user-1"},
				MaxAttendees: 10,
			},
		}
		redis := &mocks.MockRedisStore{}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		handler := NewEventHandler(&db.Stores{Mongo: mongoStore, Redis: redis})
		router.Use(func(c *gin.Context) {
			c.Set("userID", "user-1")
			c.Set("userRole", models.RoleOfficer)
			c.Set("accessLevel", 2)
			c.Next()
		})
		router.DELETE("/events/:id", handler.DeleteEventByID)

		req := httptest.NewRequest(http.MethodDelete, "/events/event-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}
		if mongoStore.DeletedEventID != "event-1" {
			t.Fatalf("expected delete for event-1, got %q", mongoStore.DeletedEventID)
		}
		if !redis.DeleteCalled {
			t.Fatal("expected redis DeleteEventHeadcount for capped event")
		}
	})

	t.Run("redis not called when unlimited", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{
			Event: &models.Event{
				ID:           "event-1",
				Status:       models.StatusClosed,
				Admins:       []string{"user-1"},
				MaxAttendees: -1,
			},
		}
		redis := &mocks.MockRedisStore{}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		handler := NewEventHandler(&db.Stores{Mongo: mongoStore, Redis: redis})
		router.Use(func(c *gin.Context) {
			c.Set("userID", "user-1")
			c.Set("userRole", models.RoleOfficer)
			c.Set("accessLevel", 2)
			c.Next()
		})
		router.DELETE("/events/:id", handler.DeleteEventByID)

		req := httptest.NewRequest(http.MethodDelete, "/events/event-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}
		if redis.DeleteCalled {
			t.Fatal("did not expect redis delete for unlimited event")
		}
	})

	t.Run("forbidden when not event admin", func(t *testing.T) {
		mongoStore := &mocks.MockMongoStore{
			Event: &models.Event{
				ID:           "event-1",
				Status:       models.StatusDraft,
				Admins:       []string{"user-1"},
				MaxAttendees: -1,
			},
		}
		redis := &mocks.MockRedisStore{}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		handler := NewEventHandler(&db.Stores{Mongo: mongoStore, Redis: redis})
		router.Use(func(c *gin.Context) {
			c.Set("userID", "user-2")
			c.Set("userRole", models.RoleOfficer)
			c.Set("accessLevel", 2)
			c.Next()
		})
		router.DELETE("/events/:id", handler.DeleteEventByID)

		req := httptest.NewRequest(http.MethodDelete, "/events/event-1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d: %s", w.Code, w.Body.String())
		}
		if mongoStore.DeletedEventID != "" {
			t.Fatal("expected mongo delete not to be called")
		}
	})
}
