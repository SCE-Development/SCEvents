package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/SCE-Development/SCEvents/pkg/registration"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type EventHandler struct {
	stores *db.Stores
}

func NewEventHandler(stores *db.Stores) *EventHandler {
	return &EventHandler{stores: stores}
}

const dateLayout = "2006-01-02"

// EventRegistrationStatus is the UI-facing per-user status attached to event responses
type EventRegistrationStatus string

const (
	EventRegistrationStatusNone       EventRegistrationStatus = "none"
	EventRegistrationStatusPending    EventRegistrationStatus = "pending"
	EventRegistrationStatusRegistered EventRegistrationStatus = "registered"
	EventRegistrationStatusWaitlisted EventRegistrationStatus = "waitlisted"
	EventRegistrationStatusRejected   EventRegistrationStatus = "rejected"
)

type EventResponse struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	Date               string                   `json:"date"`
	EndDate            string                   `json:"end_date,omitempty"`
	Time               string                   `json:"time"`
	Location           string                   `json:"location"`
	Description        string                   `json:"description"`
	Admins             []string                 `json:"admins"`
	RegistrationForm   []models.FormQuestion    `json:"registration_form"`
	MaxAttendees       int                      `json:"max_attendees"`
	CreatedAt          string                   `json:"created_at"`
	Status             string                   `json:"status"`
	Visibility         string                   `json:"visibility"`
	MinimumVisibleRole string                   `json:"minimum_visible_role,omitempty"`
	WaitlistEnabled    bool                     `json:"waitlist_enabled"`
	WaitlistSize       int                      `json:"waitlist_size,omitempty"`
	PublishDate        *time.Time               `json:"publish_date,omitempty"`
	PublishedAt        *time.Time               `json:"published_at,omitempty"`
	RegistrationStatus *EventRegistrationStatus `json:"registration_status,omitempty"`
}

func writeEventEditForbidden(c *gin.Context, ev *models.Event) {
	if len(ev.Admins) == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "this event has no dedicated admins; only site admins may modify it",
		})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": "you are not an admin of this event",
	})
}

// buildViewerFromContext builds an EventViewer from auth data stored in the Gin context
func buildViewerFromContext(c *gin.Context) models.EventViewer {
	accessLevel := 0
	if v, exists := c.Get("accessLevel"); exists {
		if n, ok := v.(int); ok {
			accessLevel = n
		}
	}

	return models.EventViewer{
		UserID:      strings.TrimSpace(c.GetString("userID")),
		AccessLevel: accessLevel,
	}
}

// GetEvents: query startDate & endDate (YYYY-MM-DD), or omit both for current UTC month; one alone is 400.
func (h *EventHandler) GetEvents(c *gin.Context) {
	startQ := strings.TrimSpace(c.Query("startDate"))
	endQ := strings.TrimSpace(c.Query("endDate"))

	var startDate, endDate string
	switch {
	case startQ == "" && endQ == "": //default case, current UTC month
		now := time.Now().UTC()
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		last := first.AddDate(0, 1, -1)
		startDate = first.Format(dateLayout)
		endDate = last.Format(dateLayout)
	case startQ != "" && endQ != "":
		if _, err := time.Parse(dateLayout, startQ); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid startDate", "expected": dateLayout})
			return
		}
		if _, err := time.Parse(dateLayout, endQ); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endDate", "expected": dateLayout})
			return
		}
		if startQ > endQ {
			c.JSON(http.StatusBadRequest, gin.H{"error": "startDate must be on or before endDate"})
			return
		}
		startDate, endDate = startQ, endQ
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provide both startDate and endDate query params, or neither for the current month default",
		})
		return
	}

	viewer := buildViewerFromContext(c)

	events, err := h.stores.Mongo.GetVisibleEvents(c.Request.Context(), viewer, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch events",
		})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusOK, buildEventResponses(events, "", nil, nil))
		return
	}

	eventIDs := collectEventIDs(events)

	registrationStatuses, err := h.stores.Mongo.GetRegistrationStatusesForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch registration statuses",
		})
		return
	}

	waitlistedEventIDs, err := h.stores.Mongo.GetWaitlistedEventIDsForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch waitlist statuses",
		})
		return
	}

	responses := buildEventResponses(events, userID, registrationStatuses, waitlistedEventIDs)
	c.JSON(http.StatusOK, responses)
}

