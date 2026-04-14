package handlers

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

const dateLayout = "2006-01-02"

// GetEvents: query startDate & endDate (YYYY-MM-DD), or omit both for current UTC month; one alone is 400.
func GetEvents(c *gin.Context) {
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
func GetEventByID(c *gin.Context) {
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
func CreateEvent(c *gin.Context) {
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create event",
		})
		return
	}

	if err := db.SetEventHeadcount(createdEvent.ID, createdEvent.MaxAttendees); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to store event headcount",
		})
		return
	}

	c.JSON(http.StatusCreated, createdEvent)
}

// deletes an event by ID
func DeleteEventByID(c *gin.Context) {
	id := c.Param("id")

	err := db.DeleteEventByID(id)
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
func UpdateEventByID(c *gin.Context) {
	id := c.Param("id")

	// Basic event-admin auth check:
	// The caller must pass their user_id as a query parameter.
	// We verify they are listed in the event's Admins before allowing the update.
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing user_id query parameter",
		})
		return
	}

	// Fetch the existing event to verify admin access
	existingEvent, err := db.GetEventByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	// Check if user is an admin of this event
	if !existingEvent.IsAdmin(userID) {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you are not an admin of this event",
		})
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
			currentRemaining, err := db.GetEventHeadcount(id)
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

			if err := db.SetEventHeadcount(id, newRemaining); err != nil {
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

func RegisterForEvent(c *gin.Context) {
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

	alreadyRegistered, err := db.IsUserRegisteredForEvent(eventID, payload.Registrant.UserID)
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

	c.JSON(http.StatusAccepted, gin.H{
		"message":  "registration request sent",
		"event_id": eventID,
	})
}
