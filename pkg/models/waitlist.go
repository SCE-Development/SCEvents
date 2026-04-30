package models

import "time"

type WaitlistEntry struct {
	EventID   string    `bson:"event_id" json:"event_id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
