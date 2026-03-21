package middleware

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	// defaultRateLimit is the maximum number of requests allowed per window per IP.
	defaultRateLimit = 60
	// defaultWindow is the duration of the rate limit window.
	defaultWindow = time.Minute
)

// bucket represents a token bucket for a single client IP.
type bucket struct {
	tokens    int
	lastReset time.Time
	mu        sync.Mutex
}

// RateLimiter holds the in-memory state for IP-based rate limiting.
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	limit   int
	window  time.Duration
}

// NewRateLimiter creates a RateLimiter with the default configuration:
// 60 requests per minute per IP. The context controls the lifetime of the
// background cleanup goroutine — cancel it on server shutdown.
func NewRateLimiter(ctx context.Context) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		limit:   defaultRateLimit,
		window:  defaultWindow,
	}

	// Periodically remove stale buckets to prevent unbounded memory growth.
	go rl.cleanup(ctx)

	return rl
}

// RateLimit returns a middleware that enforces the per-IP request rate limit.
func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := realIP(r)

		if !rl.allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allow checks whether the given IP address is within its rate limit quota.
func (rl *RateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	b, ok := rl.buckets[ip]
	if !ok {
		b = &bucket{
			tokens:    rl.limit,
			lastReset: time.Now(),
		}
		rl.buckets[ip] = b
	}
	rl.mu.Unlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	if now.Sub(b.lastReset) >= rl.window {
		b.tokens = rl.limit
		b.lastReset = now
	}

	if b.tokens <= 0 {
		return false
	}

	b.tokens--
	return true
}

// cleanup runs every 5 minutes and removes buckets that have been idle for
// longer than two window durations to bound memory usage. It stops when the
// provided context is cancelled.
func (rl *RateLimiter) cleanup(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.mu.Lock()
			for ip, b := range rl.buckets {
				b.mu.Lock()
				idle := time.Since(b.lastReset) > 2*rl.window
				b.mu.Unlock()
				if idle {
					delete(rl.buckets, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// realIP extracts the client IP address from RemoteAddr only. Chi's RealIP
// middleware already copies the trusted proxy header into RemoteAddr, so we
// must NOT read X-Real-IP or X-Forwarded-For directly — an attacker can
// spoof those headers to bypass rate limiting.
func realIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
