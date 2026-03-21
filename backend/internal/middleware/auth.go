package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const userIDKey contextKey = "userID"

// AuthConfig holds the configuration for the JWT auth middleware.
type AuthConfig struct {
	JWTSecret []byte
}

// Auth returns a middleware that validates JWT Bearer tokens from the
// Authorization header and injects the subject claim into the request context.
// Only tokens with type claim "access" are accepted, preventing refresh tokens
// from being used to call protected endpoints.
func Auth(cfg AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "authorization header required", http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				http.Error(w, "authorization header must be in 'Bearer <token>' format", http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return cfg.JWTSecret, nil
			}, jwt.WithValidMethods([]string{"HS256"}))
			if err != nil || !token.Valid {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "invalid token claims", http.StatusUnauthorized)
				return
			}

			// Reject refresh tokens presented as access tokens.
			if tokenType, _ := claims["type"].(string); tokenType != "access" {
				http.Error(w, "invalid token type", http.StatusUnauthorized)
				return
			}

			sub, err := claims.GetSubject()
			if err != nil || sub == "" {
				http.Error(w, "token missing subject claim", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, sub)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromContext retrieves the authenticated user ID from the context.
// Returns an empty string if the value is not present.
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}
