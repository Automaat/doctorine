package overview

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Automaat/doctorine/backend-go/internal/documents"
)

type fakeSource struct {
	counts     Counts
	recent     []documents.Document
	countsErr  error
	recentErr  error
	recentSeen int
}

func (f *fakeSource) Counts(context.Context) (Counts, error) {
	return f.counts, f.countsErr
}

func (f *fakeSource) RecentDocuments(_ context.Context, limit int) ([]documents.Document, error) {
	f.recentSeen = limit
	return f.recent, f.recentErr
}

func handlerWith(src source) *Handler {
	return &Handler{src: src, logger: slog.Default()}
}

func getOverview(t *testing.T, h *Handler) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/overview", http.NoBody)
	h.Get(rec, req)
	return rec
}

func TestHandlerGet(t *testing.T) {
	t.Run("renders counts and recent documents", func(t *testing.T) {
		src := &fakeSource{
			counts: Counts{Documents: 3, Illnesses: 2, Examinations: 5, Flagged: 1},
			recent: []documents.Document{{ID: 11, Title: "Scan"}},
		}
		rec := getOverview(t, handlerWith(src))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var got Response
		if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.DocumentCount != 3 || got.IllnessCount != 2 || got.ExaminationCount != 5 ||
			got.FlaggedResults != 1 {
			t.Fatalf("counts = %+v", got)
		}
		if len(got.RecentDocuments) != 1 || got.RecentDocuments[0].ID != 11 {
			t.Fatalf("recent = %+v", got.RecentDocuments)
		}
		if src.recentSeen != 5 {
			t.Fatalf("recent limit = %d, want 5", src.recentSeen)
		}
	})

	t.Run("returns 500 when counts fail", func(t *testing.T) {
		rec := getOverview(t, handlerWith(&fakeSource{countsErr: errors.New("db down")}))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})

	t.Run("returns 500 when recent documents fail", func(t *testing.T) {
		rec := getOverview(t, handlerWith(&fakeSource{recentErr: errors.New("db down")}))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}
