package examinations

import (
	"log/slog"
	"net/http"
	"regexp"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

var resultKeyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type Handler struct {
	store  *Store
	logger *slog.Logger
}

func NewHandler(store *Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

type createRequest struct {
	Title        string          `json:"title"`
	ExamDate     string          `json:"exam_date"`
	Category     string          `json:"category"`
	Facility     *string         `json:"facility"`
	ResultStatus string          `json:"result_status"`
	Summary      *string         `json:"summary"`
	Notes        *string         `json:"notes"`
	Results      []resultRequest `json:"results"`
}

type resultRequest struct {
	TestKey      string   `json:"test_key"`
	Name         string   `json:"name"`
	ValueText    *string  `json:"value_text"`
	ValueNumeric *float64 `json:"value_numeric"`
	ValuePrefix  *string  `json:"value_prefix"`
	Unit         *string  `json:"unit"`
	ReferenceMin *float64 `json:"reference_min"`
	ReferenceMax *float64 `json:"reference_max"`
	// Accepted for old clients, recalculated server-side.
	Flag         *string `json:"flag"`
	DisplayOrder int     `json:"display_order"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list examinations", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, items)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	params, detail := validateCreate(req)
	if detail != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, detail)
		return
	}
	item, err := h.store.Create(r.Context(), params)
	if err != nil {
		h.logger.Error("create examination", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, item)
}

func validateCreate(req createRequest) (CreateParams, string) {
	title := healthstatus.CleanString(req.Title)
	if title == "" {
		return CreateParams{}, "Title is required"
	}
	examDate, err := healthstatus.ParseDate(req.ExamDate, "exam_date")
	if err != nil {
		return CreateParams{}, err.Error()
	}
	category := healthstatus.CleanString(req.Category)
	if category == "" {
		category = "general"
	}
	status := healthstatus.CleanString(req.ResultStatus)
	if status == "" {
		status = "unknown"
	}
	if status != "unknown" && status != "normal" && status != "attention" && status != "urgent" {
		return CreateParams{}, "Result status must be unknown, normal, attention, or urgent"
	}
	results, detail := validateResults(req.Results)
	if detail != "" {
		return CreateParams{}, detail
	}
	return CreateParams{
		Title:        title,
		ExamDate:     examDate,
		Category:     category,
		Facility:     cleanOptional(req.Facility),
		ResultStatus: status,
		Summary:      cleanOptional(req.Summary),
		Notes:        cleanOptional(req.Notes),
		Results:      results,
	}, ""
}

func validateResults(req []resultRequest) ([]ResultParams, string) {
	if len(req) > 300 {
		return nil, "Results cannot exceed 300 rows"
	}
	results := make([]ResultParams, 0, len(req))
	seen := map[string]bool{}
	for i, item := range req {
		key := healthstatus.CleanString(item.TestKey)
		if key == "" {
			return nil, "Result test_key is required"
		}
		if len(key) > 120 || !resultKeyPattern.MatchString(key) {
			return nil, "Result test_key must use lowercase letters, numbers, and underscores"
		}
		if seen[key] {
			return nil, "Result test_key must be unique per examination"
		}
		seen[key] = true

		name := healthstatus.CleanString(item.Name)
		if name == "" {
			return nil, "Result name is required"
		}
		valueText := cleanOptional(item.ValueText)
		if valueText == nil && item.ValueNumeric == nil {
			return nil, "Result value_text or value_numeric is required"
		}
		valuePrefix := cleanOptional(item.ValuePrefix)
		if valuePrefix != nil && *valuePrefix != "<" && *valuePrefix != ">" &&
			*valuePrefix != "<=" && *valuePrefix != ">=" {
			return nil, "Result value_prefix must be <, >, <=, or >="
		}
		flag := computeFlag(item.ValueNumeric, valuePrefix, item.ReferenceMin, item.ReferenceMax)
		order := item.DisplayOrder
		if order == 0 {
			order = i + 1
		}
		results = append(results, ResultParams{
			TestKey:      key,
			Name:         name,
			ValueText:    valueText,
			ValueNumeric: item.ValueNumeric,
			ValuePrefix:  valuePrefix,
			Unit:         cleanOptional(item.Unit),
			ReferenceMin: item.ReferenceMin,
			ReferenceMax: item.ReferenceMax,
			Flag:         flag,
			DisplayOrder: order,
		})
	}
	return results, ""
}

func computeFlag(value *float64, prefix *string, referenceMin *float64, referenceMax *float64) *string {
	if value == nil {
		return nil
	}
	if referenceMin != nil && *value < *referenceMin {
		return computedFlag("L")
	}
	if referenceMax != nil && *value > *referenceMax {
		return computedFlag("H")
	}
	if prefix != nil && (*prefix == "<" || *prefix == "<=") &&
		referenceMin != nil && *value <= *referenceMin {
		return computedFlag("L")
	}
	if prefix != nil && (*prefix == ">" || *prefix == ">=") &&
		referenceMax != nil && *value >= *referenceMax {
		return computedFlag("H")
	}
	return nil
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	return healthstatus.NilIfEmpty(*value)
}

func computedFlag(value string) *string {
	return &value
}
