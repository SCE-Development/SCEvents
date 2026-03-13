package main

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	// "github.com/SCE-Development/SCEvents/pkg/types"
)

var mongoClient *mongo.Client
var mongoDB *mongo.Database

func main() {
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}

	ctx := context.Background()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = client.Disconnect(ctx)
	}()

	mongoClient = client
	mongoDB = client.Database("scevents")

	r := gin.Default()

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "response",
		})
	})

	r.Run()
}