// returns a single event by ID
func (h *EventHandler) GetEventByID(c *gin.Context) {
	id := c.Param("id")
	viewer := buildViewerFromContext(c)

	event, err := h.stores.Mongo.GetVisibleEventByID(c.Request.Context(), viewer, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusOK, buildEventResponse(*event, nil))
		return
	}

	eventIDs := []string{event.ID}

	registrationStatuses, err := h.stores.Mongo.GetRegistrationStatusesForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch registration status",
		})
		return
	}

	waitlistedEventIDs, err := h.stores.Mongo.GetWaitlistedEventIDsForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch waitlist status",
		})
		return
	}

	status := resolveEventRegistrationStatus(event.ID, registrationStatuses, waitlistedEventIDs)
	statusCopy := status

	c.JSON(http.StatusOK, buildEventResponse(*event, &statusCopy))
}

func (h *EventHandler) GetEventAttendanceSummary(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("id"))
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event id is required",
		})
		return
	}

	event, err := h.stores.Mongo.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch event",
		})
		return
	}

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")
	if !event.CanEdit(userID, userRole) {
		writeEventEditForbidden(c, event)
		return
	}

	attendeeCount, err := h.stores.Mongo.CountAcceptedRegistrationsForEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch attendance summary",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":       eventID,
		"attendee_count": attendeeCount,
	})
}

// creates a new event
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var event models.Event

	// parse JSON request body into struct
	if err := c.BindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	event.ApplyDefaults()
	event.SyncPublicationState(time.Now().UTC())

	creatorID := strings.TrimSpace(c.GetString("userID"))
	if creatorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "creator user id is required",
		})
		return
	}

	event.Admins = sanitizeAdmins(event.Admins)
	event.Admins = ensureCreatorIsEventAdmin(event.Admins, creatorID)

	if err := event.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	createdEvent, err := h.stores.Mongo.CreateEvent(c.Request.Context(), event)
	if err != nil {
		log.Printf("CreateEvent error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create event",
		})
		return
	}

	// max_attendees == -1 means the event has unlimited attendees,
	// so we skip Redis headcount tracking for that case.
	if createdEvent.MaxAttendees != -1 {
		if err := h.stores.Redis.SetEventHeadcount(c.Request.Context(), createdEvent.ID, createdEvent.MaxAttendees); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to store event headcount",
			})
			return
		}
	}

	c.JSON(http.StatusCreated, createdEvent)
}

func sanitizeAdmins(admins []string) []string {
	seen := make(map[string]struct{}, len(admins))
	normalized := make([]string, 0, len(admins))

	for _, admin := range admins {
		admin = strings.TrimSpace(admin)
		if admin == "" {
			continue
		}
		if _, ok := seen[admin]; ok {
			continue
		}
		seen[admin] = struct{}{}
		normalized = append(normalized, admin)
	}

	return normalized
}

func ensureCreatorIsEventAdmin(admins []string, creatorID string) []string {
	for _, admin := range admins {
		if admin == creatorID {
			return admins
		}
	}
	return append(admins, creatorID)
}

// deletes an event by ID
func (h *EventHandler) DeleteEventByID(c *gin.Context) {
	id := c.Param("id")

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	existingEvent, err := h.stores.Mongo.GetEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	if !existingEvent.CanEdit(userID, userRole) {
		writeEventEditForbidden(c, existingEvent)
		return
	}

	err = h.stores.Mongo.DeleteEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "event deleted successfully",
	})
}

// syncMaxAttendeesHeadcount keeps Redis headcount consistent after max_attendees changes.
func (h *EventHandler) syncMaxAttendeesHeadcount(ctx context.Context, id string, existingEvent *models.Event, fields map[string]interface{}) error {
	maxAttendees, ok := fields["max_attendees"]
	if !ok {
		return nil
	}

	newMax, ok := maxAttendees.(float64)
	if !ok {
		return nil
	}

	newMaxInt := int(newMax)

	switch {
	case newMaxInt == -1:
		// Unlimited events are not headcount-tracked in Redis.
		return h.stores.Redis.DeleteEventHeadcount(ctx, id)

	case existingEvent.MaxAttendees == -1:
		// Moving from unlimited to limited starts Redis tracking at the new max.
		return h.stores.Redis.SetEventHeadcount(ctx, id, newMaxInt)

	default:
		currentRemaining, err := h.stores.Redis.GetEventHeadcount(ctx, id)
		if err != nil {
			currentRemaining = existingEvent.MaxAttendees
		}

		seatsTaken := existingEvent.MaxAttendees - currentRemaining
		newRemaining := newMaxInt - seatsTaken
		if newRemaining < 0 {
			// Event already exceeds the new cap; remaining seats cannot be negative.
			newRemaining = 0
		}

		return h.stores.Redis.SetEventHeadcount(ctx, id, newRemaining)
	}
}

