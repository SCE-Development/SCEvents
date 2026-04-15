package registration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
)



func TestKafkaRegistrationMessage_JSONMatchesConsumerContract(t *testing.T) {
	createdAt := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	msg := models.KafkaRegistrationMessage{
		RequestID: "507f1f77bcf86cd799439011",
		EventID:   "event-abc",
		UserID:    "user-xyz",
		CreatedAt: createdAt,
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded models.KafkaRegistrationMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := validateKafkaMessage(decoded); err != nil {
		t.Fatalf("payload should pass consumer validation: %v", err)
	}
	if decoded.RequestID != msg.RequestID || decoded.EventID != msg.EventID || decoded.UserID != msg.UserID {
		t.Fatalf("round-trip mismatch: %+v vs %+v", decoded, msg)
	}
}
