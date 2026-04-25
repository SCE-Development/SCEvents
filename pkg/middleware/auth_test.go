package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter(minimumState int, clientAPIURL string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequireAuth(minimumState, clientAPIURL))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		userRole, _ := c.Get("userRole")
		c.JSON(http.StatusOK, gin.H{
			"userID":   userID,
			"userRole": userRole,
		})
	})
	return r
}

func TestRequireAuth(t *testing.T) {
	tests := []struct {
		name               string
		authHeader         string
		mockAPIStatus      int
		mockAPIResponse    interface{}
		minimumState       int
		expectedStatus     int
		expectedRole       string
		expectedUserID     string
	}{
		{
			name:           "missing auth header",
			authHeader:     "",
			minimumState:   MembershipStateMember,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "api returns unauthorized",
			authHeader:     "Bearer invalid-token",
			mockAPIStatus:  http.StatusUnauthorized,
			minimumState:   MembershipStateMember,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:       "api returns invalid json",
			authHeader: "Bearer valid-token",
			mockAPIStatus: http.StatusOK,
			mockAPIResponse: "invalid json", // It'll be sent as string without json encoding for testing fail in Unmarshal if not careful, wait I'll handle that in mock handler
			minimumState: MembershipStateMember,
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:       "api returns insufficient access level",
			authHeader: "Bearer valid-token",
			mockAPIStatus: http.StatusOK,
			mockAPIResponse: map[string]interface{}{
				"_id":         "user-1",
				"accessLevel": float64(MembershipStatePending),
			},
			minimumState:   MembershipStateMember,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:       "valid member token",
			authHeader: "Bearer valid-token",
			mockAPIStatus: http.StatusOK,
			mockAPIResponse: map[string]interface{}{
				"_id":         "user-1",
				"accessLevel": float64(MembershipStateMember),
			},
			minimumState:   MembershipStateMember,
			expectedStatus: http.StatusOK,
			expectedRole:   "User",
			expectedUserID: "user-1",
		},
		{
			name:       "valid admin token",
			authHeader: "Bearer valid-token",
			mockAPIStatus: http.StatusOK,
			mockAPIResponse: map[string]interface{}{
				"_id":         "admin-1",
				"accessLevel": float64(MembershipStateOfficer),
			},
			minimumState:   MembershipStateMember,
			expectedStatus: http.StatusOK,
			expectedRole:   "Admin",
			expectedUserID: "admin-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != tt.authHeader {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				w.WriteHeader(tt.mockAPIStatus)
				if tt.name == "api returns invalid json" {
					_, _ = w.Write([]byte(`{invalid-json`))
					return
				}
				if tt.mockAPIResponse != nil {
					_ = json.NewEncoder(w).Encode(tt.mockAPIResponse)
				}
			}))
			defer mockServer.Close()

			router := setupRouter(tt.minimumState, mockServer.URL)

			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if w.Code == http.StatusOK {
				var response map[string]string
				_ = json.Unmarshal(w.Body.Bytes(), &response)
				
				if response["userID"] != tt.expectedUserID {
					t.Errorf("expected userID %s, got %s", tt.expectedUserID, response["userID"])
				}
				if response["userRole"] != tt.expectedRole {
					t.Errorf("expected userRole %s, got %s", tt.expectedRole, response["userRole"])
				}
			}
		})
	}
}