// updates an event by ID (partial update)
func (h *EventHandler) UpdateEventByID(c *gin.Context) {
	id := c.Param("id")

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	existingEvent, err := h.stores.Mongo.GetEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	if !existingEvent.CanEdit(userID, userRole) {
		writeEventEditForbidden(c, existingEvent)
		return
	}

	// Bind request body to a map for partial update
	var fields map[string]interface{}
	if err := c.BindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	models.SanitizeUpdateFields(fields)

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no updatable fields provided",
		})
		return
	}

	// Validate patch fields for visibility/status support
	updatedEvent := existingEvent
	if err := updatedEvent.ApplyPatch(fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	updatedEvent.SyncPublicationState(time.Now().UTC())

	// Validate the merged event state after applying PATCH fields
	if err := updatedEvent.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if _, ok := fields["visibility"]; ok {
		fields["visibility"] = updatedEvent.Visibility
		fields["minimum_visible_role"] = updatedEvent.MinimumVisibleRole
	}

	if _, ok := fields["minimum_visible_role"]; ok {
		fields["minimum_visible_role"] = updatedEvent.MinimumVisibleRole
	}

	if _, ok := fields["admins"]; ok {
		fields["admins"] = sanitizeAdmins(updatedEvent.Admins)
	}

	if _, ok := fields["status"]; ok && updatedEvent.Status == models.StatusPublished {
		fields["publish_date"] = updatedEvent.PublishDate
		fields["published_at"] = updatedEvent.PublishedAt
	}

	if _, ok := fields["publish_date"]; ok {
		fields["publish_date"] = updatedEvent.PublishDate
		if updatedEvent.Status == models.StatusPublished {
			fields["status"] = updatedEvent.Status
			fields["published_at"] = updatedEvent.PublishedAt
		}
	}

	err = h.stores.Mongo.UpdateEventByID(c.Request.Context(), id, fields)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to update event",
			})
		}
		return
	}

	// If max_attendees was updated, sync the headcount in Redis
	// accounting for seats already given out
	if err := h.syncMaxAttendeesHeadcount(c.Request.Context(), id, existingEvent, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "event updated but failed to sync headcount in Redis",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "event updated successfully",
	})
}

// RegisterForEvent
func (h *EventHandler) RegisterForEvent(producer *registration.Producer) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "event id is required",
			})
			return
		}

		var payload models.RegistrationPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid JSON payload",
			})
			return
		}

		if strings.TrimSpace(payload.Registrant.UserID) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "login required to register for event",
			})
			return
		}

		alreadyRegistered, err := h.stores.Mongo.HasPendingOrAcceptedRegistration(c.Request.Context(), eventID, payload.Registrant.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to verify existing registration",
			})
			return
		}
		if alreadyRegistered {
			c.JSON(http.StatusConflict, gin.H{
				"error": "user is already registered for this event",
			})
			return
		}

		ev, err := h.stores.Mongo.GetEventByID(c.Request.Context(), eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
			return
		}

		if ev.Status == models.StatusClosed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "registration is closed for this event",
			})
			return
		}

		if ev.IsListedAdmin(payload.Registrant.UserID) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "event admins cannot register for their own event",
			})
			return
		}

		if err := ev.ValidateRegistration(payload.RegistrationFormAnswers); err != nil {
			var formErr *models.RegistrationFormValidationError
			if errors.As(err, &formErr) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": formErr.Message,
					"field": formErr.Field,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to validate registration",
			})
			return
		}

		var requestIDBytes [16]byte
		if _, err := rand.Read(requestIDBytes[:]); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create registration request",
			})
			return
		}

		req := models.RegistrationRequest{
			RequestID:  hex.EncodeToString(requestIDBytes[:]),
			EventID:    eventID,
			Registrant: payload.Registrant,
			Answers:    payload.RegistrationFormAnswers,
		}

		created, err := h.stores.Mongo.CreatePendingRegistration(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create registration request",
			})
			return
		}

		// Capacity handling:
		// Events with max_attendees == -1 are treated as unlimited and are NOT tracked in Redis
		// Therefore, skip Redis seat reservation entirely for unlimited events
		// For limited events, use Redis to atomically reserve a seat to avoid race conditions
		// under concurrent registrations
		seatTaken := true
		if ev.MaxAttendees != -1 {
			seatTaken, err = h.stores.Redis.TryTakeEventSeat(c.Request.Context(), eventID)
			if err != nil {
				_ = h.stores.Mongo.MarkRegistrationRejected(c.Request.Context(), created.RequestID, models.ReasonInternalError)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to reserve event seat",
				})
				return
			}
			if !seatTaken {
				_ = h.stores.Mongo.MarkRegistrationRejected(c.Request.Context(), created.RequestID, models.ReasonCapacityFull)
				c.JSON(http.StatusConflict, gin.H{
					"error": "event is full",
				})
				return
			}
		}

		if err := h.stores.Mongo.MarkRegistrationAccepted(c.Request.Context(), created.RequestID); err != nil {
			// Only release a Redis seat if this event is capacity-limited
			// Unlimited events do not have a Redis headcount entry
			if ev.MaxAttendees != -1 && seatTaken {
				_ = h.stores.Redis.ReleaseEventSeat(c.Request.Context(), eventID)
			}
			_ = h.stores.Mongo.MarkRegistrationRejected(c.Request.Context(), created.RequestID, models.ReasonInternalError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to send registration request",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":    "registration request sent",
			"event_id":   eventID,
			"request_id": created.RequestID,
		})
	}
}

