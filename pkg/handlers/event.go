package handlers

import (
	types "github.com/SCE-Development/SCEvents/pkg/event"
	"github.com/gin-gonic/gin"
)

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
