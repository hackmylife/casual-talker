package handler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
	"github.com/naoki-watanabe/casual-talker/backend/internal/middleware"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
	"github.com/naoki-watanabe/casual-talker/backend/internal/service"
	"golang.org/x/crypto/bcrypt"
)

// --- mock auth repository reused from helpers_test.go scope ---
// We define a separate authMockRepo here so this file compiles independently
// without relying on the service package's test-only type.

type handlerMockAuthRepo struct {
	allowedEmails map[string]*domain.AllowedEmail
	users         map[string]*domain.User  // key: email
	usersById     map[string]*domain.User  // key: id
	refreshTokens map[string]*domain.RefreshToken // key: tokenHash
	userLevels    map[string]int
}

func newHandlerMockAuthRepo() *handlerMockAuthRepo {
	return &handlerMockAuthRepo{
		allowedEmails: make(map[string]*domain.AllowedEmail),
		users:         make(map[string]*domain.User),
		usersById:     make(map[string]*domain.User),
		refreshTokens: make(map[string]*domain.RefreshToken),
		userLevels:    make(map[string]int),
	}
}

func (m *handlerMockAuthRepo) GetAllowedEmail(_ context.Context, email string) (*domain.AllowedEmail, error) {
	ae, ok := m.allowedEmails[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return ae, nil
}

func (m *handlerMockAuthRepo) CreateUser(_ context.Context, email, passwordHash string, displayName *string) (*domain.User, error) {
	u := &domain.User{
		ID:           "user-" + email,
		Email:        email,
		PasswordHash: passwordHash,
		DisplayName:  displayName,
		Level:        1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	m.users[email] = u
	m.usersById[u.ID] = u
	return u, nil
}

func (m *handlerMockAuthRepo) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *handlerMockAuthRepo) GetUserByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := m.usersById[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *handlerMockAuthRepo) CreateRefreshToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
	m.refreshTokens[tokenHash] = &domain.RefreshToken{
		ID:        "rt-" + tokenHash[:8],
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		Revoked:   false,
		CreatedAt: time.Now(),
	}
	return nil
}

func (m *handlerMockAuthRepo) GetRefreshToken(_ context.Context, tokenHash string) (*domain.RefreshToken, error) {
	rt, ok := m.refreshTokens[tokenHash]
	if !ok || rt.Revoked || time.Now().After(rt.ExpiresAt) {
		return nil, repository.ErrNotFound
	}
	return rt, nil
}

func (m *handlerMockAuthRepo) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	if rt, ok := m.refreshTokens[tokenHash]; ok {
		rt.Revoked = true
	}
	return nil
}

func (m *handlerMockAuthRepo) RevokeAllUserRefreshTokens(_ context.Context, userID string) error {
	for _, rt := range m.refreshTokens {
		if rt.UserID == userID {
			rt.Revoked = true
		}
	}
	return nil
}

func (m *handlerMockAuthRepo) UpdateUserLevel(_ context.Context, userID string, level int) error {
	u, ok := m.usersById[userID]
	if !ok {
		return repository.ErrNotFound
	}
	u.Level = level
	return nil
}

func (m *handlerMockAuthRepo) GetUserLevel(_ context.Context, _, _ string) (int, error) {
	return 1, nil
}

func (m *handlerMockAuthRepo) SetUserLevel(_ context.Context, userID, language string, level int) error {
	m.userLevels[userID+":"+language] = level
	return nil
}

func (m *handlerMockAuthRepo) GetUserLevels(_ context.Context, userID string) (map[string]int, error) {
	result := make(map[string]int)
	prefix := userID + ":"
	for k, v := range m.userLevels {
		if strings.HasPrefix(k, prefix) {
			lang := strings.TrimPrefix(k, prefix)
			result[lang] = v
		}
	}
	return result, nil
}

// --- test helpers ---

const handlerTestJWTSecret = "handler-test-secret"

// buildTestServer creates an httptest.Server with the auth routes wired up.
// Protected routes are wrapped with the auth middleware.
func buildTestServer(repo *handlerMockAuthRepo) *httptest.Server {
	svc := service.NewAuthService(repo, handlerTestJWTSecret)
	h := NewAuthHandler(svc)

	authMW := middleware.Auth(middleware.AuthConfig{
		JWTSecret: []byte(handlerTestJWTSecret),
	})

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/refresh", h.Refresh)
	mux.HandleFunc("GET /api/v1/users/me", func(w http.ResponseWriter, r *http.Request) {
		authMW(http.HandlerFunc(h.Me)).ServeHTTP(w, r)
	})

	return httptest.NewServer(mux)
}

func postJSON(t *testing.T, client *http.Client, url string, body any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func decodeBody(t *testing.T, resp *http.Response, dst any) {
	t.Helper()
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		t.Fatalf("json.Decode: %v", err)
	}
}

func hashTokenForHandler(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// --- POST /api/v1/auth/register ---

func TestRegisterHandler_Success(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	repo.allowedEmails["test@example.com"] = &domain.AllowedEmail{
		ID:    "ae-1",
		Email: "test@example.com",
	}
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/register", map[string]any{
		"email":    "test@example.com",
		"password": "password123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", resp.StatusCode)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		User         struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	decodeBody(t, resp, &body)

	if body.AccessToken == "" {
		t.Error("expected non-empty access_token")
	}
	if body.RefreshToken == "" {
		t.Error("expected non-empty refresh_token")
	}
	if body.User.Email != "test@example.com" {
		t.Errorf("unexpected user email: %s", body.User.Email)
	}
}

