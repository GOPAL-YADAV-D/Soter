package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/GOPAL-YADAV-D/Soter/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// TokenBucket represents a rate limiter for a specific entity
type TokenBucket struct {
	limiter    *rate.Limiter
	lastAccess time.Time
}

// RateLimiter manages rate limiting for users and organizations
type RateLimiter struct {
	userBuckets map[uuid.UUID]*TokenBucket
	orgBuckets  map[uuid.UUID]*TokenBucket
	mutex       sync.RWMutex
	
	// Configuration
	userRPS    rate.Limit
	userBurst  int
	orgRPS     rate.Limit
	orgBurst   int
	
	// Cleanup
	cleanupInterval time.Duration
	maxIdleTime     time.Duration
}

// NewRateLimiter creates a new rate limiter with token bucket algorithm
func NewRateLimiter(cfg *config.Config) *RateLimiter {
	rl := &RateLimiter{
		userBuckets: make(map[uuid.UUID]*TokenBucket),
		orgBuckets:  make(map[uuid.UUID]*TokenBucket),
		
		// Default limits (can be overridden per user/org from database)
		userRPS:   rate.Limit(cfg.RateLimitRPS),
		userBurst: cfg.RateLimitBurst,
		orgRPS:    rate.Limit(cfg.RateLimitRPS * 10), // Organizations get higher limits
		orgBurst:  cfg.RateLimitBurst * 10,
		
		// Cleanup settings
		cleanupInterval: 5 * time.Minute,
		maxIdleTime:     30 * time.Minute,
	}
	
	// Start cleanup routine
	go rl.cleanupRoutine()
	
	return rl
}

// GetUserLimiter gets or creates a rate limiter for a user
func (rl *RateLimiter) GetUserLimiter(userID uuid.UUID) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	bucket, exists := rl.userBuckets[userID]
	if !exists {
		bucket = &TokenBucket{
			limiter:    rate.NewLimiter(rl.userRPS, rl.userBurst),
			lastAccess: time.Now(),
		}
		rl.userBuckets[userID] = bucket
	} else {
		bucket.lastAccess = time.Now()
	}
	
	return bucket.limiter
}

// GetOrgLimiter gets or creates a rate limiter for an organization
func (rl *RateLimiter) GetOrgLimiter(orgID uuid.UUID) *rate.Limiter {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	bucket, exists := rl.orgBuckets[orgID]
	if !exists {
		bucket = &TokenBucket{
			limiter:    rate.NewLimiter(rl.orgRPS, rl.orgBurst),
			lastAccess: time.Now(),
		}
		rl.orgBuckets[orgID] = bucket
	} else {
		bucket.lastAccess = time.Now()
	}
	
	return bucket.limiter
}

// RateLimitMiddleware returns the rate limiting middleware
func (rl *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract user ID from context (set by auth middleware)
		userIDInterface, exists := c.Get("userID")
		if !exists {
			// Allow unauthenticated requests to pass through 
			// (they might be handled by other middleware)
			c.Next()
			return
		}
		
		userID, ok := userIDInterface.(uuid.UUID)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid user ID format",
			})
			c.Abort()
			return
		}
		
		// Get rate limiter for this user
		userLimiter := rl.GetUserLimiter(userID)
		
		// Check if request is allowed
		if !userLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please slow down.",
			})
			c.Abort()
			return
		}
		
		// Add rate limit headers for successful requests
		c.Header("X-RateLimit-Limit", strconv.Itoa(int(userLimiter.Limit())))
		// Calculate remaining tokens approximately
		remainingTokens := userLimiter.Burst()
		if userLimiter.Tokens() < float64(remainingTokens) {
			remainingTokens = int(userLimiter.Tokens())
		}
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remainingTokens))
		
		c.Next()
	}
}

// cleanupRoutine periodically removes unused limiters to prevent memory leaks
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.cleanupInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes limiters that haven't been accessed recently
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	
	// Clean up user buckets
	for userID, bucket := range rl.userBuckets {
		if now.Sub(bucket.lastAccess) > rl.maxIdleTime {
			delete(rl.userBuckets, userID)
		}
	}
	
	// Clean up organization buckets
	for orgID, bucket := range rl.orgBuckets {
		if now.Sub(bucket.lastAccess) > rl.maxIdleTime {
			delete(rl.orgBuckets, orgID)
		}
	}
	
	logrus.Debugf("Rate limiter cleanup: %d user buckets, %d org buckets", 
		len(rl.userBuckets), len(rl.orgBuckets))
}

// IPRateLimitMiddleware provides IP-based rate limiting as a fallback
func IPRateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	ipLimiters := make(map[string]*rate.Limiter)
	var mu sync.RWMutex
	
	// Cleanup routine for IP limiters
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		
		for range ticker.C {
			mu.Lock()
			// Remove IP limiters older than 10 minutes
			// This is a simple cleanup - in production, you might want a more sophisticated approach
			ipLimiters = make(map[string]*rate.Limiter)
			mu.Unlock()
		}
	}()
	
	return func(c *gin.Context) {
		ip := c.ClientIP()
		
		mu.Lock()
		limiter, exists := ipLimiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(requestsPerMinute)/60, requestsPerMinute) // per second with burst
			ipLimiters[ip] = limiter
		}
		mu.Unlock()
		
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "IP rate limit exceeded",
				"message": "Too many requests from this IP address.",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}