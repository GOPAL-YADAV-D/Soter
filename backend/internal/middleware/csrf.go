package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/GOPAL-YADAV-D/Soter/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CSRFProtection provides CSRF protection using double-submit cookie pattern
type CSRFProtection struct {
	secret      []byte
	cookieName  string
	headerName  string
	tokenLength int
}

// NewCSRFProtection creates a new CSRF protection middleware
func NewCSRFProtection(cfg *config.Config) *CSRFProtection {
	return &CSRFProtection{
		secret:      []byte(cfg.CSRFSecret),
		cookieName:  "csrf_token",
		headerName:  "X-CSRF-Token",
		tokenLength: 32,
	}
}

// CSRFMiddleware creates the CSRF protection middleware
func (csrf *CSRFProtection) CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip CSRF protection for GET, HEAD, OPTIONS (safe methods)
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			// Set CSRF token for safe methods
			_ = csrf.setCSRFToken(c)
			c.Next()
			return
		}

		// Skip CSRF for certain API endpoints that use API keys
		if strings.HasPrefix(c.Request.URL.Path, "/api/webhook") ||
			strings.HasPrefix(c.Request.URL.Path, "/api/health") {
			c.Next()
			return
		}

		// Validate CSRF token for unsafe methods (POST, PUT, DELETE, PATCH)
		if !csrf.validateCSRFToken(c) {
			logrus.Warnf("CSRF token validation failed for %s %s from %s",
				c.Request.Method, c.Request.URL.Path, c.ClientIP())

			c.JSON(http.StatusForbidden, gin.H{
				"error":   "CSRF token validation failed",
				"message": "Invalid or missing CSRF token",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// setCSRFToken generates and sets a new CSRF token
func (csrf *CSRFProtection) setCSRFToken(c *gin.Context) string {
	// Generate random token
	tokenBytes := make([]byte, csrf.tokenLength)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		logrus.Errorf("Failed to generate CSRF token: %v", err)
		return ""
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	// Set CSRF token in cookie (HttpOnly, Secure, SameSite)
	c.SetCookie(
		csrf.cookieName,
		token,
		3600, // 1 hour
		"/",
		"",                   // domain (empty means current domain)
		c.Request.TLS != nil, // secure (only over HTTPS)
		true,                 // httpOnly
	)

	// Also make token available to JavaScript via header (for SPA applications)
	c.Header("X-CSRF-Token", token)
	return token
}

// validateCSRFToken validates the CSRF token from cookie and header
func (csrf *CSRFProtection) validateCSRFToken(c *gin.Context) bool {
	// Get token from cookie
	cookieToken, err := c.Cookie(csrf.cookieName)
	if err != nil {
		logrus.Debugf("CSRF cookie not found: %v", err)
		return false
	}

	// Get token from header
	headerToken := c.GetHeader(csrf.headerName)
	if headerToken == "" {
		// Also check form field as fallback
		headerToken = c.PostForm("csrf_token")
	}

	if headerToken == "" {
		logrus.Debug("CSRF header token not found")
		return false
	}

	// Compare tokens using constant-time comparison
	return subtle.ConstantTimeCompare([]byte(cookieToken), []byte(headerToken)) == 1
}

// CSRFProtection returns the CSRF protection middleware
func (csrf *CSRFProtection) CSRFProtection() gin.HandlerFunc {
	return csrf.CSRFMiddleware()
}

// SecureHeaders returns the secure headers middleware
func (csrf *CSRFProtection) SecureHeaders() gin.HandlerFunc {
	return SecureHeaders()
}

// GetCSRFToken returns a handler that provides CSRF tokens
func (csrf *CSRFProtection) GetCSRFTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := csrf.setCSRFToken(c)

		c.JSON(http.StatusOK, gin.H{
			"csrf_token": token,
		})
	}
}

// GetCSRFToken extracts CSRF token for manual validation
func (csrf *CSRFProtection) GetCSRFToken(c *gin.Context) string {
	token, _ := c.Cookie(csrf.cookieName)
	return token
}

// SecureHeaders middleware adds security headers
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Prevent MIME sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Enable XSS protection
		c.Header("X-XSS-Protection", "1; mode=block")

		// Enforce HTTPS (if running on HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'"
		c.Header("Content-Security-Policy", csp)

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions Policy (Feature Policy)
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		c.Next()
	}
}

// CORSSecurityMiddleware provides secure CORS configuration
func CORSSecurityMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is in allowed list
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		} else {
			// Default to same-origin
			c.Header("Access-Control-Allow-Origin", c.Request.Host)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-CSRF-Token")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