func TestRegisterHandler_ValidationError_InvalidEmail(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/register", map[string]any{
		"email":    "not-an-email",
		"password": "password123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", resp.StatusCode)
	}
}

func TestRegisterHandler_ValidationError_PasswordTooShort(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/register", map[string]any{
		"email":    "user@example.com",
		"password": "short",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for short password, got %d", resp.StatusCode)
	}
}

func TestRegisterHandler_ValidationError_PasswordTooLong(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/register", map[string]any{
		"email":    "user@example.com",
		"password": strings.Repeat("a", 73), // max is 72
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("expected 400 for password too long, got %d", resp.StatusCode)
	}
}

func TestRegisterHandler_EmailNotInWhitelist(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/register", map[string]any{
		"email":    "unknown@example.com",
		"password": "password123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for non-whitelisted email, got %d", resp.StatusCode)
	}
}

func TestRegisterHandler_DuplicateEmail(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	repo.allowedEmails["dup@example.com"] = &domain.AllowedEmail{
		ID:    "ae-2",
		Email: "dup@example.com",
	}
	// Pre-seed the user to simulate a duplicate.
	repo.users["dup@example.com"] = &domain.User{
		ID:    "user-dup@example.com",
		Email: "dup@example.com",
	}
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/register", map[string]any{
		"email":    "dup@example.com",
		"password": "password123",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusConflict {
		t.Errorf("expected 409 for duplicate email, got %d", resp.StatusCode)
	}
}

// --- POST /api/v1/auth/login ---

func TestLoginHandler_Success(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	hash, _ := bcrypt.GenerateFromPassword([]byte("myPassword9"), 12)
	repo.users["login@example.com"] = &domain.User{
		ID:           "user-login@example.com",
		Email:        "login@example.com",
		PasswordHash: string(hash),
		Level:        1,
	}
	repo.usersById["user-login@example.com"] = repo.users["login@example.com"]
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/login", map[string]any{
		"email":    "login@example.com",
		"password": "myPassword9",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decodeBody(t, resp, &body)
	if body.AccessToken == "" || body.RefreshToken == "" {
		t.Error("expected non-empty tokens")
	}
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/login", map[string]any{
		"email":    "nobody@example.com",
		"password": "password",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// --- POST /api/v1/auth/refresh ---

func TestRefreshHandler_Success(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	repo.users["refresh@example.com"] = &domain.User{
		ID:    "user-refresh@example.com",
		Email: "refresh@example.com",
	}
	repo.usersById["user-refresh@example.com"] = repo.users["refresh@example.com"]

	rawToken := "valid-refresh-opaque-token=="
	tokenHash := hashTokenForHandler(rawToken)
	repo.refreshTokens[tokenHash] = &domain.RefreshToken{
		ID:        "rt-001",
		UserID:    "user-refresh@example.com",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   false,
	}

	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/refresh", map[string]any{
		"refresh_token": rawToken,
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	var body struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	decodeBody(t, resp, &body)
	if body.AccessToken == "" || body.RefreshToken == "" {
		t.Error("expected non-empty new tokens")
	}
}

func TestRefreshHandler_InvalidToken(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	srv := buildTestServer(repo)
	defer srv.Close()

	resp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/refresh", map[string]any{
		"refresh_token": "invalid-token",
	})
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

// --- GET /api/v1/users/me ---

func TestMeHandler_WithValidJWT(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	repo.allowedEmails["me@example.com"] = &domain.AllowedEmail{
		ID:    "ae-me",
		Email: "me@example.com",
	}
	srv := buildTestServer(repo)
	defer srv.Close()

	// First register to obtain a token.
	regResp := postJSON(t, srv.Client(), srv.URL+"/api/v1/auth/register", map[string]any{
		"email":    "me@example.com",
		"password": "password123",
	})
	defer regResp.Body.Close()

	if regResp.StatusCode != http.StatusCreated {
		t.Fatalf("register failed with status %d", regResp.StatusCode)
	}

	var regBody struct {
		AccessToken string `json:"access_token"`
		User        struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"user"`
	}
	decodeBody(t, regResp, &regBody)

	// Call /users/me with the obtained access token.
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/users/me", nil)
	if err != nil {
		t.Fatalf("http.NewRequest: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+regBody.AccessToken)

	meResp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /users/me: %v", err)
	}
	defer meResp.Body.Close()

	if meResp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", meResp.StatusCode)
	}

	var meBody struct {
		Email string `json:"email"`
	}
	decodeBody(t, meResp, &meBody)
	if meBody.Email != "me@example.com" {
		t.Errorf("unexpected email in /me response: %s", meBody.Email)
	}
}

func TestMeHandler_WithoutJWT(t *testing.T) {
	repo := newHandlerMockAuthRepo()
	srv := buildTestServer(repo)
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL + "/api/v1/users/me")
	if err != nil {
		t.Fatalf("GET /users/me: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", resp.StatusCode)
	}
}
