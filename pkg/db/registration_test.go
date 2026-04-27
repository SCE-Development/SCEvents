package db

import (
	"context"
	"testing"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCreatePendingRegistration(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		req := models.RegistrationRequest{
			RequestID: "req-1",
			EventID:   "event-1",
			Registrant: models.Registrant{
				Name:   "John Doe",
				Email:  "john@example.com",
				UserID: "user-1",
			},
		}
		result, err := store.CreatePendingRegistration(context.Background(), req)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.RequestID != "req-1" {
			t.Fatalf("expected request ID req-1, got %s", result.RequestID)
		}
		if result.Status != models.StatusPending {
			t.Fatalf("expected status pending, got %s", result.Status)
		}
		if result.ProcessedAt != nil {
			t.Fatal("expected ProcessedAt to be nil")
		}
	})
}

func TestGetRegistrationByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("found", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "req-1"},
			{Key: "event_id", Value: "event-1"},
			{Key: "status", Value: "pending"},
			{Key: "registrant", Value: bson.D{
				{Key: "name", Value: "John Doe"},
				{Key: "email", Value: "john@example.com"},
				{Key: "user_id", Value: "user-1"},
			}},
		}))
		result, err := store.GetRegistrationByID(context.Background(), "req-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.RequestID != "req-1" {
			t.Fatalf("expected request ID req-1, got %s", result.RequestID)
		}
		if result.Registrant.UserID != "user-1" {
			t.Fatalf("expected user ID user-1, got %s", result.Registrant.UserID)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		_, err := store.GetRegistrationByID(context.Background(), "nonexistent")
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestCountAcceptedRegistrationsForEvent(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns count", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "n", Value: int32(2)},
		}))
		count, err := store.CountAcceptedRegistrationsForEvent(context.Background(), "event-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if count != 2 {
			t.Fatalf("expected count 2, got %d", count)
		}
	})

	mt.Run("returns zero", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "n", Value: int32(0)},
		}))
		count, err := store.CountAcceptedRegistrationsForEvent(context.Background(), "event-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if count != 0 {
			t.Fatalf("expected count 0, got %d", count)
		}
	})
}

func TestHasAcceptedRegistration(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("exists", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "req-1"},
			{Key: "event_id", Value: "event-1"},
			{Key: "status", Value: "accepted"},
		}))
		exists, err := store.HasAcceptedRegistration(context.Background(), "event-1", "user-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !exists {
			t.Fatal("expected true, got false")
		}
	})

	mt.Run("does not exist", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		exists, err := store.HasAcceptedRegistration(context.Background(), "event-1", "user-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exists {
			t.Fatal("expected false, got true")
		}
	})
}

func TestHasPendingOrAcceptedRegistration(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("exists", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "req-1"},
			{Key: "event_id", Value: "event-1"},
			{Key: "status", Value: "pending"},
		}))
		exists, err := store.HasPendingOrAcceptedRegistration(context.Background(), "event-1", "user-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !exists {
			t.Fatal("expected true, got false")
		}
	})

	mt.Run("does not exist", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		exists, err := store.HasPendingOrAcceptedRegistration(context.Background(), "event-1", "user-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if exists {
			t.Fatal("expected false, got true")
		}
	})
}

func TestMarkRegistrationAccepted(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(1)},
			{Key: "nModified", Value: int32(1)},
		})
		if err := store.MarkRegistrationAccepted(context.Background(), "req-1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(0)},
			{Key: "nModified", Value: int32(0)},
		})
		err := store.MarkRegistrationAccepted(context.Background(), "nonexistent")
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestMarkRegistrationRejected(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(1)},
			{Key: "nModified", Value: int32(1)},
		})
		if err := store.MarkRegistrationRejected(context.Background(), "req-1", models.ReasonCapacityFull); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(0)},
			{Key: "nModified", Value: int32(0)},
		})
		err := store.MarkRegistrationRejected(context.Background(), "nonexistent", models.ReasonCapacityFull)
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestGetRegistrationStatusesForUser(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("empty userID returns empty map", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)

		result, err := store.GetRegistrationStatusesForUser(context.Background(), "", []string{"event-1"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty map, got %v", result)
		}
	})

	mt.Run("empty eventIDs returns empty map", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)

		result, err := store.GetRegistrationStatusesForUser(context.Background(), "user-1", nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty map, got %v", result)
		}
	})

	mt.Run("returns statuses for multiple events", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{
				{Key: "_id", Value: "req-1"},
				{Key: "event_id", Value: "event-1"},
				{Key: "status", Value: "accepted"},
				{Key: "registrant", Value: bson.D{{Key: "user_id", Value: "user-1"}}},
			},
			bson.D{
				{Key: "_id", Value: "req-2"},
				{Key: "event_id", Value: "event-2"},
				{Key: "status", Value: "pending"},
				{Key: "registrant", Value: bson.D{{Key: "user_id", Value: "user-1"}}},
			},
			bson.D{
				{Key: "_id", Value: "req-3"},
				{Key: "event_id", Value: "event-3"},
				{Key: "status", Value: "rejected"},
				{Key: "registrant", Value: bson.D{{Key: "user_id", Value: "user-1"}}},
			},
		))

		result, err := store.GetRegistrationStatusesForUser(context.Background(), "user-1", []string{"event-1", "event-2", "event-3"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expected := map[string]models.Status{
			"event-1": models.StatusAccepted,
			"event-2": models.StatusPending,
			"event-3": models.StatusRejected,
		}

		if len(result) != len(expected) {
			t.Fatalf("expected %d statuses, got %d", len(expected), len(result))
		}

		for eventID, status := range expected {
			if result[eventID] != status {
				t.Fatalf("expected %s => %s, got %s", eventID, status, result[eventID])
			}
		}
	})

	mt.Run("strongest status wins for same event", func(mt *mtest.T) {
		store := NewMongoStore(nil, mt.Coll, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{
				{Key: "_id", Value: "req-1"},
				{Key: "event_id", Value: "event-1"},
				{Key: "status", Value: "rejected"},
				{Key: "registrant", Value: bson.D{{Key: "user_id", Value: "user-1"}}},
			},
			bson.D{
				{Key: "_id", Value: "req-2"},
				{Key: "event_id", Value: "event-1"},
				{Key: "status", Value: "pending"},
				{Key: "registrant", Value: bson.D{{Key: "user_id", Value: "user-1"}}},
			},
			bson.D{
				{Key: "_id", Value: "req-3"},
				{Key: "event_id", Value: "event-1"},
				{Key: "status", Value: "accepted"},
				{Key: "registrant", Value: bson.D{{Key: "user_id", Value: "user-1"}}},
			},
		))

		result, err := store.GetRegistrationStatusesForUser(context.Background(), "user-1", []string{"event-1"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if result["event-1"] != models.StatusAccepted {
			t.Fatalf("expected event-1 => accepted, got %s", result["event-1"])
		}
	})
}
