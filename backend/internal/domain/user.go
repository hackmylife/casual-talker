package domain

import "time"

// User represents a registered user in the system.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  *string
	Level        int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AllowedEmail represents an email address that is permitted to register.
type AllowedEmail struct {
	ID        string
	Email     string
	InvitedBy *string
	CreatedAt time.Time
}

// RefreshToken represents a stored refresh token record used to issue new access tokens.
type RefreshToken struct {
	ID        string
	UserID    string
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}
