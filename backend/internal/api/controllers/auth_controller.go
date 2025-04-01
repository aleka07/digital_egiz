package controllers

import (
	"net/http"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/digital-egiz/backend/internal/services"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gin-gonic/gin"
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

// TokenResponse represents the login/register response body
type TokenResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	UserID    uint      `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
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

	// Generate JWT token
	token, err := user.GenerateToken(ac.jwtConfig.Secret, ac.jwtConfig.ExpirationSec)
	if err != nil {
		ac.logger.Error("Failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authentication token"})
		return
	}

	expiresAt := time.Now().Add(time.Duration(ac.jwtConfig.ExpirationSec) * time.Second)

	c.JSON(http.StatusOK, TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
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

	// Generate JWT token
	token, err := user.GenerateToken(ac.jwtConfig.Secret, ac.jwtConfig.ExpirationSec)
	if err != nil {
		ac.logger.Error("Failed to generate token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate authentication token"})
		return
	}

	expiresAt := time.Now().Add(time.Duration(ac.jwtConfig.ExpirationSec) * time.Second)

	c.JSON(http.StatusCreated, TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		UserID:    user.ID,
		Email:     user.Email,
		Role:      string(user.Role),
	})
} 