package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"backend-go/internal/app"
	"backend-go/internal/config"
	"backend-go/internal/middlewares"
)

func main() {
	cfg := config.Load()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("cannot init logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	application, err := app.NewApp(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("init app failed", zap.Error(err))
	}
	defer application.Close()

	r := chi.NewRouter()
	r.Use(middlewares.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middlewares.Recovery(logger))
	r.Use(middlewares.Logging(logger))
	r.Use(middlewares.Timeout(10 * time.Second))

	application.RegisterRoutes(r)

	addr := cfg.HTTPAddr
	if addr == "" {
		addr = ":8080"
	}

	logger.Info("backend-go listening", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, r); err != nil && err != http.ErrServerClosed {
		logger.Error("http server failed", zap.Error(err))
		os.Exit(1)
	}
}
