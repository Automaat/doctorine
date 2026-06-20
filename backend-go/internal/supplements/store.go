package supplements

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context) ([]Supplement, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, value_text, frequency, notes, created_at, updated_at
		FROM supplements
		ORDER BY name, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Supplement{}
	for rows.Next() {
		item, err := scanSupplement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Create(ctx context.Context, params CreateParams) (Supplement, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO supplements (name, value_text, frequency, notes)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, value_text, frequency, notes, created_at, updated_at
	`, params.Name, params.Value, params.Frequency, params.Notes)
	return scanSupplement(row)
}

func scanSupplement(row pgx.Row) (Supplement, error) {
	var item Supplement
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Value,
		&item.Frequency,
		&item.Notes,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Supplement{}, fmt.Errorf("scan supplement: %w", err)
	}
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)
	return item, nil
}
