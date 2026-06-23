package testorders

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

var ErrNotFound = errors.New("test order not found")

const orderColumns = `id, source, test_keys, reason, status, requested_on, due_on,
	examination_id, notes, created_at, updated_at`

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) Create(ctx context.Context, params CreateParams) (Order, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO test_orders (source, test_keys, reason, due_on, notes)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+orderColumns, params.Source, params.TestKeys, params.Reason, params.DueOn, params.Notes)
	return scanOrder(row)
}

// List returns orders, newest first. An empty status returns every order.
func (s *Store) List(ctx context.Context, status string) ([]Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+orderColumns+`
		FROM test_orders
		WHERE $1 = '' OR status = $1
		ORDER BY requested_on DESC, id DESC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orders := []Order{}
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// Update changes status, examination link, and/or notes. nil fields are left
// unchanged. It returns ErrNotFound when no order has the id.
func (s *Store) Update(ctx context.Context, id int, params UpdateParams) (Order, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE test_orders
		SET status = COALESCE($2, status),
			examination_id = CASE WHEN $3::int IS NOT NULL THEN $3 ELSE examination_id END,
			notes = COALESCE($4, notes),
			updated_at = (now() at time zone 'utc')
		WHERE id = $1
		RETURNING `+orderColumns, id, params.Status, params.ExaminationID, params.Notes)
	order, err := scanOrder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	return order, err
}

// Cancel marks an order canceled. It returns ErrNotFound when no order has the
// id, and is a no-op for an already-canceled order.
func (s *Store) Cancel(ctx context.Context, id int) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE test_orders
		SET status = 'canceled', updated_at = (now() at time zone 'utc')
		WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CompleteMatching completes every still-requested order whose test_keys are all
// covered by the given examination's result keys, linking it to that
// examination. It returns the number of orders completed.
func (s *Store) CompleteMatching(ctx context.Context, examinationID int, examTestKeys []string) (int, error) {
	if len(examTestKeys) == 0 {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE test_orders
		SET status = 'completed',
			examination_id = $1,
			updated_at = (now() at time zone 'utc')
		WHERE status = 'requested' AND test_keys <@ $2::text[]
	`, examinationID, examTestKeys)
	if err != nil {
		return 0, err
	}
	return int(tag.RowsAffected()), nil
}

func scanOrder(row pgx.Row) (Order, error) {
	var order Order
	var reason pgtype.Text
	var requestedOn pgtype.Date
	var dueOn pgtype.Date
	var examinationID pgtype.Int4
	var notes pgtype.Text
	var createdAt time.Time
	var updatedAt time.Time
	if err := row.Scan(
		&order.ID,
		&order.Source,
		&order.TestKeys,
		&reason,
		&order.Status,
		&requestedOn,
		&dueOn,
		&examinationID,
		&notes,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Order{}, fmt.Errorf("scan test order: %w", err)
	}
	order.Reason = pgTextPtr(reason)
	order.RequestedOn = healthstatus.FormatRequiredDate(requestedOn)
	order.DueOn = healthstatus.FormatDate(dueOn)
	order.ExaminationID = int4Ptr(examinationID)
	order.Notes = pgTextPtr(notes)
	order.CreatedAt = createdAt.Format(time.RFC3339)
	order.UpdatedAt = updatedAt.Format(time.RFC3339)
	if order.TestKeys == nil {
		order.TestKeys = []string{}
	}
	return order, nil
}

func pgTextPtr(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func int4Ptr(value pgtype.Int4) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int32)
	return &v
}
