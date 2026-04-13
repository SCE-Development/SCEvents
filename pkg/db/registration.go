package db

import (
	"context"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/registrationmodel"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetRegistrationsCollection() *mongo.Collection {
	return Database().Collection("registrations")
}

// CreatePendingRegistration inserts a new registration request with status "pending" before async processing
func CreatePendingRegistration(r registrationmodel.RegistrationRequest) (*registrationmodel.RegistrationRequest, error) {
	coll := GetRegistrationsCollection()

	now := time.Now().UTC()
	r.Status = registrationmodel.StatusPending
	r.DecisionReason = registrationmodel.ReasonNone
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
func GetRegistrationByID(requestID string) (*registrationmodel.RegistrationRequest, error) {
	coll := GetRegistrationsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var r registrationmodel.RegistrationRequest
	err := coll.FindOne(ctx, bson.M{"_id": requestID}).Decode(&r)
	if err != nil {
		return nil, err
	}

	return &r, nil
}

// HasAcceptedRegistration checks if a user already has an accepted registration for an event
func HasAcceptedRegistration(eventID string, userID string) (bool, error) {
	coll := GetRegistrationsCollection()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"event_id":           eventID,
		"registrant.user_id": userID,
		"status":             registrationmodel.StatusAccepted,
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
			"status":          registrationmodel.StatusAccepted,
			"decision_reason": registrationmodel.ReasonNone,
			"updated_at":      now,
			"processed_at":    now,
		},
	}

	result, err := coll.UpdateOne(
		ctx,
		bson.M{
			"_id":    requestID,
			"status": registrationmodel.StatusPending,
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
func MarkRegistrationRejected(requestID string, reason registrationmodel.DecisionReason) error {
	coll := GetRegistrationsCollection()

	now := time.Now().UTC()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"status":          registrationmodel.StatusRejected,
			"decision_reason": reason,
			"updated_at":      now,
			"processed_at":    now,
		},
	}

	result, err := coll.UpdateOne(
		ctx,
		bson.M{
			"_id":    requestID,
			"status": registrationmodel.StatusPending,
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
