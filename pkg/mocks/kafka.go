package mocks

import (
	"context"

	"github.com/SCE-Development/SCEvents/pkg/models"
)

type MockKafkaProducer struct {
	PublishedMessages []models.KafkaRegistrationMessage
	PublishErr        error
	CloseErr          error
	Closed            bool
}

func (m *MockKafkaProducer) PublishRegistration(_ context.Context, requestID string, eventID string, userID string) error {
	if m.PublishErr != nil {
		return m.PublishErr
	}
	m.PublishedMessages = append(m.PublishedMessages, models.KafkaRegistrationMessage{
		RequestID: requestID,
		EventID:   eventID,
		UserID:    userID,
	})
	return nil
}

func (m *MockKafkaProducer) Close() error {
	m.Closed = true
	return m.CloseErr
}
