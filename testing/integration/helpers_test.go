//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

func baseURL() string {
	if u := os.Getenv("SERVICE_URL"); u != "" {
		return u
	}
	return "http://localhost:8002"
}

func authHeader(userID string) string {
	return "Bearer " + userID
}

// createEvent POSTs to /events/ and returns the created event's ID.
// Uses the provided userID as both the Bearer token and event admin.
func createEvent(t interface {
	Helper()
	Fatalf(string, ...interface{})
}, userID string, body map[string]interface{}) string {
	t.Helper()

	// Ensure the creator can edit/delete in follow-up tests.
	if _, ok := body["admins"]; !ok {
		body["admins"] = []interface{}{userID}
	}

	data, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, baseURL()+"/events/", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(userID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("createEvent request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("createEvent: expected 201, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	id, ok := result["id"].(string)
	if !ok || id == "" {
		t.Fatalf("createEvent: response missing id field: %v", result)
	}
	return id
}

// registerForEvent POSTs to /events/:id/register and returns (statusCode, requestID).
func registerForEvent(eventID, userID, name, email string) (int, string) {
	body, _ := json.Marshal(map[string]interface{}{
		"registrant": map[string]interface{}{
			"name":    name,
			"email":   email,
			"user_id": userID,
		},
		"registration_form_answers": map[string]interface{}{},
	})

	req, _ := http.NewRequest(http.MethodPost, baseURL()+"/events/"+eventID+"/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(userID))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, ""
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	requestID, _ := result["request_id"].(string)
	return resp.StatusCode, requestID
}

// pollRegistrationStatus polls GET /events/registrations/:request_id?user_id=:userID
// until the status is no longer "pending" or the timeout is exceeded.
func pollRegistrationStatus(requestID, userID string, timeout time.Duration) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(baseURL() + "/events/registrations/" + requestID + "?user_id=" + userID)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()

		status, _ := result["status"].(string)
		if status != "" && status != "pending" {
			return status, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return "", fmt.Errorf("timed out after %s waiting for registration %s to be processed", timeout, requestID)
}

// defaultEvent returns a minimal valid event body with the given name and capacity.
func defaultEvent(name string, maxAttendees int) map[string]interface{} {
	return map[string]interface{}{
		"name":              name,
		"date":              "2099-12-31",
		"time":              "18:00",
		"location":          "Test Venue",
		"description":       "Integration test event",
		"max_attendees":     maxAttendees,
		"status":            "published",
		"visibility":        "public",
		"registration_form": []interface{}{},
	}
}
