package main

import (
	"context"
	"log"

	"github.com/SCE-Development/SCEvents/internal/config"
	"github.com/SCE-Development/SCEvents/pkg/database"
	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
)

type registryEntry struct {
	Collection string
	Model      any
}

var registry = []registryEntry{
	{Collection: "events", Model: models.Event{}},
}

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

	ctx := context.Background()

	var grandTotal int64
	for _, entry := range registry {
		log.Printf("migrating collection %q...", entry.Collection)

		coll := db.Database().Collection(entry.Collection)
		updated, err := database.AutoMigrateDefaults(ctx, coll, entry.Model)
		if err != nil {
			log.Fatalf("Failed migrating %s: %v", entry.Collection, err)
		}
		grandTotal += updated
	}

	log.Printf("All defaults migrated successfully! (%d total field backfills)", grandTotal)
}
