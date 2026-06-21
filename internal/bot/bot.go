package bot

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tuicli/shorter/internal/app"
	"github.com/tuicli/shorter/internal/domain"
	tele "gopkg.in/telebot.v3"
)

var ErrMissingToken = errors.New("telegram bot token is empty")

type LinkService interface {
	PreviewBulkLinks(ctx context.Context, text string) (app.BulkPreview, error)
	CreateBulkLinks(ctx context.Context, adminID int64, preview app.BulkPreview) (app.CreateBulkResult, error)
	ListLatestLinks(ctx context.Context, page int) (app.LinkPage, error)
	SearchLinks(ctx context.Context, query string, page int) (app.LinkPage, error)
	GetLink(ctx context.Context, id int64) (app.LinkView, bool, error)
	DisableLink(ctx context.Context, id int64, adminID int64) (app.LinkView, bool, error)
	EnableLink(ctx context.Context, id int64, adminID int64) (app.LinkView, bool, error)
	DeleteLink(ctx context.Context, id int64, adminID int64) (app.LinkView, bool, error)
	ExportLinksCSV(ctx context.Context, adminID int64) (app.CSVExport, error)
}

type Options struct {
	PollTimeout time.Duration
	FSMTTL      time.Duration
}

type Bot struct {
	bot      *tele.Bot
	service  LinkService
	adminIDs map[int64]struct{}
	states   *stateStore
	tracker  *messageTracker
	logger   *slog.Logger
}

func New(token string, adminUserIDs []int64, service LinkService, logger *slog.Logger, options Options) (*Bot, error) {
	if strings.TrimSpace(token) == "" {
		return nil, ErrMissingToken
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	}
	if options.PollTimeout <= 0 {
		options.PollTimeout = 30 * time.Second
	}

	tb, err := tele.NewBot(tele.Settings{
		Token:     token,
		ParseMode: tele.ModeHTML,
		Poller: &tele.LongPoller{
			Timeout:        options.PollTimeout,
			AllowedUpdates: []string{"message", "callback_query"},
		},
		OnError: func(err error, c tele.Context) {
			attrs := []any{"error", err}
			if c != nil && c.Sender() != nil {
				attrs = append(attrs, "sender_id", c.Sender().ID)
			}
			logger.Error("telegram handler failed", attrs...)
		},
	})
	if err != nil {
		return nil, err
	}

	b := &Bot{
		bot:      tb,
		service:  service,
		adminIDs: idSet(adminUserIDs),
		states:   newStateStore(options.FSMTTL),
		tracker:  newMessageTracker(),
		logger:   logger,
	}
	b.register()

	return b, nil
}

func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("telegram bot starting", "admin_count", len(b.adminIDs))
	if err := b.bot.RemoveWebhook(false); err != nil {
		return fmt.Errorf("remove telegram webhook before long polling: %w", err)
	}

	done := make(chan struct{})
	go func() {
		b.bot.Start()
		close(done)
	}()

	select {
	case <-ctx.Done():
		b.bot.Stop()
		<-done
		b.logger.Info("telegram bot stopped")
		return nil
	case <-done:
		b.logger.Info("telegram bot stopped")
		return nil
	}
}

