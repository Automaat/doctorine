package examinations

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	// webhookTimeout bounds a single delivery. Kept below the server's shutdown
	// budget so an in-flight delivery can drain before the process exits.
	webhookTimeout = 8 * time.Second
	// maxConcurrentWebhooks caps in-flight deliveries so a create flood against a
	// slow endpoint cannot spawn unbounded goroutines.
	maxConcurrentWebhooks = 16
)

// ExaminationCreatedEvent is the payload delivered to the configured webhook
// after a new examination is saved, letting an external coaching loop react to
// new (and newly out-of-range) markers.
type ExaminationCreatedEvent struct {
	ExaminationID   int      `json:"examination_id"`
	ExamDate        string   `json:"exam_date"`
	TestKeys        []string `json:"test_keys"`
	FlaggedTestKeys []string `json:"flagged_test_keys"`
}

// Notifier is invoked after an examination is created. Implementations must not
// block the request that triggered them.
type Notifier interface {
	ExaminationCreated(event ExaminationCreatedEvent)
}

// HTTPNotifier POSTs the event as JSON to a configured URL, off the request
// path. Delivery is best-effort: failures are logged, never surfaced to the
// user creating the examination.
type HTTPNotifier struct {
	url    string
	client *http.Client
	logger *slog.Logger
	sem    chan struct{}
	wg     sync.WaitGroup
}

// NewHTTPNotifier returns a notifier that POSTs to webhookURL. webhookURL must be
// non-empty; callers skip wiring a notifier when no webhook is configured. A
// malformed URL is logged at startup but still constructed (deliveries then fail
// per-request, logged at Warn).
func NewHTTPNotifier(webhookURL string, logger *slog.Logger) *HTTPNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	if parsed, err := url.ParseRequestURI(webhookURL); err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		logger.Error("DOCTORINE_WEBHOOK_URL must be an http(s) URL", "url", webhookURL, "err", err)
	}
	return &HTTPNotifier{
		url:    webhookURL,
		client: &http.Client{Timeout: webhookTimeout},
		logger: logger,
		sem:    make(chan struct{}, maxConcurrentWebhooks),
	}
}

func (n *HTTPNotifier) ExaminationCreated(event ExaminationCreatedEvent) {
	// Drop (and log) rather than block the caller or grow goroutines without
	// bound when deliveries pile up against a slow endpoint.
	select {
	case n.sem <- struct{}{}:
	default:
		n.logger.Warn("webhook concurrency limit reached, dropping delivery",
			"examination_id", event.ExaminationID)
		return
	}
	n.wg.Go(func() {
		defer func() { <-n.sem }()
		n.deliver(event)
	})
}

// Shutdown blocks until in-flight deliveries finish or ctx is done, so a
// graceful server stop does not silently drop a just-fired webhook.
func (n *HTTPNotifier) Shutdown(ctx context.Context) {
	done := make(chan struct{})
	go func() {
		n.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		n.logger.Warn("shutdown timed out with webhook deliveries still in flight")
	}
}

func (n *HTTPNotifier) deliver(event ExaminationCreatedEvent) {
	// A fresh background context: the originating request is already finished,
	// so its context would be canceled.
	ctx, cancel := context.WithTimeout(context.Background(), webhookTimeout)
	defer cancel()

	body, err := json.Marshal(event)
	if err != nil {
		n.logger.Error("marshal webhook payload", "err", err)
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		n.logger.Error("build webhook request", "err", err)
		return
	}
	req.Header.Set("content-type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		n.logger.Warn("deliver examination webhook", "err", err, "examination_id", event.ExaminationID)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusMultipleChoices {
		n.logger.Warn("examination webhook returned non-2xx",
			"status", resp.StatusCode, "examination_id", event.ExaminationID)
	}
}

// flaggedTestKeys returns the test_keys of results flagged out of range (H/L),
// in their display order.
func flaggedTestKeys(results []Result) []string {
	keys := []string{}
	for _, result := range results {
		if result.Flag != nil {
			keys = append(keys, result.TestKey)
		}
	}
	return keys
}

// allTestKeys returns every result's test_key, in display order.
func allTestKeys(results []Result) []string {
	keys := make([]string, 0, len(results))
	for _, result := range results {
		keys = append(keys, result.TestKey)
	}
	return keys
}

func buildExaminationCreatedEvent(item Examination) ExaminationCreatedEvent {
	return ExaminationCreatedEvent{
		ExaminationID:   item.ID,
		ExamDate:        item.ExamDate,
		TestKeys:        allTestKeys(item.Results),
		FlaggedTestKeys: flaggedTestKeys(item.Results),
	}
}
