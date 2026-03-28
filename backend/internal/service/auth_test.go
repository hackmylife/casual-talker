package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
)

// --- mock implementation ---

type mockAuthRepo struct {
	allowedEmails map[string]*domain.AllowedEmail
	users         map[string]*domain.User  // key: email
	usersById     map[string]*domain.User  // key: id
	refreshTokens map[string]*domain.RefreshToken // key: tokenHash
	userLevels    map[string]int // key: "userId:lang"
}

func newMockAuthRepo() *mockAuthRepo {
	return &mockAuthRepo{
		allowedEmails: make(map[string]*domain.AllowedEmail),
		users:         make(map[string]*domain.User),
		usersById:     make(map[string]*domain.User),
		refreshTokens: make(map[string]*domain.RefreshToken),
		userLevels:    make(map[string]int),
	}
}

func (m *mockAuthRepo) GetAllowedEmail(_ context.Context, email string) (*domain.AllowedEmail, error) {
	ae, ok := m.allowedEmails[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return ae, nil
}

func (m *mockAuthRepo) CreateUser(_ context.Context, email, passwordHash string, displayName *string) (*domain.User, error) {
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

func (m *mockAuthRepo) GetUserByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockAuthRepo) GetUserByID(_ context.Context, id string) (*domain.User, error) {
	u, ok := m.usersById[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return u, nil
}

func (m *mockAuthRepo) CreateRefreshToken(_ context.Context, userID, tokenHash string, expiresAt time.Time) error {
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

func (m *mockAuthRepo) GetRefreshToken(_ context.Context, tokenHash string) (*domain.RefreshToken, error) {
	rt, ok := m.refreshTokens[tokenHash]
	if !ok || rt.Revoked || time.Now().After(rt.ExpiresAt) {
		return nil, repository.ErrNotFound
	}
	return rt, nil
}

func (m *mockAuthRepo) RevokeRefreshToken(_ context.Context, tokenHash string) error {
	rt, ok := m.refreshTokens[tokenHash]
	if !ok {
		return nil
	}
	rt.Revoked = true
	return nil
}

func (m *mockAuthRepo) RevokeAllUserRefreshTokens(_ context.Context, userID string) error {
	for _, rt := range m.refreshTokens {
		if rt.UserID == userID {
			rt.Revoked = true
		}
	}
	return nil
}

func (m *mockAuthRepo) UpdateUserLevel(_ context.Context, userID string, level int) error {
	u, ok := m.usersById[userID]
	if !ok {
		return repository.ErrNotFound
	}
	u.Level = level
	return nil
}

func (m *mockAuthRepo) GetUserLevel(_ context.Context, userID, language string) (int, error) {
	key := userID + ":" + language
	if lvl, ok := m.userLevels[key]; ok {
		return lvl, nil
	}
	return 1, nil
}

func (m *mockAuthRepo) SetUserLevel(_ context.Context, userID, language string, level int) error {
	m.userLevels[userID+":"+language] = level
	return nil
}

func (m *mockAuthRepo) GetUserLevels(_ context.Context, userID string) (map[string]int, error) {
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

// --- helpers ---

const testJWTSecret = "test-secret-key"

func newTestService(repo *mockAuthRepo) *AuthService {
	return NewAuthService(repo, testJWTSecret)
}

// hashForTest returns the SHA-256 hex digest of a token string, mirroring the
// production hashToken function (which is unexported).
func hashForTest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// parseAccessToken parses and validates a JWT signed with testJWTSecret.
func parseAccessToken(t *testing.T, tokenString string) jwt.MapClaims {
	t.Helper()
	token, err := jwt.Parse(tokenString, func(tok *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		t.Fatalf("failed to parse access token: %v", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		t.Fatal("claims are not MapClaims")
	}
	return claims
}

// --- Register tests ---

func TestRegister_Success(t *testing.T) {
	repo := newMockAuthRepo()
	repo.allowedEmails["alice@example.com"] = &domain.AllowedEmail{
		ID:    "ae-1",
		Email: "alice@example.com",
	}
	svc := newTestService(repo)

	accessToken, refreshToken, user, err := svc.Register(
		context.Background(), "alice@example.com", "password123", nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accessToken == "" {
		t.Error("expected non-empty access token")
	}
	if refreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if user == nil {
		t.Fatal("expected non-nil user")
	}
	if user.Email != "alice@example.com" {
		t.Errorf("expected email %q, got %q", "alice@example.com", user.Email)
	}
}

func TestRegister_EmailNotAllowed(t *testing.T) {
	repo := newMockAuthRepo()
	svc := newTestService(repo)

	_, _, _, err := svc.Register(context.Background(), "unknown@example.com", "password123", nil)
	if !errors.Is(err, ErrEmailNotAllowed) {
		t.Errorf("expected ErrEmailNotAllowed, got %v", err)
	}
}

func TestRegister_EmailTaken(t *testing.T) {
	repo := newMockAuthRepo()
	repo.allowedEmails["bob@example.com"] = &domain.AllowedEmail{
		ID:    "ae-2",
		Email: "bob@example.com",
	}
	// Pre-populate the user so the duplicate check triggers.
	repo.users["bob@example.com"] = &domain.User{
		ID:    "user-bob@example.com",
		Email: "bob@example.com",
	}
	svc := newTestService(repo)

	_, _, _, err := svc.Register(context.Background(), "bob@example.com", "password123", nil)
	if !errors.Is(err, ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRegister_PasswordIsHashed(t *testing.T) {
	repo := newMockAuthRepo()
	repo.allowedEmails["carol@example.com"] = &domain.AllowedEmail{
		ID:    "ae-3",
		Email: "carol@example.com",
	}
	svc := newTestService(repo)

	plainPassword := "superSecret42"
	_, _, _, err := svc.Register(context.Background(), "carol@example.com", plainPassword, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := repo.users["carol@example.com"]
	if stored == nil {
		t.Fatal("user not found in mock repo after registration")
	}
	// The stored hash must not equal the plain password.
	if stored.PasswordHash == plainPassword {
		t.Error("password was stored as plain text, expected bcrypt hash")
	}
	// Verify that bcrypt can confirm the password.
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte(plainPassword)); err != nil {
		t.Errorf("bcrypt verification failed: %v", err)
	}
}

func TestRegister_JWTClaims(t *testing.T) {
	repo := newMockAuthRepo()
	repo.allowedEmails["dave@example.com"] = &domain.AllowedEmail{
		ID:    "ae-4",
		Email: "dave@example.com",
	}
	svc := newTestService(repo)

	accessToken, _, user, err := svc.Register(context.Background(), "dave@example.com", "password123", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims := parseAccessToken(t, accessToken)

	sub, err := claims.GetSubject()
	if err != nil {
		t.Fatalf("could not get subject claim: %v", err)
	}
	if sub != user.ID {
		t.Errorf("expected sub=%q, got %q", user.ID, sub)
	}

	email, _ := claims["email"].(string)
	if email != "dave@example.com" {
		t.Errorf("expected email claim %q, got %q", "dave@example.com", email)
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "access" {
		t.Errorf("expected type claim %q, got %q", "access", tokenType)
	}

	expUnix, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim missing or wrong type")
	}
	exp := time.Unix(int64(expUnix), 0)
	if !exp.After(time.Now()) {
		t.Error("expected exp to be in the future")
	}
	// Access token TTL is 15 min; give a small buffer.
	if exp.After(time.Now().Add(accessTokenTTL + time.Minute)) {
		t.Error("exp is further in the future than expected for an access token")
	}
}

// --- Login tests ---

func TestLogin_Success(t *testing.T) {
	repo := newMockAuthRepo()

	plainPW := "myPassword9"
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPW), bcryptCost)
	if err != nil {
		t.Fatalf("setup: bcrypt.GenerateFromPassword: %v", err)
	}
	repo.users["eve@example.com"] = &domain.User{
		ID:           "user-eve",
		Email:        "eve@example.com",
		PasswordHash: string(hash),
		Level:        1,
	}
	repo.usersById["user-eve"] = repo.users["eve@example.com"]

	svc := newTestService(repo)

	accessToken, refreshToken, user, err := svc.Login(context.Background(), "eve@example.com", plainPW)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if accessToken == "" || refreshToken == "" {
		t.Error("expected non-empty tokens")
	}
	if user.Email != "eve@example.com" {
		t.Errorf("unexpected user email: %s", user.Email)
	}
}

func TestLogin_UnknownEmail(t *testing.T) {
	repo := newMockAuthRepo()
	svc := newTestService(repo)

	_, _, _, err := svc.Login(context.Background(), "nobody@example.com", "password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockAuthRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("correctPW"), bcryptCost)
	repo.users["frank@example.com"] = &domain.User{
		ID:           "user-frank",
		Email:        "frank@example.com",
		PasswordHash: string(hash),
	}

	svc := newTestService(repo)

	_, _, _, err := svc.Login(context.Background(), "frank@example.com", "wrongPW")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_EmailNormalized(t *testing.T) {
	repo := newMockAuthRepo()

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass1234"), bcryptCost)
	repo.users["grace@example.com"] = &domain.User{
		ID:           "user-grace",
		Email:        "grace@example.com",
		PasswordHash: string(hash),
	}
	repo.usersById["user-grace"] = repo.users["grace@example.com"]

	svc := newTestService(repo)

	// Supply email with uppercase letters — the service must normalize it.
	_, _, user, err := svc.Login(context.Background(), "GRACE@EXAMPLE.COM", "pass1234")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.Email != "grace@example.com" {
		t.Errorf("expected normalized email, got %q", user.Email)
	}
}

// --- RefreshToken tests ---

func TestRefreshToken_Success(t *testing.T) {
	repo := newMockAuthRepo()

	repo.users["henry@example.com"] = &domain.User{
		ID:    "user-henry",
		Email: "henry@example.com",
	}
	repo.usersById["user-henry"] = repo.users["henry@example.com"]

	// Create a valid refresh token manually in the mock.
	rawToken := "some-valid-opaque-token-base64=="
	tokenHash := hashForTest(rawToken)
	repo.refreshTokens[tokenHash] = &domain.RefreshToken{
		ID:        "rt-001",
		UserID:    "user-henry",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   false,
	}

	svc := newTestService(repo)

	newAccess, newRefresh, err := svc.RefreshToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newAccess == "" || newRefresh == "" {
		t.Error("expected non-empty new tokens")
	}

	// The old token must be revoked.
	if !repo.refreshTokens[tokenHash].Revoked {
		t.Error("old refresh token was not revoked after rotation")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	repo := newMockAuthRepo()
	svc := newTestService(repo)

	_, _, err := svc.RefreshToken(context.Background(), "does-not-exist")
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRefreshToken_OldTokenRevoked(t *testing.T) {
	repo := newMockAuthRepo()

	repo.users["ivan@example.com"] = &domain.User{
		ID:    "user-ivan",
		Email: "ivan@example.com",
	}
	repo.usersById["user-ivan"] = repo.users["ivan@example.com"]

	rawToken := "ivan-refresh-token-abc=="
	tokenHash := hashForTest(rawToken)
	repo.refreshTokens[tokenHash] = &domain.RefreshToken{
		ID:        "rt-002",
		UserID:    "user-ivan",
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		Revoked:   false,
	}

	svc := newTestService(repo)

	_, _, err := svc.RefreshToken(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("unexpected error on first refresh: %v", err)
	}

	// Using the same (now revoked) token again must fail.
	_, _, err = svc.RefreshToken(context.Background(), rawToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken on second use of revoked token, got %v", err)
	}
}
