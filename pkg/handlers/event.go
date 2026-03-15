package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db"
	types "github.com/SCE-Development/SCEvents/pkg/event"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

func CreateEvent(c *gin.Context) {
	var event types.Event

	// parse JSON request body into struct
	if err := c.BindJSON(&event); err != nil {
		c.JSON(400, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	// persist the event to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	coll := db.GetEventsCollection()
	res, err := coll.InsertOne(ctx, event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create event",
		})
		return
	}

	// if Mongo generated an ID, reflect it back in the response
	if event.ID == "" {
		switch id := res.InsertedID.(type) {
		case primitive.ObjectID:
			event.ID = id.Hex()
		case string:
			event.ID = id
		}
	}

	c.JSON(201, gin.H{
		"message": "event created",
		"event":   event,
	})
}
