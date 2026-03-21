package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
)

// ErrNotFound is returned when the requested record does not exist.
var ErrNotFound = errors.New("record not found")

// AuthRepository defines data access methods for authentication-related entities.
type AuthRepository interface {
	GetAllowedEmail(ctx context.Context, email string) (*domain.AllowedEmail, error)
	CreateUser(ctx context.Context, email, passwordHash string, displayName *string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID string) error
}

// PgxAuthRepository is a pgx-backed implementation of AuthRepository.
// It can be replaced with sqlc-generated code in the future by swapping the
// implementation while keeping the AuthRepository interface unchanged.
type PgxAuthRepository struct {
	pool *pgxpool.Pool
}

// NewPgxAuthRepository creates a new PgxAuthRepository with the given connection pool.
func NewPgxAuthRepository(pool *pgxpool.Pool) *PgxAuthRepository {
	return &PgxAuthRepository{pool: pool}
}

// GetAllowedEmail retrieves an allowed email record by email address.
// Returns ErrNotFound if the email is not in the whitelist.
func (r *PgxAuthRepository) GetAllowedEmail(ctx context.Context, email string) (*domain.AllowedEmail, error) {
	const q = `SELECT id, email, invited_by, created_at FROM allowed_emails WHERE email = $1`

	row := r.pool.QueryRow(ctx, q, email)

	var ae domain.AllowedEmail
	var invitedBy *string

	err := row.Scan(&ae.ID, &ae.Email, &invitedBy, &ae.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if invitedBy != nil {
		ae.InvitedBy = invitedBy
	}

	return &ae, nil
}

// CreateUser inserts a new user record and returns the created user.
func (r *PgxAuthRepository) CreateUser(ctx context.Context, email, passwordHash string, displayName *string) (*domain.User, error) {
	const q = `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, display_name, level, created_at, updated_at`

	row := r.pool.QueryRow(ctx, q, email, passwordHash, displayName)

	return scanUser(row)
}

// GetUserByEmail retrieves a user by their email address.
// Returns ErrNotFound if no user exists with that email.
func (r *PgxAuthRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	const q = `SELECT id, email, password_hash, display_name, level, created_at, updated_at FROM users WHERE email = $1`

	row := r.pool.QueryRow(ctx, q, email)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return u, nil
}

// GetUserByID retrieves a user by their UUID.
// Returns ErrNotFound if no user exists with that ID.
func (r *PgxAuthRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	const q = `SELECT id, email, password_hash, display_name, level, created_at, updated_at FROM users WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)

	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return u, nil
}

// CreateRefreshToken inserts a new refresh token record for a user.
func (r *PgxAuthRepository) CreateRefreshToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	const q = `INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`

	_, err := r.pool.Exec(ctx, q, userID, tokenHash, expiresAt)

	return err
}

// GetRefreshToken retrieves an active, non-expired refresh token by its hash.
// Returns ErrNotFound if the token does not exist, is revoked, or has expired.
func (r *PgxAuthRepository) GetRefreshToken(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND revoked = FALSE AND expires_at > now()`

	row := r.pool.QueryRow(ctx, q, tokenHash)

	var rt domain.RefreshToken

	err := row.Scan(&rt.ID, &rt.UserID, &rt.TokenHash, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &rt, nil
}

// RevokeRefreshToken marks a single refresh token as revoked by its hash.
func (r *PgxAuthRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	const q = `UPDATE refresh_tokens SET revoked = TRUE WHERE token_hash = $1`

	_, err := r.pool.Exec(ctx, q, tokenHash)

	return err
}

// RevokeAllUserRefreshTokens revokes every refresh token belonging to the given user.
func (r *PgxAuthRepository) RevokeAllUserRefreshTokens(ctx context.Context, userID string) error {
	const q = `UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1`

	_, err := r.pool.Exec(ctx, q, userID)

	return err
}

// scanUser reads a single user row into a domain.User value.
func scanUser(row pgx.Row) (*domain.User, error) {
	var u domain.User
	var displayName *string

	err := row.Scan(
		&u.ID,
		&u.Email,
		&u.PasswordHash,
		&displayName,
		&u.Level,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	u.DisplayName = displayName

	return &u, nil
}
