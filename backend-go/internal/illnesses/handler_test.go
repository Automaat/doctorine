package illnesses

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateCreate(t *testing.T) {
	t.Run("requires a title", func(t *testing.T) {
		_, detail := validateCreate(createRequest{Title: "   "})
		if detail != "Title is required" {
			t.Fatalf("detail = %q", detail)
		}
	})

	t.Run("rejects an overlong title", func(t *testing.T) {
		_, detail := validateCreate(createRequest{Title: strings.Repeat("a", 201)})
		if detail != "Title must be 200 characters or fewer" {
			t.Fatalf("detail = %q", detail)
		}
	})

	t.Run("defaults status to active", func(t *testing.T) {
		params, detail := validateCreate(createRequest{Title: "Flu"})
		if detail != "" {
			t.Fatalf("unexpected detail %q", detail)
		}
		if params.Status != "active" {
			t.Fatalf("status = %q, want active", params.Status)
		}
	})

	t.Run("rejects an unknown status", func(t *testing.T) {
		_, detail := validateCreate(createRequest{Title: "Flu", Status: "chronic"})
		if detail != "Status must be active, monitoring, or resolved" {
			t.Fatalf("detail = %q", detail)
		}
	})

	t.Run("reports an invalid date", func(t *testing.T) {
		bad := "not-a-date"
		_, detail := validateCreate(createRequest{Title: "Flu", DiagnosedOn: &bad})
		if detail == "" {
			t.Fatal("expected a date validation error")
		}
	})

	t.Run("accepts a full record", func(t *testing.T) {
		diagnosed := "2025-01-02"
		clinician := "Dr. Smith"
		params, detail := validateCreate(createRequest{
			Title:       "Asthma",
			Status:      "monitoring",
			DiagnosedOn: &diagnosed,
			Clinician:   &clinician,
		})
		if detail != "" {
			t.Fatalf("unexpected detail %q", detail)
		}
		if params.Title != "Asthma" || params.Status != "monitoring" {
			t.Fatalf("params = %+v", params)
		}
		if params.DiagnosedOn == nil || params.Clinician == nil || *params.Clinician != "Dr. Smith" {
			t.Fatalf("optional fields not mapped: %+v", params)
		}
	})
}

type fakeRepository struct {
	items     []Illness
	created   Illness
	listErr   error
	createErr error
}

func (f *fakeRepository) List(context.Context) ([]Illness, error) {
	return f.items, f.listErr
}

func (f *fakeRepository) Create(_ context.Context, _ CreateParams) (Illness, error) {
	return f.created, f.createErr
}

func handlerWith(repo repository) *Handler {
	return &Handler{store: repo, logger: slog.Default()}
}

func newPostRequest(t *testing.T, body string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(
		context.Background(),
		http.MethodPost,
		"/api/illnesses",
		strings.NewReader(body),
	)
	req.Header.Set("content-type", "application/json")
	return req
}

func TestHandlerList(t *testing.T) {
	t.Run("returns items as JSON", func(t *testing.T) {
		h := handlerWith(&fakeRepository{items: []Illness{{ID: 1, Title: "Flu", Status: "active"}}})
		rec := httptest.NewRecorder()
		h.List(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/illnesses", http.NoBody))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		var items []Illness
		if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != 1 || items[0].Title != "Flu" {
			t.Fatalf("items = %+v", items)
		}
	})

	t.Run("returns 500 on a store error", func(t *testing.T) {
		h := handlerWith(&fakeRepository{listErr: errors.New("db down")})
		rec := httptest.NewRecorder()
		h.List(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/illnesses", http.NoBody))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})
}

func TestHandlerCreate(t *testing.T) {
	t.Run("rejects invalid JSON", func(t *testing.T) {
		h := handlerWith(&fakeRepository{})
		rec := httptest.NewRecorder()
		h.Create(rec, newPostRequest(t, "{"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("rejects a validation failure", func(t *testing.T) {
		h := handlerWith(&fakeRepository{})
		rec := httptest.NewRecorder()
		h.Create(rec, newPostRequest(t, `{"title":""}`))
		if rec.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status = %d, want 422", rec.Code)
		}
	})

	t.Run("returns 500 on a store error", func(t *testing.T) {
		h := handlerWith(&fakeRepository{createErr: errors.New("db down")})
		rec := httptest.NewRecorder()
		h.Create(rec, newPostRequest(t, `{"title":"Flu"}`))
		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	})

	t.Run("creates and returns 201", func(t *testing.T) {
		h := handlerWith(&fakeRepository{created: Illness{ID: 9, Title: "Flu", Status: "active"}})
		rec := httptest.NewRecorder()
		h.Create(rec, newPostRequest(t, `{"title":"Flu"}`))
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201", rec.Code)
		}
		var item Illness
		if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
			t.Fatal(err)
		}
		if item.ID != 9 {
			t.Fatalf("item = %+v", item)
		}
	})
}
