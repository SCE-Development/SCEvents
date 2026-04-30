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
		accessLevel, _ := c.Get("accessLevel")
		c.JSON(http.StatusOK, gin.H{
			"userID":      userID,
			"userRole":    userRole,
			"accessLevel": accessLevel,
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
			expectedStatus: http.StatusUnauthorized,
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
			name:       "officer token fails admin requirement",
			authHeader: "Bearer valid-token",
			mockAPIStatus: http.StatusOK,
			mockAPIResponse: map[string]interface{}{
				"_id":         "officer-1",
				"accessLevel": float64(MembershipStateOfficer),
			},
			minimumState:   MembershipStateAdmin,
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
				var response map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &response)

				if response["userID"] != tt.expectedUserID {
					t.Errorf("expected userID %s, got %v", tt.expectedUserID, response["userID"])
				}
				if response["userRole"] != tt.expectedRole {
					t.Errorf("expected userRole %s, got %v", tt.expectedRole, response["userRole"])
				}
			}
		})
	}
}

func setupOptionalAuthRouter(clientAPIURL string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(OptionalAuth(clientAPIURL))
	r.GET("/test", func(c *gin.Context) {
		userID, _ := c.Get("userID")
		userRole, _ := c.Get("userRole")
		accessLevel, _ := c.Get("accessLevel")
		c.JSON(http.StatusOK, gin.H{
			"userID":      userID,
			"userRole":    userRole,
			"accessLevel": accessLevel,
		})
	})
	return r
}

func TestOptionalAuth_AllowsAnonymousRequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("verification API should not be called when auth header is missing")
	}))
	defer mockServer.Close()

	router := setupOptionalAuthRouter(mockServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if _, ok := response["userID"]; ok && response["userID"] != nil {
		t.Fatalf("expected no userID, got %v", response["userID"])
	}
	if _, ok := response["userRole"]; ok && response["userRole"] != nil {
		t.Fatalf("expected no userRole, got %v", response["userRole"])
	}
}

func TestOptionalAuth_SetsContextForValidToken(t *testing.T) {
	authHeader := "Bearer valid-token"

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != authHeader {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"_id":         "user-123",
			"accessLevel": float64(MembershipStateMember),
		})
	}))
	defer mockServer.Close()

	router := setupOptionalAuthRouter(mockServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", authHeader)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid JSON, got %v", err)
	}

	if response["userID"] != "user-123" {
		t.Fatalf("expected userID user-123, got %v", response["userID"])
	}
	if response["userRole"] != "User" {
		t.Fatalf("expected userRole User, got %v", response["userRole"])
	}
	if int(response["accessLevel"].(float64)) != MembershipStateMember {
		t.Fatalf("expected accessLevel 1, got %v", response["accessLevel"])
	}
}

func TestOptionalAuth_RejectsInvalidToken(t *testing.T) {
	authHeader := "Bearer invalid-token"

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer mockServer.Close()

	router := setupOptionalAuthRouter(mockServer.URL)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", authHeader)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}
}
