package handlers
import (
	"net/http"
	"github.com/gin-gonic/gin"
	"github.com/SCE-Development/SCEvents/pkg/db"
)

func GetEventsHandler(c *gin.Context) {
	events, err := db.GetEvents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch events",
		})
		return
	}
	c.JSON(http.StatusOK, events)
}