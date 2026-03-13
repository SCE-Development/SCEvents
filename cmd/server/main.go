package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/SCE-Development/SCEvents/pkg/db"
)

type Event struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Date        string `json:"date"`
}

func main() {
	mongoURI := os.Getenv("MONGO_URI")
	if err := db.Connect(mongoURI); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := db.Disconnect(); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "response",
		})
	})

	r.POST("/events", func(c *gin.Context) {
	var event Event

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
})

	r.Run()
}