func (b *Bot) register() {
	b.bot.Handle("/start", b.adminOnly(b.handleStart))
	b.bot.Handle(tele.OnText, b.adminOnly(b.handleText))
	b.bot.Handle(&tele.Btn{Unique: btnAddLinks}, b.adminOnly(b.handleAddLinksStart))
	b.bot.Handle(&tele.Btn{Unique: btnAddConfirm}, b.adminOnly(b.handleAddConfirm))
	b.bot.Handle(&tele.Btn{Unique: btnSearch}, b.adminOnly(b.handleSearchStart))
	b.bot.Handle(&tele.Btn{Unique: btnSearchPage}, b.adminOnly(b.handleSearchPage))
	b.bot.Handle(&tele.Btn{Unique: btnLatest}, b.adminOnly(b.handleLatest))
	b.bot.Handle(&tele.Btn{Unique: btnOpen}, b.adminOnly(b.handleOpen))
	b.bot.Handle(&tele.Btn{Unique: btnDisable}, b.adminOnly(b.handleDisableConfirm))
	b.bot.Handle(&tele.Btn{Unique: btnDisableOK}, b.adminOnly(b.handleDisableOK))
	b.bot.Handle(&tele.Btn{Unique: btnEnable}, b.adminOnly(b.handleEnableConfirm))
	b.bot.Handle(&tele.Btn{Unique: btnEnableOK}, b.adminOnly(b.handleEnableOK))
	b.bot.Handle(&tele.Btn{Unique: btnDelete}, b.adminOnly(b.handleDeleteConfirm))
	b.bot.Handle(&tele.Btn{Unique: btnDeleteOK}, b.adminOnly(b.handleDeleteOK))
	b.bot.Handle(&tele.Btn{Unique: btnCSV}, b.adminOnly(b.handleCSV))
	b.bot.Handle(&tele.Btn{Unique: btnCancel}, b.adminOnly(b.handleCancel))
	b.bot.Handle(&tele.Btn{Unique: btnBack}, b.adminOnly(b.handleStart))
	b.bot.Handle(&tele.Btn{Unique: btnNoop}, b.adminOnly(func(c tele.Context) error { return nil }))
	b.bot.Handle(tele.OnCallback, b.adminOnly(func(c tele.Context) error {
		return b.upsert(c, "Кнопка устарела. Обнови меню.", mainMenu())
	}))
}

func (b *Bot) adminOnly(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		if c.Callback() != nil {
			defer func() {
				if err := c.Respond(); err != nil {
					b.logger.Debug("callback response ignored", "error", err)
				}
			}()
		}
		if !b.allowed(c.Sender()) {
			if c.Sender() != nil {
				b.logger.Warn("telegram access denied", "sender_id", c.Sender().ID)
			}
			return nil
		}
		return next(c)
	}
}

func (b *Bot) handleStart(c tele.Context) error {
	b.states.clear(contextKey(c))
	return b.upsert(c, renderMain(), mainMenu())
}

func (b *Bot) handleAddLinksStart(c tele.Context) error {
	b.states.set(contextKey(c), stateEntry{Kind: stateAddLinksConfirm})
	return b.upsert(c, renderAddPrompt(), cancelKeyboard())
}

func (b *Bot) handleSearchStart(c tele.Context) error {
	b.states.set(contextKey(c), stateEntry{Kind: stateSearchInput})
	return b.upsert(c, renderSearchPrompt(), cancelKeyboard())
}

func (b *Bot) handleText(c tele.Context) error {
	entry, ok := b.states.get(contextKey(c))
	if !ok {
		return b.handleStart(c)
	}

	switch entry.Kind {
	case stateAddLinksConfirm:
		return b.handleAddLinksText(c)
	case stateSearchInput, stateSearchResults:
		return b.handleSearchText(c)
	default:
		return b.handleStart(c)
	}
}

