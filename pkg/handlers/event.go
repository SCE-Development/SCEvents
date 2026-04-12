package handlers

import (
	"net/http"

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
	isAdmin := false
	for _, admin := range existingEvent.Admins {
		if admin == userID {
			isAdmin = true
			break
		}
	}
	if !isAdmin {
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
	delete(fields, "id")
	delete(fields, "_id")
	delete(fields, "created_at")

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

	c.JSON(http.StatusOK, gin.H{
		"message": "event updated successfully",
	})
}