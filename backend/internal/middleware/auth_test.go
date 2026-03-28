package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const authTestSecret = "middleware-test-secret"

// makeToken creates a signed JWT with the given claims using authTestSecret.
func makeToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(authTestSecret))
	if err != nil {
		t.Fatalf("makeToken: %v", err)
	}
	return signed
}

// okHandler is a minimal next handler that records whether it was called
// and echoes the userID from context.
func okHandler() (http.Handler, *bool) {
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		uid := UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(uid))
	})
	return h, &called
}

func authMiddlewareUnderTest() func(http.Handler) http.Handler {
	return Auth(AuthConfig{JWTSecret: []byte(authTestSecret)})
}

// --- tests ---

func TestAuthMiddleware_ValidAccessToken(t *testing.T) {
	next, called := okHandler()
	mw := authMiddlewareUnderTest()(next)

	token := makeToken(t, jwt.MapClaims{
		"sub":   "user-abc",
		"email": "test@example.com",
		"type":  "access",
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if !*called {
		t.Error("expected next handler to be called")
	}
	// Verify userID is injected into context.
	if rr.Body.String() != "user-abc" {
		t.Errorf("expected userID %q in context, got %q", "user-abc", rr.Body.String())
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	next, called := okHandler()
	mw := authMiddlewareUnderTest()(next)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if *called {
		t.Error("next handler must not be called without a token")
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	next, called := okHandler()
	mw := authMiddlewareUnderTest()(next)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer this.is.not.a.valid.jwt")
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
	if *called {
		t.Error("next handler must not be called for invalid token")
	}
}

func TestAuthMiddleware_RefreshTokenRejected(t *testing.T) {
	next, called := okHandler()
	mw := authMiddlewareUnderTest()(next)

	// Issue a token with type="refresh" — must be rejected.
	token := makeToken(t, jwt.MapClaims{
		"sub":  "user-abc",
		"type": "refresh",
		"exp":  time.Now().Add(7 * 24 * time.Hour).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for refresh token used as access token, got %d", rr.Code)
	}
	if *called {
		t.Error("next handler must not be called for refresh token")
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	next, called := okHandler()
	mw := authMiddlewareUnderTest()(next)

	// Token expired one minute ago.
	token := makeToken(t, jwt.MapClaims{
		"sub":  "user-abc",
		"type": "access",
		"exp":  time.Now().Add(-1 * time.Minute).Unix(),
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", rr.Code)
	}
	if *called {
		t.Error("next handler must not be called for expired token")
	}
}
