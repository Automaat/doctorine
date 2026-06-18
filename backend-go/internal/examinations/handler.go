package examinations

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
	Title        string  `json:"title"`
	ExamDate     string  `json:"exam_date"`
	Category     string  `json:"category"`
	Facility     *string `json:"facility"`
	ResultStatus string  `json:"result_status"`
	Summary      *string `json:"summary"`
	Notes        *string `json:"notes"`
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
	return CreateParams{
		Title:        title,
		ExamDate:     examDate,
		Category:     category,
		Facility:     cleanOptional(req.Facility),
		ResultStatus: status,
		Summary:      cleanOptional(req.Summary),
		Notes:        cleanOptional(req.Notes),
	}, ""
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	return healthstatus.NilIfEmpty(*value)
}
