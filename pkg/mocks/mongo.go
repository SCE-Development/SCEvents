package mocks

import (
	"context"

	"github.com/SCE-Development/SCEvents/pkg/models"
)

type MockMongoStore struct {
	Registration    *models.RegistrationRequest
	RegistrationErr error
	Event           *models.Event
	EventErr        error
	HasAccepted     bool
	HasAcceptedErr  error
	MarkAcceptedErr error
	MarkRejectedErr error
	RejectedReason  models.DecisionReason
}

func (m *MockMongoStore) GetEvents(_ context.Context, _, _ string) ([]models.Event, error) {
	return nil, nil
}
func (m *MockMongoStore) GetEventByID(_ context.Context, _ string) (*models.Event, error) {
	return m.Event, m.EventErr
}
func (m *MockMongoStore) CreateEvent(_ context.Context, e models.Event) (*models.Event, error) {
	return &e, nil
}
func (m *MockMongoStore) DeleteEventByID(_ context.Context, _ string) error {
	return nil
}
func (m *MockMongoStore) UpdateEventByID(_ context.Context, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *MockMongoStore) CreatePendingRegistration(_ context.Context, r models.RegistrationRequest) (*models.RegistrationRequest, error) {
	return &r, nil
}
func (m *MockMongoStore) GetRegistrationByID(_ context.Context, _ string) (*models.RegistrationRequest, error) {
	return m.Registration, m.RegistrationErr
}
func (m *MockMongoStore) HasAcceptedRegistration(_ context.Context, _, _ string) (bool, error) {
	return m.HasAccepted, m.HasAcceptedErr
}
func (m *MockMongoStore) HasPendingOrAcceptedRegistration(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *MockMongoStore) MarkRegistrationAccepted(_ context.Context, _ string) error {
	return m.MarkAcceptedErr
}
func (m *MockMongoStore) MarkRegistrationRejected(_ context.Context, _ string, reason models.DecisionReason) error {
	m.RejectedReason = reason
	return m.MarkRejectedErr
}
