package main

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/SCE-Development/SCEvents/internal/config"
	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/handlers"
	"github.com/SCE-Development/SCEvents/pkg/registration"
)

func main() {
	cfg := config.Load()

	if err := db.Connect(cfg.MongoURI); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if err := db.Disconnect(); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	if err := db.ConnectRedis(cfg.RedisAddr); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer func(){
		if err := db.DisconnectRedis(); err != nil {
			log.Printf("Error disconnecting Redis: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	consumer := registration.NewConsumer(
		[]string{cfg.KafkaBroker},
		cfg.KafkaTopic,
		cfg.KafkaGroupID,
	)
	
	go consumer.Run(ctx)
	
	r := gin.Default()

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{cfg.ClientURL}
	config.AllowCredentials = true
	config.AddAllowHeaders("Authorization")
	r.Use(cors.New(config))

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
		events.POST("/:id/register", handlers.RegisterForEvent)
		events.DELETE("/:id", handlers.DeleteEventByID)
		events.PATCH("/:id", handlers.UpdateEventByID)
	}

	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
