package results

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

var testKeyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

const maxTestKeys = 100

// LatestStore reads the most recent result per test_key.
type LatestStore interface {
	LatestByTestKeys(ctx context.Context, keys []string) ([]LatestResult, error)
}

type Handler struct {
	store  LatestStore
	logger *slog.Logger
}

func NewHandler(store LatestStore, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

// Latest handles GET /api/results/latest?test_keys=a,b,c. Omitting test_keys
// returns the latest result for every known key.
func (h *Handler) Latest(w http.ResponseWriter, r *http.Request) {
	keys, detail := parseTestKeys(r.URL.Query().Get("test_keys"))
	if detail != "" {
		httputil.WriteDetailError(w, http.StatusBadRequest, detail)
		return
	}
	items, err := h.store.LatestByTestKeys(r.Context(), keys)
	if err != nil {
		h.logger.Error("latest results", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, items)
}

// parseTestKeys splits and validates a comma-separated test_keys parameter,
// dropping blanks and de-duplicating. An empty parameter yields an empty slice
// (meaning "all keys").
func parseTestKeys(raw string) ([]string, string) {
	if strings.TrimSpace(raw) == "" {
		return []string{}, ""
	}
	seen := map[string]bool{}
	keys := []string{}
	for part := range strings.SplitSeq(raw, ",") {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		if len(key) > 120 || !testKeyPattern.MatchString(key) {
			return nil, "test_keys must use lowercase letters, numbers, and underscores"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
		if len(keys) > maxTestKeys {
			return nil, "test_keys cannot exceed 100 entries"
		}
	}
	return keys, ""
}
