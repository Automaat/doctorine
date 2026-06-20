package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Automaat/doctorine/backend-go/internal/auth"
	"github.com/Automaat/doctorine/backend-go/internal/db"
	"github.com/Automaat/doctorine/backend-go/internal/server"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	os.Exit(run())
}

func healthcheck() int {
	addr := envOr("DOCTORINE_ADDR", ":8000")
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		port = addr
	}
	if host == "" {
		host = "127.0.0.1"
	}
	url := "http://" + net.JoinHostPort(host, port) + "/health"
	client := &http.Client{Timeout: 3 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "healthcheck:", err)
		return 1
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "healthcheck: status", resp.StatusCode)
		return 1
	}
	return 0
}

func run() int {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := server.Config{
		Addr:         envOr("DOCTORINE_ADDR", ":8000"),
		CORSOrigins:  envOr("CORS_ORIGINS", "http://localhost:3001"),
		JWTSecret:    os.Getenv("DOCTORINE_JWT_SECRET"),
		CookieSecure: envOr("DOCTORINE_COOKIE_SECURE", "false") == "true",
		UploadDir:    envOr("DOCTORINE_UPLOAD_DIR", "./data/uploads"),
	}
	if cfg.JWTSecret == "" {
		logger.Error("DOCTORINE_JWT_SECRET is required")
		return 2
	}
	adminUsername := envOr("DOCTORINE_ADMIN_USERNAME", "admin")
	adminPassword := os.Getenv("DOCTORINE_ADMIN_PASSWORD")
	if adminPassword == "" {
		logger.Error("DOCTORINE_ADMIN_PASSWORD is required")
		return 2
	}
	seedDemoData := envOr("DOCTORINE_SEED_DEMO_DATA", "false") == "true"
	if err := os.MkdirAll(cfg.UploadDir, 0o700); err != nil {
		logger.Error("create upload dir", "err", err)
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := server.Deps{}
	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" || os.Getenv("PGHOST") != "" {
		pool, code := initDB(ctx, dsn, logger, adminUsername, adminPassword, seedDemoData)
		if code != 0 {
			return code
		}
		defer pool.Close()
		deps.Pool = pool
	} else {
		logger.Warn("no DB config; DB-backed endpoints disabled")
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           server.New(cfg, logger, deps),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	logger.Info("backend listening", "addr", cfg.Addr)

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("listen", "err", err)
			return 1
		}
	case <-ctx.Done():
		logger.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown", "err", err)
			return 1
		}
	}
	return 0
}

func initDB(
	ctx context.Context,
	dsn string,
	logger *slog.Logger,
	adminUsername string,
	adminPassword string,
	seedDemoData bool,
) (*pgxpool.Pool, int) {
	pool, err := db.New(ctx, dsn)
	if err != nil {
		logger.Error("open db pool", "err", err)
		return nil, 2
	}
	if err := db.Migrate(ctx, pool); err != nil {
		logger.Error("run migrations", "err", err)
		pool.Close()
		return nil, 2
	}
	adminHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		logger.Error("hash admin password", "err", err)
		pool.Close()
		return nil, 2
	}
	if err := auth.NewStore(pool).UpsertAdmin(ctx, adminUsername, adminHash); err != nil {
		logger.Error("seed admin user", "err", err)
		pool.Close()
		return nil, 2
	}
	logger.Info("admin user ready", "username", adminUsername)
	if seedDemoData {
		if err := db.SeedDemoData(ctx, pool); err != nil {
			logger.Error("seed demo data", "err", err)
			pool.Close()
			return nil, 2
		}
		logger.Info("demo data ready")
	}
	return pool, 0
}

func envOr(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
