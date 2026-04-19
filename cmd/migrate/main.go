package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/SCE-Development/SCEvents/internal/config"
	"github.com/SCE-Development/SCEvents/pkg/database"
	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
)

type registryEntry struct {
	Collection string
	Model      any
}

// registry maps CLI subcommands to a Mongo collection and the struct used for default tags.
// Add an entry when a new collection gets fields with default tags; Go cannot infer model types from a string alone.
var registry = map[string]registryEntry{
	"events":        {Collection: "events", Model: models.Event{}},
	"registrations": {Collection: "registrations", Model: models.RegistrationRequest{}},
}

func main() {
	if len(os.Args) < 2 || strings.TrimSpace(os.Args[1]) == "" {
		log.Print("Please specify a collection")
		os.Exit(1)
	}

	key := strings.ToLower(strings.TrimSpace(os.Args[1]))
	entry, ok := registry[key]
	if !ok {
		log.Print("Provided type doesn't exist")
		os.Exit(1)
	}

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

	log.Printf("migrating collection %q...", entry.Collection)

	coll := db.Database().Collection(entry.Collection)
	updated, err := database.AutoMigrateDefaults(ctx, coll, entry.Model)
	if err != nil {
		log.Fatalf("Failed migrating %s: %v", entry.Collection, err)
	}

	log.Printf("Defaults migrated successfully! (%d field backfills)", updated)
}
