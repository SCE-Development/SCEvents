package db

import (
	"context"

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