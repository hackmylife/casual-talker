package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
)

const (
	bcryptCost          = 12
	accessTokenTTL      = 15 * time.Minute
	refreshTokenTTL     = 7 * 24 * time.Hour
	refreshTokenRawSize = 32 // bytes of random data before base64 encoding
)

// Sentinel errors returned by AuthService methods so callers can distinguish
// application-level failures from unexpected infrastructure errors.
var (
	ErrEmailNotAllowed   = errors.New("email not in whitelist")
	ErrEmailTaken        = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken      = errors.New("invalid or expired token")
)

// AuthService handles user registration, login, token refresh, and logout.
type AuthService struct {
	repo      repository.AuthRepository
	jwtSecret []byte
}

// NewAuthService creates a new AuthService.
func NewAuthService(repo repository.AuthRepository, jwtSecret string) *AuthService {
	return &AuthService{
		repo:      repo,
		jwtSecret: []byte(jwtSecret),
	}
}

// Register creates a new user account if the email is whitelisted and not yet taken.
// It returns a signed access token, an opaque refresh token, and the new user.
func (s *AuthService) Register(ctx context.Context, email, password string, displayName *string) (accessToken, refreshToken string, user *domain.User, err error) {
	// Normalize email to prevent case-sensitivity bypass.
	email = strings.ToLower(strings.TrimSpace(email))

	// Whitelist check.
	if _, err = s.repo.GetAllowedEmail(ctx, email); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", "", nil, ErrEmailNotAllowed
		}
		return "", "", nil, fmt.Errorf("checking allowed email: %w", err)
	}

	// Duplicate email check.
	if _, err = s.repo.GetUserByEmail(ctx, email); err == nil {
		return "", "", nil, ErrEmailTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return "", "", nil, fmt.Errorf("checking existing user: %w", err)
	}

	// Hash password.
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", "", nil, fmt.Errorf("hashing password: %w", err)
	}

	// Persist the user.
	user, err = s.repo.CreateUser(ctx, email, string(hash), displayName)
	if err != nil {
		return "", "", nil, fmt.Errorf("creating user: %w", err)
	}

	// Generate tokens.
	accessToken, err = s.generateAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, tokenHash, err := s.generateRefreshToken()
	if err != nil {
		return "", "", nil, fmt.Errorf("generating refresh token: %w", err)
	}

	expiresAt := time.Now().Add(refreshTokenTTL)
	if err = s.repo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return "", "", nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return accessToken, refreshToken, user, nil
}

// Login authenticates a user and returns new access and refresh tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (accessToken, refreshToken string, user *domain.User, err error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err = s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", "", nil, ErrInvalidCredentials
		}
		return "", "", nil, fmt.Errorf("fetching user: %w", err)
	}

	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", nil, ErrInvalidCredentials
	}

	accessToken, err = s.generateAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", nil, fmt.Errorf("generating access token: %w", err)
	}

	refreshToken, tokenHash, err := s.generateRefreshToken()
	if err != nil {
		return "", "", nil, fmt.Errorf("generating refresh token: %w", err)
	}

	expiresAt := time.Now().Add(refreshTokenTTL)
	if err = s.repo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt); err != nil {
		return "", "", nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return accessToken, refreshToken, user, nil
}

// RefreshToken validates the provided opaque refresh token, revokes it, and
// issues a fresh access token together with a new refresh token.
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (newAccessToken, newRefreshToken string, err error) {
	tokenHash := hashToken(refreshToken)

	record, err := s.repo.GetRefreshToken(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("fetching refresh token: %w", err)
	}

	// Revoke the old token before issuing new ones (token rotation).
	if err = s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return "", "", fmt.Errorf("revoking old refresh token: %w", err)
	}

	user, err := s.repo.GetUserByID(ctx, record.UserID)
	if err != nil {
		return "", "", fmt.Errorf("fetching user for refresh: %w", err)
	}

	newAccessToken, err = s.generateAccessToken(user.ID, user.Email)
	if err != nil {
		return "", "", fmt.Errorf("generating access token: %w", err)
	}

	newRefreshToken, newTokenHash, err := s.generateRefreshToken()
	if err != nil {
		return "", "", fmt.Errorf("generating refresh token: %w", err)
	}

	expiresAt := time.Now().Add(refreshTokenTTL)
	if err = s.repo.CreateRefreshToken(ctx, user.ID, newTokenHash, expiresAt); err != nil {
		return "", "", fmt.Errorf("storing new refresh token: %w", err)
	}

	return newAccessToken, newRefreshToken, nil
}

// Logout revokes the given refresh token so it can no longer be used.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	tokenHash := hashToken(refreshToken)

	if err := s.repo.RevokeRefreshToken(ctx, tokenHash); err != nil {
		return fmt.Errorf("revoking refresh token: %w", err)
	}

	return nil
}

// generateAccessToken creates a short-lived signed JWT for the given user.
// Claims: sub (userID), email, type ("access"), iat, exp (15 min).
func (s *AuthService) generateAccessToken(userID, email string) (string, error) {
	now := time.Now()

	claims := jwt.MapClaims{
		"sub":   userID,
		"email": email,
		"type":  "access",
		"iat":   now.Unix(),
		"exp":   now.Add(accessTokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString(s.jwtSecret)
}

// generateRefreshToken returns a random opaque token string and its SHA-256
// hex digest. Only the digest is stored in the database; the raw token is
// sent to the client and never persisted.
func (s *AuthService) generateRefreshToken() (token, tokenHash string, err error) {
	raw := make([]byte, refreshTokenRawSize)
	if _, err = rand.Read(raw); err != nil {
		return "", "", fmt.Errorf("reading random bytes: %w", err)
	}

	token = base64.URLEncoding.EncodeToString(raw)
	tokenHash = hashToken(token)

	return token, tokenHash, nil
}

// GetUserByID fetches a user by their UUID. Used by the /users/me endpoint.
func (s *AuthService) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fetching user by id: %w", err)
	}
	return user, nil
}

// hashToken returns the hex-encoded SHA-256 digest of the given token string.
func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
