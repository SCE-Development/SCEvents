//go:build integration

package integration_test

import (
	"net/http"
	"testing"
	"time"
)

func isRegistrationSubmitted(status int) bool {
	return status == http.StatusCreated || status == http.StatusAccepted
}

// TestRegistrationFlow_BasicRegistration creates an event, registers a user,
// and verifies the registration is accepted by the Kafka consumer.
func TestRegistrationFlow_BasicRegistration(t *testing.T) {
	t.Parallel()
	eventID := createEvent(t, "admin-basic-reg", defaultEvent("Basic Registration Test", 100))

	status, requestID := registerForEvent(eventID, "user-basic-reg", "Alice Smith", "alice@example.com")
	if !isRegistrationSubmitted(status) {
		t.Fatalf("expected 201/202 for registration submission, got %d", status)
	}
	if requestID == "" {
		t.Fatal("expected non-empty request_id in response")
	}

	finalStatus, err := pollRegistrationStatus(requestID, "user-basic-reg", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if finalStatus != "accepted" {
		t.Errorf("expected registration status 'accepted', got %q", finalStatus)
	}
}

// TestRegistrationFlow_DuplicateRegistration verifies that registering the same
// user for the same event twice is rejected with 409.
func TestRegistrationFlow_DuplicateRegistration(t *testing.T) {
	t.Parallel()
	eventID := createEvent(t, "admin-dup-reg", defaultEvent("Duplicate Registration Test", 100))

	// first registration
	status, requestID := registerForEvent(eventID, "user-dup-reg", "Bob Jones", "bob@example.com")
	if !isRegistrationSubmitted(status) {
		t.Fatalf("first registration: expected 201/202, got %d", status)
	}

	// wait for the first to be accepted so we're in an accepted state, not just pending
	if _, err := pollRegistrationStatus(requestID, "user-dup-reg", 10*time.Second); err != nil {
		t.Fatal(err)
	}

	// second registration — same user, same event
	status, _ = registerForEvent(eventID, "user-dup-reg", "Bob Jones", "bob@example.com")
	if status != http.StatusConflict {
		t.Errorf("duplicate registration: expected 409 Conflict, got %d", status)
	}
}

// TestRegistrationFlow_CapacityFull creates an event with capacity 1, fills it,
// then verifies a second user's registration is rejected as capacity_full.
func TestRegistrationFlow_CapacityFull(t *testing.T) {
	t.Parallel()
	eventID := createEvent(t, "admin-cap-full", defaultEvent("Capacity Full Test", 1))

	// register first user — should be accepted
	status, requestID := registerForEvent(eventID, "user-cap-first", "Carol White", "carol@example.com")
	if !isRegistrationSubmitted(status) {
		t.Fatalf("first registration: expected 201/202, got %d", status)
	}
	firstStatus, err := pollRegistrationStatus(requestID, "user-cap-first", 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if firstStatus != "accepted" {
		t.Fatalf("first registration: expected 'accepted', got %q", firstStatus)
	}

	// register second user — event is full, should be rejected immediately
	status, requestID = registerForEvent(eventID, "user-cap-second", "Dave Green", "dave@example.com")
	if status != http.StatusConflict {
		t.Fatalf("second registration: expected 409 Conflict when full, got %d", status)
	}
	if requestID != "" {
		secondStatus, err := pollRegistrationStatus(requestID, "user-cap-second", 10*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if secondStatus != "rejected" {
			t.Errorf("second registration: expected 'rejected' (capacity_full), got %q", secondStatus)
		}
	}
}
