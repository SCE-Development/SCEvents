package event

import (
	"context"
	"time"

	dbmongo "github.com/SCE-Development/SCEvents/pkg/db/mongo"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type mongoStore struct {
	events *mongo.Collection
}

func NewMongoStore(events *mongo.Collection) *mongoStore {
	return &mongoStore{events: events}
}

func GetEventsCollection() *mongo.Collection {
	return dbmongo.Database().Collection("events")
}

// GetEvents returns events overlapping [startDate, endDate] (YYYY-MM-DD strings).
func GetEvents(startDate, endDate string) ([]models.Event, error) {
	coll := GetEventsCollection()
	ctx := context.Background()

	// date <= endDate and (single-day: date >= startDate else end_date >= startDate).
	filter := bson.M{
		"$and": bson.A{
			bson.M{"date": bson.M{"$lte": endDate}},
			bson.M{
				"$or": bson.A{
					bson.M{
						"$and": bson.A{
							bson.M{"$or": bson.A{
								bson.M{"end_date": bson.M{"$exists": false}},
								bson.M{"end_date": nil},
								bson.M{"end_date": ""},
							}},
							bson.M{"date": bson.M{"$gte": startDate}},
						},
					},
					bson.M{"end_date": bson.M{"$gte": startDate}},
				},
			},
		},
	}

	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	events := make([]models.Event, 0)
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}

	return events, nil
}

// retrieves an event from the database by ID
func GetEventByID(id string) (*models.Event, error) {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var e models.Event
	err := coll.FindOne(ctx, bson.M{"_id": id}).Decode(&e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// creates a new event in the database
func CreateEvent(e models.Event) (*models.Event, error) {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if e.ID == "" {
		e.ID = primitive.NewObjectID().Hex()
	}

	_, err := coll.InsertOne(ctx, e)
	if err != nil {
		return nil, err
	}

	return &e, nil
}

// deletes an event by ID
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

// UpdateEventByID performs a partial update on an event document.
// Only fields present in the provided map are updated via $set.
func UpdateEventByID(id string, fields map[string]interface{}) error {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": fields,
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

func (s *mongoStore) GetEvents(ctx context.Context, startDate, endDate string) ([]models.Event, error) {
	filter := bson.M{
		"$and": bson.A{
			bson.M{"date": bson.M{"$lte": endDate}},
			bson.M{
				"$or": bson.A{
					bson.M{
						"$and": bson.A{
							bson.M{"$or": bson.A{
								bson.M{"end_date": bson.M{"$exists": false}},
								bson.M{"end_date": nil},
								bson.M{"end_date": ""},
							}},
							bson.M{"date": bson.M{"$gte": startDate}},
						},
					},
					bson.M{"end_date": bson.M{"$gte": startDate}},
				},
			},
		},
	}
	cursor, err := s.events.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	events := make([]models.Event, 0)
	if err := cursor.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func (s *mongoStore) GetEventByID(ctx context.Context, id string) (*models.Event, error) {
	var e models.Event
	err := s.events.FindOne(ctx, bson.M{"_id": id}).Decode(&e)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *mongoStore) CreateEvent(ctx context.Context, e models.Event) (*models.Event, error) {
	if e.ID == "" {
		e.ID = primitive.NewObjectID().Hex()
	}
	_, err := s.events.InsertOne(ctx, e)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (s *mongoStore) DeleteEventByID(ctx context.Context, id string) error {
	result, err := s.events.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *mongoStore) UpdateEventByID(ctx context.Context, id string, fields map[string]interface{}) error {
	update := bson.M{
		"$set": fields,
	}
	result, err := s.events.UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}
