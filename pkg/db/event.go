package db

import (
	"context"
	"time"

	event "github.com/SCE-Development/SCEvents/pkg/event"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetEvents() ([]event.Event, error) {
	coll := GetEventsCollection()
	ctx := context.Background()

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	events := make([]event.Event, 0)
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

// creates a new event in the database
func CreateEvent(e event.Event) (*event.Event, error) {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := coll.InsertOne(ctx, e)
	if err != nil {
		return nil, err
	}

	// if Mongo generated an ID, reflect it back in the event
	if e.ID == "" {
		switch id := res.InsertedID.(type) {
		case primitive.ObjectID:
			e.ID = id.Hex()
		case string:
			e.ID = id
		}
	}

	return &e, nil
}

func DeleteEventByID(id string) error {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := coll.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}

	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func UpdateEventByID(id string, e event.Event) error {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert the event struct to a map for $set update
	// We want to avoid overwriting _id and created_at if they are not intended to be changed
	update := bson.M{
		"$set": bson.M{
			"name":              e.Name,
			"date":              e.Date,
			"time":              e.Time,
			"location":          e.Location,
			"description":       e.Description,
			"admins":            e.Admins,
			"registration_form": e.RegistrationForm,
			"max_attendees":     e.MaxAttendees,
			"status":            e.Status,
		},
	}

	result, err := coll.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}

	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}
