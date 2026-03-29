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

	"go.uber.org/zap"

	"github.com/mechatracker/api/internal/config"
	"github.com/mechatracker/api/internal/router"
)

func main() {
	// ── Logger ────────────────────────────────────────────────────────────────
	// Development gets a human-friendly console logger; production gets JSON.
	// The logger is passed down rather than used as a global.
	log, err := buildLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync() //nolint:errcheck

	// ── Config ────────────────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("failed to load config", zap.Error(err))
	}

	log.Info("starting mechatracker api",
		zap.String("env", cfg.Env),
		zap.Int("port", cfg.Port),
	)

	// ── Router ────────────────────────────────────────────────────────────────
	r := router.New(cfg, log)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
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
