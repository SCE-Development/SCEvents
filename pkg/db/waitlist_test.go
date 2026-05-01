package db

import (
	"context"
	"testing"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestHasWaitlistEntry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("found", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{{Key: "event_id", Value: "event-1"}, {Key: "user_id", Value: "user-1"}},
		))

		ok, err := store.HasWaitlistEntry(context.Background(), "event-1", "user-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !ok {
			t.Fatal("expected true, got false")
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))

		ok, err := store.HasWaitlistEntry(context.Background(), "event-1", "user-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if ok {
			t.Fatal("expected false, got true")
		}
	})
}

func TestCountWaitlistEntries(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("counts entries", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{{Key: "n", Value: int32(3)}},
		))

		count, err := store.CountWaitlistEntries(context.Background(), "event-1")
		if err != nil {
			mt.Fatalf("expected no error, got %v", err)
		}
		if count != 3 {
			mt.Fatalf("expected 3, got %d", count)
		}
	})
}

func TestCreateWaitlistEntry(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)
		mt.AddMockResponses(mtest.CreateSuccessResponse())

		err := store.CreateWaitlistEntry(context.Background(), models.WaitlistEntry{
			EventID: "event-1",
			UserID:  "user-1",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	mt.Run("insert error", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)

		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   0,
			Code:    11000,
			Message: "duplicate key error",
		}))

		err := store.CreateWaitlistEntry(context.Background(), models.WaitlistEntry{
			EventID: "event-1",
			UserID:  "user-1",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetWaitlistedEventIDsForUser(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns matching waitlisted event IDs", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{
				{Key: "event_id", Value: "event-1"},
				{Key: "user_id", Value: "user-1"},
			},
			bson.D{
				{Key: "event_id", Value: "event-2"},
				{Key: "user_id", Value: "user-1"},
			},
		))

		result, err := store.GetWaitlistedEventIDsForUser(
			context.Background(),
			"user-1",
			[]string{"event-1", "event-2", "event-3"},
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) != 2 {
			t.Fatalf("expected 2 event IDs, got %d", len(result))
		}
		if !result["event-1"] {
			t.Fatal("expected event-1 to be present")
		}
		if !result["event-2"] {
			t.Fatal("expected event-2 to be present")
		}
		if result["event-3"] {
			t.Fatal("did not expect event-3 to be present")
		}
	})

	mt.Run("returns empty map when userID is blank", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)

		result, err := store.GetWaitlistedEventIDsForUser(context.Background(), "", []string{"event-1"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(result))
		}
	})

	mt.Run("returns empty map when eventIDs is empty", func(mt *mtest.T) {
		store := NewMongoStore(nil, nil, mt.Coll)

		result, err := store.GetWaitlistedEventIDsForUser(context.Background(), "user-1", nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) != 0 {
			t.Fatalf("expected empty map, got %d entries", len(result))
		}
	})
}
