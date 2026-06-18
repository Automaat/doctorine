package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
