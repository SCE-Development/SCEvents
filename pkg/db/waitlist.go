package db

import (
	"context"
	"strings"
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

func (s *mongoStore) CreateWaitlistEntry(ctx context.Context, entry models.WaitlistEntry) error {
	entry.CreatedAt = time.Now().UTC()
	_, err := s.waitlists.InsertOne(ctx, entry)
	return err
}

func (s *mongoStore) HasWaitlistEntry(ctx context.Context, eventID string, userID string) (bool, error) {
	filter := bson.M{
		"event_id": eventID,
		"user_id":  userID,
	}

	err := s.waitlists.FindOne(ctx, filter).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *mongoStore) CountWaitlistEntries(ctx context.Context, eventID string) (int64, error) {
	filter := bson.M{
		"event_id": eventID,
	}
	return s.waitlists.CountDocuments(ctx, filter)
}

// GetWaitlistedEventIDsForUser returns a set of event IDs the user is currently waitlisted for
func (s *mongoStore) GetWaitlistedEventIDsForUser(ctx context.Context, userID string, eventIDs []string) (map[string]bool, error) {
	result := make(map[string]bool)

	if strings.TrimSpace(userID) == "" || len(eventIDs) == 0 {
		return result, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{
		"user_id": userID,
		"event_id": bson.M{
			"$in": eventIDs,
		},
	}

	cursor, err := s.waitlists.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var entries []models.WaitlistEntry
	if err := cursor.All(ctx, &entries); err != nil {
		return nil, err
	}

	for _, entry := range entries {
		result[entry.EventID] = true
	}

	return result, nil
}
