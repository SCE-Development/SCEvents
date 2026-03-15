package handlers

import (
	"net/http"

	types "github.com/SCE-Development/SCEvents/pkg/event"
	"github.com/gin-gonic/gin"
	"github.com/SCE-Development/SCEvents/pkg/db"
)

// returns a single event by ID
func GetEventByIDHandler(c *gin.Context) {
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

func CreateEvent(c *gin.Context) {
	var event types.Event

	// parse JSON request body into struct
	if err := c.BindJSON(&event); err != nil {
		c.JSON(400, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	// pretend we saved it to a database
	c.JSON(201, gin.H{
		"message": "event created",
		"event":   event,
	})
}
