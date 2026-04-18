package middleware

import (
	"encoding/json"
	"io"
	"log"
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

		req, _ := http.NewRequest("POST", clientAPIURL+"/api/Auth/verify", nil)
		req.Header.Set("Authorization", authHeader)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			// if not success, the token was invalid or the API is down
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token verification failed"})
			return
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read token verification response"})
			return
		}

		log.Printf("API Response: %s\n", string(bodyBytes))

		// parse Clark's response (decoded JWT payload)
		var user map[string]interface{}
		if err := json.Unmarshal(bodyBytes, &user); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse token verification response"})
			return
		}

		// read accessLevel (float64 due to JSON unmarshaling)
		accessLevel, ok := user["accessLevel"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Invalid user data from Clark API"})
			return
		}

		if int(accessLevel) < minimumState {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient privileges"})
			return
		}

		role := "User"
		if int(accessLevel) >= MembershipStateOfficer {
			role = "Admin"
		}

		userID, _ := user["_id"].(string)

		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Next()
	}
}
