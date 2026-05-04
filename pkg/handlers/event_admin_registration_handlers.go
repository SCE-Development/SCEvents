package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func (h *EventHandler) loadEventAndAuthorizeAdmin(ctx context.Context, eventID, userID, userRole string) bool {
	ev, err := h.stores.Mongo.GetEventByID(ctx, eventID)
	if err != nil {
		return false
	}
	return ev.CanEdit(userID, userRole)
}

func parsePagination(c *gin.Context) (int64, int64, bool) {
	limit := int64(50)
	offset := int64(0)

	if limitQ := strings.TrimSpace(c.Query("limit")); limitQ != "" {
		parsed, err := strconv.ParseInt(limitQ, 10, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
			return 0, 0, false
		}
		limit = parsed
	}

	if offsetQ := strings.TrimSpace(c.Query("offset")); offsetQ != "" {
		parsed, err := strconv.ParseInt(offsetQ, 10, 64)
		if err != nil || parsed < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be a non-negative integer"})
			return 0, 0, false
		}
		offset = parsed
	}

	return limit, offset, true
}

func (h *EventHandler) ListEventRegistrations(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("id"))
	if eventID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event id is required"})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	userRole := strings.TrimSpace(c.GetString("userRole"))

	limit, offset, ok := parsePagination(c)
	if !ok {
		return
	}

	if !h.loadEventAndAuthorizeAdmin(c.Request.Context(), eventID, userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not an admin of this event"})
		return
	}

	registrations, err := h.stores.Mongo.ListRegistrationsByEventID(c.Request.Context(), eventID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch event registrations"})
		return
	}

	statusCounts, err := h.stores.Mongo.CountRegistrationsByStatusForEvent(c.Request.Context(), eventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch registration dashboard summary"})
		return
	}

	summary := models.RegistrationDashboardSummary{
		Pending:  statusCounts[models.StatusPending],
		Accepted: statusCounts[models.StatusAccepted],
		Rejected: statusCounts[models.StatusRejected],
	}
	summary.Total = summary.Pending + summary.Accepted + summary.Rejected

	c.JSON(http.StatusOK, gin.H{
		"event_id":      eventID,
		"summary":       summary,
		"attendees":     registrations,
		"registrations": registrations,
		"limit":         limit,
		"offset":        offset,
	})
}

func (h *EventHandler) GetEventRegistrationByRequestID(c *gin.Context) {
	eventID := strings.TrimSpace(c.Param("id"))
	requestID := strings.TrimSpace(c.Param("request_id"))
	if eventID == "" || requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "event id and request id are required"})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "login required"})
		return
	}
	userRole := strings.TrimSpace(c.GetString("userRole"))

	if !h.loadEventAndAuthorizeAdmin(c.Request.Context(), eventID, userID, userRole) {
		c.JSON(http.StatusForbidden, gin.H{"error": "you are not an admin of this event"})
		return
	}

	registrationReq, err := h.stores.Mongo.GetRegistrationByEventAndRequestID(c.Request.Context(), eventID, requestID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{"error": "registration request not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch registration request"})
		return
	}

	c.JSON(http.StatusOK, registrationReq)
}
