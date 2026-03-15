package db

import (
	"context"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	client   *mongo.Client
	database *mongo.Database
)

const (
	defaultURI = "mongodb://localhost:27017"
	dbName     = "scevents"
)

// Connect initializes the global MongoDB client and database using the provided URI.
// If the URI is empty, it falls back to MONGO_URI and then to a local default.
func Connect(uri string) error {
	if uri == "" {
		uri = os.Getenv("MONGO_URI")
	}
	if uri == "" {
		uri = defaultURI
	}

	ctx := context.Background()
	c, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}

	client = c
	database = c.Database(dbName)

	return nil
}

// Disconnect closes the global MongoDB client if it has been initialized.
func Disconnect() error {
	if client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Disconnect(ctx)
	client = nil
	database = nil

	return err
}

// Client returns the initialized MongoDB client.
func Client() *mongo.Client {
	return client
}

// Database returns the initialized MongoDB database handle.
func Database() *mongo.Database {
	return database
}

func GetEventsCollection() *mongo.Collection {
	return Database().Collection("events")
}