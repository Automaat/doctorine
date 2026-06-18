package examinations

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

func (s *Store) List(ctx context.Context) ([]Examination, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, exam_date, category, facility, result_status, summary, notes, created_at, updated_at
		FROM examinations
		ORDER BY exam_date DESC, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Examination{}
	for rows.Next() {
		item, err := scanExamination(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Create(ctx context.Context, params CreateParams) (Examination, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO examinations (title, exam_date, category, facility, result_status, summary, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, exam_date, category, facility, result_status, summary, notes, created_at, updated_at
	`, params.Title, params.ExamDate, params.Category, params.Facility, params.ResultStatus, params.Summary, params.Notes)
	return scanExamination(row)
}

func scanExamination(row pgx.Row) (Examination, error) {
	var item Examination
	var examDate pgtype.Date
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.Title,
		&examDate,
		&item.Category,
		&item.Facility,
		&item.ResultStatus,
		&item.Summary,
		&item.Notes,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Examination{}, fmt.Errorf("scan examination: %w", err)
	}
	item.ExamDate = healthstatus.FormatRequiredDate(examDate)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)
	return item, nil
}