func (h *EventHandler) JoinEventWaitlist(c *gin.Context) {
	eventID := c.Param("id")
	if strings.TrimSpace(eventID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event id is required",
		})
		return
	}

	userID := c.GetString("userID")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "login required to join waitlist",
		})
		return
	}

	ev, err := h.stores.Mongo.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	if !ev.WaitlistEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "waitlist is not enabled for this event",
		})
		return
	}

	if ev.WaitlistSize <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "waitlist is not configured for this event",
		})
		return
	}

	alreadyRegistered, err := h.stores.Mongo.HasAcceptedRegistration(c.Request.Context(), eventID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to verify existing registration",
		})
		return
	}
	if alreadyRegistered {
		c.JSON(http.StatusConflict, gin.H{
			"error": "user is already registered for this event",
		})
		return
	}

	alreadyWaitlisted, err := h.stores.Mongo.HasWaitlistEntry(c.Request.Context(), eventID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to verify existing waitlist entry",
		})
		return
	}
	if alreadyWaitlisted {
		c.JSON(http.StatusConflict, gin.H{
			"error": "user is already on the waitlist for this event",
		})
		return
	}

	count, err := h.stores.Mongo.CountWaitlistEntries(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to count waitlist entries",
		})
		return
	}

	if count >= int64(ev.WaitlistSize) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "waitlist is full",
		})
		return
	}

	entry := models.WaitlistEntry{
		EventID: eventID,
		UserID:  userID,
	}

	if err := h.stores.Mongo.CreateWaitlistEntry(c.Request.Context(), entry); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "user is already on the waitlist for this event",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to join waitlist",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "joined waitlist successfully",
		"event_id": eventID,
	})
}

func (h *EventHandler) GetRegistrationStatus(c *gin.Context) {
	requestID := c.Param("request_id")
	if strings.TrimSpace(requestID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request id is required",
		})
		return
	}

	userID := c.Query("user_id")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing user_id query parameter",
		})
		return
	}

	req, err := h.stores.Mongo.GetRegistrationByID(c.Request.Context(), requestID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "registration request not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch registration status",
		})
		return
	}

	if req.Registrant.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you do not have access to this registration request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id":      req.RequestID,
		"event_id":        req.EventID,
		"status":          req.Status,
		"decision_reason": req.DecisionReason,
		"created_at":      req.CreatedAt,
		"updated_at":      req.UpdatedAt,
		"processed_at":    req.ProcessedAt,
	})
}

// buildEventResponses attaches registration_status only when user context is available
func buildEventResponse(event models.Event, registrationStatus *EventRegistrationStatus) EventResponse {
	return EventResponse{
		ID:                 event.ID,
		Name:               event.Name,
		Date:               event.Date,
		EndDate:            event.EndDate,
		Time:               event.Time,
		Location:           event.Location,
		Description:        event.Description,
		Admins:             event.Admins,
		RegistrationForm:   event.RegistrationForm,
		MaxAttendees:       event.MaxAttendees,
		CreatedAt:          event.CreatedAt,
		Status:             event.Status,
		Visibility:         event.Visibility,
		MinimumVisibleRole: event.MinimumVisibleRole,
		WaitlistEnabled:    event.WaitlistEnabled,
		WaitlistSize:       event.WaitlistSize,
		PublishDate:        event.PublishDate,
		PublishedAt:        event.PublishedAt,
		RegistrationStatus: registrationStatus,
	}
}

