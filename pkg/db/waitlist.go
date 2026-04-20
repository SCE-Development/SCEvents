package db

import (
	"context"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetWaitlistCollection() *mongo.Collection {
	return Database().Collection("waitlists")
}

func InitWaitlistIndexes() error {
	coll := GetWaitlistCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "user_id", Value: 1},
			},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
			},
		},
	})
	return err
}

func CreateWaitlistEntry(entry models.WaitlistEntry) error {
	coll := GetWaitlistCollection()

	entry.CreatedAt = time.Now().UTC()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := coll.InsertOne(ctx, entry)
	return err
}

func HasWaitlistEntry(eventID string, userID string) (bool, error) {
	coll := GetWaitlistCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"event_id": eventID,
		"user_id":  userID,
	}

	err := coll.FindOne(ctx, filter).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func CountWaitlistEntries(eventID string) (int64, error) {
	coll := GetWaitlistCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"event_id": eventID,
	}

	return coll.CountDocuments(ctx, filter)
}