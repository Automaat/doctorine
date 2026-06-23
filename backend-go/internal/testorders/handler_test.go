package testorders

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeRepo struct {
	created    CreateParams
	createErr  error
	listStatus string
	list       []Order
	updated    UpdateParams
	updateErr  error
	cancelErr  error
	canceled   int
}

func (f *fakeRepo) Create(_ context.Context, params CreateParams) (Order, error) {
	f.created = params
	if f.createErr != nil {
		return Order{}, f.createErr
	}
	return Order{ID: 1, Source: params.Source, TestKeys: params.TestKeys, Status: StatusRequested}, nil
}

func (f *fakeRepo) List(_ context.Context, status string) ([]Order, error) {
	f.listStatus = status
	return f.list, nil
}

func (f *fakeRepo) Update(_ context.Context, _ int, params UpdateParams) (Order, error) {
	f.updated = params
	if f.updateErr != nil {
		return Order{}, f.updateErr
	}
	return Order{ID: 1, Status: StatusCompleted}, nil
}

func (f *fakeRepo) Cancel(_ context.Context, id int) error {
	f.canceled = id
	return f.cancelErr
}

func handler(repo Repository) *Handler {
	return &Handler{store: repo, logger: slog.Default()}
}

func post(h *Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/test-orders", strings.NewReader(body))
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	h.Create(rec, req)
	return rec
}

func withID(req *http.Request, id string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
}

func TestCreateValidatesAndDefaults(t *testing.T) {
	repo := &fakeRepo{}
	rec := post(handler(repo), `{"test_keys":["ferrytyna"," ferrytyna ","tsh"],"reason":" baseline "}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", rec.Code, rec.Body.String())
	}
	if repo.created.Source != "coach" {
		t.Fatalf("source = %q, want coach default", repo.created.Source)
	}
	if len(repo.created.TestKeys) != 2 || repo.created.TestKeys[0] != "ferrytyna" || repo.created.TestKeys[1] != "tsh" {
		t.Fatalf("test_keys = %v, want deduped [ferrytyna tsh]", repo.created.TestKeys)
	}
	if repo.created.Reason == nil || *repo.created.Reason != "baseline" {
		t.Fatalf("reason = %v, want baseline", repo.created.Reason)
	}
}

func TestCreateRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"empty keys":   `{"test_keys":[]}`,
		"missing keys": `{"reason":"x"}`,
		"bad key":      `{"test_keys":["Ferritin"]}`,
		"blank key":    `{"test_keys":["ferrytyna",""]}`,
		"bad due_on":   `{"test_keys":["tsh"],"due_on":"06-01-2026"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if rec := post(handler(&fakeRepo{}), body); rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want 422 (%s)", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListFiltersByStatus(t *testing.T) {
	repo := &fakeRepo{list: []Order{{ID: 1, Status: StatusRequested}}}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/test-orders?status=requested", http.NoBody)
	rec := httptest.NewRecorder()
	handler(repo).List(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if repo.listStatus != "requested" {
		t.Fatalf("list status = %q, want requested", repo.listStatus)
	}
	var out []Order
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil || len(out) != 1 {
		t.Fatalf("out = %v err = %v", out, err)
	}
}

func TestListRejectsBadStatus(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/test-orders?status=bogus", http.NoBody)
	rec := httptest.NewRecorder()
	handler(&fakeRepo{}).List(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestUpdateLinksExamination(t *testing.T) {
	repo := &fakeRepo{}
	req := withID(httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/test-orders/1",
		strings.NewReader(`{"status":"completed","examination_id":7}`)), "1")
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	handler(repo).Update(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", rec.Code, rec.Body.String())
	}
	if repo.updated.Status == nil || *repo.updated.Status != "completed" {
		t.Fatalf("status = %v, want completed", repo.updated.Status)
	}
	if repo.updated.ExaminationID == nil || *repo.updated.ExaminationID != 7 {
		t.Fatalf("examination_id = %v, want 7", repo.updated.ExaminationID)
	}
}

func TestUpdateRejectsEmptyPatch(t *testing.T) {
	req := withID(httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/test-orders/1",
		strings.NewReader(`{}`)), "1")
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	handler(&fakeRepo{}).Update(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", rec.Code)
	}
}

func TestUpdateNotFound(t *testing.T) {
	repo := &fakeRepo{updateErr: ErrNotFound}
	req := withID(httptest.NewRequestWithContext(context.Background(), http.MethodPatch, "/api/test-orders/9",
		strings.NewReader(`{"status":"canceled"}`)), "9")
	req.Header.Set("content-type", "application/json")
	rec := httptest.NewRecorder()
	handler(repo).Update(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDeleteCancels(t *testing.T) {
	repo := &fakeRepo{}
	req := withID(httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/test-orders/3", http.NoBody), "3")
	rec := httptest.NewRecorder()
	handler(repo).Delete(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
	if repo.canceled != 3 {
		t.Fatalf("canceled id = %d, want 3", repo.canceled)
	}
}

func TestDeleteNotFound(t *testing.T) {
	repo := &fakeRepo{cancelErr: ErrNotFound}
	req := withID(httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/test-orders/9", http.NoBody), "9")
	rec := httptest.NewRecorder()
	handler(repo).Delete(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestDeleteRejectsBadID(t *testing.T) {
	req := withID(httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "/api/test-orders/x", http.NoBody), "x")
	rec := httptest.NewRecorder()
	handler(&fakeRepo{}).Delete(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}
