package examinations

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const webhookTimeout = 10 * time.Second

// ExaminationCreatedEvent is the payload delivered to the configured webhook
// after a new examination is saved, letting an external coaching loop react to
// new (and newly out-of-range) markers.
type ExaminationCreatedEvent struct {
	ExaminationID   int      `json:"examination_id"`
	ExamDate        string   `json:"exam_date"`
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
}

// NewHTTPNotifier returns a notifier that POSTs to url. url must be non-empty;
// callers skip wiring a notifier when no webhook is configured.
func NewHTTPNotifier(url string, logger *slog.Logger) *HTTPNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &HTTPNotifier{
		url:    url,
		client: &http.Client{Timeout: webhookTimeout},
		logger: logger,
	}
}

func (n *HTTPNotifier) ExaminationCreated(event ExaminationCreatedEvent) {
	go n.deliver(event)
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

func buildExaminationCreatedEvent(item Examination) ExaminationCreatedEvent {
	return ExaminationCreatedEvent{
		ExaminationID:   item.ID,
		ExamDate:        item.ExamDate,
		FlaggedTestKeys: flaggedTestKeys(item.Results),
	}
}
