package handlers
import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"

	"github.com/SCE-Development/SCEvents/pkg/db"
	eventtypes "github.com/SCE-Development/SCEvents/pkg/event"
)
func GetEventsHandler(c *gin.Context) {
	coll := db.Database().Collection("events")
	ctx := c.Request.Context()

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch events",
		})
		return
	}
	defer cursor.Close(ctx)

	var events []eventtypes.Event
	if err := cursor.All(ctx, &events); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to decode events",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
	})
}