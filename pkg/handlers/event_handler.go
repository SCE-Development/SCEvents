package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/db"
	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	stores *db.Stores
}

func NewEventHandler(stores *db.Stores) *EventHandler {
	return &EventHandler{stores: stores}
}

const dateLayout = "2006-01-02"

type EventRegistrationStatus string

const (
	EventRegistrationStatusNone       EventRegistrationStatus = "none"
	EventRegistrationStatusPending    EventRegistrationStatus = "pending"
	EventRegistrationStatusRegistered EventRegistrationStatus = "registered"
	EventRegistrationStatusWaitlisted EventRegistrationStatus = "waitlisted"
	EventRegistrationStatusRejected   EventRegistrationStatus = "rejected"
)

type EventResponse struct {
	ID                 string                   `json:"id"`
	Name               string                   `json:"name"`
	Date               string                   `json:"date"`
	EndDate            string                   `json:"end_date,omitempty"`
	Time               string                   `json:"time"`
	Location           string                   `json:"location"`
	Description        string                   `json:"description"`
	Admins             []string                 `json:"admins"`
	RegistrationForm   []models.FormQuestion    `json:"registration_form"`
	MaxAttendees       int                      `json:"max_attendees"`
	CreatedAt          string                   `json:"created_at"`
	Status             string                   `json:"status"`
	Visibility         string                   `json:"visibility"`
	MinimumVisibleRole string                   `json:"minimum_visible_role,omitempty"`
	WaitlistEnabled    bool                     `json:"waitlist_enabled"`
	WaitlistSize       int                      `json:"waitlist_size,omitempty"`
	PublishDate        *time.Time               `json:"publish_date,omitempty"`
	PublishedAt        *time.Time               `json:"published_at,omitempty"`
	RegistrationStatus *EventRegistrationStatus `json:"registration_status,omitempty"`
}

func writeEventEditForbidden(c *gin.Context, ev *models.Event) {
	if len(ev.Admins) == 0 {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "this event has no dedicated admins; only site admins may modify it",
		})
		return
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": "you are not an admin of this event",
	})
}

func buildViewerFromContext(c *gin.Context) models.EventViewer {
	accessLevel := 0
	if v, exists := c.Get("accessLevel"); exists {
		if n, ok := v.(int); ok {
			accessLevel = n
		}
	}

	return models.EventViewer{
		UserID:      strings.TrimSpace(c.GetString("userID")),
		AccessLevel: accessLevel,
	}
}
