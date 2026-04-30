package models

import (
	"fmt"
	"strings"
	"time"
)

const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusClosed    = "closed"
)

const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

const (
	RoleMember  = "member"
	RoleOfficer = "officer"
	RoleAdmin   = "admin"
)

type AnswerDetails struct {
	MaxChars int `bson:"max_chars,omitempty" json:"max_chars,omitempty"`
}

type FormAnswer struct {
	Value any `bson:"value,omitempty" json:"value,omitempty"` // can be string, []string, number, boolean, or nil
}

type FormQuestion struct {
	ID            string         `bson:"id" json:"id"`
	Type          string         `bson:"type" json:"type"` // textbox, multiple_choice, dropdown, checkbox
	Question      string         `bson:"question" json:"question"`
	Required      bool           `bson:"required" json:"required"`
	AnswerDetails *AnswerDetails `bson:"answer_details,omitempty" json:"answer_details,omitempty"`
	AnswerOptions []string       `bson:"answer_options,omitempty" json:"answer_options,omitempty"`
}

type Event struct {
	ID               string         `bson:"_id" json:"id"`
	Name             string         `bson:"name" json:"name"`
	Date             string         `bson:"date" json:"date"`
	EndDate          string         `bson:"end_date,omitempty" json:"end_date,omitempty"`
	Time             string         `bson:"time" json:"time"`
	Location         string         `bson:"location" json:"location"`
	Description      string         `bson:"description" json:"description"`
	Admins           []string       `bson:"admins" json:"admins" default:"[]"`
	RegistrationForm []FormQuestion `bson:"registration_form" json:"registration_form" default:"[]"`
	// MaxAttendees controls event capacity
	// -1 means unlimited capacity (no Redis headcount tracking)
	// Values > 0 are tracked in Redis for concurrency-safe registration
	MaxAttendees       int        `bson:"max_attendees" json:"max_attendees" default:"-1"`
	CreatedAt          string     `bson:"created_at" json:"created_at"`
	Status             string     `bson:"status" json:"status" default:"draft"`
	Visibility         string     `bson:"visibility" json:"visibility" default:"public"`
	MinimumVisibleRole string     `bson:"minimum_visible_role,omitempty" json:"minimum_visible_role,omitempty"`
	WaitlistEnabled    bool       `bson:"waitlist_enabled" json:"waitlist_enabled"`
	WaitlistSize       int        `bson:"waitlist_size,omitempty" json:"waitlist_size,omitempty"`
	PublishDate        *time.Time `bson:"publish_date,omitempty" json:"publish_date,omitempty"`
	PublishedAt        *time.Time `bson:"published_at,omitempty" json:"published_at,omitempty"`
}

type RegistrationFormValidationError struct {
	Field   string
	Message string
}

func (e *RegistrationFormValidationError) Error() string {
	return e.Message
}

// ApplyDefaults sets safe defaults for newly created events.
func (e *Event) ApplyDefaults() {
	if strings.TrimSpace(e.Status) == "" {
		e.Status = StatusDraft
	}

	if strings.TrimSpace(e.Visibility) == "" {
		e.Visibility = VisibilityPublic
	}

	e.normalize()
}

// normalize enforces internal consistency for dependent fields.
func (e *Event) normalize() {
	e.Status = strings.TrimSpace(e.Status)
	e.Visibility = strings.TrimSpace(e.Visibility)
	e.MinimumVisibleRole = strings.TrimSpace(e.MinimumVisibleRole)

	if e.Visibility == VisibilityPublic {
		e.MinimumVisibleRole = ""
	}

	if !e.WaitlistEnabled {
		e.WaitlistSize = 0
	}
}

// Validate validates the event as a whole.
// Use this for creates and for PATCH after applying incoming fields onto an existing event.
func (e *Event) Validate() error {
	e.normalize()

	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(e.Date) == "" {
		return fmt.Errorf("date is required")
	}
	if strings.TrimSpace(e.Time) == "" {
		return fmt.Errorf("time is required")
	}
	if strings.TrimSpace(e.Location) == "" {
		return fmt.Errorf("location is required")
	}

	if err := ValidateStatus(e.Status); err != nil {
		return err
	}
	if err := ValidateVisibility(e.Visibility); err != nil {
		return err
	}
	if err := ValidateMinimumVisibleRole(e.Visibility, e.MinimumVisibleRole); err != nil {
		return err
	}
	if e.MaxAttendees == 0 || e.MaxAttendees < -1 {
		return fmt.Errorf("max_attendees must be greater than 0, or -1 for no limit")
	}

	if e.WaitlistEnabled && e.WaitlistSize <= 0 {
		return fmt.Errorf("waitlist_size must be greater than 0 when waitlist_enabled is true")
	}

	if e.PublishDate != nil && e.PublishDate.IsZero() {
		return fmt.Errorf("publish_date must be a valid datetime")
	}

	if e.Status == StatusClosed && e.PublishDate != nil {
		return fmt.Errorf("closed events cannot have a publish_date")
	}

	if strings.TrimSpace(e.EndDate) != "" {
		date, err := time.Parse("2006-01-02", e.Date)
		if err != nil {
			return fmt.Errorf("date is not a valid date (expected YYYY-MM-DD)")
		}
		endDate, err := time.Parse("2006-01-02", e.EndDate)
		if err != nil {
			return fmt.Errorf("end_date is not a valid date (expected YYYY-MM-DD)")
		}
		if endDate.Before(date) {
			return fmt.Errorf("end_date must be on or after start date")
		}
	}

	return nil
}

