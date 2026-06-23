package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"
)

// PersonalTokenPrefix marks a long-lived personal access token so the auth
// middleware can route it to the token store instead of the session store. The
// trailing underscore keeps it visually distinct; session tokens are bare
// base64url and never carry this prefix.
const PersonalTokenPrefix = "dpat_"

// Token scopes. ScopeFull grants the same access as a logged-in session;
// ScopeRead restricts the token to safe (read-only) HTTP methods.
const (
	ScopeFull = "full"
	ScopeRead = "read"
)

// PersonalToken is the stored metadata for a long-lived API token. The raw
// token is shown once at creation and never persisted; only its hash is kept.
type PersonalToken struct {
	ID         int
	UserID     int
	Name       string
	Scope      string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

// PersonalTokenStore persists and manages a user's long-lived API tokens.
type PersonalTokenStore interface {
	CreatePersonalToken(
		ctx context.Context,
		userID int,
		tokenHash string,
		name string,
		scope string,
		expiresAt *time.Time,
	) (PersonalToken, error)
	ListPersonalTokens(ctx context.Context, userID int) ([]PersonalToken, error)
	RevokePersonalToken(ctx context.Context, userID int, id int) error
	// PersonalTokenUser resolves a presented token hash to its owner and scope,
	// rejecting revoked or expired tokens with ErrNotFound and recording use.
	PersonalTokenUser(ctx context.Context, tokenHash string) (*User, string, error)
}

// GeneratePersonalToken returns a new prefixed personal access token and its
// SHA-256 hash. Only the hash is ever stored.
func GeneratePersonalToken() (string, string, error) {
	var raw [32]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", "", err
	}
	token := PersonalTokenPrefix + base64.RawURLEncoding.EncodeToString(raw[:])
	return token, HashSessionToken(token), nil
}
