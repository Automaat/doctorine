package examinations

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Automaat/doctorine/backend-go/internal/db"
)

// TestCreateFiresWebhook drives the real Create handler end-to-end (real store,
// spy notifier) so it actually verifies the create path invokes the notifier
// with the flagged markers — deleting the notify call from Create would fail
// this. Skipped unless DOCTORINE_TEST_DATABASE_URL is set.
func TestCreateFiresWebhook(t *testing.T) {
	dsn := os.Getenv("DOCTORINE_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set DOCTORINE_TEST_DATABASE_URL to run the create-webhook DB test")
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
	if _, err := pool.Exec(ctx, `TRUNCATE examination_results, examinations RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("reset: %v", err)
	}

	spy := &spyNotifier{}
	h := NewHandler(NewStore(pool), slog.Default(), spy)

	body := `{
		"title": "Webhook labs",
		"exam_date": "2026-06-02",
		"results": [
			{"test_key": "ferrytyna", "name": "Ferrytyna", "value_numeric": 5, "reference_min": 30, "reference_max": 400},
			{"test_key": "tsh", "name": "TSH", "value_numeric": 2, "reference_min": 0.3, "reference_max": 4}
		]
	}`
	req := httptest.NewRequestWithContext(ctx, http.MethodPost, "/api/examinations", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	spy.mu.Lock()
	defer spy.mu.Unlock()
	if spy.calls != 1 {
		t.Fatalf("notifier called %d times, want 1", spy.calls)
	}
	// ferrytyna 5 < min 30 → flagged L; tsh in range → not flagged.
	if len(spy.event.FlaggedTestKeys) != 1 || spy.event.FlaggedTestKeys[0] != "ferrytyna" {
		t.Fatalf("flagged = %v, want [ferrytyna]", spy.event.FlaggedTestKeys)
	}
	if spy.event.ExaminationID == 0 || spy.event.ExamDate != "2026-06-02" {
		t.Fatalf("event = %+v", spy.event)
	}
}
