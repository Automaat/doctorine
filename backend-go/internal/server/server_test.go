package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthWithoutDB(t *testing.T) {
	handler := New(Config{}, nil, Deps{})
	req := httptest.NewRequest(http.MethodGet, "/health", http.NoBody)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
