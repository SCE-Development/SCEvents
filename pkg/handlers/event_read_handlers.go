package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/gin-gonic/gin"
)

func (h *EventHandler) GetEvents(c *gin.Context) {
	startQ := strings.TrimSpace(c.Query("startDate"))
	endQ := strings.TrimSpace(c.Query("endDate"))

	var startDate, endDate string
	switch {
	case startQ == "" && endQ == "":
		now := time.Now().UTC()
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		last := first.AddDate(0, 1, -1)
		startDate = first.Format(dateLayout)
		endDate = last.Format(dateLayout)
	case startQ != "" && endQ != "":
		if _, err := time.Parse(dateLayout, startQ); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid startDate", "expected": dateLayout})
			return
		}
		if _, err := time.Parse(dateLayout, endQ); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid endDate", "expected": dateLayout})
			return
		}
		if startQ > endQ {
			c.JSON(http.StatusBadRequest, gin.H{"error": "startDate must be on or before endDate"})
			return
		}
		startDate, endDate = startQ, endQ
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provide both startDate and endDate query params, or neither for the current month default",
		})
		return
	}

	viewer := buildViewerFromContext(c)

	events, err := h.stores.Mongo.GetVisibleEvents(c.Request.Context(), viewer, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch events",
		})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusOK, buildEventResponses(events, "", nil, nil))
		return
	}

	eventIDs := collectEventIDs(events)

	registrationStatuses, err := h.stores.Mongo.GetRegistrationStatusesForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch registration statuses",
		})
		return
	}

	waitlistedEventIDs, err := h.stores.Mongo.GetWaitlistedEventIDsForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch waitlist statuses",
		})
		return
	}

	responses := buildEventResponses(events, userID, registrationStatuses, waitlistedEventIDs)
	c.JSON(http.StatusOK, responses)
}

func (h *EventHandler) GetEventByID(c *gin.Context) {
	id := c.Param("id")
	viewer := buildViewerFromContext(c)

	event, err := h.stores.Mongo.GetVisibleEventByID(c.Request.Context(), viewer, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "event not found",
		})
		return
	}

	userID := strings.TrimSpace(c.GetString("userID"))
	if userID == "" {
		c.JSON(http.StatusOK, buildEventResponse(*event, nil))
		return
	}

	eventIDs := []string{event.ID}

	registrationStatuses, err := h.stores.Mongo.GetRegistrationStatusesForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch registration status",
		})
		return
	}

	waitlistedEventIDs, err := h.stores.Mongo.GetWaitlistedEventIDsForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch waitlist status",
		})
		return
	}

	status := resolveEventRegistrationStatus(event.ID, registrationStatuses, waitlistedEventIDs)
	statusCopy := status

	c.JSON(http.StatusOK, buildEventResponse(*event, &statusCopy))
}

func buildEventResponse(event models.Event, registrationStatus *EventRegistrationStatus) EventResponse {
	return EventResponse{
		ID:                 event.ID,
		Name:               event.Name,
		Date:               event.Date,
		EndDate:            event.EndDate,
		Time:               event.Time,
		Location:           event.Location,
		Description:        event.Description,
		Admins:             event.Admins,
		RegistrationForm:   event.RegistrationForm,
		MaxAttendees:       event.MaxAttendees,
		CreatedAt:          event.CreatedAt,
		Status:             event.Status,
		Visibility:         event.Visibility,
		MinimumVisibleRole: event.MinimumVisibleRole,
		WaitlistEnabled:    event.WaitlistEnabled,
		WaitlistSize:       event.WaitlistSize,
		PublishDate:        event.PublishDate,
		PublishedAt:        event.PublishedAt,
		RegistrationStatus: registrationStatus,
	}
}

func resolveEventRegistrationStatus(
	eventID string,
	registrationStatuses map[string]models.Status,
	waitlistedEventIDs map[string]bool,
) EventRegistrationStatus {
	if status, ok := registrationStatuses[eventID]; ok {
		switch status {
		case models.StatusAccepted:
			return EventRegistrationStatusRegistered
		case models.StatusPending:
			return EventRegistrationStatusPending
		case models.StatusRejected:
			if waitlistedEventIDs[eventID] {
				return EventRegistrationStatusWaitlisted
			}
			return EventRegistrationStatusRejected
		}
	}

	if waitlistedEventIDs[eventID] {
		return EventRegistrationStatusWaitlisted
	}

	return EventRegistrationStatusNone
}

func collectEventIDs(events []models.Event) []string {
	ids := make([]string, 0, len(events))
	for _, event := range events {
		ids = append(ids, event.ID)
	}
	return ids
}

func buildEventResponses(
	events []models.Event,
	userID string,
	registrationStatuses map[string]models.Status,
	waitlistedEventIDs map[string]bool,
) []EventResponse {
	responses := make([]EventResponse, 0, len(events))

	for _, event := range events {
		if strings.TrimSpace(userID) == "" {
			responses = append(responses, buildEventResponse(event, nil))
			continue
		}

		status := resolveEventRegistrationStatus(event.ID, registrationStatuses, waitlistedEventIDs)
		statusCopy := status
		responses = append(responses, buildEventResponse(event, &statusCopy))
	}

	return responses
}
