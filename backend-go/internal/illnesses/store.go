package illnesses

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context) ([]Illness, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, status, diagnosed_on, resolved_on, clinician, notes, created_at, updated_at
		FROM illnesses
		ORDER BY
			CASE status WHEN 'active' THEN 0 WHEN 'monitoring' THEN 1 ELSE 2 END,
			created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Illness{}
	for rows.Next() {
		item, err := scanIllness(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Create(ctx context.Context, params CreateParams) (Illness, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO illnesses (title, status, diagnosed_on, resolved_on, clinician, notes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, title, status, diagnosed_on, resolved_on, clinician, notes, created_at, updated_at
	`, params.Title, params.Status, params.DiagnosedOn, params.ResolvedOn, params.Clinician, params.Notes)
	return scanIllness(row)
}

func scanIllness(row pgx.Row) (Illness, error) {
	var item Illness
	var diagnosedOn pgtype.Date
	var resolvedOn pgtype.Date
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.Title,
		&item.Status,
		&diagnosedOn,
		&resolvedOn,
		&item.Clinician,
		&item.Notes,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Illness{}, fmt.Errorf("scan illness: %w", err)
	}
	item.DiagnosedOn = healthstatus.FormatDate(diagnosedOn)
	item.ResolvedOn = healthstatus.FormatDate(resolvedOn)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)
	return item, nil
}
