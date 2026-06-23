package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"
)

// SessionStore persists and looks up opaque, revocable server-side sessions.
type SessionStore interface {
	CreateSession(ctx context.Context, userID int, tokenHash string, expiresAt time.Time) error
	SessionUser(ctx context.Context, tokenHash string) (*User, error)
	RevokeSession(ctx context.Context, tokenHash string) error
}

// UserStore looks up users during authentication.
type UserStore interface {
	GetByUsername(ctx context.Context, username string) (*User, error)
}

// GenerateSessionToken returns a new opaque session token and its SHA-256 hash.
// Only the hash is ever stored, so a database leak cannot reveal usable tokens.
func GenerateSessionToken() (string, string, error) {
	for {
		var raw [32]byte
		if _, err := rand.Read(raw[:]); err != nil {
			return "", "", err
		}
		token := base64.RawURLEncoding.EncodeToString(raw[:])
		// Never emit a session token that collides with the personal access
		// token prefix, so the auth middleware can route by prefix unambiguously.
		if strings.HasPrefix(token, PersonalTokenPrefix) {
			continue
		}
		return token, HashSessionToken(token), nil
	}
}

// HashSessionToken hashes a presented token so it can be matched against the
// stored session hash.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
