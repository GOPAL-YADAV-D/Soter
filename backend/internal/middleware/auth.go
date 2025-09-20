package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/GOPAL-YADAV-D/Soter/internal/models"
	"github.com/GOPAL-YADAV-D/Soter/internal/services"
)

const (
	AuthContextKey = "auth_context"
	UserContextKey = "user"
)

// AuthMiddleware creates authentication middleware
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		token := tokenParts[1]

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			logrus.WithError(err).Debug("Token validation failed")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Create auth context
		authContext := &models.AuthContext{
			UserID:          claims.UserID,
			OrganizationID:  claims.OrganizationID,
			Role:            claims.Role,
			IsAuthenticated: true,
		}

		// Add to context
		c.Set(AuthContextKey, authContext)
		c.Set(UserContextKey, claims)

		// Add to request context for GraphQL
		ctx := context.WithValue(c.Request.Context(), AuthContextKey, authContext)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// OptionalAuthMiddleware creates optional authentication middleware
// This allows endpoints to work with or without authentication
func OptionalAuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth header, continue without authentication
			c.Next()
			return
		}

		// Check if it's a Bearer token
		tokenParts := strings.Split(authHeader, " ")
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			// Invalid format, continue without authentication
			c.Next()
			return
		}

		token := tokenParts[1]

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			// Invalid token, continue without authentication
			logrus.WithError(err).Debug("Optional token validation failed")
			c.Next()
			return
		}

		// Create auth context
		authContext := &models.AuthContext{
			UserID:          claims.UserID,
			OrganizationID:  claims.OrganizationID,
			Role:            claims.Role,
			IsAuthenticated: true,
		}

		// Add to context
		c.Set(AuthContextKey, authContext)
		c.Set(UserContextKey, claims)

		// Add to request context for GraphQL
		ctx := context.WithValue(c.Request.Context(), AuthContextKey, authContext)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// RequireRole creates middleware that requires a specific role
func RequireRole(requiredRole models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		authContext, exists := c.Get(AuthContextKey)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		auth, ok := authContext.(*models.AuthContext)
		if !ok || !auth.IsAuthenticated {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		if !auth.Role.HasPermission(requiredRole) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":         "Insufficient permissions",
				"required_role": string(requiredRole),
				"user_role":     string(auth.Role),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAdmin creates middleware that requires admin role
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

// RequireMember creates middleware that requires member role or higher
func RequireMember() gin.HandlerFunc {
	return RequireRole(models.RoleMember)
}

// RequireViewer creates middleware that requires viewer role or higher
func RequireViewer() gin.HandlerFunc {
	return RequireRole(models.RoleViewer)
}

// GetAuthContext extracts authentication context from gin context
func GetAuthContext(c *gin.Context) (*models.AuthContext, error) {
	authContext, exists := c.Get(AuthContextKey)
	if !exists {
		return nil, fmt.Errorf("authentication required")
	}

	auth, ok := authContext.(*models.AuthContext)
	if !ok || !auth.IsAuthenticated {
		return nil, fmt.Errorf("authentication required")
	}

	return auth, nil
}

// GetUserFromContext extracts user claims from gin context
func GetUserFromContext(c *gin.Context) (*services.JWTClaims, error) {
	userContext, exists := c.Get(UserContextKey)
	if !exists {
		return nil, fmt.Errorf("authentication required")
	}

	claims, ok := userContext.(*services.JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid user context")
	}

	return claims, nil
}

// RequireOrganizationAccess creates middleware that ensures user belongs to the organization
func RequireOrganizationAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		authContext, err := GetAuthContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
			})
			c.Abort()
			return
		}

		// Check if user has access to the organization
		if authContext.OrganizationID == uuid.Nil {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "No organization access",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware creates rate limiting middleware
func RateLimitMiddleware() gin.HandlerFunc {
	// TODO: Implement proper rate limiting with Redis
	// For now, this is a placeholder
	return func(c *gin.Context) {
		// Basic rate limiting logic would go here
		c.Next()
	}
}
