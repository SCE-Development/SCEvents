package models

import "strings"

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
	Admins           []string       `bson:"admins" json:"admins"`
	RegistrationForm []FormQuestion `bson:"registration_form" json:"registration_form"`
	MaxAttendees     int            `bson:"max_attendees" json:"max_attendees"`
	CreatedAt        string         `bson:"created_at" json:"created_at"`
	Status           string         `bson:"status" json:"status"` // draft, published, closed
}

type RegistrationFormValidationError struct {
	Field   string
	Message string
}

func (e *RegistrationFormValidationError) Error() string {
	return e.Message
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
