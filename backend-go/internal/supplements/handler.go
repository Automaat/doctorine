package supplements

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
	Name      string  `json:"name"`
	Value     string  `json:"value"`
	Frequency string  `json:"frequency"`
	Notes     *string `json:"notes"`
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.store.List(r.Context())
	if err != nil {
		h.logger.Error("list supplements", "err", err)
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
		h.logger.Error("create supplement", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, item)
}

func validateCreate(req createRequest) (CreateParams, string) {
	name := healthstatus.CleanString(req.Name)
	if name == "" {
		return CreateParams{}, "Name is required"
	}
	if len(name) > 200 {
		return CreateParams{}, "Name must be 200 characters or fewer"
	}
	value := healthstatus.CleanString(req.Value)
	if value == "" {
		return CreateParams{}, "Value is required"
	}
	if len(value) > 120 {
		return CreateParams{}, "Value must be 120 characters or fewer"
	}
	frequency := healthstatus.CleanString(req.Frequency)
	if frequency == "" {
		return CreateParams{}, "Frequency is required"
	}
	if len(frequency) > 120 {
		return CreateParams{}, "Frequency must be 120 characters or fewer"
	}
	return CreateParams{
		Name:      name,
		Value:     value,
		Frequency: frequency,
		Notes:     cleanOptional(req.Notes),
	}, ""
}

func cleanOptional(value *string) *string {
	if value == nil {
		return nil
	}
	return healthstatus.NilIfEmpty(*value)
}
