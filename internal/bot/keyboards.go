package bot

import (
	"strconv"

	"github.com/tuicli/shorter/internal/app"
	"github.com/tuicli/shorter/internal/domain"
	tele "gopkg.in/telebot.v3"
)

const (
	btnAddLinks   = "add_links"
	btnAddConfirm = "add_confirm"
	btnSearch     = "search"
	btnSearchPage = "search_page"
	btnLatest     = "latest"
	btnOpen       = "open"
	btnDisable    = "disable"
	btnDisableOK  = "disable_ok"
	btnEnable     = "enable"
	btnEnableOK   = "enable_ok"
	btnDelete     = "delete"
	btnDeleteOK   = "delete_ok"
	btnCSV        = "csv"
	btnCancel     = "cancel"
	btnBack       = "back"
	btnNoop       = "noop"
	sourceLatest  = "latest"
	sourceSearch  = "search"
)

func mainMenu() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data("➕ Добавить ссылки", btnAddLinks)),
		menu.Row(menu.Data("🔎 Найти", btnSearch), menu.Data("🕘 Последние", btnLatest, "1")),
		menu.Row(menu.Data("⬇️ CSV", btnCSV)),
	)
	return menu
}

func cancelKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(menu.Data("✖️ Отмена", btnCancel)))
	return menu
}

func previewKeyboard(hasNew bool) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	if hasNew {
		menu.Inline(menu.Row(
			menu.Data("✔️ Подтвердить", btnAddConfirm),
			menu.Data("✖️ Отмена", btnCancel),
		))
		return menu
	}
	menu.Inline(menu.Row(menu.Data("✖️ Отмена", btnCancel)))
	return menu
}

func listKeyboard(page app.LinkPage, source string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	rows := make([]tele.Row, 0, len(page.Links)+2)
	for _, item := range page.Links {
		rows = append(rows, menu.Row(
			menu.Data(linkListLabel(item), btnOpen, formatLinkContext(item.Link.ID, source, page.Page)),
		))
	}
	if page.TotalPages > 1 {
		prev := page.Page - 1
		if prev < 1 {
			prev = page.TotalPages
		}
		next := page.Page + 1
		if next > page.TotalPages {
			next = 1
		}
		pageButton := btnLatest
		if source == sourceSearch {
			pageButton = btnSearchPage
		}
		rows = append(rows, menu.Row(
			menu.Data("◁", pageButton, strconv.Itoa(prev)),
			menu.Data(strconv.Itoa(page.Page)+"/"+strconv.Itoa(page.TotalPages), btnNoop),
			menu.Data("▷", pageButton, strconv.Itoa(next)),
		))
	}
	rows = append(rows, menu.Row(menu.Data("◀️ Назад", btnBack)))
	menu.Inline(rows...)
	return menu
}

func detailKeyboard(link app.LinkView, source string, page int) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	data := formatLinkContext(link.Link.ID, source, page)
	rows := []tele.Row{}
	switch link.Link.Status {
	case domain.StatusActive:
		rows = append(rows, menu.Row(menu.Data("⏸ Выключить", btnDisable, data)))
	case domain.StatusDisabled:
		rows = append(rows, menu.Row(menu.Data("▶️ Включить", btnEnable, data)))
	}
	if link.Link.Status == domain.StatusActive || link.Link.Status == domain.StatusDisabled {
		rows = append(rows, menu.Row(menu.Data("✖️ Удалить", btnDelete, data)))
	}
	rows = append(rows, menu.Row(backToListButton(menu, source, page)))
	menu.Inline(rows...)
	return menu
}

func confirmActionKeyboard(action string, okButton string, data string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(
		menu.Row(menu.Data(action, okButton, data)),
		menu.Row(menu.Data("◀️ Назад", btnOpen, data)),
	)
	return menu
}

func backToListButton(menu *tele.ReplyMarkup, source string, page int) tele.Btn {
	if page < 1 {
		page = 1
	}
	if source == sourceSearch {
		return menu.Data("◀️ Назад", btnSearchPage, strconv.Itoa(page))
	}
	return menu.Data("◀️ Назад", btnLatest, strconv.Itoa(page))
}

func formatLinkContext(id int64, source string, page int) string {
	if page < 1 {
		page = 1
	}
	if source != sourceSearch {
		source = sourceLatest
	}
	return strconv.FormatInt(id, 10) + "|" + source + "|" + strconv.Itoa(page)
}
