package middleware

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

// these should sync with Clark's membership states
// https://github.com/SCE-Development/Clark/blob/a53f28bdb4e0b02e94d481102784922da4fc6e6d/src/Enums.js#L23-L30
const (
	MembershipStateBanned    = -2
	MembershipStatePending   = -1
	MembershipStateNonMember = 0
	MembershipStateMember    = 1
	MembershipStateOfficer   = 2
	MembershipStateAdmin     = 3
)

func RequireAuth(minimumState int, clientAPIURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No authorization header"})
			return
		}

		userID, role, accessLevel, err := verifyAuthHeader(authHeader, clientAPIURL)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token verification failed"})
			return
		}

		if accessLevel < minimumState {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges"})
			return
		}

		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Set("accessLevel", accessLevel)
		c.Next()
	}
}

func verifyAuthHeader(authHeader string, clientAPIURL string) (string, string, int, error) {
	req, err := http.NewRequest("POST", clientAPIURL+"/api/Auth/verify", nil)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create token verification request")
	}
	req.Header.Set("Authorization", authHeader)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", "", 0, fmt.Errorf("token verification failed")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read token verification response")
	}

	// parse Clark's response (decoded JWT payload)
	var user map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &user); err != nil {
		return "", "", 0, fmt.Errorf("failed to parse token verification response")
	}

	// read accessLevel (float64 due to JSON unmarshaling)
	accessLevelFloat, ok := user["accessLevel"].(float64)
	if !ok {
		return "", "", 0, fmt.Errorf("invalid user data from Clark API")
	}

	accessLevel := int(accessLevelFloat)

	role := "non_member"
	switch {
	case accessLevel >= MembershipStateAdmin:
		role = "admin"
	case accessLevel >= MembershipStateOfficer:
		role = "officer"
	case accessLevel >= MembershipStateMember:
		role = "member"
	}

	userID, _ := user["_id"].(string)

	return userID, role, accessLevel, nil
}

// OptionalAuth enriches the request with user context when a valid token is present,
// but still allows anonymous access for public endpoints
func OptionalAuth(clientAPIURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		userID, role, accessLevel, err := verifyAuthHeader(authHeader, clientAPIURL)
		if err != nil {
			// if not success, the token was invalid or the user is pending, proceed anonymously
			c.Next()
			return
		}

		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Set("accessLevel", accessLevel)
		c.Next()
	}
}
