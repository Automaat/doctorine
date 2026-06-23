package testorders

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

var testKeyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

const maxTestKeys = 100

// Repository persists test orders.
type Repository interface {
	Create(ctx context.Context, params CreateParams) (Order, error)
	List(ctx context.Context, status string) ([]Order, error)
	Update(ctx context.Context, id int, params UpdateParams) (Order, error)
	Cancel(ctx context.Context, id int) error
}

type Handler struct {
	store  Repository
	logger *slog.Logger
}

func NewHandler(store Repository, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{store: store, logger: logger}
}

type createRequest struct {
	Source   string   `json:"source"`
	TestKeys []string `json:"test_keys"`
	Reason   *string  `json:"reason"`
	DueOn    *string  `json:"due_on"`
	Notes    *string  `json:"notes"`
}

type updateRequest struct {
	Status        *string `json:"status"`
	ExaminationID *int    `json:"examination_id"`
	Notes         *string `json:"notes"`
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
	order, err := h.store.Create(r.Context(), params)
	if err != nil {
		h.logger.Error("create test order", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, order)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	status := healthstatus.CleanString(r.URL.Query().Get("status"))
	if status != "" && !validStatus(status) {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid status")
		return
	}
	orders, err := h.store.List(r.Context(), status)
	if err != nil {
		h.logger.Error("list test orders", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, orders)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	var req updateRequest
	if err := httputil.DecodeJSON(r, &req); err != nil {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	params, detail := validateUpdate(req)
	if detail != "" {
		httputil.WriteDetailError(w, http.StatusUnprocessableEntity, detail)
		return
	}
	order, err := h.store.Update(r.Context(), id, params)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Test order not found")
			return
		}
		h.logger.Error("update test order", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, order)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.store.Cancel(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Test order not found")
			return
		}
		h.logger.Error("cancel test order", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateCreate(req createRequest) (CreateParams, string) {
	keys, detail := validateTestKeys(req.TestKeys)
	if detail != "" {
		return CreateParams{}, detail
	}
	source := healthstatus.CleanString(req.Source)
	if source == "" {
		source = "coach"
	}
	if len(source) > 40 {
		return CreateParams{}, "Source is too long"
	}
	var dueOn *time.Time
	if parsed, ok, err := healthstatus.ParseOptionalDate(req.DueOn, "due_on"); err != nil {
		return CreateParams{}, err.Error()
	} else if ok {
		dueOn = &parsed
	}
	return CreateParams{
		Source:   source,
		TestKeys: keys,
		Reason:   cleanOptional(req.Reason),
		DueOn:    dueOn,
		Notes:    cleanOptional(req.Notes),
	}, ""
}

func validateUpdate(req updateRequest) (UpdateParams, string) {
	var params UpdateParams
	if req.Status != nil {
		status := healthstatus.CleanString(*req.Status)
		if !validStatus(status) {
			return UpdateParams{}, "Status must be requested, completed, or canceled"
		}
		params.Status = &status
	}
	if req.ExaminationID != nil {
		if *req.ExaminationID <= 0 {
			return UpdateParams{}, "examination_id must be positive"
		}
		params.ExaminationID = req.ExaminationID
	}
	params.Notes = cleanOptional(req.Notes)
	if params.Status == nil && params.ExaminationID == nil && params.Notes == nil {
		return UpdateParams{}, "No fields to update"
	}
	return params, ""
}

func validateTestKeys(raw []string) ([]string, string) {
	if len(raw) == 0 {
		return nil, "test_keys is required"
	}
	if len(raw) > maxTestKeys {
		return nil, "test_keys cannot exceed 100 entries"
	}
	seen := map[string]bool{}
	keys := make([]string, 0, len(raw))
	for _, item := range raw {
		key := healthstatus.CleanString(item)
		if key == "" {
			return nil, "test_keys must not contain blanks"
		}
		if len(key) > 120 || !testKeyPattern.MatchString(key) {
			return nil, "test_keys must use lowercase letters, numbers, and underscores"
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		keys = append(keys, key)
	}
	return keys, ""
}

func validStatus(status string) bool {
	return status == StatusRequested || status == StatusCompleted || status == StatusCancelled
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	return healthstatus.NilIfEmpty(*value)
}

func parseID(w http.ResponseWriter, raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		httputil.WriteDetailError(w, http.StatusBadRequest, "Invalid id")
		return 0, false
	}
	return id, true
}
