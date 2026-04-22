package registration

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
		store := NewMongoStore(mt.Coll)
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
		store := NewMongoStore(mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{"_id", "req-1"},
			{"event_id", "event-1"},
			{"status", "pending"},
			{"registrant", bson.D{
				{"name", "John Doe"},
				{"email", "john@example.com"},
				{"user_id", "user-1"},
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
		store := NewMongoStore(mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		_, err := store.GetRegistrationByID(context.Background(), "nonexistent")
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestHasAcceptedRegistration(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("exists", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{"_id", "req-1"},
			{"event_id", "event-1"},
			{"status", "accepted"},
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
		store := NewMongoStore(mt.Coll)
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
		store := NewMongoStore(mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{"_id", "req-1"},
			{"event_id", "event-1"},
			{"status", "pending"},
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
		store := NewMongoStore(mt.Coll)
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
		store := NewMongoStore(mt.Coll)
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"n", int32(1)},
			{"nModified", int32(1)},
		})
		if err := store.MarkRegistrationAccepted(context.Background(), "req-1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll)
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"n", int32(0)},
			{"nModified", int32(0)},
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
		store := NewMongoStore(mt.Coll)
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"n", int32(1)},
			{"nModified", int32(1)},
		})
		if err := store.MarkRegistrationRejected(context.Background(), "req-1", models.ReasonCapacityFull); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll)
		mt.AddMockResponses(bson.D{
			{"ok", 1},
			{"n", int32(0)},
			{"nModified", int32(0)},
		})
		err := store.MarkRegistrationRejected(context.Background(), "nonexistent", models.ReasonCapacityFull)
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}
