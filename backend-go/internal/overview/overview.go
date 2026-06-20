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

// Counts holds the dashboard summary totals.
type Counts struct {
	Documents    int
	Illnesses    int
	Examinations int
	Flagged      int
}

// source supplies the data the handler renders, so handler behavior can be
// tested without a database.
type source interface {
	Counts(ctx context.Context) (Counts, error)
	RecentDocuments(ctx context.Context, limit int) ([]documents.Document, error)
}

type pgxSource struct {
	pool      *pgxpool.Pool
	documents *documents.Store
}

func (s pgxSource) Counts(ctx context.Context) (Counts, error) {
	var counts Counts
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM documents),
			(SELECT count(*) FROM illnesses WHERE status <> 'resolved'),
			(SELECT count(*) FROM examinations),
			(SELECT count(*) FROM examination_results WHERE flag IS NOT NULL)
	`).Scan(&counts.Documents, &counts.Illnesses, &counts.Examinations, &counts.Flagged)
	if err != nil {
		return Counts{}, err
	}
	return counts, nil
}

func (s pgxSource) RecentDocuments(ctx context.Context, limit int) ([]documents.Document, error) {
	return s.documents.Recent(ctx, limit)
}

type Handler struct {
	src    source
	logger *slog.Logger
}

func NewHandler(pool *pgxpool.Pool, docs *documents.Store, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	return &Handler{src: pgxSource{pool: pool, documents: docs}, logger: logger}
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
	counts, err := h.src.Counts(ctx)
	if err != nil {
		return Response{}, err
	}
	recent, err := h.src.RecentDocuments(ctx, 5)
	if err != nil {
		return Response{}, err
	}
	return Response{
		DocumentCount:    counts.Documents,
		IllnessCount:     counts.Illnesses,
		ExaminationCount: counts.Examinations,
		FlaggedResults:   counts.Flagged,
		RecentDocuments:  recent,
	}, nil
}
