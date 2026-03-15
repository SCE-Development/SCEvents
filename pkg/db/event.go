package db

import (
	"context"
	"time"

	event "github.com/SCE-Development/SCEvents/pkg/event"
	"go.mongodb.org/mongo-driver/bson"
)

func GetEvents() ([]event.Event, error) {
	coll := GetEventsCollection()
	ctx := context.Background()

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var events []event.Event
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}

	return events, nil
}

// retrieves an event from the database by ID
func GetEventByID(id string) (*event.Event, error) {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var e event.Event
	err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}
