package services_test

import (
	"testing"
	"time"

	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	testutils "github.com/digital-egiz/backend/tests/utils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func TestUserService_GetByID(t *testing.T) {
	// Setup test environment
	ts := testutils.NewTestSetup(t)
	defer ts.Cleanup()

	// Create user model for database migration
	ts.SetupTestDatabase(&models.User{})

	// Create test service
	userService := services.NewUserService(ts.DB, ts.Logger)

	// Test case: Get non-existent user
	t.Run("Should return error when user does not exist", func(t *testing.T) {
		// Attempt to get non-existent user
		user, err := userService.GetByID(999)

		// Assert error and nil user
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	// Test case: Create and get user
	t.Run("Should return user when user exists", func(t *testing.T) {
		// Create test user in database
		testUser := &models.User{
			Email:     "test@example.com",
			Password:  "hashed_password", // In real scenario, this would be hashed
			FirstName: "Test",
			LastName:  "User",
			Role:      models.RoleUser,
			Active:    true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		result := ts.DB.DB.Create(testUser)
		assert.NoError(t, result.Error)
		assert.NotZero(t, testUser.ID)

		// Get user by ID
		user, err := userService.GetByID(testUser.ID)

		// Assert success
		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUser.ID, user.ID)
		assert.Equal(t, testUser.Email, user.Email)
		assert.Equal(t, testUser.FirstName, user.FirstName)
		assert.Equal(t, testUser.LastName, user.LastName)
		assert.Equal(t, testUser.Role, user.Role)
		assert.Equal(t, testUser.Active, user.Active)
	})
}

func TestUserService_Authenticate(t *testing.T) {
	// Setup test environment
	ts := testutils.NewTestSetup(t)
	defer ts.Cleanup()

	// Create user model for database migration
	ts.SetupTestDatabase(&models.User{})

	// Create test service
	userService := services.NewUserService(ts.DB, ts.Logger)

	// Test case: Invalid credentials for non-existent user
	t.Run("Should return error for non-existent user", func(t *testing.T) {
		// Attempt to authenticate non-existent user
		user, err := userService.Authenticate("nonexistent@example.com", "password123")

		// Assert error and nil user
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "invalid credentials")
	})

	// Test case: Create user and authenticate
	t.Run("Should successfully authenticate user with correct credentials", func(t *testing.T) {
		// Create test user
		email := "auth.test@example.com"
		password := "securePassword123"

		// Hash the password directly
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		assert.NoError(t, err)

		// Create user directly in the database bypassing hooks
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

		// Skip the BeforeCreate hooks to avoid double-hashing
		result := ts.DB.DB.Session(&gorm.Session{SkipHooks: true}).Create(user)
		assert.NoError(t, result.Error)
		assert.NotZero(t, user.ID)

		// Verify the hashed password was stored correctly
		var storedUser models.User
		err = ts.DB.DB.First(&storedUser, user.ID).Error
		assert.NoError(t, err)

		// Manual check that the password verification works
		err = bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(password))
		assert.NoError(t, err, "Password should be correctly hashed and verifiable")

		// Now try the actual service authentication
		authenticatedUser, err := userService.Authenticate(email, password)

		// Assert success
		assert.NoError(t, err)
		assert.NotNil(t, authenticatedUser)
		assert.Equal(t, email, authenticatedUser.Email)
		assert.Equal(t, user.ID, authenticatedUser.ID)
	})

	// Test case: Invalid password
	t.Run("Should return error for incorrect password", func(t *testing.T) {
		// Create test user
		email := "password.test@example.com"
		password := "correctPassword123"

		// Hash the password directly
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		assert.NoError(t, err)

		// Create user directly in the database bypassing hooks
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

		// Skip the BeforeCreate hooks to avoid double-hashing
		result := ts.DB.DB.Session(&gorm.Session{SkipHooks: true}).Create(user)
		assert.NoError(t, result.Error)
		assert.NotZero(t, user.ID)

		// Attempt to authenticate with wrong password
		authenticatedUser, err := userService.Authenticate(email, "wrongPassword456")

		// Assert error and nil user
		assert.Error(t, err)
		assert.Nil(t, authenticatedUser)
		assert.Contains(t, err.Error(), "invalid credentials")
	})
}

func TestUserService_Create(t *testing.T) {
	// Setup test environment
	ts := testutils.NewTestSetup(t)
	defer ts.Cleanup()

	// Create user model for database migration
	ts.SetupTestDatabase(&models.User{})

	// Create test service
	userService := services.NewUserService(ts.DB, ts.Logger)

	// Test case: Create user with valid data
	t.Run("Should create user with valid data", func(t *testing.T) {
		// Create user data
		user := &models.User{
			Email:     "new.user@example.com",
			Password:  "securePassword123",
			FirstName: "New",
			LastName:  "User",
			Role:      models.RoleUser,
			Active:    true,
		}

		// Create user
		err := userService.Create(user)

		// Assert success
		assert.NoError(t, err)
		assert.NotZero(t, user.ID)

		// Verify the user was created in the database
		var createdUser models.User
		result := ts.DB.DB.First(&createdUser, user.ID)
		assert.NoError(t, result.Error)
		assert.Equal(t, user.Email, createdUser.Email)
		assert.Equal(t, user.FirstName, createdUser.FirstName)
		assert.Equal(t, user.LastName, createdUser.LastName)
		assert.Equal(t, user.Role, createdUser.Role)
	})

	// Test case: Attempt to create user with duplicate email
	t.Run("Should return error when creating user with duplicate email", func(t *testing.T) {
		// Create user data with same email
		user := &models.User{
			Email:     "new.user@example.com", // Same as previous test
			Password:  "anotherPassword456",
			FirstName: "Another",
			LastName:  "User",
			Role:      models.RoleUser,
			Active:    true,
		}

		// Attempt to create user
		err := userService.Create(user)

		// Assert error due to duplicate email
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "email already exists")
	})
}
