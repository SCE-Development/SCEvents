package registration

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/mocks"
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

func TestValidateKafkaMessage(t *testing.T) {
	valid := models.KafkaRegistrationMessage{
		RequestID: "req-1",
		EventID:   "event-1",
		UserID:    "user-1",
	}

	t.Run("valid message", func(t *testing.T) {
		if err := validateKafkaMessage(valid); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("missing request_id", func(t *testing.T) {
		msg := valid
		msg.RequestID = ""
		err := validateKafkaMessage(msg)
		if err == nil {
			t.Fatal("expected error for missing request_id")
		}
		if err.Error() != "missing request_id" {
			t.Fatalf("expected 'missing request_id', got %q", err.Error())
		}
	})

	t.Run("missing event_id", func(t *testing.T) {
		msg := valid
		msg.EventID = ""
		err := validateKafkaMessage(msg)
		if err == nil {
			t.Fatal("expected error for missing event_id")
		}
		if err.Error() != "missing event_id" {
			t.Fatalf("expected 'missing event_id', got %q", err.Error())
		}
	})

	t.Run("missing user_id", func(t *testing.T) {
		msg := valid
		msg.UserID = ""
		err := validateKafkaMessage(msg)
		if err == nil {
			t.Fatal("expected error for missing user_id")
		}
		if err.Error() != "missing user_id" {
			t.Fatalf("expected 'missing user_id', got %q", err.Error())
		}
	})
}

func TestProducerClose(t *testing.T) {
	t.Run("nil producer", func(t *testing.T) {
		var p *Producer
		if err := p.Close(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("nil writer", func(t *testing.T) {
		p := &Producer{writer: nil}
		if err := p.Close(); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestMockKafkaProducer(t *testing.T) {
	t.Run("records published messages", func(t *testing.T) {
		mock := &mocks.MockKafkaProducer{}
		err := mock.PublishRegistration(context.Background(), "req-1", "event-1", "user-1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(mock.PublishedMessages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(mock.PublishedMessages))
		}
		if mock.PublishedMessages[0].RequestID != "req-1" {
			t.Fatalf("expected request ID req-1, got %s", mock.PublishedMessages[0].RequestID)
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		mock := &mocks.MockKafkaProducer{PublishErr: errors.New("broker down")}
		err := mock.PublishRegistration(context.Background(), "req-1", "event-1", "user-1")
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "broker down" {
			t.Fatalf("expected 'broker down', got %q", err.Error())
		}
		if len(mock.PublishedMessages) != 0 {
			t.Fatalf("expected 0 messages on error, got %d", len(mock.PublishedMessages))
		}
	})

	t.Run("close tracks state", func(t *testing.T) {
		mock := &mocks.MockKafkaProducer{}
		if mock.Closed {
			t.Fatal("expected closed=false before Close()")
		}
		_ = mock.Close()
		if !mock.Closed {
			t.Fatal("expected closed=true after Close()")
		}
	})
}
