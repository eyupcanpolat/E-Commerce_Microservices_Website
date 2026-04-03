// Package ratelimit provides a simple in-memory rate limiter for the API Gateway.
// Uses token bucket algorithm per IP address.
// Limits: 60 requests/minute per IP by default (configurable via env).
package ratelimit

import (
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"eticaret/shared/logger"
	"eticaret/shared/response"
)

// bucket holds the token bucket state for a single IP.
type bucket struct {
	tokens    float64
	lastRefil time.Time
}

// Limiter is a per-IP token bucket rate limiter.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	rate     float64 // tokens per second
	capacity float64 // max tokens (burst)
}

// NewLimiter creates a Limiter.
// ratePerMinute: how many requests per minute are allowed per IP.
func NewLimiter(ratePerMinute int) *Limiter {
	l := &Limiter{
		buckets:  make(map[string]*bucket),
		rate:     float64(ratePerMinute) / 60.0,
		capacity: float64(ratePerMinute) / 4, // burst = 25% of rate/min
	}
	// Background cleanup of stale buckets
	go l.cleanup()
	return l
}

// NewLimiterFromEnv reads RATE_LIMIT_PER_MINUTE env variable, defaults to 60.
func NewLimiterFromEnv() *Limiter {
	rate := 60
	if v := os.Getenv("RATE_LIMIT_PER_MINUTE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			rate = n
		}
	}
	logger.Info("Rate limiter initialized", "requests_per_minute", rate)
	return NewLimiter(rate)
}

// Allow returns true if the given IP is within rate limits.
func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	b, ok := l.buckets[ip]
	if !ok {
		b = &bucket{tokens: l.capacity, lastRefil: time.Now()}
		l.buckets[ip] = b
	}

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(b.lastRefil).Seconds()
	b.tokens += elapsed * l.rate
	if b.tokens > l.capacity {
		b.tokens = l.capacity
	}
	b.lastRefil = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Middleware returns an http.Handler that enforces rate limiting.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)
		if !l.Allow(ip) {
			logger.Warn("Rate limit exceeded", "ip", ip, "path", r.URL.Path)
			w.Header().Set("Retry-After", "60")
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(int(l.capacity)))
			response.JSON(w, http.StatusTooManyRequests, map[string]interface{}{
				"success": false,
				"error":   "Çok fazla istek gönderdiniz. Lütfen bir dakika bekleyin.",
				"code":    "RATE_LIMIT_EXCEEDED",
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// cleanup removes stale bucket entries every 5 minutes.
func (l *Limiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.mu.Lock()
		cutoff := time.Now().Add(-10 * time.Minute)
		for ip, b := range l.buckets {
			if b.lastRefil.Before(cutoff) {
				delete(l.buckets, ip)
			}
		}
		l.mu.Unlock()
	}
}

// realIP extracts the real client IP, respecting X-Forwarded-For and X-Real-IP.
func realIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the chain
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Strip port from RemoteAddr
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
