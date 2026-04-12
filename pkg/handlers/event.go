package handlers

import (
	"errors"
	"net/http"
	"strings"
	"github.com/SCE-Development/SCEvents/pkg/db"
	event "github.com/SCE-Development/SCEvents/pkg/event"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// returns the MongoDB events collection
func GetEvents(c *gin.Context) {
	events, err := db.GetEvents()
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
	var event event.Event

	// parse JSON request body into struct
	if err := c.BindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON payload",
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

	// Strip immutable fields that should not be overwritten
	event.SanitizeUpdateFields(fields)

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no updatable fields provided",
		})
		return
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

	var payload event.RegistrationPayload
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
		var formErr *event.RegistrationFormValidationError
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