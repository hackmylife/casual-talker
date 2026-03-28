package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestRateLimiter creates a RateLimiter with custom limit and window values
// so tests can run quickly without waiting for the production 1-minute window.
func newTestRateLimiter(ctx context.Context, limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*bucket),
		limit:   limit,
		window:  window,
	}
	go rl.cleanup(ctx)
	return rl
}

// dummyHandler is a minimal next handler that always returns 200 OK.
var dummyHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

// sendRequest fires a single request through the rate-limit middleware and
// returns the response recorder so the caller can inspect the status code and
// response headers.
func sendRequest(mw http.Handler, remoteAddr string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	rr := httptest.NewRecorder()
	mw.ServeHTTP(rr, req)
	return rr
}

// --- tests ---

func TestRateLimit_WithinLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Allow 3 requests per 10 seconds.
	rl := newTestRateLimiter(ctx, 3, 10*time.Second)
	mw := rl.RateLimit(dummyHandler)

	for i := 0; i < 3; i++ {
		rr := sendRequest(mw, "192.0.2.1:1234")
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}
}

func TestRateLimit_ExceedsLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Allow only 2 requests per 10 seconds.
	rl := newTestRateLimiter(ctx, 2, 10*time.Second)
	mw := rl.RateLimit(dummyHandler)

	// First two requests succeed.
	for i := 0; i < 2; i++ {
		rr := sendRequest(mw, "10.0.0.1:9999")
		if rr.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, rr.Code)
		}
	}

	// Third request must be rejected.
	rr := sendRequest(mw, "10.0.0.1:9999")
	if rr.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after exceeding limit, got %d", rr.Code)
	}

	// Retry-After header must be present.
	if rr.Header().Get("Retry-After") == "" {
		t.Error("expected Retry-After header on 429 response")
	}
}

func TestRateLimit_WindowReset(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Allow 1 request per 50 milliseconds so the window resets fast.
	rl := newTestRateLimiter(ctx, 1, 50*time.Millisecond)
	mw := rl.RateLimit(dummyHandler)

	const ip = "172.16.0.1:8080"

	// Use the single allowed token.
	rr := sendRequest(mw, ip)
	if rr.Code != http.StatusOK {
		t.Fatalf("initial request: expected 200, got %d", rr.Code)
	}

	// Immediately a second request must be rejected.
	rr = sendRequest(mw, ip)
	if rr.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: expected 429, got %d", rr.Code)
	}

	// Wait for the window to expire, then the request must succeed again.
	time.Sleep(60 * time.Millisecond)

	rr = sendRequest(mw, ip)
	if rr.Code != http.StatusOK {
		t.Errorf("post-reset request: expected 200, got %d", rr.Code)
	}
}
