package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/decision"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/operations"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/config"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/middleware"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/proxy"
	"github.com/camellia-oleifera-smart-harvest/api-gateway/internal/platform/store"
	systemhttp "github.com/camellia-oleifera-smart-harvest/api-gateway/internal/system"
)

func main() {
	cfgPath := flag.String("config", "", "path to gateway.yaml (default: $CAMELLIA_GATEWAY_CONFIG or tooling/config/gateway.yaml)")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}

	// Set up structured logger.
	var logLevel slog.Level
	switch cfg.Logging.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var logHandler slog.Handler
	opts := &slog.HandlerOptions{Level: logLevel}
	if cfg.Logging.Format == "text" {
		logHandler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		logHandler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(logHandler)

	rp, err := proxy.New(cfg.Upstream, logger)
	if err != nil {
		logger.Error("failed to create proxy", "error", err)
		os.Exit(1)
	}

	db, err := store.Open(cfg.DB)
	if err != nil {
		logger.Error("failed to open gateway data store", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("failed to close gateway data store", "error", err)
		}
	}()

	decisionRepo := decision.NewRepository(db)
	operationsRepo := operations.NewRepository(db)
	decisionHandler := decision.NewHandler(decisionRepo, logger)
	operationsHandler := operations.NewHandler(operationsRepo, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", systemhttp.Health(cfg.Upstream, logger))
	decisionHandler.Register(mux)
	operationsHandler.Register(mux)
	mux.Handle("/", rp)

	var h http.Handler = mux
	h = middleware.Auth(cfg.Auth, logger)(h)
	h = middleware.RateLimit(cfg.RateLimit, logger)(h)
	h = middleware.CORS(cfg.CORS)(h)
	h = middleware.Logging(logger)(h)
	h = middleware.RequestID(h)

	srv := &http.Server{
		Addr:         cfg.Addr(),
		Handler:      h,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeoutS) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeoutS) * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		logger.Info("gateway listening",
			"addr", cfg.Addr(),
			"upstream", cfg.Upstream.BaseURL,
			"auth", cfg.Auth.Enabled,
			"rate_limit", cfg.RateLimit.Enabled,
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("listen error", "error", err)
			os.Exit(1)
		}
	}()

	<-done
	logger.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", "error", err)
	}
	logger.Info("gateway stopped")
}