func ValidateStatus(status string) error {
	switch strings.TrimSpace(status) {
	case StatusDraft, StatusPublished, StatusClosed:
		return nil
	default:
		return fmt.Errorf("invalid status: must be one of draft, published, closed")
	}
}

func ValidateVisibility(visibility string) error {
	switch strings.TrimSpace(visibility) {
	case VisibilityPublic, VisibilityPrivate:
		return nil
	default:
		return fmt.Errorf("invalid visibility: must be one of public, private")
	}
}

func ValidateRole(role string) error {
	switch strings.TrimSpace(role) {
	case RoleMember, RoleOfficer, RoleAdmin:
		return nil
	default:
		return fmt.Errorf("invalid minimum_visible_role: must be one of member, officer, admin")
	}
}

func ValidateMinimumVisibleRole(visibility string, role string) error {
	visibility = strings.TrimSpace(visibility)
	role = strings.TrimSpace(role)

	switch visibility {
	case VisibilityPublic:
		if role != "" {
			return fmt.Errorf("minimum_visible_role must be empty when visibility is public")
		}
		return nil

	case VisibilityPrivate:
		if role == "" {
			return fmt.Errorf("minimum_visible_role is required when visibility is private")
		}
		return ValidateRole(role)

	default:
		return fmt.Errorf("invalid visibility: must be one of public, private")
	}
}

// IsListedAdmin reports whether the given userID is explicitly listed in the event's Admins list.
func (e *Event) IsListedAdmin(userID string) bool {
	for _, admin := range e.Admins {
		if admin == userID {
			return true
		}
	}
	return false
}

// CanEdit reports whether the caller may modify this event.
// If Admins is empty, the event is treated as admin-less and any site admin may edit it.
// Otherwise, only user IDs explicitly listed in Admins may edit it.
func (e *Event) CanEdit(userID, callerSiteRole string) bool {
	if len(e.Admins) == 0 {
		return strings.EqualFold(strings.TrimSpace(callerSiteRole), RoleAdmin)
	}
	return e.IsListedAdmin(userID)
}

func (e *Event) ValidateRegistration(answers map[string]any) error {
	for _, question := range e.RegistrationForm {
		if !question.Required {
			continue
		}

		answer, exists := answers[question.ID]
		if !exists {
			return &RegistrationFormValidationError{
				Field:   question.ID,
				Message: "missing required answer",
			}
		}

		switch v := answer.(type) {
		case string:
			if strings.TrimSpace(v) == "" {
				return &RegistrationFormValidationError{
					Field:   question.ID,
					Message: "required answer cannot be empty",
				}
			}
		case []any:
			if len(v) == 0 {
				return &RegistrationFormValidationError{
					Field:   question.ID,
					Message: "required answer cannot be empty",
				}
			}
		}
	}
	return nil
}

// SanitizeUpdateFields removes immutable fields from an update map
// to prevent them from being overwritten.
func SanitizeUpdateFields(fields map[string]interface{}) {
	delete(fields, "id")
	delete(fields, "_id")
	delete(fields, "created_at")
	delete(fields, "published_at")
}

