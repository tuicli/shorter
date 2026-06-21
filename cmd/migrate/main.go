package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tuicli/shorter/internal/config"
	"github.com/tuicli/shorter/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger, os.Args[1:]); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.ValidateMigrate(); err != nil {
		return err
	}

	direction := "up"
	if len(args) > 0 {
		direction = args[0]
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	logger.Info("running migrations", "direction", direction, "dir", cfg.MigrationsDir)
	if err := postgres.Migrate(ctx, db, cfg.MigrationsDir, direction); err != nil {
		return err
	}
	logger.Info("migrations finished", "direction", direction)
	return nil
}
