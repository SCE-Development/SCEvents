package db

import (
	"context"
	"strings"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// buildDateRangeFilter constructs a MongoDB filter to find events that occur within the specified date range, accounting for both single-day and multi-day events
func buildDateRangeFilter(startDate, endDate string) bson.M {
	return bson.M{
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
}

// buildVisibilityFilter constructs a MongoDB filter to determine which events are visible to a given viewer based on their access level and the event's visibility settings
func buildVisibilityFilter(viewer models.EventViewer) bson.M {
	// Site admin can view everything
	if viewer.AccessLevel >= 3 {
		return bson.M{}
	}

	conditions := bson.A{}

	// Event admins can view their own events, even if draft
	if strings.TrimSpace(viewer.UserID) != "" {
		conditions = append(conditions, bson.M{"admins": viewer.UserID})
	}

	// Published public events are visible to everyone
	conditions = append(conditions, bson.M{
		"$and": bson.A{
			bson.M{"status": models.StatusPublished},
			bson.M{"visibility": models.VisibilityPublic},
		},
	})

	// Published private events depend on access level
	privateRoleConditions := bson.A{}
	if viewer.AccessLevel >= 1 {
		privateRoleConditions = append(privateRoleConditions, bson.M{"minimum_visible_role": models.RoleMember})
	}
	if viewer.AccessLevel >= 2 {
		privateRoleConditions = append(privateRoleConditions, bson.M{"minimum_visible_role": models.RoleOfficer})
	}
	if viewer.AccessLevel >= 3 {
		privateRoleConditions = append(privateRoleConditions, bson.M{"minimum_visible_role": models.RoleAdmin})
	}

	if len(privateRoleConditions) > 0 {
		conditions = append(conditions, bson.M{
			"$and": bson.A{
				bson.M{"status": models.StatusPublished},
				bson.M{"visibility": models.VisibilityPrivate},
				bson.M{"$or": privateRoleConditions},
			},
		})
	}

	return bson.M{"$or": conditions}
}

// GetVisiblleEvents retrieves events from the database that are visible to the specified viewer and fall within the given date range, 
// applying appropriate filters based on event status, visibility, and viewer access level
func (s *mongoStore) GetVisibleEvents(ctx context.Context, viewer models.EventViewer, startDate, endDate string) ([]models.Event, error) {
	filter := bson.M{
		"$and": bson.A{
			buildDateRangeFilter(startDate, endDate),
			buildVisibilityFilter(viewer),
		},
	}

	cursor, err := s.events.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cursor.Close(ctx) }()

	var events []models.Event
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

// GetVisibleEventByID retrieves an event by ID and checks if it's visible to the specified viewer based on event status, visibility, and viewer access level
func (s *mongoStore) GetVisibleEventByID(ctx context.Context, viewer models.EventViewer, id string) (*models.Event, error) {
	filter := bson.M{
		"$and": bson.A{
			bson.M{"_id": id},
			buildVisibilityFilter(viewer),
		},
	}

	var e models.Event
	if err := s.events.FindOne(ctx, filter).Decode(&e); err != nil {
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
	defer func() {
		_ = cursor.Close(ctx)
	}()
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

// PublishDueEvents promotes due draft events to published
// It is safe to run repeatedly because only draft events are updated
func (s *mongoStore) PublishDueEvents(ctx context.Context, now time.Time) (int64, error) {
	now = now.UTC()

	filter := bson.M{
		"status": models.StatusDraft,
		"publish_date": bson.M{
			"$ne":  nil,
			"$lte": now,
		},
	}

	update := bson.M{
		"$set": bson.M{
			"status":       models.StatusPublished,
			"published_at": now,
		},
	}

	result, err := s.events.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}

	return result.ModifiedCount, nil
}

// InitEventIndexes creates necessary indexes on the events collection to optimize query performance for common access patterns, such as filtering by status, visibility, date, and publish date
func InitEventIndexes() error {
	coll := GetEventsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "visibility", Value: 1},
				{Key: "date", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "publish_date", Value: 1},
				{Key: "status", Value: 1},
			},
		},
		{
			Keys: bson.D{
				{Key: "admins", Value: 1},
			},
		},
	})

	return err
}
