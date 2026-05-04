package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/SCE-Development/SCEvents/pkg/registration"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

func (h *EventHandler) RegisterForEvent(producer *registration.Producer) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventID := c.Param("id")
		if eventID == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "event id is required",
			})
			return
		}

		var payload models.RegistrationPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "invalid JSON payload",
			})
			return
		}

		if strings.TrimSpace(payload.Registrant.UserID) == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "login required to register for event",
			})
			return
		}

		alreadyRegistered, err := h.stores.Mongo.HasPendingOrAcceptedRegistration(c.Request.Context(), eventID, payload.Registrant.UserID)
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

		ev, err := h.stores.Mongo.GetEventByID(c.Request.Context(), eventID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "event not found",
			})
			return
		}

		if ev.Status == models.StatusClosed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "registration is closed for this event",
			})
			return
		}

		if ev.IsListedAdmin(payload.Registrant.UserID) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "event admins cannot register for their own event",
			})
			return
		}

		if err := ev.ValidateRegistration(payload.RegistrationFormAnswers); err != nil {
			var formErr *models.RegistrationFormValidationError
			if errors.As(err, &formErr) {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": formErr.Message,
					"field": formErr.Field,
				})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to validate registration",
			})
			return
		}

		var requestIDBytes [16]byte
		if _, err := rand.Read(requestIDBytes[:]); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create registration request",
			})
			return
		}

		req := models.RegistrationRequest{
			RequestID:  hex.EncodeToString(requestIDBytes[:]),
			EventID:    eventID,
			Registrant: payload.Registrant,
			Answers:    payload.RegistrationFormAnswers,
		}

		created, err := h.stores.Mongo.CreatePendingRegistration(c.Request.Context(), req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to create registration request",
			})
			return
		}

		seatTaken := true
		if ev.MaxAttendees != -1 {
			seatTaken, err = h.stores.Redis.TryTakeEventSeat(c.Request.Context(), eventID)
			if err != nil {
				_ = h.stores.Mongo.MarkRegistrationRejected(c.Request.Context(), created.RequestID, models.ReasonInternalError)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "failed to reserve event seat",
				})
				return
			}
			if !seatTaken {
				_ = h.stores.Mongo.MarkRegistrationRejected(c.Request.Context(), created.RequestID, models.ReasonCapacityFull)
				c.JSON(http.StatusConflict, gin.H{
					"error": "event is full",
				})
				return
			}
		}

		if err := h.stores.Mongo.MarkRegistrationAccepted(c.Request.Context(), created.RequestID); err != nil {
			if ev.MaxAttendees != -1 && seatTaken {
				_ = h.stores.Redis.ReleaseEventSeat(c.Request.Context(), eventID)
			}
			_ = h.stores.Mongo.MarkRegistrationRejected(c.Request.Context(), created.RequestID, models.ReasonInternalError)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to send registration request",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"message":    "registration request sent",
			"event_id":   eventID,
			"request_id": created.RequestID,
		})
	}
}

func (h *EventHandler) GetRegistrationStatus(c *gin.Context) {
	requestID := c.Param("request_id")
	if strings.TrimSpace(requestID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "request id is required",
		})
		return
	}

	userID := c.Query("user_id")
	if strings.TrimSpace(userID) == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "missing user_id query parameter",
		})
		return
	}

	req, err := h.stores.Mongo.GetRegistrationByID(c.Request.Context(), requestID)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "registration request not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to fetch registration status",
		})
		return
	}

	if req.Registrant.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "you do not have access to this registration request",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"request_id":      req.RequestID,
		"event_id":        req.EventID,
		"status":          req.Status,
		"decision_reason": req.DecisionReason,
		"created_at":      req.CreatedAt,
		"updated_at":      req.UpdatedAt,
		"processed_at":    req.ProcessedAt,
	})
}

func (h *EventHandler) GetMyRegistrationState(c *gin.Context) {
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

	eventIDs := []string{eventID}
	registrationStatuses, err := h.stores.Mongo.GetRegistrationStatusesForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch registration status"})
		return
	}

	waitlistedEventIDs, err := h.stores.Mongo.GetWaitlistedEventIDsForUser(c.Request.Context(), userID, eventIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch waitlist status"})
		return
	}

	status := resolveEventRegistrationStatus(eventID, registrationStatuses, waitlistedEventIDs)
	c.JSON(http.StatusOK, gin.H{
		"registration_status": status,
	})
}
