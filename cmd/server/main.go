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

	// Get Redis address from environment variable
	redisAddr := os.Getenv("REDIS_ADDR")
	if err := db.ConnectRedis(redisAddr); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer func(){
		if err := db.DisconnectRedis(); err != nil {
			log.Printf("Error disconnecting Redis: %v", err)
		}
	}()

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "response",
		})
	})

	events := r.Group("/events")
	{
		events.GET("/", handlers.GetEvents)
		events.GET("/:id", handlers.GetEventByID)
		events.POST("/", handlers.CreateEvent)
	}

	r.Run()
}
