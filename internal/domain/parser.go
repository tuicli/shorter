package domain

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var httpURLPattern = regexp.MustCompile(`https?://\S+`)

type ParsedRow struct {
	Line        int
	OriginalURL string
	Title       string
	Warnings    []string
}

type InvalidLine struct {
	Line   int
	Reason string
}

type LocalDuplicate struct {
	Line        int
	OriginalURL string
	FirstLine   int
}

type ParseResult struct {
	Rows            []ParsedRow
	LocalDuplicates []LocalDuplicate
	InvalidLines    []InvalidLine
}

func ParseBulkLinks(text string, maxLines int) (ParseResult, error) {
	lines := strings.Split(text, "\n")
	if maxLines > 0 && len(lines) > maxLines {
		return ParseResult{}, fmt.Errorf("%w: got %d, max %d", ErrTooManyLines, len(lines), maxLines)
	}

	result := ParseResult{
		Rows:            []ParsedRow{},
		LocalDuplicates: []LocalDuplicate{},
		InvalidLines:    []InvalidLine{},
	}
	seen := map[string]int{}
	for index, rawLine := range lines {
		lineNumber := index + 1
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		match := httpURLPattern.FindStringIndex(line)
		if match == nil {
			result.InvalidLines = append(result.InvalidLines, InvalidLine{
				Line:   lineNumber,
				Reason: "URL не найден",
			})
			continue
		}

		rawURL := line[match[0]:match[1]]
		originalURL := trimURLTail(rawURL)
		if !validHTTPURL(originalURL) {
			result.InvalidLines = append(result.InvalidLines, InvalidLine{
				Line:   lineNumber,
				Reason: "некорректный URL",
			})
			continue
		}

		if firstLine, ok := seen[originalURL]; ok {
			result.LocalDuplicates = append(result.LocalDuplicates, LocalDuplicate{
				Line:        lineNumber,
				OriginalURL: originalURL,
				FirstLine:   firstLine,
			})
			continue
		}
		seen[originalURL] = lineNumber

		title := cleanTitle(line[match[1]:])
		if title == "" {
			title = DeriveTitle(originalURL)
		}

		warnings := []string{}
		if extra := httpURLPattern.FindStringIndex(line[match[1]:]); extra != nil {
			warnings = append(warnings, "extra_url_ignored")
		}

		result.Rows = append(result.Rows, ParsedRow{
			Line:        lineNumber,
			OriginalURL: originalURL,
			Title:       title,
			Warnings:    warnings,
		})
	}

	return result, nil
}

func trimURLTail(rawURL string) string {
	return strings.TrimRightFunc(rawURL, func(r rune) bool {
		switch r {
		case '.', ',', ';', '!', '?', ')', ']', '}', '"', '\'':
			return true
		default:
			return false
		}
	})
}

func cleanTitle(raw string) string {
	title := strings.TrimSpace(raw)
	title = strings.TrimLeftFunc(title, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '—' || r == '–' || r == ':'
	})
	title = strings.TrimSpace(title)
	if !utf8.ValidString(title) {
		return ""
	}
	return title
}

func validHTTPURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	return parsed.Host != ""
}
