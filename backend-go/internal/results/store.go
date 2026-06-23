package results

import (
	"context"
	"fmt"

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

// LatestByTestKeys returns the most recent result per test_key, ordered by
// test_key. When keys is empty every known test_key is returned; otherwise only
// the requested keys that have at least one result appear. Recency is decided by
// the owning examination's exam_date (newest wins), breaking ties on result id.
func (s *Store) LatestByTestKeys(ctx context.Context, keys []string) ([]LatestResult, error) {
	// pgx encodes a nil slice as SQL NULL, which would make cardinality()
	// return NULL and filter out every row. Normalize so nil means "all keys".
	if keys == nil {
		keys = []string{}
	}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (er.test_key)
			er.test_key,
			er.name,
			e.exam_date,
			er.value_text,
			er.value_numeric,
			er.value_prefix,
			er.unit,
			er.reference_min,
			er.reference_max,
			er.flag
		FROM examination_results er
		JOIN examinations e ON e.id = er.examination_id
		WHERE cardinality($1::text[]) = 0 OR er.test_key = ANY($1::text[])
		ORDER BY er.test_key, e.exam_date DESC, er.id DESC
	`, keys)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []LatestResult{}
	for rows.Next() {
		item, err := scanLatest(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanLatest(row pgx.Row) (LatestResult, error) {
	var item LatestResult
	var examDate pgtype.Date
	var valueNumeric pgtype.Float8
	var referenceMin pgtype.Float8
	var referenceMax pgtype.Float8
	if err := row.Scan(
		&item.TestKey,
		&item.Name,
		&examDate,
		&item.ValueText,
		&valueNumeric,
		&item.ValuePrefix,
		&item.Unit,
		&referenceMin,
		&referenceMax,
		&item.Flag,
	); err != nil {
		return LatestResult{}, fmt.Errorf("scan latest result: %w", err)
	}
	item.ExamDate = healthstatus.FormatRequiredDate(examDate)
	item.ValueNumeric = float64Ptr(valueNumeric)
	item.ReferenceMin = float64Ptr(referenceMin)
	item.ReferenceMax = float64Ptr(referenceMax)
	return item, nil
}

func float64Ptr(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	return &value.Float64
}
