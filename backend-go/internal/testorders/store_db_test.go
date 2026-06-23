package testorders

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/db"
)

func testStore(t *testing.T) (*Store, *pgxpool.Pool) {
	t.Helper()
	dsn := os.Getenv("DOCTORINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCTORINE_TEST_DATABASE_URL to run the test orders DB test")
	}
	ctx := context.Background()
	pool, err := db.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		pool.Close()
		t.Fatalf("migrate: %v", err)
	}
	if _, err := pool.Exec(ctx, `TRUNCATE test_orders, examination_results, examinations RESTART IDENTITY CASCADE`); err != nil {
		pool.Close()
		t.Fatalf("reset: %v", err)
	}
	return NewStore(pool), pool
}

func TestStoreCRUDAndListFilter(t *testing.T) {
	store, pool := testStore(t)
	defer pool.Close()
	ctx := context.Background()

	reason := "baseline before volume"
	order, err := store.Create(ctx, CreateParams{Source: "coach", TestKeys: []string{"ferrytyna", "testosteron"}, Reason: &reason})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if order.Status != StatusRequested || order.Source != "coach" || len(order.TestKeys) != 2 {
		t.Fatalf("created order = %+v", order)
	}
	if order.RequestedOn == "" {
		t.Fatal("requested_on should default to today")
	}

	requested, err := store.List(ctx, StatusRequested)
	if err != nil || len(requested) != 1 {
		t.Fatalf("list requested = (%v, %v), want one", requested, err)
	}
	if completed, _ := store.List(ctx, StatusCompleted); len(completed) != 0 {
		t.Fatalf("list completed = %v, want none", completed)
	}

	// Cancel.
	if err := store.Cancel(ctx, order.ID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if err := store.Cancel(ctx, 99999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cancel missing err = %v, want ErrNotFound", err)
	}
	after, _ := store.List(ctx, StatusCanceled)
	if len(after) != 1 || after[0].Status != StatusCanceled {
		t.Fatalf("after cancel = %+v", after)
	}
}

func TestStoreUpdateLinksExamination(t *testing.T) {
	store, pool := testStore(t)
	defer pool.Close()
	ctx := context.Background()

	order, err := store.Create(ctx, CreateParams{Source: "coach", TestKeys: []string{"tsh"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	examID := insertExam(ctx, t, pool, "2026-06-01")
	status := StatusCompleted
	updated, err := store.Update(ctx, order.ID, UpdateParams{Status: &status, ExaminationID: &examID})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.Status != StatusCompleted || updated.ExaminationID == nil || *updated.ExaminationID != examID {
		t.Fatalf("updated = %+v", updated)
	}
	if _, err := store.Update(ctx, 99999, UpdateParams{Status: &status}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("update missing err = %v, want ErrNotFound", err)
	}

	// Linking to a non-existent examination is a foreign-key violation, mapped
	// to a typed error (rendered 422) rather than an opaque failure.
	badExam := 9999999
	if _, err := store.Update(ctx, order.ID, UpdateParams{ExaminationID: &badExam}); !errors.Is(err, ErrExaminationNotFound) {
		t.Fatalf("update bad examination err = %v, want ErrExaminationNotFound", err)
	}
}

func TestCompleteMatching(t *testing.T) {
	store, pool := testStore(t)
	defer pool.Close()
	ctx := context.Background()

	covered, err := store.Create(ctx, CreateParams{TestKeys: []string{"ferrytyna", "tsh"}})
	if err != nil {
		t.Fatalf("create covered: %v", err)
	}
	uncovered, err := store.Create(ctx, CreateParams{TestKeys: []string{"ferrytyna", "glukoza"}})
	if err != nil {
		t.Fatalf("create uncovered: %v", err)
	}

	examID := insertExam(ctx, t, pool, "2026-06-02")
	// Examination covers ferrytyna + tsh (+ extra), so only `covered` matches.
	n, err := store.CompleteMatching(ctx, examID, []string{"ferrytyna", "tsh", "witamina_d_25_oh"})
	if err != nil {
		t.Fatalf("complete matching: %v", err)
	}
	if n != 1 {
		t.Fatalf("completed %d, want 1", n)
	}

	all, _ := store.List(ctx, "")
	byID := map[int]Order{}
	for _, o := range all {
		byID[o.ID] = o
	}
	if c := byID[covered.ID]; c.Status != StatusCompleted || c.ExaminationID == nil || *c.ExaminationID != examID {
		t.Fatalf("covered order = %+v, want completed+linked", c)
	}
	if u := byID[uncovered.ID]; u.Status != StatusRequested {
		t.Fatalf("uncovered order = %+v, want still requested", u)
	}

	// Empty exam keys complete nothing.
	if n, _ := store.CompleteMatching(ctx, examID, nil); n != 0 {
		t.Fatalf("empty keys completed %d, want 0", n)
	}
}

func insertExam(ctx context.Context, t *testing.T, pool *pgxpool.Pool, date string) int {
	t.Helper()
	var id int
	if err := pool.QueryRow(ctx, `INSERT INTO examinations (title, exam_date) VALUES ($1,$2) RETURNING id`,
		"exam "+date, date).Scan(&id); err != nil {
		t.Fatalf("insert exam: %v", err)
	}
	return id
}
