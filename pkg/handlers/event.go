package handlers

import (
	"net/http"

	"github.com/SCE-Development/SCEvents/pkg/db"
	event "github.com/SCE-Development/SCEvents/pkg/event"
	"github.com/gin-gonic/gin"
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
