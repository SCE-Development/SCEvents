package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/SCE-Development/SCEvents/internal/config"
	dbmongo "github.com/SCE-Development/SCEvents/pkg/db/mongo"
	dbredis "github.com/SCE-Development/SCEvents/pkg/db/redis"
	"github.com/SCE-Development/SCEvents/pkg/db/stores"
	dbwaitlist "github.com/SCE-Development/SCEvents/pkg/db/waitlist"
	"github.com/SCE-Development/SCEvents/pkg/handlers"
	"github.com/SCE-Development/SCEvents/pkg/middleware"
	"github.com/SCE-Development/SCEvents/pkg/registration"
)

func main() {
	cfg := config.Load()

	if err := dbmongo.Connect(cfg.MongoURI); err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	if err := dbwaitlist.InitWaitlistIndexes(); err != nil {
		log.Fatalf("Failed to initialize waitlist indexes: %v", err)
	}
	defer func() {
		if err := dbmongo.Disconnect(); err != nil {
			log.Printf("Error disconnecting MongoDB: %v", err)
		}
	}()

	if err := dbredis.ConnectRedis(cfg.RedisAddr); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer func() {
		if err := dbredis.DisconnectRedis(); err != nil {
			log.Printf("Error disconnecting Redis: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	redisStore := dbredis.NewRedisStore(dbredis.RedisClient())
	appStores := &stores.Stores{
		Redis: redisStore,
	}

	producer := registration.NewProducer(
		[]string{cfg.KafkaBroker},
		cfg.KafkaTopic,
	)

	consumer := registration.NewConsumer(
		[]string{cfg.KafkaBroker},
		cfg.KafkaTopic,
		cfg.KafkaGroupID,
		appStores,
	)

	eventHandler := handlers.NewEventHandler(appStores)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		consumer.Run(ctx)
	}()

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
		events.GET("/", eventHandler.GetEvents)
		events.GET("/:id", eventHandler.GetEventByID)
		events.GET("/registrations/:request_id", eventHandler.GetRegistrationStatus)

		protected := events.Group("/")
		protected.Use(middleware.RequireAuth(middleware.MembershipStateNonMember, cfg.ClientAPIURL))
		{
			protected.POST("/", eventHandler.CreateEvent)
			protected.POST("/:id/register", eventHandler.RegisterForEvent(producer))
			protected.POST("/:id/waitlist", eventHandler.JoinEventWaitlist)
			protected.DELETE("/:id", eventHandler.DeleteEventByID)
			protected.PATCH("/:id", eventHandler.UpdateEventByID)
		}
	}

	srv := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: r,
	}

	errChan := make(chan error, 1)

	// Start the HTTP server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// Block until a signal is received or a background service crashes
	gracefulShutdown(cancel, &wg, errChan,
		srv.Shutdown,
		func(_ context.Context) error { return consumer.Close() },
		func(_ context.Context) error { return producer.Close() },
	)
}

// gracefulShutdown blocks until a stop signal or critical error is received.
// It then cancels the global context, runs all closers with a timeout, and
// waits for tracked goroutines to finish.
func gracefulShutdown(cancel context.CancelFunc, wg *sync.WaitGroup, errChan <-chan error, closers ...func(context.Context) error) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received or a background service crashes
	select {
	case err := <-errChan:
		log.Printf("critical error: %v", err)
	case <-quit:
		log.Println("shutting down...")
	}

	// 1. Cancel the global context to tell background workers (like Kafka) to stop looping
	cancel()

	// 2. Give cleanup tasks up to 15 seconds to finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()

	// 3. Execute all provided closer functions
	for _, closer := range closers {
		if err := closer(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}

	// Wait for all goroutines to exit.
	wg.Wait()
	log.Println("shutdown complete.")
}
