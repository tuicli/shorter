package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	TelegramBotToken      string
	AdminUserIDs          []int64
	BaseURL               string
	DatabaseURL           string
	BackendHTTPAddr       string
	MigrationsDir         string
	CodeLength            int
	CodeGenerationRetries int
	AddLinksMaxLines      int
	LinksPageSize         int
	FSMTTL                time.Duration
	CSVExportMaxRows      int
	BotPollTimeout        time.Duration
}

func Load() (Config, error) {
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	adminIDs, err := parseAdminIDs(get(lookup, "ADMIN_USER_IDS", ""))
	if err != nil {
		return Config{}, err
	}

	baseURL, err := normalizeBaseURL(get(lookup, "BASE_URL", ""))
	if err != nil && strings.TrimSpace(get(lookup, "BASE_URL", "")) != "" {
		return Config{}, err
	}

	return Config{
		TelegramBotToken:      strings.TrimSpace(get(lookup, "TELEGRAM_BOT_TOKEN", "")),
		AdminUserIDs:          adminIDs,
		BaseURL:               baseURL,
		DatabaseURL:           strings.TrimSpace(get(lookup, "DATABASE_URL", "")),
		BackendHTTPAddr:       strings.TrimSpace(get(lookup, "BACKEND_HTTP_ADDR", ":8080")),
		MigrationsDir:         strings.TrimSpace(get(lookup, "MIGRATIONS_DIR", "migrations")),
		CodeLength:            getInt(lookup, "CODE_LENGTH", 8),
		CodeGenerationRetries: getInt(lookup, "CODE_GENERATION_RETRIES", 20),
		AddLinksMaxLines:      getInt(lookup, "ADD_LINKS_MAX_LINES", 200),
		LinksPageSize:         getInt(lookup, "LINKS_PAGE_SIZE", 10),
		FSMTTL:                time.Duration(getInt(lookup, "FSM_TTL_MINUTES", 30)) * time.Minute,
		CSVExportMaxRows:      getInt(lookup, "CSV_EXPORT_MAX_ROWS", 10000),
		BotPollTimeout:        time.Duration(getInt(lookup, "BOT_POLL_TIMEOUT_SECONDS", 30)) * time.Second,
	}, nil
}

func (c Config) ValidateBot() error {
	if strings.TrimSpace(c.TelegramBotToken) == "" {
		return errors.New("TELEGRAM_BOT_TOKEN is required")
	}
	if len(c.AdminUserIDs) == 0 {
		return errors.New("ADMIN_USER_IDS must contain at least one id")
	}
	if err := c.validateShared(); err != nil {
		return err
	}
	return nil
}

func (c Config) ValidateBackend() error {
	if c.BackendHTTPAddr == "" {
		return errors.New("BACKEND_HTTP_ADDR is required")
	}
	return c.validateShared()
}

func (c Config) ValidateMigrate() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.MigrationsDir) == "" {
		return errors.New("MIGRATIONS_DIR is required")
	}
	return nil
}

func (c Config) validateShared() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return errors.New("BASE_URL is required")
	}
	if c.CodeLength < 4 || c.CodeLength > 32 {
		return errors.New("CODE_LENGTH must be between 4 and 32")
	}
	if c.CodeGenerationRetries < 1 {
		return errors.New("CODE_GENERATION_RETRIES must be positive")
	}
	if c.AddLinksMaxLines < 1 {
		return errors.New("ADD_LINKS_MAX_LINES must be positive")
	}
	if c.LinksPageSize < 1 {
		return errors.New("LINKS_PAGE_SIZE must be positive")
	}
	if c.CSVExportMaxRows < 1 {
		return errors.New("CSV_EXPORT_MAX_ROWS must be positive")
	}
	return nil
}

func parseAdminIDs(raw string) ([]int64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []int64{}, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err != nil || id <= 0 {
			return nil, fmt.Errorf("ADMIN_USER_IDS contains invalid id %q", part)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func normalizeBaseURL(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("BASE_URL is invalid: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", errors.New("BASE_URL must use http or https")
	}
	if parsed.Host == "" {
		return "", errors.New("BASE_URL must include host")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("BASE_URL must not include query or fragment")
	}

	return raw, nil
}

func get(lookup func(string) (string, bool), key string, fallback string) string {
	value, ok := lookup(key)
	if !ok {
		return fallback
	}
	return value
}

func getInt(lookup func(string) (string, bool), key string, fallback int) int {
	raw := strings.TrimSpace(get(lookup, key, ""))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
