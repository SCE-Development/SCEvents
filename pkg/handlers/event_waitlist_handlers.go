package handlers

import (
	"net/http"
	"strings"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func (h *EventHandler) JoinEventWaitlist(c *gin.Context) {
	eventID := c.Param("id")
	if strings.TrimSpace(eventID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event id is required",
		})
		return
	}

	userID := c.GetString("userID")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "login required to join waitlist",
		})
		return
	}

	ev, err := h.stores.Mongo.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	if !ev.WaitlistEnabled {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "waitlist is not enabled for this event",
		})
		return
	}

	if ev.WaitlistSize <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "waitlist is not configured for this event",
		})
		return
	}

	alreadyRegistered, err := h.stores.Mongo.HasAcceptedRegistration(c.Request.Context(), eventID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to verify existing registration",
		})
		return
	}
	if alreadyRegistered {
		c.JSON(http.StatusConflict, gin.H{
			"error": "user is already registered for this event",
		})
		return
	}

	alreadyWaitlisted, err := h.stores.Mongo.HasWaitlistEntry(c.Request.Context(), eventID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to verify existing waitlist entry",
		})
		return
	}
	if alreadyWaitlisted {
		c.JSON(http.StatusConflict, gin.H{
			"error": "user is already on the waitlist for this event",
		})
		return
	}

	count, err := h.stores.Mongo.CountWaitlistEntries(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to count waitlist entries",
		})
		return
	}

	if count >= int64(ev.WaitlistSize) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "waitlist is full",
		})
		return
	}

	entry := models.WaitlistEntry{
		EventID: eventID,
		UserID:  userID,
	}

	if err := h.stores.Mongo.CreateWaitlistEntry(c.Request.Context(), entry); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "user is already on the waitlist for this event",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to join waitlist",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "joined waitlist successfully",
		"event_id": eventID,
	})
}
