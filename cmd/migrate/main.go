// Command migrate backfills MongoDB documents with default values declared
// via `default:"..."` struct tags on registered models.
//
// Usage:
//
//	MONGO_URI=mongodb://localhost:27017 go run ./cmd/migrate
//
// Re-running is safe: UpdateMany uses {$exists: false}, so already-migrated
// documents are not touched.
package main

import (
	"context"
	"log"

	"github.com/SCE-Development/SCEvents/internal/config"
	"github.com/SCE-Development/SCEvents/pkg/database"
	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
)

// registryEntry maps a MongoDB collection name to the Go model whose
// struct tags describe the defaults for that collection.
type registryEntry struct {
	Collection string
	Model      any
}

// registry is the list of (collection, model) pairs to migrate. Add new
// models here as the schema grows.
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
