package results

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/db"
)

// TestLatestByTestKeys exercises the real DISTINCT ON query: it must pick the
// newest result per test_key by exam_date and honor the optional key filter.
// Skipped unless DOCTORINE_TEST_DATABASE_URL is set.
func TestLatestByTestKeys(t *testing.T) {
	dsn := os.Getenv("DOCTORINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCTORINE_TEST_DATABASE_URL to run the latest-results DB test")
	}
	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	resetResults(ctx, t, pool)
	store := NewStore(pool)

	// Two examinations, ferrytyna measured in both; the newer date must win.
	oldExam := insertExam(ctx, t, pool, "2026-01-01")
	newExam := insertExam(ctx, t, pool, "2026-06-01")
	insertResult(ctx, t, pool, oldExam, "ferrytyna", "Ferrytyna", 30)
	insertResult(ctx, t, pool, newExam, "ferrytyna", "Ferrytyna", 80)
	insertResult(ctx, t, pool, oldExam, "tsh", "TSH", 2)

	all, err := store.LatestByTestKeys(ctx, []string{})
	if err != nil {
		t.Fatalf("latest all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("latest all = %d rows, want 2 (%+v)", len(all), all)
	}
	byKey := map[string]LatestResult{}
	for _, r := range all {
		byKey[r.TestKey] = r
	}
	ferr := byKey["ferrytyna"]
	if ferr.ValueNumeric == nil || *ferr.ValueNumeric != 80 || ferr.ExamDate != "2026-06-01" {
		t.Fatalf("ferrytyna latest = %+v, want value 80 on 2026-06-01", ferr)
	}

	filtered, err := store.LatestByTestKeys(ctx, []string{"tsh"})
	if err != nil {
		t.Fatalf("latest filtered: %v", err)
	}
	if len(filtered) != 1 || filtered[0].TestKey != "tsh" {
		t.Fatalf("filtered = %+v, want only tsh", filtered)
	}

	// A nil slice must behave like an empty slice ("all keys"), not encode as
	// SQL NULL and return nothing.
	viaNil, err := store.LatestByTestKeys(ctx, nil)
	if err != nil {
		t.Fatalf("latest nil: %v", err)
	}
	if len(viaNil) != 2 {
		t.Fatalf("latest nil = %d rows, want 2", len(viaNil))
	}
}

func TestTrendByTestKey(t *testing.T) {
	dsn := os.Getenv("DOCTORINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCTORINE_TEST_DATABASE_URL to run the trend DB test")
	}
	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	resetResults(ctx, t, pool)
	store := NewStore(pool)

	jan := insertExam(ctx, t, pool, "2026-01-01")
	mar := insertExam(ctx, t, pool, "2026-03-01")
	insertResult(ctx, t, pool, mar, "ferrytyna", "Ferrytyna", 80)
	insertResult(ctx, t, pool, jan, "ferrytyna", "Ferrytyna", 30)
	// A textual-only row for the same key must be excluded from the series.
	insertText(ctx, t, pool, jan, "opis", "Opis", "prawidłowy")

	series, err := store.TrendByTestKey(ctx, "ferrytyna", 36500)
	if err != nil {
		t.Fatalf("trend: %v", err)
	}
	if len(series) != 2 {
		t.Fatalf("trend = %d points, want 2 (%+v)", len(series), series)
	}
	// Oldest first.
	if series[0].ExamDate != "2026-01-01" || series[1].ExamDate != "2026-03-01" {
		t.Fatalf("trend order = %+v, want ascending by date", series)
	}

	// A narrow window must exclude the older point.
	recent, err := store.TrendByTestKey(ctx, "ferrytyna", 1)
	if err != nil {
		t.Fatalf("trend recent: %v", err)
	}
	for _, p := range recent {
		if p.ExamDate == "2026-01-01" {
			t.Fatalf("1-day window leaked an old point: %+v", recent)
		}
	}
}

func resetResults(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(ctx, `TRUNCATE examination_results, examinations RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset: %v", err)
	}
}

func insertExam(ctx context.Context, t *testing.T, pool *pgxpool.Pool, date string) int {
	t.Helper()
	var id int
	err := pool.QueryRow(ctx, `
		INSERT INTO examinations (title, exam_date) VALUES ($1, $2) RETURNING id
	`, "exam "+date, date).Scan(&id)
	if err != nil {
		t.Fatalf("insert exam: %v", err)
	}
	return id
}

func insertResult(ctx context.Context, t *testing.T, pool *pgxpool.Pool, examID int, key, name string, value float64) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO examination_results (examination_id, test_key, name, value_numeric)
		VALUES ($1, $2, $3, $4)
	`, examID, key, name, value)
	if err != nil {
		t.Fatalf("insert result: %v", err)
	}
}

func insertText(ctx context.Context, t *testing.T, pool *pgxpool.Pool, examID int, key, name, text string) {
	t.Helper()
	_, err := pool.Exec(ctx, `
		INSERT INTO examination_results (examination_id, test_key, name, value_text)
		VALUES ($1, $2, $3, $4)
	`, examID, key, name, text)
	if err != nil {
		t.Fatalf("insert text result: %v", err)
	}
}
