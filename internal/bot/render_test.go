package bot

import (
	"testing"

	"github.com/tuicli/shorter/internal/app"
	"github.com/tuicli/shorter/internal/domain"
)

func TestRenderListShowsCopyableLinkLabels(t *testing.T) {
	page := app.LinkPage{
		Links: []app.LinkView{
			{
				Link: domain.ShortLink{
					Title: "a&b",
				},
				ShortURL: "https://s.example/A",
			},
			{
				Link: domain.ShortLink{
					Title: "abcdefghi",
				},
				ShortURL: "https://s.example/B",
			},
		},
		Total: 28,
	}

	got := renderList("🕘 <b>Последние</b>", page)
	want := "🕘 <b>Последние</b>\n\n" +
		"Всего: <code>28</code>\n\n" +
		"a&amp;b - https://s.example/A\n" +
		"abcdef.. - https://s.example/B"
	if got != want {
		t.Fatalf("renderList() = %q, want %q", got, want)
	}
}

func TestRenderListEmpty(t *testing.T) {
	got := renderList("🔎 <b>Найти</b>", app.LinkPage{})
	want := "🔎 <b>Найти</b>\n\nНичего не найдено."
	if got != want {
		t.Fatalf("renderList() = %q, want %q", got, want)
	}
}
