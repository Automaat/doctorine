package weights

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Automaat/doctorine/backend-go/internal/healthstatus"
	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

const maxWeightKg = 1000

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
	MeasuredOn string  `json:"measured_on"`
	WeightKg   float64 `json:"weight_kg"`
	Notes      *string `json:"notes"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list weights", "err", err)
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
		h.logger.Error("create weight", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, item)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, chi.URLParam(r, "id"))
	if !ok {
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrEntryNotFound) {
			httputil.WriteDetailError(w, http.StatusNotFound, "Weight entry not found")
			return
		}
		h.logger.Error("delete weight", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validateCreate(req createRequest) (CreateParams, string) {
	measuredOn, err := healthstatus.ParseDate(req.MeasuredOn, "Date")
	if err != nil {
		return CreateParams{}, err.Error()
	}
	if req.WeightKg <= 0 {
		return CreateParams{}, "Weight must be greater than 0"
	}
	if req.WeightKg >= maxWeightKg {
		return CreateParams{}, "Weight must be less than 1000 kg"
	}
	return CreateParams{
		MeasuredOn: measuredOn,
		WeightKg:   req.WeightKg,
		Notes:      cleanOptional(req.Notes),
	}, ""
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
