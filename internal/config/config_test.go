package config

import "testing"

func TestLoadAndValidateBot(t *testing.T) {
	values := map[string]string{
		"TELEGRAM_BOT_TOKEN":       "token",
		"ADMIN_USER_IDS":           "123,456",
		"BASE_URL":                 "https://s.example/",
		"DATABASE_URL":             "postgres://shorter:pass@localhost/shorter?sslmode=disable",
		"BACKEND_HTTP_ADDR":        ":9090",
		"CODE_LENGTH":              "10",
		"CODE_GENERATION_RETRIES":  "7",
		"ADD_LINKS_MAX_LINES":      "50",
		"LINKS_PAGE_SIZE":          "10",
		"CSV_EXPORT_MAX_ROWS":      "100",
		"BOT_POLL_TIMEOUT_SECONDS": "15",
	}

	cfg, err := load(func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	})
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if err := cfg.ValidateBot(); err != nil {
		t.Fatalf("ValidateBot returned error: %v", err)
	}
	if cfg.BaseURL != "https://s.example" {
		t.Fatalf("BaseURL = %q", cfg.BaseURL)
	}
	if cfg.CodeLength != 10 {
		t.Fatalf("CodeLength = %d", cfg.CodeLength)
	}
}

func TestValidateBotRequiresAdmins(t *testing.T) {
	cfg := Config{
		TelegramBotToken:      "token",
		BaseURL:               "https://s.example",
		DatabaseURL:           "postgres://shorter:pass@localhost/shorter?sslmode=disable",
		CodeLength:            8,
		CodeGenerationRetries: 20,
		AddLinksMaxLines:      200,
		LinksPageSize:         10,
		CSVExportMaxRows:      10000,
	}

	if err := cfg.ValidateBot(); err == nil {
		t.Fatal("ValidateBot returned nil, want error")
	}
}
