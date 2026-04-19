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
	"github.com/SCE-Development/SCEvents/pkg/middleware"
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
	defer func() {
		if err := db.DisconnectRedis(); err != nil {
			log.Printf("Error disconnecting Redis: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	producer := registration.NewProducer(
		[]string{cfg.KafkaBroker},
		cfg.KafkaTopic,
	)

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
		events.GET("/registrations/:request_id", handlers.GetRegistrationStatus)

		protected := events.Group("/")
		protected.Use(middleware.RequireAuth(middleware.MembershipStateNonMember, cfg.ClientAPIURL))
		{
			protected.POST("/", handlers.CreateEvent)
			protected.POST("/:id/register", handlers.RegisterForEvent(producer))
			protected.DELETE("/:id", handlers.DeleteEventByID)
			protected.PATCH("/:id", handlers.UpdateEventByID)
		}
	}

	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
