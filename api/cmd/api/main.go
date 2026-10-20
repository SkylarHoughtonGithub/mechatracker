// Package main is the entry point for the mechatracker API server.
//
// @title           mechatracker API
// @version         0.1.0
// @description     Repair tracking platform for mechanics, auto shops, and fleet managers.
//
// @contact.name    mechatracker support
//
// @host            localhost:8080
// @BasePath        /v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and a Google OAuth ID token. Browser clients should rely on the session cookie minted by POST /auth/google instead.
//
// @securityDefinitions.apikey CookieAuth
// @in cookie
// @name mt_session
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/router"
	"github.com/mechatracker/api/internal/store"
	"github.com/mechatracker/api/internal/tracing"

	_ "github.com/mechatracker/api/openapi"
)

// version is set at build time via -ldflags "-X main.version=<sha>".
var version = "dev"

func main() {
	log, err := buildLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	log.Info("starting mechatracker api",
		zap.String("env", cfg.Env),
		zap.String("version", version),
		zap.Int("port", cfg.Port),
	)

	// ── Tracing ───────────────────────────────────────────────
	// Non-fatal: a missing or unreachable OTel collector drops traces
	// but keeps the API serving.
	shutdownTracing := tracing.Init(context.Background(), cfg.Env, version, log)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTracing(ctx)
	}()

	// ── Database ──────────────────────────────────────────────
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatal("failed to open db pool", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal("db ping failed", zap.Error(err))
	}
	log.Info("database connected")

	// ── sqlc Querier ──────────────────────────────────────────
	q := store.New(pool)

	// ── Router ────────────────────────────────────────────────
	r := router.New(cfg, log, q)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info("listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown error", zap.Error(err))
	}
	log.Info("stopped")
}

func buildLogger() (*zap.Logger, error) {
	if os.Getenv("ENV") == "production" {
		return zap.NewProduction()
	}
	return zap.NewDevelopment()
}
