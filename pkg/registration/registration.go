package registration

import "time"

type Status string
type DecisionReason string

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
)

type Registrant struct {
	Name   string `bson:"name" json:"name" binding:"required"`
	Email  string `bson:"email" json:"email" binding:"required,email"`
	UserID string `bson:"user_id" json:"user_id"`
}

// SubmitRegistrationRequest represents the incoming API payload from the frontend when a user registers for an event
type SubmitRegistrationRequest struct {
	Registrant  Registrant     `bson:"registrant" json:"registrant" binding:"required"`
	Answers     map[string]any `bson:"answers" json:"answers" binding:"required"`
	SubmittedAt string         `bson:"submitted_at,omitempty" json:"submitted_at,omitempty"`
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
