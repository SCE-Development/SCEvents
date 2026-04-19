package registration

import (
	"context"
	"encoding/json"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

// NewProducer builds a Kafka writer for registration reference messages.
// Writes are blocking to ensure publish errors reflect Kafka failures (e.g. outbox rollback).
func NewProducer(brokers []string, topic string) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireOne,
	}

	return &Producer{
		writer: w,
	}
}

// PublishRegistration emits a lightweight reference message to Kafka that the consumer can use
// to process an existing pending registration request stored in MongoDB.
func (p *Producer) PublishRegistration(ctx context.Context, requestID string, eventID string, userID string) error {
	createdAt := time.Now().UTC()
	payload := models.KafkaRegistrationMessage{
		RequestID: requestID,
		EventID:   eventID,
		UserID:    userID,
		CreatedAt: createdAt,
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.writer.WriteMessages(ctx, kafka.Message{
		Value: b,
		Time:  createdAt,
	})
}

func (p *Producer) Close() error {
	if p == nil || p.writer == nil {
		return nil
	}
	return p.writer.Close()
}
