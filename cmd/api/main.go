// Subscriptions Aggregator API.
//
//	@title       EM Subscriptions API
//	@version     1.0
//	@description REST service for aggregating user online subscriptions.
//	@host        localhost:8080
//	@BasePath    /api/v1
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/sekigo/em-subscriptions/docs"

	"github.com/sekigo/em-subscriptions/internal/config"
	"github.com/sekigo/em-subscriptions/internal/handler"
	mw "github.com/sekigo/em-subscriptions/internal/middleware"
	"github.com/sekigo/em-subscriptions/internal/logger"
	"github.com/sekigo/em-subscriptions/internal/repository"
	"github.com/sekigo/em-subscriptions/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("starting service", "env", cfg.Env, "addr", cfg.HTTP.Host+":"+cfg.HTTP.Port)

	// Run migrations before opening the pool — fail fast if schema is broken.
	log.Info("running migrations", "source", cfg.DB.MigrationsPath)
	if err := repository.RunMigrations(cfg.DB.MigrationsPath, cfg.DB.DSN()); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	log.Info("migrations applied")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := repository.NewPool(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	defer pool.Close()
	log.Info("connected to postgres")

	repo := repository.NewSubscriptionRepo(pool)
	svc := service.New(repo)
	h := handler.New(svc, log)

	r := chi.NewRouter()
	r.Use(mw.RequestID)
	r.Use(mw.Logger(log))
	r.Use(mw.Recoverer(log))

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", h.Register)

	r.Get("/swagger/*", httpSwagger.WrapHandler)

	srv := &http.Server{
		Addr:         cfg.HTTP.Host + ":" + cfg.HTTP.Port,
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
		ErrorLog:     slog.NewLogLogger(log.Handler(), slog.LevelError),
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
		close(serverErr)
	}()

	// Wait for SIGINT/SIGTERM or a server error.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-stop:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("http server: %w", err)
		}
	}

	shutdownStart := time.Now()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	log.Info("server stopped gracefully", "shutdown_ms", time.Since(shutdownStart).Milliseconds())
	return nil
}
