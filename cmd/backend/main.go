package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tuicli/shorter/internal/app"
	"github.com/tuicli/shorter/internal/config"
	"github.com/tuicli/shorter/internal/domain"
	"github.com/tuicli/shorter/internal/httpserver"
	"github.com/tuicli/shorter/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("backend stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.ValidateBackend(); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	repo := postgres.NewRepository(db)
	service := app.NewService(repo, domain.RandomCodeGenerator{Length: cfg.CodeLength}, app.Options{
		BaseURL:               cfg.BaseURL,
		AddLinksMaxLines:      cfg.AddLinksMaxLines,
		LinksPageSize:         cfg.LinksPageSize,
		CodeGenerationRetries: cfg.CodeGenerationRetries,
		CSVExportMaxRows:      cfg.CSVExportMaxRows,
	})
	server := &http.Server{
		Addr:              cfg.BackendHTTPAddr,
		Handler:           httpserver.New(service, logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("backend listening", "addr", cfg.BackendHTTPAddr)
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
