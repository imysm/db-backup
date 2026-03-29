package router

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens   float64
	maxTokens float64
	refillRate float64 // tokens per second
	lastTime  time.Time
	mu        sync.Mutex
}

func newTokenBucket(maxTokens, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:    maxTokens,
		maxTokens: maxTokens,
		refillRate: refillRate,
		lastTime:  time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}
	tb.lastTime = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// rateLimiter manages per-IP rate limiting with periodic cleanup.
type rateLimiter struct {
	buckets    map[string]*tokenBucket
	mu         sync.RWMutex
	maxTokens  float64
	refillRate float64
	maxEntries int
}

func newRateLimiter(maxTokens, refillRate float64, maxEntries int) *rateLimiter {
	rl := &rateLimiter{
		buckets:    make(map[string]*tokenBucket),
		maxTokens:  maxTokens,
		refillRate: refillRate,
		maxEntries: maxEntries,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) getBucket(ip string) *tokenBucket {
	rl.mu.RLock()
	b, ok := rl.buckets[ip]
	rl.mu.RUnlock()
	if ok {
		return b
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()
	// Double-check
	if b, ok = rl.buckets[ip]; ok {
		return b
	}
	// Evict if at capacity
	if len(rl.buckets) >= rl.maxEntries {
		// Simple: just don't create new bucket, let the request through
		// to avoid OOM. In production you'd use LRU.
		return newTokenBucket(rl.maxTokens, rl.refillRate)
	}
	b = newTokenBucket(rl.maxTokens, rl.refillRate)
	rl.buckets[ip] = b
	return b
}

func (rl *rateLimiter) allow(ip string) bool {
	return rl.getBucket(ip).allow()
}

func (rl *rateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		rl.mu.Lock()
		// Remove buckets not accessed recently
		if len(rl.buckets) > rl.maxEntries/2 {
			// Simple cleanup: clear all (buckets are cheap, they'll be recreated)
			// For production, use LRU with access time tracking
			rl.buckets = make(map[string]*tokenBucket, len(rl.buckets)/2)
		}
		rl.mu.Unlock()
	}
}

// clientIP extracts the client IP from the request.
func clientIP(c *gin.Context) string {
	// Check X-Forwarded-For first, then X-Real-Ip, then RemoteAddr
	if xff := c.GetHeader("X-Forwarded-For"); xff != "" {
		ips := stringsFunc(xff)
		if len(ips) > 0 {
			return ips[0]
		}
	}
	if xri := c.GetHeader("X-Real-Ip"); xri != "" {
		return xri
	}
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

// stringsFunc is a simple split function to avoid importing "strings" circularly.
// Actually let's just use net/http package for parsing.
// We need this to parse X-Forwarded-For.
func stringsFunc(s string) []string {
	// Simple comma-split
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

// RateLimitMiddleware creates a gin middleware for rate limiting.
func RateLimitMiddleware(maxTokens, refillRate float64, maxEntries int) gin.HandlerFunc {
	rl := newRateLimiter(maxTokens, refillRate, maxEntries)
	return func(c *gin.Context) {
		ip := clientIP(c)
		if !rl.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests",
			})
			return
		}
		c.Next()
	}
}
