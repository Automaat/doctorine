package documents

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

var ErrNotFound = errors.New("document not found")

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) List(ctx context.Context) ([]Document, error) {
	rows, err := s.pool.Query(ctx, listSQL()+` ORDER BY d.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Document{}
	for rows.Next() {
		item, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Recent(ctx context.Context, limit int) ([]Document, error) {
	rows, err := s.pool.Query(ctx, listSQL()+` ORDER BY d.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Document{}
	for rows.Next() {
		item, err := scanDocument(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) Create(ctx context.Context, params CreateParams) (Document, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO documents (
			title, document_type, issued_at, original_filename, storage_name, content_type,
			size_bytes, sha256_hex, notes, illness_id, examination_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, params.Title, params.DocumentType, params.IssuedAt, params.OriginalFilename, params.StorageName,
		params.ContentType, params.SizeBytes, params.SHA256Hex, params.Notes, params.IllnessID,
		params.ExaminationID)
	var id int
	if err := row.Scan(&id); err != nil {
		return Document{}, err
	}
	return s.Get(ctx, id)
}

func (s *Store) Get(ctx context.Context, id int) (Document, error) {
	row := s.pool.QueryRow(ctx, listSQL()+` WHERE d.id = $1`, id)
	item, err := scanDocument(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Document{}, ErrNotFound
		}
		return Document{}, err
	}
	return item, nil
}

func (s *Store) Delete(ctx context.Context, id int) (Document, error) {
	item, err := s.Get(ctx, id)
	if err != nil {
		return Document{}, err
	}
	if _, err := s.pool.Exec(ctx, `DELETE FROM documents WHERE id = $1`, id); err != nil {
		return Document{}, err
	}
	return item, nil
}

func listSQL() string {
	return `
		SELECT
			d.id, d.title, d.document_type, d.issued_at, d.original_filename, d.storage_name,
			d.content_type, d.size_bytes, d.sha256_hex, d.notes, d.illness_id, i.title,
			d.examination_id, e.title, d.created_at
		FROM documents d
		LEFT JOIN illnesses i ON i.id = d.illness_id
		LEFT JOIN examinations e ON e.id = d.examination_id`
}

func scanDocument(row pgx.Row) (Document, error) {
	var item Document
	var issuedAt pgtype.Date
	var createdAt time.Time
	if err := row.Scan(
		&item.ID,
		&item.Title,
		&item.DocumentType,
		&issuedAt,
		&item.OriginalFilename,
		&item.StorageName,
		&item.ContentType,
		&item.SizeBytes,
		&item.SHA256Hex,
		&item.Notes,
		&item.IllnessID,
		&item.IllnessTitle,
		&item.ExaminationID,
		&item.ExaminationTitle,
		&createdAt,
	); err != nil {
		return Document{}, fmt.Errorf("scan document: %w", err)
	}
	item.IssuedAt = healthstatus.FormatDate(issuedAt)
	item.CreatedAt = createdAt.Format(time.RFC3339)
	return item, nil
}
