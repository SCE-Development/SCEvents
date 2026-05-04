package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func (h *EventHandler) CreateEvent(c *gin.Context) {
	var event models.Event

	if err := c.BindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	event.ApplyDefaults()
	event.SyncPublicationState(time.Now().UTC())

	creatorID := strings.TrimSpace(c.GetString("userID"))
	if creatorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "creator user id is required",
		})
		return
	}

	event.Admins = sanitizeAdmins(event.Admins)
	event.Admins = ensureCreatorIsEventAdmin(event.Admins, creatorID)

	if err := event.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	createdEvent, err := h.stores.Mongo.CreateEvent(c.Request.Context(), event)
	if err != nil {
		log.Printf("CreateEvent error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create event",
		})
		return
	}

	if createdEvent.MaxAttendees != -1 {
		if err := h.stores.Redis.SetEventHeadcount(c.Request.Context(), createdEvent.ID, createdEvent.MaxAttendees); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to store event headcount",
			})
			return
		}
	}

	c.JSON(http.StatusCreated, createdEvent)
}

func sanitizeAdmins(admins []string) []string {
	seen := make(map[string]struct{}, len(admins))
	normalized := make([]string, 0, len(admins))

	for _, admin := range admins {
		admin = strings.TrimSpace(admin)
		if admin == "" {
			continue
		}
		if _, ok := seen[admin]; ok {
			continue
		}
		seen[admin] = struct{}{}
		normalized = append(normalized, admin)
	}

	return normalized
}

func ensureCreatorIsEventAdmin(admins []string, creatorID string) []string {
	for _, admin := range admins {
		if admin == creatorID {
			return admins
		}
	}
	return append(admins, creatorID)
}

func (h *EventHandler) DeleteEventByID(c *gin.Context) {
	id := c.Param("id")

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	existingEvent, err := h.stores.Mongo.GetEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	if !existingEvent.CanEdit(userID, userRole) {
		writeEventEditForbidden(c, existingEvent)
		return
	}

	if existingEvent.Status != models.StatusDraft && existingEvent.Status != models.StatusClosed {
		c.JSON(http.StatusConflict, gin.H{
			"error": "only draft or closed events can be deleted; close the event first",
		})
		return
	}

	err = h.stores.Mongo.DeleteEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	if h.stores.Redis != nil && existingEvent.MaxAttendees != -1 {
		if err := h.stores.Redis.DeleteEventHeadcount(c.Request.Context(), id); err != nil {
			log.Printf("delete event: redis headcount cleanup failed for %s: %v", id, err)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "event deleted successfully",
	})
}

func (h *EventHandler) syncMaxAttendeesHeadcount(ctx context.Context, id string, existingEvent *models.Event, fields map[string]interface{}) error {
	maxAttendees, ok := fields["max_attendees"]
	if !ok {
		return nil
	}

	newMax, ok := maxAttendees.(float64)
	if !ok {
		return nil
	}

	newMaxInt := int(newMax)

	switch {
	case newMaxInt == -1:
		return h.stores.Redis.DeleteEventHeadcount(ctx, id)
	case existingEvent.MaxAttendees == -1:
		return h.stores.Redis.SetEventHeadcount(ctx, id, newMaxInt)
	default:
		currentRemaining, err := h.stores.Redis.GetEventHeadcount(ctx, id)
		if err != nil {
			currentRemaining = existingEvent.MaxAttendees
		}

		seatsTaken := existingEvent.MaxAttendees - currentRemaining
		newRemaining := newMaxInt - seatsTaken
		if newRemaining < 0 {
			newRemaining = 0
		}

		return h.stores.Redis.SetEventHeadcount(ctx, id, newRemaining)
	}
}

func (h *EventHandler) UpdateEventByID(c *gin.Context) {
	id := c.Param("id")

	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	existingEvent, err := h.stores.Mongo.GetEventByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	if !existingEvent.CanEdit(userID, userRole) {
		writeEventEditForbidden(c, existingEvent)
		return
	}

	var fields map[string]interface{}
	if err := c.BindJSON(&fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid JSON payload",
		})
		return
	}

	models.SanitizeUpdateFields(fields)

	if len(fields) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "no updatable fields provided",
		})
		return
	}

	updatedEvent := existingEvent
	if err := updatedEvent.ApplyPatch(fields); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	updatedEvent.SyncPublicationState(time.Now().UTC())

	if updatedEvent.Status == models.StatusClosed {
		updatedEvent.PublishDate = nil
		fields["publish_date"] = nil
	}

	if err := updatedEvent.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if _, ok := fields["visibility"]; ok {
		fields["visibility"] = updatedEvent.Visibility
		fields["minimum_visible_role"] = updatedEvent.MinimumVisibleRole
	}

	if _, ok := fields["minimum_visible_role"]; ok {
		fields["minimum_visible_role"] = updatedEvent.MinimumVisibleRole
	}

	if _, ok := fields["admins"]; ok {
		fields["admins"] = sanitizeAdmins(updatedEvent.Admins)
	}

	if _, ok := fields["status"]; ok && updatedEvent.Status == models.StatusPublished {
		fields["publish_date"] = updatedEvent.PublishDate
		fields["published_at"] = updatedEvent.PublishedAt
	}

	if _, ok := fields["publish_date"]; ok {
		fields["publish_date"] = updatedEvent.PublishDate
		if updatedEvent.Status == models.StatusPublished {
			fields["status"] = updatedEvent.Status
			fields["published_at"] = updatedEvent.PublishedAt
		}
	}

	err = h.stores.Mongo.UpdateEventByID(c.Request.Context(), id, fields)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to update event",
			})
		}
		return
	}

	if err := h.syncMaxAttendeesHeadcount(c.Request.Context(), id, existingEvent, fields); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "event updated but failed to sync headcount in Redis",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "event updated successfully",
	})
}
