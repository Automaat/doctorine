package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("user not found")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) UpsertAdmin(ctx context.Context, username string, passwordHash string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("admin username is required")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO users (username, password_hash, is_admin, display_name)
		VALUES ($1, $2, true, $1)
		ON CONFLICT (username) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
			is_admin = true,
			updated_at = now() at time zone 'utc'
	`, username, passwordHash)
	return err
}

func (s *Store) GetByUsername(ctx context.Context, username string) (*User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, is_admin, display_name, created_at
		FROM users
		WHERE username = $1
	`, strings.TrimSpace(username))
	return scanUser(row)
}

func (s *Store) CreateSession(
	ctx context.Context,
	userID int,
	tokenHash string,
	expiresAt time.Time,
) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

// SessionUser returns the user owning a session whose token hash matches and is
// neither revoked nor expired. It returns ErrNotFound otherwise.
func (s *Store) SessionUser(ctx context.Context, tokenHash string) (*User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT u.id, u.username, u.password_hash, u.is_admin, u.display_name, u.created_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
			AND s.revoked_at IS NULL
			AND s.expires_at > (now() at time zone 'utc')
	`, tokenHash)
	return scanUser(row)
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE sessions
		SET revoked_at = (now() at time zone 'utc')
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	return err
}

func scanUser(row pgx.Row) (*User, error) {
	var user User
	if err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.DisplayName,
		&user.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}
