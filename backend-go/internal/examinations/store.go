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
	ids := []int{}
	for rows.Next() {
		item, err := scanExamination(rows)
		if err != nil {
			return nil, err
		}
		item.Results = []Result{}
		ids = append(ids, item.ID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return items, nil
	}
	results, err := s.listResults(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := map[int][]Result{}
	for _, result := range results {
		byID[result.ExaminationID] = append(byID[result.ExaminationID], result)
	}
	for i := range items {
		items[i].Results = byID[items[i].ID]
		if items[i].Results == nil {
			items[i].Results = []Result{}
		}
	}
	return items, nil
}

func (s *Store) Create(ctx context.Context, params CreateParams) (Examination, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Examination{}, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		INSERT INTO examinations (title, exam_date, category, facility, result_status, summary, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, exam_date, category, facility, result_status, summary, notes, created_at, updated_at
	`, params.Title, params.ExamDate, params.Category, params.Facility, params.ResultStatus, params.Summary, params.Notes)
	item, err := scanExamination(row)
	if err != nil {
		return Examination{}, err
	}
	item.Results = []Result{}
	for _, resultParams := range params.Results {
		result, err := insertResult(ctx, tx, item.ID, resultParams)
		if err != nil {
			return Examination{}, err
		}
		item.Results = append(item.Results, result)
	}
	if err := tx.Commit(ctx); err != nil {
		return Examination{}, err
	}
	return item, nil
}

func (s *Store) listResults(ctx context.Context, examinationIDs []int) ([]Result, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, examination_id, test_key, name, value_text, value_numeric, value_prefix, unit,
			reference_min, reference_max, flag, display_order, created_at, updated_at
		FROM examination_results
		WHERE examination_id = ANY($1::int[])
		ORDER BY examination_id, display_order, name
	`, examinationIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Result{}
	for rows.Next() {
		item, err := scanResult(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func insertResult(ctx context.Context, tx pgx.Tx, examinationID int, params ResultParams) (Result, error) {
	row := tx.QueryRow(ctx, `
		INSERT INTO examination_results (
			examination_id, test_key, name, value_text, value_numeric, value_prefix, unit,
			reference_min, reference_max, flag, display_order
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, examination_id, test_key, name, value_text, value_numeric, value_prefix,
			unit, reference_min, reference_max, flag, display_order, created_at, updated_at
	`, examinationID, params.TestKey, params.Name, params.ValueText, params.ValueNumeric, params.ValuePrefix,
		params.Unit, params.ReferenceMin, params.ReferenceMax, params.Flag, params.DisplayOrder)
	return scanResult(row)
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

func scanResult(row pgx.Row) (Result, error) {
	var item Result
	var valueNumeric pgtype.Float8
	var referenceMin pgtype.Float8
	var referenceMax pgtype.Float8
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.ExaminationID,
		&item.TestKey,
		&item.Name,
		&item.ValueText,
		&valueNumeric,
		&item.ValuePrefix,
		&item.Unit,
		&referenceMin,
		&referenceMax,
		&item.Flag,
		&item.DisplayOrder,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Result{}, fmt.Errorf("scan examination result: %w", err)
	}
	item.ValueNumeric = float64Ptr(valueNumeric)
	item.ReferenceMin = float64Ptr(referenceMin)
	item.ReferenceMax = float64Ptr(referenceMax)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)
	return item, nil
}

func float64Ptr(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}
