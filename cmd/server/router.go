package main

import (
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/SCE-Development/SCEvents/internal/config"
	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/handlers"
	"github.com/SCE-Development/SCEvents/pkg/middleware"
	"github.com/SCE-Development/SCEvents/pkg/registration"
)

func setupRouter(cfg config.AppConfig) (*gin.Engine, *registration.Producer) {
	redisStore := db.NewRedisStore(db.RedisClient())
	stores := &db.Stores{
		Redis: redisStore,
	}

	producer := registration.NewProducer(
		[]string{cfg.KafkaBroker},
		cfg.KafkaTopic,
	)

	r := gin.Default()

	// Set up CORS
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{cfg.ClientURL}
	config.AllowCredentials = true
	config.AddAllowHeaders("Authorization")
	r.Use(cors.New(config))

	// Create handlers
	eventHandler := handlers.NewEventHandler(stores)

	// Ping route
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "response",
		})
	})

	// Events routes
	events := r.Group("/events")
	{
		events.GET("/", eventHandler.GetEvents)
		events.GET("/:id", eventHandler.GetEventByID)
		events.GET("/registrations/:request_id", eventHandler.GetRegistrationStatus)

		protected := events.Group("/")
		protected.Use(middleware.RequireAuth(middleware.MembershipStateNonMember, cfg.ClientAPIURL))
		{
			protected.POST("/", eventHandler.CreateEvent)
			protected.DELETE("/:id", eventHandler.DeleteEventByID)
			protected.PATCH("/:id", eventHandler.UpdateEventByID)
			protected.POST("/:id/register", eventHandler.RegisterForEvent(producer))
			protected.POST("/:id/waitlist", eventHandler.JoinEventWaitlist)
		}
	}

	return r, producer
}