func (b *Bot) handleAddLinksText(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	preview, err := b.service.PreviewBulkLinks(ctx, c.Text())
	if err != nil {
		if errors.Is(err, domain.ErrTooManyLines) {
			return b.upsert(c, "Слишком много строк. Разбей пачку на меньшие части.", cancelKeyboard())
		}
		b.logger.Error("preview bulk links failed", "error", err, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось разобрать ссылки. Попробуй ещё раз.", cancelKeyboard())
	}

	b.states.set(contextKey(c), stateEntry{
		Kind:    stateAddLinksConfirm,
		Preview: preview,
	})
	return b.upsert(c, renderPreview(preview), previewKeyboard(preview.HasNewRows()))
}

func (b *Bot) handleAddConfirm(c tele.Context) error {
	entry, ok := b.states.take(contextKey(c), stateAddLinksConfirm)
	if !ok || !entry.Preview.HasNewRows() {
		return b.upsert(c, "Проверка устарела. Отправь ссылки ещё раз.", cancelKeyboard())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := b.service.CreateBulkLinks(ctx, c.Sender().ID, entry.Preview)
	if err != nil {
		b.logger.Error("create bulk links failed", "error", err, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось создать ссылки. Попробуй позже.", mainMenu())
	}

	return b.upsert(c, renderCreateResult(result), mainMenu())
}

func (b *Bot) handleSearchText(c tele.Context) error {
	query := strings.TrimSpace(c.Text())
	if query == "" {
		return b.upsert(c, renderSearchPrompt(), cancelKeyboard())
	}

	b.states.set(contextKey(c), stateEntry{
		Kind:  stateSearchResults,
		Query: query,
	})
	return b.showSearchPage(c, query, 1)
}

func (b *Bot) handleSearchPage(c tele.Context) error {
	entry, ok := b.states.get(contextKey(c))
	if !ok || entry.Query == "" {
		b.states.set(contextKey(c), stateEntry{Kind: stateSearchInput})
		return b.upsert(c, renderSearchPrompt(), cancelKeyboard())
	}
	return b.showSearchPage(c, entry.Query, parsePage(c.Data()))
}

func (b *Bot) showSearchPage(c tele.Context, query string, page int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := b.service.SearchLinks(ctx, query, page)
	if err != nil {
		b.logger.Error("search links failed", "error", err, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось найти ссылки. Попробуй позже.", mainMenu())
	}

	return b.upsert(c, renderList("🔎 <b>Найти</b>", result), listKeyboard(result, sourceSearch))
}

func (b *Bot) handleLatest(c tele.Context) error {
	return b.showLatest(c, parsePage(c.Data()))
}

func (b *Bot) showLatest(c tele.Context, page int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := b.service.ListLatestLinks(ctx, page)
	if err != nil {
		b.logger.Error("list latest links failed", "error", err, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось открыть последние ссылки. Попробуй позже.", mainMenu())
	}

	return b.upsert(c, renderList("🕘 <b>Последние</b>", result), listKeyboard(result, sourceLatest))
}

func (b *Bot) handleOpen(c tele.Context) error {
	id, source, page := parseLinkContext(c.Data())
	return b.showLink(c, id, source, page)
}

func (b *Bot) showLink(c tele.Context, id int64, source string, page int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	item, ok, err := b.service.GetLink(ctx, id)
	if err != nil {
		b.logger.Error("get link failed", "error", err, "link_id", id, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось открыть ссылку. Попробуй позже.", mainMenu())
	}
	if !ok {
		return b.upsert(c, renderNotFound(), listBackKeyboard(source, page))
	}
	return b.upsert(c, renderDetail(item), detailKeyboard(item, source, page))
}

func (b *Bot) handleDisableConfirm(c tele.Context) error {
	id, source, page := parseLinkContext(c.Data())
	item, ok, err := b.getItemForAction(c, id)
	if err != nil || !ok {
		return b.actionReadError(c, err, source, page)
	}
	return b.upsert(c, renderDisableConfirm(item), confirmActionKeyboard("⏸ Выключить", btnDisableOK, formatLinkContext(id, source, page)))
}

func (b *Bot) handleEnableConfirm(c tele.Context) error {
	id, source, page := parseLinkContext(c.Data())
	item, ok, err := b.getItemForAction(c, id)
	if err != nil || !ok {
		return b.actionReadError(c, err, source, page)
	}
	return b.upsert(c, renderEnableConfirm(item), confirmActionKeyboard("▶️ Включить", btnEnableOK, formatLinkContext(id, source, page)))
}

func (b *Bot) handleDeleteConfirm(c tele.Context) error {
	id, source, page := parseLinkContext(c.Data())
	item, ok, err := b.getItemForAction(c, id)
	if err != nil || !ok {
		return b.actionReadError(c, err, source, page)
	}
	return b.upsert(c, renderDeleteConfirm(item), confirmActionKeyboard("✖️ Удалить", btnDeleteOK, formatLinkContext(id, source, page)))
}

func (b *Bot) handleDisableOK(c tele.Context) error {
	id, source, page := parseLinkContext(c.Data())
	return b.updateStatus(c, id, source, page, b.service.DisableLink)
}

func (b *Bot) handleEnableOK(c tele.Context) error {
	id, source, page := parseLinkContext(c.Data())
	return b.updateStatus(c, id, source, page, b.service.EnableLink)
}

func (b *Bot) handleDeleteOK(c tele.Context) error {
	id, source, page := parseLinkContext(c.Data())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, ok, err := b.service.DeleteLink(ctx, id, c.Sender().ID)
	if err != nil {
		b.logger.Error("delete link failed", "error", err, "link_id", id, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось удалить ссылку. Попробуй позже.", listBackKeyboard(source, page))
	}
	if !ok {
		return b.upsert(c, renderNotFound(), listBackKeyboard(source, page))
	}
	if source == sourceSearch {
		return b.handleSearchPage(c)
	}
	return b.showLatest(c, page)
}

func (b *Bot) updateStatus(
	c tele.Context,
	id int64,
	source string,
	page int,
	update func(context.Context, int64, int64) (app.LinkView, bool, error),
) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	item, ok, err := update(ctx, id, c.Sender().ID)
	if err != nil {
		b.logger.Error("update link status failed", "error", err, "link_id", id, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось изменить ссылку. Попробуй позже.", listBackKeyboard(source, page))
	}
	if !ok {
		return b.upsert(c, renderNotFound(), listBackKeyboard(source, page))
	}
	return b.upsert(c, renderDetail(item), detailKeyboard(item, source, page))
}

func (b *Bot) handleCSV(c tele.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	export, err := b.service.ExportLinksCSV(ctx, c.Sender().ID)
	if err != nil {
		b.logger.Error("export csv failed", "error", err, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось выгрузить CSV. Попробуй позже.", mainMenu())
	}

	doc := &tele.Document{
		File:     tele.FromReader(bytes.NewReader(export.Content)),
		FileName: export.Filename,
	}
	if _, err := c.Bot().Send(c.Recipient(), doc); err != nil {
		b.logger.Error("send csv failed", "error", err, "sender_id", c.Sender().ID)
		return b.upsert(c, "CSV создан, но Telegram не принял файл. Попробуй позже.", mainMenu())
	}

	return b.upsert(c, renderCSVReady(export), mainMenu())
}

func (b *Bot) handleCancel(c tele.Context) error {
	b.states.clear(contextKey(c))
	return b.upsert(c, "Отменено.", mainMenu())
}

func (b *Bot) getItemForAction(c tele.Context, id int64) (app.LinkView, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return b.service.GetLink(ctx, id)
}

func (b *Bot) actionReadError(c tele.Context, err error, source string, page int) error {
	if err != nil {
		b.logger.Error("read link for action failed", "error", err, "sender_id", c.Sender().ID)
		return b.upsert(c, "Не получилось открыть ссылку. Попробуй позже.", listBackKeyboard(source, page))
	}
	return b.upsert(c, renderNotFound(), listBackKeyboard(source, page))
}

func (b *Bot) upsert(c tele.Context, text string, markup *tele.ReplyMarkup) error {
	key := contextKey(c)
	opts := []any{tele.NoPreview}
	if markup != nil {
		opts = append(opts, markup)
	}

	if c.Callback() != nil {
		if msg := c.Message(); msg != nil && msg.ID != 0 {
			b.tracker.set(key, msg.ID)
		}
		err := c.Edit(text, opts...)
		if err == nil || isMessageNotModified(err) {
			return nil
		}
		b.logger.Debug("telegram edit fallback to send", "error", err)
	}

	if oldID, ok := b.tracker.get(key); ok && c.Chat() != nil {
		msg := &tele.Message{ID: oldID, Chat: c.Chat()}
		if err := c.Bot().Delete(msg); err == nil {
			b.tracker.clearIf(key, oldID)
		}
	}

	msg, err := c.Bot().Send(c.Recipient(), text, opts...)
	if err != nil {
		return err
	}
	if msg != nil {
		b.tracker.set(key, msg.ID)
	}
	return nil
}

func listBackKeyboard(source string, page int) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	menu.Inline(menu.Row(backToListButton(menu, source, page)))
	return menu
}

func idSet(ids []int64) map[int64]struct{} {
	out := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func (b *Bot) allowed(user *tele.User) bool {
	if user == nil {
		return false
	}
	_, ok := b.adminIDs[user.ID]
	return ok
}

func contextKey(c tele.Context) int64 {
	if c == nil || c.Sender() == nil {
		return 0
	}
	chatID := int64(0)
	if c.Chat() != nil {
		chatID = c.Chat().ID
	}
	return scopeKey(c.Sender().ID, chatID)
}

func parsePage(raw string) int {
	page, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || page < 1 {
		return 1
	}
	return page
}

func parseLinkContext(raw string) (int64, string, int) {
	parts := strings.Split(raw, "|")
	if len(parts) != 3 {
		return 0, sourceLatest, 1
	}
	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || id < 1 {
		id = 0
	}
	source := parts[1]
	if source != sourceSearch {
		source = sourceLatest
	}
	return id, source, parsePage(parts[2])
}

func isMessageNotModified(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "message is not modified")
}
