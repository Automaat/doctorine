package testorders

import (
	"context"
	"log/slog"
	"time"

	"github.com/Automaat/doctorine/backend-go/internal/examinations"
)

const completeTimeout = 5 * time.Second

// MatchStore completes orders covered by a created examination.
type MatchStore interface {
	CompleteMatching(ctx context.Context, examinationID int, examTestKeys []string) (int, error)
}

// Completer implements examinations.Notifier: when an examination is created it
// auto-completes every still-requested order whose test_keys the examination
// covers, linking the order to it. Runs synchronously so the order state is
// consistent by the time the create response returns.
type Completer struct {
	store  MatchStore
	logger *slog.Logger
}

func NewCompleter(store MatchStore, logger *slog.Logger) *Completer {
	if logger == nil {
		logger = slog.Default()
	}
	return &Completer{store: store, logger: logger}
}

func (c *Completer) ExaminationCreated(event examinations.ExaminationCreatedEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), completeTimeout)
	defer cancel()
	completed, err := c.store.CompleteMatching(ctx, event.ExaminationID, event.TestKeys)
	if err != nil {
		c.logger.Error("auto-complete test orders", "err", err, "examination_id", event.ExaminationID)
		return
	}
	if completed > 0 {
		c.logger.Info("auto-completed test orders", "count", completed, "examination_id", event.ExaminationID)
	}
}
