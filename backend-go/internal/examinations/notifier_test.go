package examinations

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
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

func TestHTTPNotifierBoundsConcurrencyAndDrains(t *testing.T) {
	release := make(chan struct{})
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		<-release // block until the test lets deliveries finish
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	notifier := NewHTTPNotifier(srv.URL, slog.Default())
	// Fire more than the concurrency cap; the excess is dropped, not queued.
	for i := range maxConcurrentWebhooks + 5 {
		notifier.ExaminationCreated(ExaminationCreatedEvent{ExaminationID: i})
	}

	// Shutdown must wait for in-flight deliveries: with the server still
	// blocking, the drain only returns once its context is canceled.
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	notifier.Shutdown(ctx)
	if time.Since(start) < 100*time.Millisecond {
		t.Fatal("Shutdown returned before draining in-flight deliveries")
	}
	close(release)
	if got := hits.Load(); got > maxConcurrentWebhooks {
		t.Fatalf("delivered %d concurrently, want <= %d", got, maxConcurrentWebhooks)
	}
}

type spyNotifier struct {
	mu    sync.Mutex
	event ExaminationCreatedEvent
	calls int
}

func (s *spyNotifier) ExaminationCreated(event ExaminationCreatedEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.event = event
	s.calls++
}
