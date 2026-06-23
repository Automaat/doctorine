package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (s *Store) CreatePersonalToken(
	ctx context.Context,
	userID int,
	tokenHash string,
	name string,
	scope string,
	expiresAt *time.Time,
) (PersonalToken, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO personal_access_tokens (user_id, token_hash, name, scope, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, name, scope, expires_at, last_used_at, revoked_at, created_at
	`, userID, tokenHash, name, scope, expiresAt)
	return scanPersonalToken(row)
}

func (s *Store) ListPersonalTokens(ctx context.Context, userID int) ([]PersonalToken, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, user_id, name, scope, expires_at, last_used_at, revoked_at, created_at
		FROM personal_access_tokens
		WHERE user_id = $1 AND revoked_at IS NULL
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := []PersonalToken{}
	for rows.Next() {
		token, err := scanPersonalToken(rows)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}
	return tokens, rows.Err()
}

// RevokePersonalToken revokes a token the user owns. It returns ErrNotFound when
// no matching live token exists, so a caller cannot revoke another user's token.
func (s *Store) RevokePersonalToken(ctx context.Context, userID int, id int) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE personal_access_tokens
		SET revoked_at = (now() at time zone 'utc')
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// PersonalTokenUser validates a token hash and, in the same statement, records
// last_used_at. Revoked or expired tokens yield ErrNotFound.
func (s *Store) PersonalTokenUser(ctx context.Context, tokenHash string) (*User, string, error) {
	var user User
	var scope string
	err := s.pool.QueryRow(ctx, `
		UPDATE personal_access_tokens t
		SET last_used_at = (now() at time zone 'utc')
		FROM users u
		WHERE t.token_hash = $1
			AND t.user_id = u.id
			AND t.revoked_at IS NULL
			AND (t.expires_at IS NULL OR t.expires_at > (now() at time zone 'utc'))
		RETURNING t.scope, u.id, u.username, u.password_hash, u.is_admin, u.display_name, u.created_at
	`, tokenHash).Scan(
		&scope,
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.IsAdmin,
		&user.DisplayName,
		&user.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	return &user, scope, nil
}

func scanPersonalToken(row pgx.Row) (PersonalToken, error) {
	var token PersonalToken
	var expiresAt pgtype.Timestamp
	var lastUsedAt pgtype.Timestamp
	var revokedAt pgtype.Timestamp
	if err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.Name,
		&token.Scope,
		&expiresAt,
		&lastUsedAt,
		&revokedAt,
		&token.CreatedAt,
	); err != nil {
		return PersonalToken{}, err
	}
	token.ExpiresAt = timestampPtr(expiresAt)
	token.LastUsedAt = timestampPtr(lastUsedAt)
	token.RevokedAt = timestampPtr(revokedAt)
	return token, nil
}

func timestampPtr(value pgtype.Timestamp) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
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
