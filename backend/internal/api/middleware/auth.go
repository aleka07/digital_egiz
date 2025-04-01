package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/db/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware provides JWT authentication middleware for Gin
type AuthMiddleware struct {
	jwtConfig *config.JWTConfig
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(jwtConfig *config.JWTConfig) *AuthMiddleware {
	return &AuthMiddleware{
		jwtConfig: jwtConfig,
	}
}

// RequireAuth middleware ensures that a valid JWT token is present in the request
func (am *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			return
		}

		// Check if the Authorization header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer {token}"})
			return
		}

		// Parse and validate the token
		token := parts[1]
		claims, err := validateToken(token, am.jwtConfig.Secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		// Set user claims in context for later use
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// RequireRole middleware ensures that the authenticated user has the required role
func (am *AuthMiddleware) RequireRole(role models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if user is authenticated
		userRole, exists := c.Get("user_role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "User is not authenticated"})
			return
		}

		// Check if user has the required role
		if userRole != string(models.RoleAdmin) && userRole != string(role) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			return
		}

		c.Next()
	}
}

// RequireAdmin middleware ensures that the authenticated user has admin role
func (am *AuthMiddleware) RequireAdmin() gin.HandlerFunc {
	return am.RequireRole(models.RoleAdmin)
}

// validateToken validates the JWT token and returns the claims
func validateToken(tokenString string, secretKey string) (*models.Claims, error) {
	if secretKey == "" {
		return nil, errors.New("JWT secret key is not configured")
	}

	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Validate the alg is what we expect
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}

		return []byte(secretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errors.New("token has expired")
		}
		return nil, errors.New("invalid token")
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
} 