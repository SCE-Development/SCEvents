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

func GetRegistrationsCollection() *mongo.Collection {
	return Database().Collection("registrations")
}

func InitRegistrationIndexes() error {
	coll := GetRegistrationsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{
				{Key: "event_id", Value: 1},
				{Key: "status", Value: 1},
			},
			Options: options.Index().SetName("event_id_status"),
		},
		{
			Keys: bson.D{
				{Key: "registrant.user_id", Value: 1},
				{Key: "event_id", Value: 1},
			},
			Options: options.Index().SetName("registrant_user_event"),
		},
	})
	return err
}

// CreatePendingRegistration inserts a new registration request with status "pending" before async processing
func CreatePendingRegistration(r models.RegistrationRequest) (*models.RegistrationRequest, error) {
	coll := GetRegistrationsCollection()

	now := time.Now().UTC()
	r.Status = models.StatusPending
	r.DecisionReason = models.ReasonNone
	r.CreatedAt = now
	r.UpdatedAt = now
	r.ProcessedAt = nil

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := coll.InsertOne(ctx, r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// GetRegistrationByID fetches a registration request by its request_id
func GetRegistrationByID(requestID string) (*models.RegistrationRequest, error) {
	coll := GetRegistrationsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var r models.RegistrationRequest
	err := coll.FindOne(ctx, bson.M{"_id": requestID}).Decode(&r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

func CountAcceptedRegistrationsForEvent(eventID string) (int64, error) {
	coll := GetRegistrationsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return coll.CountDocuments(ctx, bson.M{
		"event_id": eventID,
		"status":   models.StatusAccepted,
	})
}

// HasAcceptedRegistration checks if a user already has an accepted registration for an event
func HasAcceptedRegistration(eventID string, userID string) (bool, error) {
	coll := GetRegistrationsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"event_id":           eventID,
		"registrant.user_id": userID,
		"status":             models.StatusAccepted,
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

// HasPendingOrAcceptedRegistration checks if a user has an in-flight or accepted registration for an event.
// This is used to make the HTTP registration endpoint idempotent-ish for double submits.
func HasPendingOrAcceptedRegistration(eventID string, userID string) (bool, error) {
	coll := GetRegistrationsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"event_id":           eventID,
		"registrant.user_id": userID,
		"status": bson.M{
			"$in": []models.Status{models.StatusPending, models.StatusAccepted},
		},
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

// MarkRegistrationAccepted transitions a pending request to accepted and records processing time
func MarkRegistrationAccepted(requestID string) error {
	coll := GetRegistrationsCollection()

	now := time.Now().UTC()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":          models.StatusAccepted,
			"decision_reason": models.ReasonNone,
			"updated_at":      now,
			"processed_at":    now,
		},
	}

	result, err := coll.UpdateOne(
		ctx,
		bson.M{
			"_id":    requestID,
			"status": models.StatusPending,
		},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

// MarkRegistrationRejected transitions a pending request to rejected with a reason
func MarkRegistrationRejected(requestID string, reason models.DecisionReason) error {
	coll := GetRegistrationsCollection()

	now := time.Now().UTC()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":          models.StatusRejected,
			"decision_reason": reason,
			"updated_at":      now,
			"processed_at":    now,
		},
	}

	result, err := coll.UpdateOne(
		ctx,
		bson.M{
			"_id":    requestID,
			"status": models.StatusPending,
		},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}

	return nil
}

func (s *mongoStore) CreatePendingRegistration(ctx context.Context, r models.RegistrationRequest) (*models.RegistrationRequest, error) {
	now := time.Now().UTC()
	r.Status = models.StatusPending
	r.DecisionReason = models.ReasonNone
	r.CreatedAt = now
	r.UpdatedAt = now
	r.ProcessedAt = nil
	_, err := s.registrations.InsertOne(ctx, r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *mongoStore) GetRegistrationByID(ctx context.Context, requestID string) (*models.RegistrationRequest, error) {
	var r models.RegistrationRequest
	err := s.registrations.FindOne(ctx, bson.M{"_id": requestID}).Decode(&r)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *mongoStore) CountAcceptedRegistrationsForEvent(ctx context.Context, eventID string) (int64, error) {
	return s.registrations.CountDocuments(ctx, bson.M{
		"event_id": eventID,
		"status":   models.StatusAccepted,
	})
}

func (s *mongoStore) HasAcceptedRegistration(ctx context.Context, eventID, userID string) (bool, error) {
	filter := bson.M{
		"event_id":           eventID,
		"registrant.user_id": userID,
		"status":             models.StatusAccepted,
	}
	err := s.registrations.FindOne(ctx, filter).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *mongoStore) HasPendingOrAcceptedRegistration(ctx context.Context, eventID, userID string) (bool, error) {
	filter := bson.M{
		"event_id":           eventID,
		"registrant.user_id": userID,
		"status": bson.M{
			"$in": []models.Status{models.StatusPending, models.StatusAccepted},
		},
	}
	err := s.registrations.FindOne(ctx, filter).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *mongoStore) MarkRegistrationAccepted(ctx context.Context, requestID string) error {
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"status":          models.StatusAccepted,
			"decision_reason": models.ReasonNone,
			"updated_at":      now,
			"processed_at":    now,
		},
	}
	result, err := s.registrations.UpdateOne(
		ctx,
		bson.M{
			"_id":    requestID,
			"status": models.StatusPending,
		},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *mongoStore) MarkRegistrationRejected(ctx context.Context, requestID string, reason models.DecisionReason) error {
	now := time.Now().UTC()
	update := bson.M{
		"$set": bson.M{
			"status":          models.StatusRejected,
			"decision_reason": reason,
			"updated_at":      now,
			"processed_at":    now,
		},
	}
	result, err := s.registrations.UpdateOne(
		ctx,
		bson.M{
			"_id":    requestID,
			"status": models.StatusPending,
		},
		update,
	)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

// Higher values win when multiple registration records exist for the same user-event pair
func registrationStatusPriority(status models.Status) int {
	switch status {
	case models.StatusAccepted:
		return 3
	case models.StatusPending:
		return 2
	case models.StatusRejected:
		return 1
	default:
		return 0
	}
}

// GetRegistrationStatusesForUser batch-loads the caller's strongest registration state for each event
func (s *mongoStore) GetRegistrationStatusesForUser(ctx context.Context, userID string, eventIDs []string) (map[string]models.Status, error) {
	result := make(map[string]models.Status)

	if strings.TrimSpace(userID) == "" || len(eventIDs) == 0 {
		return result, nil
	}

	filter := bson.M{
		"registrant.user_id": userID,
		"event_id": bson.M{
			"$in": eventIDs,
		},
	}

	cursor, err := s.registrations.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var requests []models.RegistrationRequest
	if err := cursor.All(ctx, &requests); err != nil {
		return nil, err
	}

	for _, req := range requests {
		current, exists := result[req.EventID]
		if !exists || registrationStatusPriority(req.Status) > registrationStatusPriority(current) {
			result[req.EventID] = req.Status
		}
	}

	return result, nil
}
