package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

type PublicAttendee struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
}

func formatPublicAttendeeName(fullName string) string {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return "Registered attendee"
	}

	parts := strings.Fields(fullName)
	if len(parts) == 1 {
		return parts[0]
	}

	last := []rune(parts[len(parts)-1])
	if len(last) == 0 {
		return parts[0]
	}

	return parts[0] + " " + string(last[0]) + "."
}

func (h *EventHandler) GetEventAttendanceSummary(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("id"))
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event id is required",
		})
		return
	}

	_, err := h.stores.Mongo.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch event",
		})
		return
	}

	attendeeCount, err := h.stores.Mongo.CountAcceptedRegistrationsForEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch attendance summary",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":       eventID,
		"attendee_count": attendeeCount,
	})
}

func (h *EventHandler) GetEventAttendees(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("id"))
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "event id is required",
		})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "login required",
		})
		return
	}
	userRole := strings.TrimSpace(c.GetString("userRole"))

	event, err := h.stores.Mongo.GetEventByID(c.Request.Context(), eventID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch event",
		})
		return
	}
	canView := event.CanEdit(userID, userRole)
	if !canView {
		hasAccepted, err := h.stores.Mongo.HasAcceptedRegistration(c.Request.Context(), eventID, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to verify attendee access",
			})
			return
		}
		canView = hasAccepted
	}

	if !canView {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "only registered attendees and event organizers can view the attendee list",
		})
		return
	}

	registrations, err := h.stores.Mongo.ListAcceptedRegistrationsForEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch attendees",
		})
		return
	}

	attendees := make([]PublicAttendee, 0, len(registrations))
	for _, r := range registrations {
		attendees = append(attendees, PublicAttendee{
			UserID: strings.TrimSpace(r.Registrant.UserID),
			Name:   formatPublicAttendeeName(r.Registrant.Name),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"event_id":  eventID,
		"attendees": attendees,
	})
}
