package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/Automaat/doctorine/backend-go/internal/httputil"
)

type pinger interface {
	Ping(context.Context) error
}

type healthResponse struct {
	Status   string `json:"status"`
	Database string `json:"database"`
}

func healthHandler(logger *slog.Logger, pool pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "not_configured"
		if pool != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(ctx); err != nil {
				logger.Warn("health db ping", "err", err)
				httputil.WriteJSON(w, http.StatusServiceUnavailable, healthResponse{
					Status:   "degraded",
					Database: "down",
				})
				return
			}
			dbStatus = "up"
		}
		httputil.WriteJSON(w, http.StatusOK, healthResponse{Status: "ok", Database: dbStatus})
	}
}
