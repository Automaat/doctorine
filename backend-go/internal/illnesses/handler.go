package illnesses

import (
	"log/slog"
	"net/http"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

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
	Title       string  `json:"title"`
	Status      string  `json:"status"`
	DiagnosedOn *string `json:"diagnosed_on"`
	ResolvedOn  *string `json:"resolved_on"`
	Clinician   *string `json:"clinician"`
	Notes       *string `json:"notes"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list illnesses", "err", err)
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
		h.logger.Error("create illness", "err", err)
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
	if len(title) > 200 {
		return CreateParams{}, "Title must be 200 characters or fewer"
	}
	status := healthstatus.CleanString(req.Status)
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "monitoring" && status != "resolved" {
		return CreateParams{}, "Status must be active, monitoring, or resolved"
	}
	diagnosedOn, err := healthstatus.ParseOptionalDate(req.DiagnosedOn, "diagnosed_on")
	if err != nil {
		return CreateParams{}, err.Error()
	}
	resolvedOn, err := healthstatus.ParseOptionalDate(req.ResolvedOn, "resolved_on")
	if err != nil {
		return CreateParams{}, err.Error()
	}
	return CreateParams{
		Title:       title,
		Status:      status,
		DiagnosedOn: diagnosedOn,
		ResolvedOn:  resolvedOn,
		Clinician:   cleanOptional(req.Clinician),
		Notes:       cleanOptional(req.Notes),
	}, ""
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	return healthstatus.NilIfEmpty(*value)
}
