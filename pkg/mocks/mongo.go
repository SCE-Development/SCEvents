package mocks

import (
	"context"
	"time"

	"github.com/SCE-Development/SCEvents/pkg/models"
)

type MockMongoStore struct {
	Registration               *models.RegistrationRequest
	Registrations              []models.RegistrationRequest
	RegistrationErr            error
	RegistrationsErr           error
	StatusCounts               map[models.Status]int64
	StatusCountsErr            error
	Event                      *models.Event
	EventErr                   error
	AttendeeCount              int64
	AttendeeCountErr           error
	CountCalled                bool
	HasAccepted                bool
	HasAcceptedErr             error
	MarkAcceptedErr            error
	MarkRejectedErr            error
	RejectedReason             models.DecisionReason
	RegistrationStatuses       map[string]models.Status
	RegistrationStatusesErr    error
	WaitlistedEventIDs         map[string]bool
	WaitlistedEventIDsErr      error
	Events                     []models.Event
	EventsErr                  error
	CreatedEvent               *models.Event
	PublishedDueCount          int64
    PublishDueEventsErr        error
}

func (m *MockMongoStore) GetVisibleEvents(_ context.Context, _ models.EventViewer, _, _ string) ([]models.Event, error) {
	return m.Events, m.EventsErr
}
func (m *MockMongoStore) GetVisibleEventByID(_ context.Context, _ models.EventViewer, _ string) (*models.Event, error) {
	return m.Event, m.EventErr
}
func (m *MockMongoStore) GetEvents(_ context.Context, _, _ string) ([]models.Event, error) {
	return m.Events, m.EventsErr
}
func (m *MockMongoStore) GetEventByID(_ context.Context, _ string) (*models.Event, error) {
	return m.Event, m.EventErr
}
func (m *MockMongoStore) CreateEvent(_ context.Context, e models.Event) (*models.Event, error) {
	m.CreatedEvent = &e
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
func (m *MockMongoStore) ListRegistrationsByEventID(_ context.Context, _ string, _, _ int64) ([]models.RegistrationRequest, error) {
	return m.Registrations, m.RegistrationsErr
}
func (m *MockMongoStore) GetRegistrationByEventAndRequestID(_ context.Context, _, _ string) (*models.RegistrationRequest, error) {
	return m.Registration, m.RegistrationErr
}
func (m *MockMongoStore) CountRegistrationsByStatusForEvent(_ context.Context, _ string) (map[models.Status]int64, error) {
	if m.StatusCounts == nil {
		return map[models.Status]int64{
			models.StatusPending:  0,
			models.StatusAccepted: 0,
			models.StatusRejected: 0,
		}, m.StatusCountsErr
	}
	return m.StatusCounts, m.StatusCountsErr
}
func (m *MockMongoStore) CountAcceptedRegistrationsForEvent(_ context.Context, _ string) (int64, error) {
	m.CountCalled = true
	return m.AttendeeCount, m.AttendeeCountErr
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
func (m *MockMongoStore) GetRegistrationStatusesForUser(_ context.Context, _ string, _ []string) (map[string]models.Status, error) {
	return m.RegistrationStatuses, m.RegistrationStatusesErr
}
func (m *MockMongoStore) HasWaitlistEntry(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *MockMongoStore) CountWaitlistEntries(_ context.Context, _ string) (int64, error) {
	return 0, nil
}
func (m *MockMongoStore) CreateWaitlistEntry(_ context.Context, _ models.WaitlistEntry) error {
	return nil
}
func (m *MockMongoStore) GetWaitlistedEventIDsForUser(_ context.Context, _ string, _ []string) (map[string]bool, error) {
	return m.WaitlistedEventIDs, m.WaitlistedEventIDsErr
}
func (m *MockMongoStore) PublishDueEvents(_ context.Context, _ time.Time) (int64, error) {
	return m.PublishedDueCount, m.PublishDueEventsErr
}
func (m *MockMongoStore) ListAcceptedRegistrationsForEvent(_ context.Context, _ string) ([]models.RegistrationRequest, error) {
	return m.Registrations, m.RegistrationsErr
}
