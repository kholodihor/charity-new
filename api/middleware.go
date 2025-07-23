package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kholodihor/charity/token"
)

const (
	authorizationHeaderKey  = "authorization"
	authorizationTypeBearer = "bearer"
	authorizationPayloadKey = "authorization_payload"
)

// AuthMiddleware creates a gin middleware for authorization
func authMiddleware(tokenMaker token.Maker) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authorizationHeader := ctx.GetHeader(authorizationHeaderKey)

		if len(authorizationHeader) == 0 {
			err := errors.New("authorization header is not provided")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		fields := strings.Fields(authorizationHeader)
		if len(fields) < 2 {
			err := errors.New("invalid authorization header format")
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		authorizationType := strings.ToLower(fields[0])
		if authorizationType != authorizationTypeBearer {
			err := fmt.Errorf("unsupported authorization type %s", authorizationType)
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		accessToken := fields[1]
		payload, err := tokenMaker.VerifyToken(accessToken, token.TokenTypeAccessToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, errorResponse(err))
			return
		}

		ctx.Set(authorizationPayloadKey, payload)
		ctx.Next()
	}
}

// Visitor represents a client making requests
type Visitor struct {
	requests []time.Time
	mutex    sync.RWMutex
}

// RateLimiter represents a simple in-memory rate limiter
type RateLimiter struct {
	visitors map[string]*Visitor
	mutex    sync.RWMutex
	rate     int           // requests per minute
	window   time.Duration // time window
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*Visitor),
		rate:     requestsPerMinute,
		window:   time.Minute,
	}
	
	// Start cleanup goroutine to remove old visitors
	go rl.cleanupVisitors()
	
	return rl
}

// Allow checks if a request from the given IP is allowed
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	visitor, exists := rl.visitors[ip]
	if !exists {
		visitor = &Visitor{
			requests: make([]time.Time, 0),
		}
		rl.visitors[ip] = visitor
	}
	
	visitor.mutex.Lock()
	defer visitor.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rl.window)
	
	// Remove old requests outside the time window
	validRequests := make([]time.Time, 0)
	for _, reqTime := range visitor.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	visitor.requests = validRequests
	
	// Check if we're under the rate limit
	if len(visitor.requests) >= rl.rate {
		return false
	}
	
	// Add current request
	visitor.requests = append(visitor.requests, now)
	return true
}

// cleanupVisitors removes visitors that haven't made requests recently
func (rl *RateLimiter) cleanupVisitors() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		cutoff := now.Add(-10 * time.Minute) // Remove visitors inactive for 10 minutes
		
		for ip, visitor := range rl.visitors {
			visitor.mutex.RLock()
			lastRequest := time.Time{}
			if len(visitor.requests) > 0 {
				lastRequest = visitor.requests[len(visitor.requests)-1]
			}
			visitor.mutex.RUnlock()
			
			if lastRequest.Before(cutoff) {
				delete(rl.visitors, ip)
			}
		}
		rl.mutex.Unlock()
	}
}

// RateLimitMiddleware creates a gin middleware for rate limiting
func RateLimitMiddleware(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ip := ctx.ClientIP()
		
		if !rateLimiter.Allow(ip) {
			ctx.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
				"message": "too many requests, please try again later",
			})
			return
		}
		
		ctx.Next()
	}
}
