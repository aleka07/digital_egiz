package middleware_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/digital-egiz/backend/internal/api/middleware"
	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db/models"
	testutils "github.com/digital-egiz/backend/tests/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_RequireAuth(t *testing.T) {
	// Create test setup
	ts := testutils.NewTestSetup(t)
	defer ts.Cleanup()

	// Create JWT config
	jwtConfig := &config.JWTConfig{
		Secret:                 "test-secret-key",
		ExpirationHours:        1,
		RefreshSecret:          "test-refresh-secret",
		RefreshExpirationHours: 168, // 7 days
	}

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtConfig)

	// Setup test route
	ts.Router.GET("/protected", authMiddleware.RequireAuth(), func(c *gin.Context) {
		// Get user ID from context
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id not set in context"})
			return
		}

		// Return user ID in response
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	// Setup test route for admin access
	ts.Router.GET("/admin", authMiddleware.RequireAuth(), authMiddleware.RequireAdmin(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
	})

	t.Run("Should return 401 when no token provided", func(t *testing.T) {
		// Execute request without token
		resp := ts.ExecuteRequest("GET", "/protected", nil, nil)

		// Assert response
		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message
		assert.Contains(t, response["error"], "Authorization header is required")
	})

	t.Run("Should return 401 when invalid token format provided", func(t *testing.T) {
		// Execute request with invalid token format
		resp := ts.ExecuteRequest("GET", "/protected", nil, map[string]string{
			"Authorization": "InvalidFormat token123",
		})

		// Assert response
		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message
		assert.Contains(t, response["error"], "Authorization header format must be Bearer")
	})

	t.Run("Should return 401 when invalid token provided", func(t *testing.T) {
		// Execute request with invalid token
		resp := ts.ExecuteRequest("GET", "/protected", nil, map[string]string{
			"Authorization": "Bearer invalid-token",
		})

		// Assert response
		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message
		assert.Contains(t, response["error"], "invalid token")
	})

	t.Run("Should return 401 when expired token provided", func(t *testing.T) {
		// Create expired JWT token
		claims := &models.Claims{
			UserID: 1,
			Email:  "test@example.com",
			Role:   string(models.RoleUser),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			},
		}

		// Generate token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtConfig.Secret))
		assert.NoError(t, err)

		// Execute request with expired token
		resp := ts.ExecuteRequest("GET", "/protected", nil, map[string]string{
			"Authorization": "Bearer " + tokenString,
		})

		// Assert response
		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message contains token expired indication
		assert.Contains(t, response["error"], "Token has expired")
		assert.Equal(t, "token_expired", response["code"])
	})

	t.Run("Should return 200 when valid token provided", func(t *testing.T) {
		// Create valid JWT token
		claims := &models.Claims{
			UserID: 1,
			Email:  "test@example.com",
			Role:   string(models.RoleUser),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		// Generate token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtConfig.Secret))
		assert.NoError(t, err)

		// Execute request with valid token
		resp := ts.ExecuteRequest("GET", "/protected", nil, map[string]string{
			"Authorization": "Bearer " + tokenString,
		})

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)

		// Parse response
		var response map[string]interface{}
		ts.ParseResponse(resp, &response)

		// Assert user ID in response
		assert.Equal(t, float64(1), response["user_id"])
	})

	t.Run("Should return 403 when non-admin accesses admin route", func(t *testing.T) {
		// Create valid JWT token for non-admin
		claims := &models.Claims{
			UserID: 2,
			Email:  "user@example.com",
			Role:   string(models.RoleUser),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		// Generate token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtConfig.Secret))
		assert.NoError(t, err)

		// Execute request with valid token but non-admin user
		resp := ts.ExecuteRequest("GET", "/admin", nil, map[string]string{
			"Authorization": "Bearer " + tokenString,
		})

		// Assert response
		assert.Equal(t, http.StatusForbidden, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message
		assert.Contains(t, response["error"], "Insufficient permissions")
	})

	t.Run("Should return 200 when admin accesses admin route", func(t *testing.T) {
		// Create valid JWT token for admin
		claims := &models.Claims{
			UserID: 3,
			Email:  "admin@example.com",
			Role:   string(models.RoleAdmin),
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		// Generate token
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString([]byte(jwtConfig.Secret))
		assert.NoError(t, err)

		// Execute request with valid token for admin
		resp := ts.ExecuteRequest("GET", "/admin", nil, map[string]string{
			"Authorization": "Bearer " + tokenString,
		})

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert success message
		assert.Equal(t, "admin access granted", response["message"])
	})
}