// resolveEventRegistrationStatus maps persisted registration/waitlist records into one UI-facing status
func resolveEventRegistrationStatus(
	eventID string,
	registrationStatuses map[string]models.Status,
	waitlistedEventIDs map[string]bool,
) EventRegistrationStatus {
	if status, ok := registrationStatuses[eventID]; ok {
		switch status {
		case models.StatusAccepted:
			return EventRegistrationStatusRegistered
		case models.StatusPending:
			return EventRegistrationStatusPending
		case models.StatusRejected:
			if waitlistedEventIDs[eventID] {
				return EventRegistrationStatusWaitlisted
			}
			return EventRegistrationStatusRejected
		}
	}

	if waitlistedEventIDs[eventID] {
		return EventRegistrationStatusWaitlisted
	}

	return EventRegistrationStatusNone
}

func collectEventIDs(events []models.Event) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}

func buildEventResponses(
	events []models.Event,
	userID string,
	registrationStatuses map[string]models.Status,
	waitlistedEventIDs map[string]bool,
) []EventResponse {
	responses := make([]EventResponse, 0, len(events))

	for _, event := range events {
		if strings.TrimSpace(userID) == "" {
			responses = append(responses, buildEventResponse(event, nil))
			continue
		}

		status := resolveEventRegistrationStatus(event.ID, registrationStatuses, waitlistedEventIDs)
		statusCopy := status
		responses = append(responses, buildEventResponse(event, &statusCopy))
	}

	return responses
}

func (h *EventHandler) loadEventAndAuthorizeAdmin(ctx context.Context, eventID, userID, userRole string) bool {
	ev, err := h.stores.Mongo.GetEventByID(ctx, eventID)
	if err != nil {
		return false
	}
	return ev.CanEdit(userID, userRole)
}

func parsePagination(c *gin.Context) (int64, int64, bool) {
	limit := int64(50)
	offset := int64(0)

	if limitQ := strings.TrimSpace(c.Query("limit")); limitQ != "" {
		parsed, err := strconv.ParseInt(limitQ, 10, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return 0, 0, false
		}
		limit = parsed
	}

	if offsetQ := strings.TrimSpace(c.Query("offset")); offsetQ != "" {
		parsed, err := strconv.ParseInt(offsetQ, 10, 64)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be a non-negative integer"})
			return 0, 0, false
		}
		offset = parsed
	}

	return limit, offset, true
}

func (h *EventHandler) ListEventRegistrations(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("id"))
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event id is required"})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	userRole := strings.TrimSpace(c.GetString("userRole"))

	limit, offset, ok := parsePagination(c)
	if !ok {
		return
	}

	if !h.loadEventAndAuthorizeAdmin(c.Request.Context(), eventID, userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not an admin of this event"})
		return
	}

	registrations, err := h.stores.Mongo.ListRegistrationsByEventID(c.Request.Context(), eventID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch event registrations"})
		return
	}

	statusCounts, err := h.stores.Mongo.CountRegistrationsByStatusForEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch registration dashboard summary"})
		return
	}

	summary := models.RegistrationDashboardSummary{
		Pending:  statusCounts[models.StatusPending],
		Accepted: statusCounts[models.StatusAccepted],
		Rejected: statusCounts[models.StatusRejected],
	}
	summary.Total = summary.Pending + summary.Accepted + summary.Rejected

	c.JSON(http.StatusOK, gin.H{
		"event_id":      eventID,
		"summary":       summary,
		"attendees":     registrations,
		"registrations": registrations,
		"limit":         limit,
		"offset":        offset,
	})
}

func (h *EventHandler) GetEventRegistrationByRequestID(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("id"))
	requestID := strings.TrimSpace(c.Param("request_id"))
	if eventID == "" || requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event id and request id are required"})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	userRole := strings.TrimSpace(c.GetString("userRole"))

	if !h.loadEventAndAuthorizeAdmin(c.Request.Context(), eventID, userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not an admin of this event"})
		return
	}

	registrationReq, err := h.stores.Mongo.GetRegistrationByEventAndRequestID(c.Request.Context(), eventID, requestID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "registration request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch registration request"})
		return
	}

	c.JSON(http.StatusOK, registrationReq)
}
