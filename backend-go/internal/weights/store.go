package weights

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
)

var ErrEntryNotFound = errors.New("weight entry not found")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context) ([]Entry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, measured_on, weight_kg, notes, created_at, updated_at
		FROM weight_entries
		ORDER BY measured_on DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Entry{}
	for rows.Next() {
		item, err := scanEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Create records a measurement, replacing an existing entry for the same day so
// the time series keeps one point per date.
func (s *Store) Create(ctx context.Context, params CreateParams) (Entry, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO weight_entries (measured_on, weight_kg, notes)
		VALUES ($1, $2, $3)
		ON CONFLICT (measured_on) DO UPDATE SET
			weight_kg = EXCLUDED.weight_kg,
			notes = EXCLUDED.notes,
			updated_at = now() at time zone 'utc'
		RETURNING id, measured_on, weight_kg, notes, created_at, updated_at
	`, params.MeasuredOn, params.WeightKg, params.Notes)
	return scanEntry(row)
}

func (s *Store) Delete(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM weight_entries WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrEntryNotFound
	}
	return nil
}

func scanEntry(row pgx.Row) (Entry, error) {
	var item Entry
	var measuredOn time.Time
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&item.ID,
		&measuredOn,
		&item.WeightKg,
		&item.Notes,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Entry{}, fmt.Errorf("scan weight entry: %w", err)
	}
	item.MeasuredOn = measuredOn.Format(healthstatus.DateLayout)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)
	return item, nil
}
