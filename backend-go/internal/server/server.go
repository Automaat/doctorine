package server

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/auth"
	"github.com/Automaat/doctorine/backend-go/internal/documents"
	"github.com/Automaat/doctorine/backend-go/internal/examinations"
	"github.com/Automaat/doctorine/backend-go/internal/illnesses"
	"github.com/Automaat/doctorine/backend-go/internal/metrics"
	"github.com/Automaat/doctorine/backend-go/internal/overview"
	"github.com/Automaat/doctorine/backend-go/internal/results"
	"github.com/Automaat/doctorine/backend-go/internal/supplements"
	"github.com/Automaat/doctorine/backend-go/internal/weights"
)

const requestTimeout = 60 * time.Second

type Config struct {
	Addr         string
	CORSOrigins  string
	CookieSecure bool
	UploadDir    string
}

type Deps struct {
	Pool *pgxpool.Pool
}

func New(cfg Config, logger *slog.Logger, deps Deps) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(requestObserver(logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(requestTimeout))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   splitOrigins(cfg.CORSOrigins),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}))

	var healthPool pinger
	if deps.Pool != nil {
		healthPool = deps.Pool
	}
	r.Get("/health", healthHandler(logger, healthPool))
	r.Handle("/metrics", metrics.Handler())

	if deps.Pool != nil {
		registerRoutes(r, cfg, deps.Pool, logger)
	}
	return r
}

func requestObserver(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			status := ww.Status()
			if status == 0 {
				status = http.StatusOK
			}
			route := chi.RouteContext(r.Context()).RoutePattern()
			duration := time.Since(start)
			metrics.ObserveRequest(r.Method, route, status, duration)
			logger.Info("request",
				"request_id", middleware.GetReqID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"route", route,
				"status", status,
				"bytes", ww.BytesWritten(),
				"latency_ms", float64(duration.Microseconds())/1000.0,
			)
		})
	}
}

func registerRoutes(r chi.Router, cfg Config, pool *pgxpool.Pool, logger *slog.Logger) {
	store := auth.NewStore(pool)
	authHandler := auth.NewHandler(store, cfg.CookieSecure, logger)

	r.Post("/api/auth/login", authHandler.Login)
	r.Post("/api/auth/logout", authHandler.Logout)

	r.Group(func(r chi.Router) {
		r.Use(auth.Authenticate(store))
		r.Get("/api/auth/me", authHandler.Me)
		r.Get("/api/tokens", authHandler.ListTokens)
		r.Post("/api/tokens", authHandler.CreateToken)
		r.Delete("/api/tokens/{id}", authHandler.RevokeToken)

		documentStore := documents.NewStore(pool)
		documentsHandler := documents.NewHandler(documentStore, cfg.UploadDir, logger)
		illnessHandler := illnesses.NewHandler(illnesses.NewStore(pool), logger)
		examinationHandler := examinations.NewHandler(examinations.NewStore(pool), logger)
		supplementHandler := supplements.NewHandler(supplements.NewStore(pool), logger)
		weightHandler := weights.NewHandler(weights.NewStore(pool), logger)
		resultHandler := results.NewHandler(results.NewStore(pool), logger)
		overviewHandler := overview.NewHandler(pool, documentStore, logger)

		r.Get("/api/overview", overviewHandler.Get)
		r.Get("/api/illnesses", illnessHandler.List)
		r.Post("/api/illnesses", illnessHandler.Create)
		r.Get("/api/supplements", supplementHandler.List)
		r.Post("/api/supplements", supplementHandler.Create)
		r.Get("/api/weights", weightHandler.List)
		r.Post("/api/weights", weightHandler.Create)
		r.Delete("/api/weights/{id}", weightHandler.Delete)
		r.Get("/api/results/latest", resultHandler.Latest)
		r.Get("/api/results/trend/{test_key}", resultHandler.Trend)
		r.Get("/api/examinations", examinationHandler.List)
		r.Get("/api/examinations/{id}", examinationHandler.Get)
		r.Post("/api/examinations", examinationHandler.Create)
		r.Delete("/api/examinations/{id}", examinationHandler.Delete)
		r.Get("/api/documents", documentsHandler.List)
		r.Post("/api/documents", documentsHandler.Upload)
		r.Get("/api/documents/{id}/download", documentsHandler.Download)
		r.Delete("/api/documents/{id}", documentsHandler.Delete)
	})
}

func splitOrigins(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned != "" {
			out = append(out, cleaned)
		}
	}
	return out
}
