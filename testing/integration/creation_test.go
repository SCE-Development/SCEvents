//go:build integration

package integration_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// TestEventCreationFlow_CreateAndRetrieve creates an event and verifies it can be
// fetched back with matching fields.
func TestEventCreationFlow_CreateAndRetrieve(t *testing.T) {
	body := defaultEvent("Create And Retrieve Test", 50)
	eventID := createEvent(t, "admin-create-retrieve", body)

	resp, err := http.Get(baseURL() + "/events/" + eventID)
	if err != nil {
		t.Fatalf("GET event failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET event: expected 200, got %d", resp.StatusCode)
	}

	var event map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&event)

	if event["id"] != eventID {
		t.Errorf("expected id %q, got %q", eventID, event["id"])
	}
	if event["name"] != body["name"] {
		t.Errorf("expected name %q, got %q", body["name"], event["name"])
	}
	if event["location"] != body["location"] {
		t.Errorf("expected location %q, got %q", body["location"], event["location"])
	}
}

// TestEventCreationFlow_Update creates an event, patches the name and location,
// then verifies the updated values are returned.
func TestEventCreationFlow_Update(t *testing.T) {
	eventID := createEvent(t, "admin-update", defaultEvent("Update Test Original", 50))

	patch, _ := json.Marshal(map[string]interface{}{
		"name":     "Update Test Renamed",
		"location": "New Venue",
	})
	req, _ := http.NewRequest(http.MethodPatch, baseURL()+"/events/"+eventID, bytes.NewReader(patch))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader("admin-update"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("PATCH event failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PATCH event: expected 200, got %d", resp.StatusCode)
	}

	// verify via GET
	resp, err = http.Get(baseURL() + "/events/" + eventID)
	if err != nil {
		t.Fatalf("GET after PATCH failed: %v", err)
	}
	defer resp.Body.Close()

	var event map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&event)

	if event["name"] != "Update Test Renamed" {
		t.Errorf("expected updated name, got %q", event["name"])
	}
	if event["location"] != "New Venue" {
		t.Errorf("expected updated location, got %q", event["location"])
	}
}

// TestEventCreationFlow_Delete creates an event, deletes it, and verifies a
// subsequent GET returns 404. Only draft or closed events may be deleted (see DeleteEventByID).
func TestEventCreationFlow_Delete(t *testing.T) {
	body := defaultEvent("Delete Test", 50)
	body["status"] = "draft"
	eventID := createEvent(t, "admin-delete", body)

	req, _ := http.NewRequest(http.MethodDelete, baseURL()+"/events/"+eventID, nil)
	req.Header.Set("Authorization", authHeader("admin-delete"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("DELETE event failed: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE event: expected 200, got %d", resp.StatusCode)
	}

	// verify the event is gone
	resp, err = http.Get(baseURL() + "/events/" + eventID)
	if err != nil {
		t.Fatalf("GET after DELETE failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 after deletion, got %d", resp.StatusCode)
	}
}
