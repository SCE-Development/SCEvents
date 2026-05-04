package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/SCE-Development/SCEvents/pkg/config"
	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/migration"
	"github.com/SCE-Development/SCEvents/pkg/models"
)

func main() {
	if len(os.Args) < 2 || strings.TrimSpace(os.Args[1]) == "" {
		log.Print("Please specify a collection")
		os.Exit(1)
	}

	key := strings.ToLower(strings.TrimSpace(os.Args[1]))
	entry, ok := models.MigrationRegistry[key]
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
	updated, err := migration.AutoMigrateDefaults(ctx, coll, entry.Model)
	if err != nil {
		log.Fatalf("Failed migrating %s: %v", entry.Collection, err)
	}

	log.Printf("Defaults migrated successfully! (%d field backfills)", updated)
}
