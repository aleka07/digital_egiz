package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestSetup contains utilities for testing
type TestSetup struct {
	Router   *gin.Engine
	DB       *db.Database
	Logger   *utils.Logger
	Config   *config.Config
	Cleanup  func()
	Requires *require.Assertions
}

// NewTestSetup creates a new test setup with in-memory SQLite database
func NewTestSetup(t require.TestingT) *TestSetup {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test logger directly using zap for tests
	zapConfig := zap.NewDevelopmentConfig()
	zapConfig.OutputPaths = []string{"stdout"}
	zapLogger, err := zapConfig.Build()
	if err != nil {
		require.FailNow(t, "Failed to create zap logger", err)
	}

	logger := &utils.Logger{
		Logger: zapLogger,
	}

	// Create test config
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:                 "test-secret-key-for-testing-only",
			ExpirationHours:        1,
			RefreshSecret:          "test-refresh-secret-key-for-testing-only",
			RefreshExpirationHours: 168, // 7 days
		},
		Database: config.DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			DBName:   "test_db",
			SSLMode:  "disable",
			TimeZone: "UTC",
		},
		Log: config.LogConfig{
			Level:  "info",
			Format: "json",
		},
	}

	// Create in-memory SQLite database
	gormDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		require.FailNow(t, "Failed to create in-memory database", err)
	}

	// Create database wrapper (compatible with the real db.Database)
	database := &db.Database{
		DB: gormDB,
	}

	// Create test router
	router := gin.New()
	router.Use(gin.Recovery())

	// Create cleanup function
	cleanup := func() {
		zapLogger.Sync()
		sqlDB, _ := gormDB.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	return &TestSetup{
		Router:   router,
		DB:       database,
		Logger:   logger,
		Config:   cfg,
		Cleanup:  cleanup,
		Requires: require.New(t),
	}
}

// ExecuteRequest executes a test request and returns the response
func (ts *TestSetup) ExecuteRequest(method, path string, body interface{}, headers map[string]string) *httptest.ResponseRecorder {
	// Create request
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		ts.Requires.NoError(err, "Failed to marshal request body")
	}

	req, err := http.NewRequest(method, path, bytes.NewBuffer(reqBody))
	ts.Requires.NoError(err, "Failed to create request")

	// Set content type if request has body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Set additional headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Execute request
	resp := httptest.NewRecorder()
	ts.Router.ServeHTTP(resp, req)

	return resp
}

// ParseResponse parses the JSON response into the provided struct
func (ts *TestSetup) ParseResponse(response *httptest.ResponseRecorder, target interface{}) {
	err := json.Unmarshal(response.Body.Bytes(), target)
	ts.Requires.NoError(err, "Failed to parse response body: %s", response.Body.String())
}

// SetupTestDatabase creates and migrates test database tables
func (ts *TestSetup) SetupTestDatabase(models ...interface{}) {
	err := ts.DB.DB.AutoMigrate(models...)
	ts.Requires.NoError(err, "Failed to migrate database")
}

// CreateTestAuthToken creates a JWT token for testing authenticated endpoints
func (ts *TestSetup) CreateTestAuthToken(userID uint, email string, role models.Role) string {
	// Create claims for the token
	claims := &models.Claims{
		UserID: userID,
		Email:  email,
		Role:   string(role),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "digital-egiz-test",
		},
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token
	tokenString, err := token.SignedString([]byte(ts.Config.JWT.Secret))
	ts.Requires.NoError(err, "Failed to sign JWT token")

	return tokenString
}

// TempFile creates a temporary file with the given content
func (ts *TestSetup) TempFile(content string) (string, func()) {
	tempFile, err := os.CreateTemp("", "test-*.tmp")
	ts.Requires.NoError(err, "Failed to create temp file")

	_, err = tempFile.WriteString(content)
	ts.Requires.NoError(err, "Failed to write to temp file")

	err = tempFile.Close()
	ts.Requires.NoError(err, "Failed to close temp file")

	cleanup := func() {
		os.Remove(tempFile.Name())
	}

	return tempFile.Name(), cleanup
}

// SeedTestUser creates a test user in the database for testing
func (ts *TestSetup) SeedTestUser(email, password string, isAdmin bool) uint {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	ts.Requires.NoError(err, "Failed to hash password")

	// Create user record
	user := &models.User{
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: "Test",
		LastName:  "User",
		Role:      models.RoleUser,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Set admin role if needed
	if isAdmin {
		user.Role = models.RoleAdmin
	}

	result := ts.DB.DB.Create(user)
	ts.Requires.NoError(result.Error, "Failed to create test user")

	return user.ID
}
