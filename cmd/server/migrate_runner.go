package main

import (
	"context"
	"log"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/migration"
	"github.com/SCE-Development/SCEvents/pkg/models"
)

func runMigrationCycle(ctx context.Context) {
	for key, entry := range models.MigrationRegistry {
		coll := db.Database().Collection(entry.Collection)

		updated, err := migration.AutoMigrateDefaults(ctx, coll, entry.Model)
		if err != nil {
			log.Printf("auto-migrate failed for %s (%s): %v", key, entry.Collection, err)
			continue
		}

		if updated > 0 {
			log.Printf("auto-migrate updated %d fields for %s (%s)", updated, key, entry.Collection)
		}
	}
}

func runAutoMigrations(ctx context.Context, interval time.Duration) {
	// run once at startup
	runMigrationCycle(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Print("auto-migrate runner stopped")
			return
		case <-ticker.C:
			runMigrationCycle(ctx)
		}
	}
}