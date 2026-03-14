package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/handlers"
)

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

	r.GET("/events", handlers.GetEventsHandler)
	
	r.Run()
}
