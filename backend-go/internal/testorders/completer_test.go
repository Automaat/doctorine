package testorders

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/Automaat/doctorine/backend-go/internal/examinations"
)

type fakeMatchStore struct {
	gotExamID int
	gotKeys   []string
	completed int
	err       error
}

func (f *fakeMatchStore) CompleteMatching(_ context.Context, examID int, keys []string) (int, error) {
	f.gotExamID, f.gotKeys = examID, keys
	return f.completed, f.err
}

func TestCompleterPassesExamKeys(t *testing.T) {
	store := &fakeMatchStore{completed: 1}
	c := NewCompleter(store, slog.Default())
	c.ExaminationCreated(examinations.ExaminationCreatedEvent{
		ExaminationID: 5,
		TestKeys:      []string{"ferrytyna", "tsh"},
	})
	if store.gotExamID != 5 {
		t.Fatalf("exam id = %d, want 5", store.gotExamID)
	}
	if len(store.gotKeys) != 2 || store.gotKeys[0] != "ferrytyna" {
		t.Fatalf("keys = %v, want all examination keys", store.gotKeys)
	}
}

func TestCompleterSwallowsStoreError(t *testing.T) {
	// A failed auto-complete must not panic; it is best-effort and logged.
	c := NewCompleter(&fakeMatchStore{err: errors.New("boom")}, slog.Default())
	c.ExaminationCreated(examinations.ExaminationCreatedEvent{ExaminationID: 1, TestKeys: []string{"tsh"}})
}
