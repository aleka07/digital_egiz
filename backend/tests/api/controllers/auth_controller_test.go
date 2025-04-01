package controllers_test

import (
	"net/http"
	"testing"

	"github.com/digital-egiz/backend/internal/api/controllers"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	testutils "github.com/digital-egiz/backend/tests/utils"
	"github.com/stretchr/testify/assert"
)

func TestAuthController_RegisterAndLogin(t *testing.T) {
	// Setup test environment
	ts := testutils.NewTestSetup(t)
	defer ts.Cleanup()

	// Create user model for database migration
	ts.SetupTestDatabase(&models.User{})

	// Create user service
	userService := services.NewUserService(ts.DB, ts.Logger)

	// Create auth controller
	authController := controllers.NewAuthController(userService, &ts.Config.JWT, ts.Logger)

	// Register auth routes
	authGroup := ts.Router.Group("/api")
	authController.RegisterRoutes(authGroup)

	// Test case: Register a new user
	t.Run("Should register a new user successfully", func(t *testing.T) {
		// Create register request body
		registerRequest := map[string]interface{}{
			"email":      "test@example.com",
			"password":   "securePassword123",
			"first_name": "Test",
			"last_name":  "User",
		}

		// Execute register request
		resp := ts.ExecuteRequest("POST", "/api/register", registerRequest, nil)

		// Assert response
		assert.Equal(t, http.StatusCreated, resp.Code)

		// Parse response
		var response map[string]interface{}
		ts.ParseResponse(resp, &response)

		// Assert user ID and token in response
		assert.NotNil(t, response["user"])
		assert.NotNil(t, response["access_token"])
		assert.NotNil(t, response["refresh_token"])

		// Verify user fields
		user := response["user"].(map[string]interface{})
		assert.Equal(t, registerRequest["email"], user["email"])
		assert.Equal(t, registerRequest["first_name"], user["first_name"])
		assert.Equal(t, registerRequest["last_name"], user["last_name"])
		assert.False(t, user["is_admin"].(bool))
	})

	// Test case: Register with duplicate email
	t.Run("Should return error when registering with duplicate email", func(t *testing.T) {
		// Create register request with duplicate email
		registerRequest := map[string]interface{}{
			"email":      "test@example.com", // Same as previous test
			"password":   "anotherPassword456",
			"first_name": "Another",
			"last_name":  "User",
		}

		// Execute register request
		resp := ts.ExecuteRequest("POST", "/api/register", registerRequest, nil)

		// Assert response (should fail with conflict)
		assert.Equal(t, http.StatusConflict, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message
		assert.Contains(t, response["error"], "already exists")
	})

	// Test case: Login with valid credentials
	t.Run("Should login successfully with valid credentials", func(t *testing.T) {
		// Create login request
		loginRequest := map[string]interface{}{
			"email":    "test@example.com",
			"password": "securePassword123",
		}

		// Execute login request
		resp := ts.ExecuteRequest("POST", "/api/login", loginRequest, nil)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)

		// Parse response
		var response map[string]interface{}
		ts.ParseResponse(resp, &response)

		// Assert user and tokens in response
		assert.NotNil(t, response["user"])
		assert.NotNil(t, response["access_token"])
		assert.NotNil(t, response["refresh_token"])

		// Verify user fields
		user := response["user"].(map[string]interface{})
		assert.Equal(t, "test@example.com", user["email"])
		assert.Equal(t, "Test", user["first_name"])
		assert.Equal(t, "User", user["last_name"])
	})

	// Test case: Login with invalid credentials
	t.Run("Should fail login with invalid credentials", func(t *testing.T) {
		// Create login request with wrong password
		loginRequest := map[string]interface{}{
			"email":    "test@example.com",
			"password": "wrongPassword",
		}

		// Execute login request
		resp := ts.ExecuteRequest("POST", "/api/login", loginRequest, nil)

		// Assert response (should fail with unauthorized)
		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message
		assert.Contains(t, response["error"], "invalid credentials")
	})

	// Test case: Refresh token
	t.Run("Should refresh access token with valid refresh token", func(t *testing.T) {
		// First login to get a refresh token
		loginRequest := map[string]interface{}{
			"email":    "test@example.com",
			"password": "securePassword123",
		}

		loginResp := ts.ExecuteRequest("POST", "/api/login", loginRequest, nil)
		assert.Equal(t, http.StatusOK, loginResp.Code)

		var loginResponse map[string]interface{}
		ts.ParseResponse(loginResp, &loginResponse)
		refreshToken := loginResponse["refresh_token"].(string)

		// Create refresh token request
		refreshRequest := map[string]interface{}{
			"refresh_token": refreshToken,
		}

		// Execute refresh token request
		resp := ts.ExecuteRequest("POST", "/api/refresh", refreshRequest, nil)

		// Assert response
		assert.Equal(t, http.StatusOK, resp.Code)

		// Parse response
		var response map[string]interface{}
		ts.ParseResponse(resp, &response)

		// Assert new access token in response
		assert.NotNil(t, response["access_token"])
		assert.NotEqual(t, loginResponse["access_token"], response["access_token"])
	})

	// Test case: Refresh with invalid token
	t.Run("Should fail refresh with invalid refresh token", func(t *testing.T) {
		// Create refresh token request with invalid token
		refreshRequest := map[string]interface{}{
			"refresh_token": "invalid-refresh-token",
		}

		// Execute refresh token request
		resp := ts.ExecuteRequest("POST", "/api/refresh", refreshRequest, nil)

		// Assert response (should fail with unauthorized)
		assert.Equal(t, http.StatusUnauthorized, resp.Code)

		// Parse response
		var response map[string]string
		ts.ParseResponse(resp, &response)

		// Assert error message
		assert.Contains(t, response["error"], "invalid refresh token")
	})
}
