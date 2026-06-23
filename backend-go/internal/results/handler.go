package results

import (
	"context"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

var testKeyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

const (
	maxTestKeys    = 100
	defaultDays    = 365
	maxTrendDays   = 36500
	maxTestKeySize = 120
)

// Reader reads results for the coaching endpoints.
type Reader interface {
	LatestByTestKeys(ctx context.Context, keys []string) ([]LatestResult, error)
	TrendByTestKey(ctx context.Context, testKey string, days int) ([]TrendPoint, error)
}

type Handler struct {
	store  Reader
	logger *slog.Logger
}

func NewHandler(store Reader, logger *slog.Logger) *Handler {
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

// Trend handles GET /api/results/trend/{test_key}?days=365, returning the dated
// numeric series for a single marker over the window (default 365 days).
func (h *Handler) Trend(w http.ResponseWriter, r *http.Request) {
	testKey := strings.TrimSpace(chi.URLParam(r, "test_key"))
	if testKey == "" || len(testKey) > maxTestKeySize || !testKeyPattern.MatchString(testKey) {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid test_key")
		return
	}
	days, detail := parseDays(r.URL.Query().Get("days"))
	if detail != "" {
		httputil.WriteDetailError(w, http.StatusBadRequest, detail)
		return
	}
	points, err := h.store.TrendByTestKey(r.Context(), testKey, days)
	if err != nil {
		h.logger.Error("result trend", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, points)
}

// parseDays validates the optional days window. A blank value defaults to 365.
func parseDays(raw string) (int, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultDays, ""
	}
	days, err := strconv.Atoi(raw)
	if err != nil || days < 1 {
		return 0, "days must be a positive integer"
	}
	if days > maxTrendDays {
		return 0, "days cannot exceed 36500"
	}
	return days, ""
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
