package models

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRegistrationPayloadBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/test", func(c *gin.Context) {
		var payload RegistrationPayload
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	tests := []struct {
		name       string
		payloadMap map[string]interface{}
		wantErr    bool
	}{
		{
			name: "valid payload",
			payloadMap: map[string]interface{}{
				"registrant": map[string]interface{}{
					"name":  "Test User",
					"email": "test@example.com",
				},
				"registration_form_answers": map[string]interface{}{
					"q1": "a1",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			payloadMap: map[string]interface{}{
				"registrant": map[string]interface{}{
					"email": "test@example.com",
				},
				"registration_form_answers": map[string]interface{}{
					"q1": "a1",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			payloadMap: map[string]interface{}{
				"registrant": map[string]interface{}{
					"name":  "Test User",
					"email": "invalid-email",
				},
				"registration_form_answers": map[string]interface{}{
					"q1": "a1",
				},
			},
			wantErr: true,
		},
		{
			name: "missing email",
			payloadMap: map[string]interface{}{
				"registrant": map[string]interface{}{
					"name": "Test User",
				},
				"registration_form_answers": map[string]interface{}{
					"q1": "a1",
				},
			},
			wantErr: true,
		},
		{
			name: "missing registration form answers",
			payloadMap: map[string]interface{}{
				"registrant": map[string]interface{}{
					"name":  "Test User",
					"email": "test@example.com",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.payloadMap)
			req, _ := http.NewRequest(http.MethodPost, "/test", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.wantErr {
				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status 400 for bad payload, got %d", w.Code)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("expected status 200 for good payload, got %d", w.Code)
				}
			}
		})
	}
}
