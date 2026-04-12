package event

import (
	"strings"
	"time"
)

type RegistrationFormValidationError struct {
	Field   string
	Message string
}

func (e *RegistrationFormValidationError) Error() string {
	return e.Message
}

type Registrant struct {
	Name   string `json:"name" binding:"required"`
	Email  string `json:"email" binding:"required,email"`
	UserID string `json:"user_id,omitempty"`
}

type RegistrationPayload struct {
	Registrant              Registrant      `json:"registrant" binding:"required"`
	RegistrationFormAnswers map[string]any  `json:"registration_form_answers" binding:"required"`
	Timestamp               time.Time       `json:"timestamp"`
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
