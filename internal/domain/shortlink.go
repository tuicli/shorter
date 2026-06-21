package domain

import (
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type LinkStatus string

const (
	StatusActive   LinkStatus = "active"
	StatusDisabled LinkStatus = "disabled"
	StatusDeleted  LinkStatus = "deleted"
)

type ShortLink struct {
	ID                   int64
	Code                 string
	OriginalURL          string
	Title                string
	Status               LinkStatus
	CreatedByTelegramID  int64
	UpdatedByTelegramID  int64
	DisabledAt           *time.Time
	DisabledByTelegramID int64
	DeletedAt            *time.Time
	DeletedByTelegramID  int64
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type NewShortLink struct {
	Code                string
	OriginalURL         string
	Title               string
	CreatedByTelegramID int64
}

func (s LinkStatus) Valid() bool {
	switch s {
	case StatusActive, StatusDisabled, StatusDeleted:
		return true
	default:
		return false
	}
}

func IsValidCode(code string) bool {
	if len(code) < 4 || len(code) > 32 {
		return false
	}
	for _, r := range code {
		isDigit := r >= '0' && r <= '9'
		isLower := r >= 'a' && r <= 'z'
		isUpper := r >= 'A' && r <= 'Z'
		if !isDigit && !isLower && !isUpper {
			return false
		}
	}
	return true
}

func BuildShortURL(baseURL string, code string) string {
	return strings.TrimRight(baseURL, "/") + "/" + code
}

func CompactTitle(title string) string {
	title = strings.TrimSpace(title)
	if utf8.RuneCountInString(title) <= 7 {
		return title
	}

	runes := []rune(title)
	return string(runes[:6]) + ".."
}

func DeriveTitle(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "link"
	}

	host := strings.TrimSpace(parsed.Host)
	if host == "" {
		return "link"
	}

	path := strings.Trim(parsed.EscapedPath(), "/")
	if path == "" {
		return host
	}

	parts := strings.Split(path, "/")
	last := strings.TrimSpace(parts[len(parts)-1])
	if last == "" {
		return host
	}
	return host + "/" + last
}
