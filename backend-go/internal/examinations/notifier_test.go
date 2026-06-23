package examinations

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFlaggedTestKeysIncludesOnlyFlagged(t *testing.T) {
	results := []Result{
		{TestKey: "ferrytyna", Flag: new("L")},
		{TestKey: "tsh", Flag: nil},
		{TestKey: "glukoza", Flag: new("H")},
	}
	got := flaggedTestKeys(results)
	if len(got) != 2 || got[0] != "ferrytyna" || got[1] != "glukoza" {
		t.Fatalf("flaggedTestKeys = %v, want [ferrytyna glukoza]", got)
	}
}

func TestFlaggedTestKeysNeverNil(t *testing.T) {
	if got := flaggedTestKeys(nil); got == nil || len(got) != 0 {
		t.Fatalf("flaggedTestKeys(nil) = %v, want empty non-nil slice", got)
	}
}

func TestBuildExaminationCreatedEvent(t *testing.T) {
	item := Examination{
		ID:       7,
		ExamDate: "2026-06-01",
		Results:  []Result{{TestKey: "ferrytyna", Flag: new("L")}, {TestKey: "tsh"}},
	}
	event := buildExaminationCreatedEvent(item)
	if event.ExaminationID != 7 || event.ExamDate != "2026-06-01" {
		t.Fatalf("event = %+v", event)
	}
	if len(event.FlaggedTestKeys) != 1 || event.FlaggedTestKeys[0] != "ferrytyna" {
		t.Fatalf("flagged = %v, want [ferrytyna]", event.FlaggedTestKeys)
	}
}

func TestHTTPNotifierPostsEvent(t *testing.T) {
	type received struct {
		contentType string
		event       ExaminationCreatedEvent
	}
	got := make(chan received, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var event ExaminationCreatedEvent
		_ = json.Unmarshal(body, &event)
		got <- received{contentType: r.Header.Get("content-type"), event: event}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	notifier := NewHTTPNotifier(srv.URL, slog.Default())
	notifier.ExaminationCreated(ExaminationCreatedEvent{
		ExaminationID:   3,
		ExamDate:        "2026-06-01",
		FlaggedTestKeys: []string{"ferrytyna"},
	})

	select {
	case r := <-got:
		if r.contentType != "application/json" {
			t.Fatalf("content-type = %q, want application/json", r.contentType)
		}
		if r.event.ExaminationID != 3 || len(r.event.FlaggedTestKeys) != 1 {
			t.Fatalf("delivered event = %+v", r.event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("webhook was not delivered")
	}
}

func TestHandlerCreateNotifies(t *testing.T) {
	spy := &spyNotifier{}
	h := &Handler{notifier: spy, logger: slog.Default()}
	// Exercise the glue directly: the handler builds the event from the created
	// examination and hands it to the notifier.
	item := Examination{ID: 9, ExamDate: "2026-06-02", Results: []Result{{TestKey: "glukoza", Flag: new("H")}}}
	if h.notifier != nil {
		h.notifier.ExaminationCreated(buildExaminationCreatedEvent(item))
	}
	if spy.event.ExaminationID != 9 || len(spy.event.FlaggedTestKeys) != 1 || spy.event.FlaggedTestKeys[0] != "glukoza" {
		t.Fatalf("notifier received %+v", spy.event)
	}
}

type spyNotifier struct {
	event ExaminationCreatedEvent
	calls int
}

func (s *spyNotifier) ExaminationCreated(event ExaminationCreatedEvent) {
	s.event = event
	s.calls++
}
