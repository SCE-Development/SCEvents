package mocks

import (
	"context"
)

type MockRedisStore struct {
	SeatAvailable bool
	SeatErr       error
	ReleaseCalled bool
	ReleaseErr    error

	SetCalled   bool
	SetCapacity int
	SetErr      error

	GetCalled    bool
	GetRemaining int
	GetErr       error

	DeleteCalled bool
	DeleteErr    error
}

func (m *MockRedisStore) SetEventHeadcount(_ context.Context, _ string, capacity int) error {
	m.SetCalled = true
	m.SetCapacity = capacity
	return m.SetErr
}
func (m *MockRedisStore) GetEventHeadcount(_ context.Context, _ string) (int, error) {
	m.GetCalled = true
	return m.GetRemaining, m.GetErr
}
func (m *MockRedisStore) DeleteEventHeadcount(ctx context.Context, eventID string) error {
	m.DeleteCalled = true
	return m.DeleteErr
}
func (m *MockRedisStore) TryTakeEventSeat(_ context.Context, _ string) (bool, error) {
	return m.SeatAvailable, m.SeatErr
}
func (m *MockRedisStore) ReleaseEventSeat(_ context.Context, _ string) error {
	m.ReleaseCalled = true
	return m.ReleaseErr
}
func (m *MockRedisStore) IsUserRegisteredForEvent(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (m *MockRedisStore) Close() error {
	return nil
}
