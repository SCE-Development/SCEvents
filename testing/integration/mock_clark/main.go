package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// Mock Clark auth server for use in integration and load tests.
// Echoes the Bearer token as _id with accessLevel 3 (admin). Create/update/delete
// routes require MembershipStateOfficer (2), so the mock must be at least officer.
func main() {
	http.HandleFunc("/api/Auth/verify", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		auth := r.Header.Get("Authorization")
		userID := strings.TrimPrefix(auth, "Bearer ")
		if userID == "" {
			http.Error(w, "missing token", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"_id":         userID,
			"accessLevel": 3,
		})
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("mock Clark API listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
