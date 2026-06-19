package overview

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/documents"
	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

type Response struct {
	DocumentCount    int                  `json:"document_count"`
	IllnessCount     int                  `json:"illness_count"`
	ExaminationCount int                  `json:"examination_count"`
	FlaggedResults   int                  `json:"flagged_results"`
	RecentDocuments  []documents.Document `json:"recent_documents"`
}

type Handler struct {
	pool      *pgxpool.Pool
	documents *documents.Store
	logger    *slog.Logger
}

func NewHandler(pool *pgxpool.Pool, documents *documents.Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{pool: pool, documents: documents, logger: logger}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	response, err := h.load(r.Context())
	if err != nil {
		h.logger.Error("load overview", "err", err)
		httputil.WriteDetailError(w, http.StatusInternalServerError, "Internal Server Error")
		return
	}
	httputil.WriteJSON(w, http.StatusOK, response)
}

func (h *Handler) load(ctx context.Context) (Response, error) {
	var response Response
	err := h.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM documents),
			(SELECT count(*) FROM illnesses WHERE status <> 'resolved'),
			(SELECT count(*) FROM examinations),
			(SELECT count(*) FROM examination_results WHERE flag IS NOT NULL)
	`).Scan(
		&response.DocumentCount,
		&response.IllnessCount,
		&response.ExaminationCount,
		&response.FlaggedResults,
	)
	if err != nil {
		return Response{}, err
	}
	recent, err := h.documents.Recent(ctx, 5)
	if err != nil {
		return Response{}, err
	}
	response.RecentDocuments = recent
	return response, nil
}
