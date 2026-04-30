package main

import (
	"context"
	"log"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db"
)

func startPublisher(ctx context.Context, stores *db.Stores) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			updated, err := stores.Mongo.PublishDueEvents(ctx, time.Now().UTC())
			if err != nil {
				log.Printf("publish scheduler error: %v", err)
				continue
			}
			if updated > 0 {
				log.Printf("auto-published %d event(s)", updated)
			}
		}
	}
}