// ApplyPatch applies supported PATCH fields onto the event.
// It performs type validation for incoming map values.
// After ApplyPatch, call Validate() on the event.
func (e *Event) ApplyPatch(fields map[string]interface{}) error {
	for key, value := range fields {
		switch key {
		case "name":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("name must be a string")
			}
			e.Name = s

		case "date":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("date must be a string")
			}
			e.Date = s

		case "end_date":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("end_date must be a string")
			}
			e.EndDate = s

		case "time":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("time must be a string")
			}
			e.Time = s

		case "location":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("location must be a string")
			}
			e.Location = s

		case "description":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("description must be a string")
			}
			e.Description = s

		case "status":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("status must be a string")
			}
			e.Status = s

		case "visibility":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("visibility must be a string")
			}
			e.Visibility = s

		case "minimum_visible_role":
			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("minimum_visible_role must be a string")
			}
			e.MinimumVisibleRole = s

		case "max_attendees":
			n, ok := value.(float64)
			if !ok {
				return fmt.Errorf("max_attendees must be a number")
			}
			e.MaxAttendees = int(n)

		case "waitlist_enabled":
			b, ok := value.(bool)
			if !ok {
				return fmt.Errorf("waitlist_enabled must be a boolean")
			}
			e.WaitlistEnabled = b

		case "waitlist_size":
			n, ok := value.(float64)
			if !ok {
				return fmt.Errorf("waitlist_size must be a number")
			}
			e.WaitlistSize = int(n)

		case "registration_form":
			arr, ok := value.([]interface{})
			if !ok {
				return fmt.Errorf("registration_form must be an array")
			}
			questions := make([]FormQuestion, 0, len(arr))
			for i, item := range arr {
				qMap, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("registration_form[%d] must be an object", i)
				}
				q := FormQuestion{}
				if id, ok := qMap["id"].(string); ok {
					q.ID = id
				}
				if typ, ok := qMap["type"].(string); ok {
					q.Type = typ
				}
				if question, ok := qMap["question"].(string); ok {
					q.Question = question
				}
				if required, ok := qMap["required"].(bool); ok {
					q.Required = required
				}
				if details, ok := qMap["answer_details"].(map[string]interface{}); ok {
					ad := &AnswerDetails{}
					if maxChars, ok := details["max_chars"].(float64); ok {
						ad.MaxChars = int(maxChars)
					}
					q.AnswerDetails = ad
				}
				if opts, ok := qMap["answer_options"].([]interface{}); ok {
					for _, opt := range opts {
						if s, ok := opt.(string); ok {
							q.AnswerOptions = append(q.AnswerOptions, s)
						}
					}
				}
				questions = append(questions, q)
			}
			e.RegistrationForm = questions

		case "admins":
			arr, ok := value.([]interface{})
			if !ok {
				return fmt.Errorf("admins must be an array")
			}
			seen := make(map[string]struct{}, len(arr))
			admins := make([]string, 0, len(arr))
			for i, item := range arr {
				admin, ok := item.(string)
				if !ok {
					return fmt.Errorf("admins[%d] must be a string", i)
				}
				admin = strings.TrimSpace(admin)
				if admin == "" {
					return fmt.Errorf("admins[%d] cannot be empty", i)
				}
				if _, exists := seen[admin]; exists {
					continue
				}
				seen[admin] = struct{}{}
				admins = append(admins, admin)
			}
			if len(admins) == 0 {
				return fmt.Errorf("admins must include at least one user")
			}
			e.Admins = admins

		case "publish_date":
			if value == nil {
				e.PublishDate = nil
				break
			}

			s, ok := value.(string)
			if !ok {
				return fmt.Errorf("publish_date must be an RFC3339 datetime string or null")
			}

			parsed, err := time.Parse(time.RFC3339, s)
			if err != nil {
				return fmt.Errorf("publish_date must be a valid RFC3339 datetime")
			}
			parsed = parsed.UTC()
			e.PublishDate = &parsed
		}
	}

	e.normalize()
	return nil
}

// ShouldAutoPublish reports whether the event should be automatically published based on its publish_date and current time
func (e *Event) ShouldAutoPublish(now time.Time) bool {
	if e.PublishDate == nil {
		return false
	}
	if e.Status == StatusPublished || e.Status == StatusClosed {
		return false
	}
	return !e.PublishDate.After(now.UTC())
}

// SyncPublicationState updates the event's status to published if it should be auto-published, and sets publish_date and published_at if missing
func (e *Event) SyncPublicationState(now time.Time) {
	now = now.UTC()

	if e.Status == StatusPublished {
		if e.PublishDate == nil {
			t := now
			e.PublishDate = &t
		}
		if e.PublishedAt == nil {
			t := now
			e.PublishedAt = &t
		}
		return
	}

	if e.ShouldAutoPublish(now) {
		e.Status = StatusPublished
		if e.PublishedAt == nil {
			t := now
			e.PublishedAt = &t
		}
	}
}

// CanView reports whether the given viewer can see this event based on its status, visibility, and minimum visible role
func (e *Event) CanView(viewer EventViewer) bool {
	// Site admins can view everything
	if viewer.AccessLevel >= 3 {
		return true
	}

	// Event-specific admins can view their own events, even if draft
	if e.IsListedAdmin(viewer.UserID) {
		return true
	}

	// Unpublished events are not visible to regular viewers
	if e.Status != StatusPublished {
		return false
	}

	// Public events are visible to everyone once published
	if e.Visibility == VisibilityPublic {
		return true
	}

	// Private published events require a minimum role
	required := 0
	switch e.MinimumVisibleRole {
	case RoleMember:
		required = 1
	case RoleOfficer:
		required = 2
	case RoleAdmin:
		required = 3
	default:
		return false
	}

	return viewer.AccessLevel >= required
}
