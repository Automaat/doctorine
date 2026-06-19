package examinations

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
)

var ErrResultDefinitionNotFound = errors.New("result definition not found")

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
		resultParams, err := resolveResultDefinition(ctx, tx, resultParams)
		if err != nil {
			return Examination{}, err
		}
		resultParams.Flag = computeFlag(
			resultParams.ValueNumeric,
			resultParams.ValuePrefix,
			resultParams.ReferenceMin,
			resultParams.ReferenceMax,
		)
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
		SELECT
			er.id,
			er.examination_id,
			er.definition_id,
			er.test_key,
			er.name,
			er.value_text,
			er.value_numeric,
			er.value_prefix,
			er.unit,
			er.reference_min,
			er.reference_max,
			er.flag,
			er.display_order,
			er.created_at,
			er.updated_at,
			rd.id,
			rd.test_key,
			rd.name,
			rd.unit,
			rd.reference_min,
			rd.reference_max,
			rd.category,
			rd.created_at,
			rd.updated_at
		FROM examination_results er
		LEFT JOIN result_definitions rd ON rd.id = er.definition_id
		WHERE er.examination_id = ANY($1::int[])
		ORDER BY er.examination_id, er.display_order, er.name
	`, examinationIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Result{}
	for rows.Next() {
		item, err := scanJoinedResult(rows)
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
			examination_id, definition_id, test_key, name, value_text, value_numeric, value_prefix, unit,
			reference_min, reference_max, flag, display_order
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, examination_id, definition_id, test_key, name, value_text, value_numeric, value_prefix,
			unit, reference_min, reference_max, flag, display_order, created_at, updated_at
	`, examinationID, params.DefinitionID, params.TestKey, params.Name, params.ValueText, params.ValueNumeric, params.ValuePrefix,
		params.Unit, params.ReferenceMin, params.ReferenceMax, params.Flag, params.DisplayOrder)
	return scanResult(row)
}

func resolveResultDefinition(ctx context.Context, tx pgx.Tx, params ResultParams) (ResultParams, error) {
	if params.DefinitionID != nil {
		var definition ResultDefinition
		var referenceMin pgtype.Float8
		var referenceMax pgtype.Float8
		err := tx.QueryRow(ctx, `
			SELECT id, test_key, name, unit, reference_min, reference_max
			FROM result_definitions
			WHERE id = $1
		`, *params.DefinitionID).Scan(
			&definition.ID,
			&definition.TestKey,
			&definition.Name,
			&definition.Unit,
			&referenceMin,
			&referenceMax,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return ResultParams{}, ErrResultDefinitionNotFound
		}
		if err != nil {
			return ResultParams{}, err
		}
		params.DefinitionID = &definition.ID
		params.TestKey = definition.TestKey
		params.Name = definition.Name
		params.Unit = definition.Unit
		params.ReferenceMin = float64Ptr(referenceMin)
		params.ReferenceMax = float64Ptr(referenceMax)
		return params, nil
	}

	var id int
	err := tx.QueryRow(ctx, `
		INSERT INTO result_definitions (test_key, name, unit, reference_min, reference_max)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (test_key) DO UPDATE SET
			name = EXCLUDED.name,
			unit = COALESCE(EXCLUDED.unit, result_definitions.unit),
			reference_min = COALESCE(EXCLUDED.reference_min, result_definitions.reference_min),
			reference_max = COALESCE(EXCLUDED.reference_max, result_definitions.reference_max),
			updated_at = now() at time zone 'utc'
		RETURNING id
	`, params.TestKey, params.Name, params.Unit, params.ReferenceMin, params.ReferenceMax).Scan(&id)
	if err != nil {
		return ResultParams{}, err
	}
	params.DefinitionID = &id
	return params, nil
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
	var definitionID pgtype.Int4
	var valueNumeric pgtype.Float8
	var referenceMin pgtype.Float8
	var referenceMax pgtype.Float8
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.ExaminationID,
		&definitionID,
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
	item.DefinitionID = intPtr(definitionID)
	item.ValueNumeric = float64Ptr(valueNumeric)
	item.ReferenceMin = float64Ptr(referenceMin)
	item.ReferenceMax = float64Ptr(referenceMax)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)
	return item, nil
}

func scanJoinedResult(row pgx.Row) (Result, error) {
	var item Result
	var definitionID pgtype.Int4
	var valueNumeric pgtype.Float8
	var referenceMin pgtype.Float8
	var referenceMax pgtype.Float8
	var createdAt time.Time
	var updatedAt time.Time
	var joinedDefinitionID pgtype.Int4
	var joinedTestKey pgtype.Text
	var joinedName pgtype.Text
	var joinedUnit pgtype.Text
	var joinedReferenceMin pgtype.Float8
	var joinedReferenceMax pgtype.Float8
	var joinedCategory pgtype.Text
	var joinedCreatedAt pgtype.Timestamp
	var joinedUpdatedAt pgtype.Timestamp
	if err := row.Scan(
		&item.ID,
		&item.ExaminationID,
		&definitionID,
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
		&joinedDefinitionID,
		&joinedTestKey,
		&joinedName,
		&joinedUnit,
		&joinedReferenceMin,
		&joinedReferenceMax,
		&joinedCategory,
		&joinedCreatedAt,
		&joinedUpdatedAt,
	); err != nil {
		return Result{}, fmt.Errorf("scan joined examination result: %w", err)
	}
	item.DefinitionID = intPtr(definitionID)
	item.ValueNumeric = float64Ptr(valueNumeric)
	item.ReferenceMin = float64Ptr(referenceMin)
	item.ReferenceMax = float64Ptr(referenceMax)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	item.UpdatedAt = updatedAt.Format(time.RFC3339)
	if joinedDefinitionID.Valid {
		item.Definition = &ResultDefinition{
			ID:           int(joinedDefinitionID.Int32),
			TestKey:      joinedTestKey.String,
			Name:         joinedName.String,
			Unit:         pgTextPtr(joinedUnit),
			ReferenceMin: float64Ptr(joinedReferenceMin),
			ReferenceMax: float64Ptr(joinedReferenceMax),
			Category:     joinedCategory.String,
			CreatedAt:    timestampString(joinedCreatedAt),
			UpdatedAt:    timestampString(joinedUpdatedAt),
		}
	}
	return item, nil
}

func float64Ptr(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}

func pgTextPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func timestampString(value pgtype.Timestamp) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
}

func intPtr(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	item := int(value.Int32)
	return &item
}
