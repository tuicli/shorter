package domain

import (
	"errors"
	"testing"
)

func TestParseBulkLinks(t *testing.T) {
	input := stringsJoinLines(
		"21. https://rfnd.io/t/phet8/?pid=2854&sub1=05a_aa33aa&erid=2SDnjchJUYg — a_aa33aa",
		"22. https://rfnd.io/t/phet8/?pid=2854&sub1=05b_bb44bb second link",
		"https://example.com/path",
		"broken line",
	)

	result, err := ParseBulkLinks(input, 10)
	if err != nil {
		t.Fatalf("ParseBulkLinks returned error: %v", err)
	}

	if len(result.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(result.Rows))
	}
	if result.Rows[0].Title != "a_aa33aa" {
		t.Fatalf("first title = %q", result.Rows[0].Title)
	}
	if result.Rows[1].Title != "second link" {
		t.Fatalf("second title = %q", result.Rows[1].Title)
	}
	if result.Rows[2].Title != "example.com/path" {
		t.Fatalf("derived title = %q", result.Rows[2].Title)
	}
	if len(result.InvalidLines) != 1 || result.InvalidLines[0].Line != 4 {
		t.Fatalf("invalid lines = %#v, want line 4", result.InvalidLines)
	}
}

func TestParseBulkLinksLocalDuplicates(t *testing.T) {
	input := stringsJoinLines(
		"https://example.com/a first",
		"https://example.com/a second",
	)

	result, err := ParseBulkLinks(input, 10)
	if err != nil {
		t.Fatalf("ParseBulkLinks returned error: %v", err)
	}

	if len(result.Rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(result.Rows))
	}
	if len(result.LocalDuplicates) != 1 {
		t.Fatalf("duplicates = %d, want 1", len(result.LocalDuplicates))
	}
	if result.LocalDuplicates[0].Line != 2 || result.LocalDuplicates[0].FirstLine != 1 {
		t.Fatalf("duplicate = %#v, want line 2 first line 1", result.LocalDuplicates[0])
	}
}

func TestParseBulkLinksTooManyLines(t *testing.T) {
	_, err := ParseBulkLinks("https://example.com/a\nhttps://example.com/b", 1)
	if !errors.Is(err, ErrTooManyLines) {
		t.Fatalf("err = %v, want ErrTooManyLines", err)
	}
}

func TestCompactTitle(t *testing.T) {
	if got := CompactTitle("a_aa33aa"); got != "a_aa33.." {
		t.Fatalf("CompactTitle = %q", got)
	}
	if got := CompactTitle("abcdefg"); got != "abcdefg" {
		t.Fatalf("CompactTitle short = %q", got)
	}
}

func stringsJoinLines(lines ...string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	return out
}
