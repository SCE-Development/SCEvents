package models

import (
	"encoding/json"
	"fmt"
	"strings"
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
	ID                 string         `bson:"_id" json:"id"`
	Name               string         `bson:"name" json:"name"`
	Date               string         `bson:"date" json:"date"`
	EndDate            string         `bson:"end_date,omitempty" json:"end_date,omitempty"`
	Time               string         `bson:"time" json:"time"`
	Location           string         `bson:"location" json:"location"`
	Description        string         `bson:"description" json:"description"`
	Admins             []string       `bson:"admins" json:"admins"`
	RegistrationForm   []FormQuestion `bson:"registration_form" json:"registration_form"`
	MaxAttendees       int            `bson:"max_attendees" json:"max_attendees"`
	CreatedAt          string         `bson:"created_at" json:"created_at"`
	Status             string         `bson:"status" json:"status"`
	Visibility         string         `bson:"visibility" json:"visibility"`
	MinimumVisibleRole string         `bson:"minimum_visible_role,omitempty" json:"minimum_visible_role,omitempty"`
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
	if e.MaxAttendees < 0 {
		return fmt.Errorf("max_attendees cannot be negative")
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

// IsAdmin checks if the given userID is in the event's Admins list.
func (e *Event) IsAdmin(userID string) bool {
	for _, admin := range e.Admins {
		if admin == userID {
			return true
		}
	}
	return false
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

		case "registration_form":
			data, err := json.Marshal(value)
			if err != nil {
				return fmt.Errorf("registration_form must be a valid array")
			}

			var form []FormQuestion
			if err := json.Unmarshal(data, &form); err != nil {
				return fmt.Errorf("registration_form must be a valid array")
			}
			e.RegistrationForm = form

		case "admins":
			// supported by persistence model, but not yet patchable here
			return fmt.Errorf("%s cannot be updated through this endpoint yet", key)
		}
	}

	e.normalize()
	return nil
}
