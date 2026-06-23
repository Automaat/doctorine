package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthWithoutDB(t *testing.T) {
	handler, _ := New(Config{}, nil, Deps{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
