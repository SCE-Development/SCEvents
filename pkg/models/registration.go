package models

import "time"

type Status string
type DecisionReason string

// Status values describe the lifecycle of a registration request in persistence
const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
	StatusRejected Status = "rejected"
)

const (
	ReasonNone             DecisionReason = ""
	ReasonEventNotFound    DecisionReason = "event_not_found"
	ReasonDuplicateUser    DecisionReason = "duplicate_user"
	ReasonCapacityFull     DecisionReason = "capacity_full"
	ReasonInvalidPayload   DecisionReason = "invalid_payload"
	ReasonAlreadyProcessed DecisionReason = "already_processed"
	ReasonInternalError    DecisionReason = "internal_error"
	ReasonEventClosed      DecisionReason = "event_closed"
)

type Registrant struct {
	Name   string `bson:"name" json:"name" binding:"required"`
	Email  string `bson:"email" json:"email" binding:"required,email"`
	UserID string `bson:"user_id" json:"user_id,omitempty"`
}

type RegistrationPayload struct {
	Registrant              Registrant     `bson:"registrant" json:"registrant" binding:"required"`
	RegistrationFormAnswers map[string]any `bson:"registration_form_answers" json:"registration_form_answers" binding:"required"`
	Timestamp               time.Time      `bson:"timestamp" json:"timestamp"`
}

// RegistrationRequest is the source-of-truth document representing a user's registration lifecycle (pending to accepted/rejected)
type RegistrationRequest struct {
	RequestID      string         `bson:"_id" json:"request_id"`
	EventID        string         `bson:"event_id" json:"event_id"`
	Registrant     Registrant     `bson:"registrant" json:"registrant"`
	Answers        map[string]any `bson:"answers" json:"answers"`
	Status         Status         `bson:"status" json:"status"`
	DecisionReason DecisionReason `bson:"decision_reason,omitempty" json:"decision_reason,omitempty"`
	CreatedAt      time.Time      `bson:"created_at" json:"created_at"`
	UpdatedAt      time.Time      `bson:"updated_at" json:"updated_at"`
	ProcessedAt    *time.Time     `bson:"processed_at,omitempty" json:"processed_at,omitempty"`
}

// KafkaRegistrationMessage is a lightweight reference payload used by the consumer to process a registration request
type KafkaRegistrationMessage struct {
	RequestID string    `json:"request_id"`
	EventID   string    `json:"event_id"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

type RegistrationDashboardSummary struct {
	Total    int64 `json:"total"`
	Accepted int64 `json:"accepted"`
	Pending  int64 `json:"pending"`
	Rejected int64 `json:"rejected"`
}
