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
