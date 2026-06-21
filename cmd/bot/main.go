package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tuicli/shorter/internal/app"
	"github.com/tuicli/shorter/internal/bot"
	"github.com/tuicli/shorter/internal/config"
	"github.com/tuicli/shorter/internal/domain"
	"github.com/tuicli/shorter/internal/storage/postgres"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("bot stopped with error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.ValidateBot(); err != nil {
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
	adminBot, err := bot.New(cfg.TelegramBotToken, cfg.AdminUserIDs, service, logger, bot.Options{
		PollTimeout: cfg.BotPollTimeout,
		FSMTTL:      cfg.FSMTTL,
	})
	if err != nil {
		return err
	}

	return adminBot.Start(ctx)
}
