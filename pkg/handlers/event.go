package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
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

	events, err := db.GetEvents(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch events",
		})
		return
	}
	c.JSON(http.StatusOK, events)
}

// returns a single event by ID
func (h *EventHandler) GetEventByID(c *gin.Context) {
	id := c.Param("id")

	event, err := db.GetEventByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	c.JSON(http.StatusOK, event)
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

	if err := event.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	createdEvent, err := db.CreateEvent(event)
	if err != nil {
		log.Printf("CreateEvent error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create event",
		})
		return
	}

	if err := h.stores.Redis.SetEventHeadcount(c.Request.Context(), createdEvent.ID, createdEvent.MaxAttendees); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to store event headcount",
		})
		return
	}

	c.JSON(http.StatusCreated, createdEvent)
}

// deletes an event by ID
func (h *EventHandler) DeleteEventByID(c *gin.Context) {
	id := c.Param("id")

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	existingEvent, err := db.GetEventByID(id)
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

	err = db.DeleteEventByID(id)
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

// updates an event by ID (partial update)
func (h *EventHandler) UpdateEventByID(c *gin.Context) {
	id := c.Param("id")

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	existingEvent, err := db.GetEventByID(id)
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

	err = db.UpdateEventByID(id, fields)
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
	if maxAttendees, ok := fields["max_attendees"]; ok {
		if newMax, ok := maxAttendees.(float64); ok {
			// Get current remaining seats from Redis
			currentRemaining, err := h.stores.Redis.GetEventHeadcount(c.Request.Context(), id)
			if err != nil {
				// If Redis key doesn't exist yet, no seats have been taken
				currentRemaining = existingEvent.MaxAttendees
			}

			// Calculate how many seats have already been given out
			seatsTaken := existingEvent.MaxAttendees - currentRemaining

			// Set the new headcount: new max minus seats already taken
			newRemaining := int(newMax) - seatsTaken
			if newRemaining < 0 {
				newRemaining = 0
			}

			if err := h.stores.Redis.SetEventHeadcount(c.Request.Context(), id, newRemaining); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "event updated but failed to sync headcount in Redis",
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "event updated successfully",
	})
}

// RegisterForEvent writes a pending registration to MongoDB, publishes a reference message to Kafka, and returns 202 Accepted.
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

		alreadyRegistered, err := db.HasPendingOrAcceptedRegistration(eventID, payload.Registrant.UserID)
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

		ev, err := db.GetEventByID(eventID)
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

		if ev.IsAdmin(payload.Registrant.UserID) {
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

		created, err := db.CreatePendingRegistration(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create registration request",
			})
			return
		}

		if err := producer.PublishRegistration(c.Request.Context(), created.RequestID, eventID, payload.Registrant.UserID); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "failed to queue registration for processing",
			})
			return
		}

		c.JSON(http.StatusAccepted, gin.H{
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

	ev, err := db.GetEventByID(eventID)
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

	alreadyRegistered, err := db.HasAcceptedRegistration(eventID, userID)
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

	alreadyWaitlisted, err := db.HasWaitlistEntry(eventID, userID)
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

	count, err := db.CountWaitlistEntries(eventID)
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

	if err := db.CreateWaitlistEntry(entry); err != nil {
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

	req, err := db.GetRegistrationByID(requestID)
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
