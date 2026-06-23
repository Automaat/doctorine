package results

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeStore struct {
	gotKeys     []string
	results     []LatestResult
	err         error
	gotTrendKey string
	gotDays     int
	trend       []TrendPoint
	trendErr    error
}

func (f *fakeStore) LatestByTestKeys(_ context.Context, keys []string) ([]LatestResult, error) {
	f.gotKeys = keys
	return f.results, f.err
}

func (f *fakeStore) TrendByTestKey(_ context.Context, testKey string, days int) ([]TrendPoint, error) {
	f.gotTrendKey, f.gotDays = testKey, days
	return f.trend, f.trendErr
}

func newHandler(store Reader) *Handler {
	return &Handler{store: store, logger: slog.Default()}
}

func do(h *Handler, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/results/latest"+query, http.NoBody)
	rec := httptest.NewRecorder()
	h.Latest(rec, req)
	return rec
}

func doTrend(h *Handler, testKey, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/results/trend/"+testKey+query, http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("test_key", testKey)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()
	h.Trend(rec, req)
	return rec
}

func TestLatestReturnsResults(t *testing.T) {
	value := 42.0
	store := &fakeStore{results: []LatestResult{{TestKey: "ferrytyna", Name: "Ferrytyna", ExamDate: "2026-06-01", ValueNumeric: &value}}}
	rec := do(newHandler(store), "?test_keys=ferrytyna,tsh")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if len(store.gotKeys) != 2 || store.gotKeys[0] != "ferrytyna" || store.gotKeys[1] != "tsh" {
		t.Fatalf("gotKeys = %v, want [ferrytyna tsh]", store.gotKeys)
	}
	var out []LatestResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].TestKey != "ferrytyna" {
		t.Fatalf("out = %+v", out)
	}
}

func TestLatestNoParamMeansAllKeys(t *testing.T) {
	store := &fakeStore{results: []LatestResult{}}
	rec := do(newHandler(store), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if store.gotKeys == nil || len(store.gotKeys) != 0 {
		t.Fatalf("gotKeys = %v, want empty non-nil slice", store.gotKeys)
	}
	if rec.Body.String() != "[]\n" {
		t.Fatalf("body = %q, want empty array", rec.Body.String())
	}
}

func TestLatestDeduplicatesAndTrims(t *testing.T) {
	store := &fakeStore{results: []LatestResult{}}
	do(newHandler(store), "?test_keys=tsh,%20tsh%20,,glukoza")
	if len(store.gotKeys) != 2 || store.gotKeys[0] != "tsh" || store.gotKeys[1] != "glukoza" {
		t.Fatalf("gotKeys = %v, want [tsh glukoza]", store.gotKeys)
	}
}

func TestLatestRejectsInvalidKey(t *testing.T) {
	rec := do(newHandler(&fakeStore{}), "?test_keys=Ferritin")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestLatestSurfacesStoreError(t *testing.T) {
	rec := do(newHandler(&fakeStore{err: errors.New("boom")}), "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestTrendReturnsSeries(t *testing.T) {
	v1, v2 := 30.0, 80.0
	store := &fakeStore{trend: []TrendPoint{{ExamDate: "2026-01-01", ValueNumeric: &v1}, {ExamDate: "2026-06-01", ValueNumeric: &v2}}}
	rec := doTrend(newHandler(store), "ferrytyna", "?days=180")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if store.gotTrendKey != "ferrytyna" || store.gotDays != 180 {
		t.Fatalf("store got key=%q days=%d, want ferrytyna/180", store.gotTrendKey, store.gotDays)
	}
	var out []TrendPoint
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0].ExamDate != "2026-01-01" {
		t.Fatalf("out = %+v", out)
	}
}

func TestTrendDefaultsDays(t *testing.T) {
	store := &fakeStore{trend: []TrendPoint{}}
	if rec := doTrend(newHandler(store), "tsh", ""); rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if store.gotDays != 365 {
		t.Fatalf("days = %d, want default 365", store.gotDays)
	}
}

func TestTrendRejectsInvalidKey(t *testing.T) {
	if rec := doTrend(newHandler(&fakeStore{}), "TSH", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestTrendRejectsBadDays(t *testing.T) {
	for _, days := range []string{"?days=0", "?days=-5", "?days=abc", "?days=99999999"} {
		if rec := doTrend(newHandler(&fakeStore{}), "tsh", days); rec.Code != http.StatusBadRequest {
			t.Fatalf("days %q: status = %d, want 400", days, rec.Code)
		}
	}
}

func TestTrendSurfacesStoreError(t *testing.T) {
	store := &fakeStore{trendErr: errors.New("boom")}
	if rec := doTrend(newHandler(store), "tsh", ""); rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
