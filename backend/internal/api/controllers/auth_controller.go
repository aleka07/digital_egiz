package controllers

import (
	"net/http"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

// LoginRequest represents the login request body
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
}

// TokenResponse represents the login/register/token refresh response body
type TokenResponse struct {
	Token        string    `json:"token"`         // Access token
	RefreshToken string    `json:"refresh_token"` // Refresh token for obtaining new access tokens
	ExpiresAt    time.Time `json:"expires_at"`    // When the access token expires
	UserID       uint      `json:"user_id"`       // ID of the authenticated user
	Email        string    `json:"email"`         // Email of the authenticated user
	Role         string    `json:"role"`          // Role of the authenticated user
}

// RefreshRequest represents the token refresh request body
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// AuthController handles authentication-related endpoints
type AuthController struct {
	userService *services.UserService
	jwtConfig   *config.JWTConfig
	logger      *utils.Logger
}

// NewAuthController creates a new authentication controller
func NewAuthController(userService *services.UserService, jwtConfig *config.JWTConfig, logger *utils.Logger) *AuthController {
	return &AuthController{
		userService: userService,
		jwtConfig:   jwtConfig,
		logger:      logger.Named("auth_controller"),
	}
}

// RegisterRoutes registers the controller's routes with the router group
func (ac *AuthController) RegisterRoutes(router *gin.RouterGroup) {
	auth := router.Group("/auth")
	{
		auth.POST("/login", ac.Login)
		auth.POST("/register", ac.Register)
		auth.POST("/refresh", ac.RefreshToken)
	}
}

// Login handles user authentication and returns a JWT token
// @Summary Login user
// @Description Authenticate user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param login_request body LoginRequest true "Login credentials"
// @Success 200 {object} TokenResponse "Login successful"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Invalid credentials"
// @Failure 500 {object} map[string]string "Server error"
// @Router /auth/login [post]
func (ac *AuthController) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := ac.userService.Authenticate(req.Email, req.Password)
	if err != nil {
		ac.logger.Warn("Login failed", zap.String("email", req.Email), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	// Update last login time
	if err := ac.userService.UpdateLastLogin(user.ID); err != nil {
		ac.logger.Error("Failed to update last login time", zap.Uint("user_id", user.ID), zap.Error(err))
	}

	// Generate access token
	token, err := user.GenerateToken(ac.jwtConfig.Secret, ac.jwtConfig.ExpirationHours*3600)
	if err != nil {
		ac.logger.Error("Failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authentication token"})
		return
	}

	// Generate refresh token (longer expiration)
	refreshToken, err := user.GenerateToken(ac.jwtConfig.RefreshSecret, ac.jwtConfig.RefreshExpirationHours*3600)
	if err != nil {
		ac.logger.Error("Failed to generate refresh token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	expiresAt := time.Now().Add(time.Duration(ac.jwtConfig.ExpirationHours) * time.Hour)

	c.JSON(http.StatusOK, TokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		UserID:       user.ID,
		Email:        user.Email,
		Role:         string(user.Role),
	})
}

// Register handles user registration
// @Summary Register new user
// @Description Register a new user and return JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param register_request body RegisterRequest true "Registration information"
// @Success 201 {object} TokenResponse "Registration successful"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 409 {object} map[string]string "Email already exists"
// @Failure 500 {object} map[string]string "Server error"
// @Router /auth/register [post]
func (ac *AuthController) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create new user
	user := &models.User{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Role:      models.RoleUser,
		Active:    true,
	}

	// Save user to database
	if err := ac.userService.Create(user); err != nil {
		ac.logger.Warn("Registration failed", zap.String("email", req.Email), zap.Error(err))

		// Check if the error is due to email already existing
		if err.Error() == "email already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": "Email is already registered"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		return
	}

	// Generate access token
	token, err := user.GenerateToken(ac.jwtConfig.Secret, ac.jwtConfig.ExpirationHours*3600)
	if err != nil {
		ac.logger.Error("Failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authentication token"})
		return
	}

	// Generate refresh token
	refreshToken, err := user.GenerateToken(ac.jwtConfig.RefreshSecret, ac.jwtConfig.RefreshExpirationHours*3600)
	if err != nil {
		ac.logger.Error("Failed to generate refresh token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate refresh token"})
		return
	}

	expiresAt := time.Now().Add(time.Duration(ac.jwtConfig.ExpirationHours) * time.Hour)

	c.JSON(http.StatusCreated, TokenResponse{
		Token:        token,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		UserID:       user.ID,
		Email:        user.Email,
		Role:         string(user.Role),
	})
}

// RefreshToken handles token refresh
// @Summary Refresh JWT token
// @Description Exchange a valid refresh token for a new access token
// @Tags auth
// @Accept json
// @Produce json
// @Param refresh_request body RefreshRequest true "Refresh token"
// @Success 200 {object} TokenResponse "Token refresh successful"
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 401 {object} map[string]string "Invalid refresh token"
// @Failure 500 {object} map[string]string "Server error"
// @Router /auth/refresh [post]
func (ac *AuthController) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse and validate the refresh token
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(req.RefreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the algorithm
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(ac.jwtConfig.RefreshSecret), nil
	})

	if err != nil || !token.Valid {
		ac.logger.Warn("Invalid refresh token", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Get user from database using claims
	user, err := ac.userService.GetByID(claims.UserID)
	if err != nil {
		ac.logger.Error("Failed to find user for refresh token", zap.Uint("user_id", claims.UserID), zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Check if user is active
	if !user.Active {
		ac.logger.Warn("Inactive user attempted token refresh", zap.Uint("user_id", user.ID))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	// Generate new access token
	newToken, err := user.GenerateToken(ac.jwtConfig.Secret, ac.jwtConfig.ExpirationHours*3600)
	if err != nil {
		ac.logger.Error("Failed to generate new token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new token"})
		return
	}

	// Generate new refresh token
	newRefreshToken, err := user.GenerateToken(ac.jwtConfig.RefreshSecret, ac.jwtConfig.RefreshExpirationHours*3600)
	if err != nil {
		ac.logger.Error("Failed to generate new refresh token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate new refresh token"})
		return
	}

	expiresAt := time.Now().Add(time.Duration(ac.jwtConfig.ExpirationHours) * time.Hour)

	c.JSON(http.StatusOK, TokenResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
		UserID:       user.ID,
		Email:        user.Email,
		Role:         string(user.Role),
	})
}
