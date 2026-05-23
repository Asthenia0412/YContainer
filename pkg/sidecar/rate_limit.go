package sidecar

import (
	"math"
	"net/http"
	"sync"
	"time"
)

type TokenBucket struct {
	rate     float64
	burst    int
	tokens   float64
	lastTime time.Time
	mu       sync.Mutex
}

func NewTokenBucket(rate float64, burst int) *TokenBucket {
	return &TokenBucket{
		rate:     rate,
		burst:    burst,
		tokens:   float64(burst),
		lastTime: time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastTime).Seconds()
	tb.tokens = math.Min(tb.tokens+elapsed*tb.rate, float64(tb.burst))
	tb.lastTime = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

type RateLimiter struct {
	bucket  *TokenBucket
	config  RateLimitConfig
}

type RateLimitConfig struct {
	Enabled     bool
	RequestsPer int
	PerSeconds  float64
	Burst       int
}

func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	return &RateLimiter{
		bucket: NewTokenBucket(
			float64(config.RequestsPer)/config.PerSeconds,
			config.Burst,
		),
		config: config,
	}
}

func RateLimit(limiter *RateLimiter) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			if !limiter.bucket.Allow() {
				w.Header().Set("Retry-After", "1")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}