package bot

import (
	"html"
	"strconv"
	"strings"
	"time"

	"github.com/tuicli/shorter/internal/app"
	"github.com/tuicli/shorter/internal/domain"
)

const (
	previewSampleLimit = 10
	resultSampleLimit  = 50
)

func renderMain() string {
	return strings.Join([]string{
		"🔗 <b>Ссылки</b>",
		"",
		"Добавь пачку ссылок, найди созданную или выгрузи CSV.",
	}, "\n")
}

func renderAddPrompt() string {
	return strings.Join([]string{
		"➕ <b>Добавить ссылки</b>",
		"",
		"Отправь ссылки текстом. Одна строка - одна ссылка.",
	}, "\n")
}

func renderSearchPrompt() string {
	return strings.Join([]string{
		"🔎 <b>Найти</b>",
		"",
		"Отправь часть ссылки, названия или кода.",
	}, "\n")
}

func renderPreview(preview app.BulkPreview) string {
	parts := []string{
		"➕ <b>Проверка ссылок</b>",
		"",
		"Новые: <code>" + strconv.Itoa(len(preview.NewRows)) + "</code>",
		"Дубли: <code>" + strconv.Itoa(len(preview.DuplicateRows)) + "</code>",
		"Ошибки: <code>" + strconv.Itoa(len(preview.InvalidLines)) + "</code>",
	}

	if len(preview.NewRows) > 0 {
		parts = append(parts, "", "<b>Будут созданы:</b>")
		for i, row := range preview.NewRows {
			if i >= previewSampleLimit {
				parts = append(parts, "…ещё "+strconv.Itoa(len(preview.NewRows)-previewSampleLimit))
				break
			}
			parts = append(parts, "стр. "+strconv.Itoa(row.Line)+": "+escape(row.Title)+" - "+escape(shorten(row.OriginalURL, 80)))
		}
	}

	if len(preview.DuplicateRows) > 0 {
		parts = append(parts, "", "<b>Дубли будут пропущены:</b>")
		for i, row := range preview.DuplicateRows {
			if i >= previewSampleLimit {
				parts = append(parts, "…ещё "+strconv.Itoa(len(preview.DuplicateRows)-previewSampleLimit))
				break
			}
			if row.Kind == "existing" {
				parts = append(parts, "стр. "+strconv.Itoa(row.Line)+": "+escape(row.ExistingShortURL)+" ("+escape(string(row.ExistingStatus))+")")
				continue
			}
			parts = append(parts, "стр. "+strconv.Itoa(row.Line)+": дубль строки "+strconv.Itoa(row.FirstLine))
		}
	}

	if len(preview.InvalidLines) > 0 {
		parts = append(parts, "", "<b>Ошибки:</b>")
		for i, row := range preview.InvalidLines {
			if i >= previewSampleLimit {
				parts = append(parts, "…ещё "+strconv.Itoa(len(preview.InvalidLines)-previewSampleLimit))
				break
			}
			parts = append(parts, "стр. "+strconv.Itoa(row.Line)+": "+escape(row.Reason))
		}
	}

	return strings.Join(parts, "\n")
}

func renderCreateResult(result app.CreateBulkResult) string {
	parts := []string{
		"✔️ <b>Готово</b>",
		"",
		"Создано: <code>" + strconv.Itoa(len(result.Created)) + "</code>",
		"Пропущено дублей: <code>" + strconv.Itoa(len(result.SkippedDuplicates)) + "</code>",
		"Ошибки: <code>" + strconv.Itoa(len(result.Failures)) + "</code>",
	}

	if len(result.Created) > 0 {
		parts = append(parts, "", "<b>Короткие ссылки:</b>")
		for i, created := range result.Created {
			if i >= resultSampleLimit {
				parts = append(parts, "…ещё "+strconv.Itoa(len(result.Created)-resultSampleLimit))
				break
			}
			parts = append(parts, escape(created.Link.Title)+" - "+escape(created.ShortURL))
		}
	}

	if len(result.Failures) > 0 {
		parts = append(parts, "", "<b>Не созданы:</b>")
		for i, failure := range result.Failures {
			if i >= previewSampleLimit {
				parts = append(parts, "…ещё "+strconv.Itoa(len(result.Failures)-previewSampleLimit))
				break
			}
			parts = append(parts, "стр. "+strconv.Itoa(failure.Line)+": "+escape(failure.Error))
		}
	}

	return strings.Join(parts, "\n")
}

func renderList(title string, page app.LinkPage) string {
	if len(page.Links) == 0 {
		return title + "\n\nНичего не найдено."
	}
	return title + "\n\nВсего: <code>" + strconv.Itoa(page.Total) + "</code>"
}

func renderDetail(item app.LinkView) string {
	link := item.Link
	return strings.Join([]string{
		"🔗 <b>Ссылка</b>",
		"",
		"<b>Название:</b> " + escape(link.Title),
		"<b>Статус:</b> <code>" + escape(string(link.Status)) + "</code>",
		"<b>Короткая:</b> " + escape(item.ShortURL),
		"<b>Оригинал:</b> " + escape(link.OriginalURL),
		"<b>Код:</b> <code>" + escape(link.Code) + "</code>",
		"<b>Создана:</b> <code>" + escape(formatTime(link.CreatedAt)) + "</code>",
	}, "\n")
}

func renderDisableConfirm(item app.LinkView) string {
	return strings.Join([]string{
		"⏸ <b>Выключить ссылку?</b>",
		"",
		escape(item.Link.Title),
		escape(item.ShortURL),
		"",
		"Она перестанет редиректить.",
	}, "\n")
}

func renderEnableConfirm(item app.LinkView) string {
	return strings.Join([]string{
		"▶️ <b>Включить ссылку?</b>",
		"",
		escape(item.Link.Title),
		escape(item.ShortURL),
		"",
		"Она снова начнет редиректить.",
	}, "\n")
}

func renderDeleteConfirm(item app.LinkView) string {
	return strings.Join([]string{
		"✖️ <b>Удалить ссылку?</b>",
		"",
		escape(item.Link.Title),
		escape(item.ShortURL),
		"",
		"Она пропадет из списков и перестанет редиректить.",
	}, "\n")
}

func renderCSVReady(export app.CSVExport) string {
	lines := []string{
		"⬇️ <b>CSV готов</b>",
		"",
		"Строк: <code>" + strconv.Itoa(export.Rows) + "</code>",
	}
	if export.Capped {
		lines = append(lines, "Достигнут лимит выгрузки.")
	}
	return strings.Join(lines, "\n")
}

func renderNotFound() string {
	return "Ссылка не найдена или уже удалена."
}

func escape(value string) string {
	return html.EscapeString(value)
}

func shorten(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit-2]) + ".."
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Local().Format("2006-01-02 15:04")
}

func statusActionLabel(status domain.LinkStatus) string {
	switch status {
	case domain.StatusActive:
		return "⏸ Выключить"
	case domain.StatusDisabled:
		return "▶️ Включить"
	default:
		return ""
	}
}
