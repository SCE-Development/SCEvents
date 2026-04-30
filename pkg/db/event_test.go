package db

import (
	"context"
	"testing"
	"time"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestCreateEvent(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		event := models.Event{
			ID:           "event-1",
			Name:         "Test Event",
			Date:         "2026-05-01",
			Time:         "10:00",
			Location:     "Room 101",
			MaxAttendees: 50,
		}
		result, err := store.CreateEvent(context.Background(), event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ID != "event-1" {
			t.Fatalf("expected ID event-1, got %s", result.ID)
		}
	})

	mt.Run("generates ID when empty", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		mt.AddMockResponses(mtest.CreateSuccessResponse())
		event := models.Event{
			Name:     "Test Event",
			Date:     "2026-05-01",
			Time:     "10:00",
			Location: "Room 101",
		}
		result, err := store.CreateEvent(context.Background(), event)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ID == "" {
			t.Fatal("expected generated ID, got empty")
		}
	})

	mt.Run("duplicate key error", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		mt.AddMockResponses(mtest.CreateWriteErrorsResponse(mtest.WriteError{
			Index:   0,
			Code:    11000,
			Message: "duplicate key error",
		}))
		event := models.Event{
			ID:   "event-1",
			Name: "Test Event",
			Date: "2026-05-01",
			Time: "10:00",
		}
		_, err := store.CreateEvent(context.Background(), event)
		if err == nil {
			t.Fatal("expected error for duplicate key")
		}
	})
}

func TestGetEventByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch, bson.D{
			{Key: "_id", Value: "event-1"},
			{Key: "name", Value: "Test Event"},
			{Key: "date", Value: "2026-05-01"},
			{Key: "time", Value: "10:00"},
			{Key: "location", Value: "Room 101"},
			{Key: "max_attendees", Value: int32(50)},
		}))
		result, err := store.GetEventByID(context.Background(), "event-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if result.ID != "event-1" {
			t.Fatalf("expected ID event-1, got %s", result.ID)
		}
		if result.Name != "Test Event" {
			t.Fatalf("expected name 'Test Event', got %s", result.Name)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()
		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))
		_, err := store.GetEventByID(context.Background(), "nonexistent")
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestDeleteEventByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: int32(1)}})
		if err := store.DeleteEventByID(context.Background(), "event-1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		mt.AddMockResponses(bson.D{{Key: "ok", Value: 1}, {Key: "n", Value: int32(0)}})
		err := store.DeleteEventByID(context.Background(), "nonexistent")
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestUpdateEventByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("success", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(1)},
			{Key: "nModified", Value: int32(1)},
		})
		err := store.UpdateEventByID(context.Background(), "event-1", map[string]interface{}{
			"name": "Updated Name",
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(0)},
			{Key: "nModified", Value: int32(0)},
		})
		err := store.UpdateEventByID(context.Background(), "nonexistent", map[string]interface{}{
			"name": "Updated Name",
		})
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestGetVisibleEvents(t *testing.T) {
func TestGetVisibleEvents(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns matching visible events", func(mt *mtest.T) {
	mt.Run("returns matching visible events", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{
				{Key: "_id", Value: "event-1"},
				{Key: "name", Value: "Event One"},
				{Key: "status", Value: models.StatusPublished},
				{Key: "visibility", Value: models.VisibilityPublic},
				{Key: "status", Value: models.StatusPublished},
				{Key: "visibility", Value: models.VisibilityPublic},
				{Key: "date", Value: "2026-05-01"},
				{Key: "time", Value: "10:00"},
				{Key: "location", Value: "Room 101"},
			},
			bson.D{
				{Key: "_id", Value: "event-2"},
				{Key: "name", Value: "Event Two"},
				{Key: "status", Value: models.StatusPublished},
				{Key: "visibility", Value: models.VisibilityPublic},
				{Key: "status", Value: models.StatusPublished},
				{Key: "visibility", Value: models.VisibilityPublic},
				{Key: "date", Value: "2026-05-15"},
				{Key: "time", Value: "14:00"},
				{Key: "location", Value: "Room 202"},
			},
		))

		events, err := store.GetVisibleEvents(context.Background(), models.EventViewer{}, "2026-05-01", "2026-05-31")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}
		if events[0].ID != "event-1" {
			t.Fatalf("expected first event ID event-1, got %s", events[0].ID)
		}
	})

	mt.Run("empty result", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))

		events, err := store.GetVisibleEvents(context.Background(), models.EventViewer{}, "2026-05-01", "2026-05-31")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(events) != 0 {
			t.Fatalf("expected 0 events, got %d", len(events))
		}
	})
}

func TestGetVisibleEventByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{
				{Key: "_id", Value: "event-1"},
				{Key: "name", Value: "Visible Event"},
				{Key: "status", Value: models.StatusPublished},
				{Key: "visibility", Value: models.VisibilityPublic},
				{Key: "date", Value: "2026-05-01"},
				{Key: "time", Value: "10:00"},
				{Key: "location", Value: "Room 101"},
			},
		))

		event, err := store.GetVisibleEventByID(context.Background(), models.EventViewer{}, "event-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if event.ID != "event-1" {
			t.Fatalf("expected event-1, got %s", event.ID)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))

		_, err := store.GetVisibleEventByID(context.Background(), models.EventViewer{}, "missing")
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestPublishDueEvents(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("publishes due draft events", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(2)},
			{Key: "nModified", Value: int32(2)},
		})

		updated, err := store.PublishDueEvents(context.Background(), time.Now().UTC())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if updated != 2 {
			t.Fatalf("expected 2 updated docs, got %d", updated)
		}
	})

	mt.Run("returns error on command failure", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    10107,
			Message: "not primary",
		}))

		_, err := store.PublishDueEvents(context.Background(), time.Now().UTC())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestGetVisibleEventByID(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch,
			bson.D{
				{Key: "_id", Value: "event-1"},
				{Key: "name", Value: "Visible Event"},
				{Key: "status", Value: models.StatusPublished},
				{Key: "visibility", Value: models.VisibilityPublic},
				{Key: "date", Value: "2026-05-01"},
				{Key: "time", Value: "10:00"},
				{Key: "location", Value: "Room 101"},
			},
		))

		event, err := store.GetVisibleEventByID(context.Background(), models.EventViewer{}, "event-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if event.ID != "event-1" {
			t.Fatalf("expected event-1, got %s", event.ID)
		}
	})

	mt.Run("not found", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)
		ns := mt.Coll.Database().Name() + "." + mt.Coll.Name()

		mt.AddMockResponses(mtest.CreateCursorResponse(0, ns, mtest.FirstBatch))

		_, err := store.GetVisibleEventByID(context.Background(), models.EventViewer{}, "missing")
		if err != mongo.ErrNoDocuments {
			t.Fatalf("expected ErrNoDocuments, got %v", err)
		}
	})
}

func TestPublishDueEvents(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("publishes due draft events", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)

		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: int32(2)},
			{Key: "nModified", Value: int32(2)},
		})

		updated, err := store.PublishDueEvents(context.Background(), time.Now().UTC())
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if updated != 2 {
			t.Fatalf("expected 2 updated docs, got %d", updated)
		}
	})

	mt.Run("returns error on command failure", func(mt *mtest.T) {
		store := NewMongoStore(mt.Coll, nil, nil)

		mt.AddMockResponses(mtest.CreateCommandErrorResponse(mtest.CommandError{
			Code:    10107,
			Message: "not primary",
		}))

		_, err := store.PublishDueEvents(context.Background(), time.Now().UTC())
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
