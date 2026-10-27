// Package bot is the Telegram chat interface: logging by text or voice, confirmation
// cards with fix and undo buttons, and a few commands. All money writes go through
// capture and ledger; the bot only renders and routes.
package bot

import (
	"context"
	"errors"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/stt"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/i18n"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
)

const (
	maxVoiceSeconds = 60
	maxVoiceBytes   = 2 << 20
)

type Config struct {
	// MiniAppURL is the public HTTPS URL of the Mini App; empty hides app buttons.
	MiniAppURL string
}

type Bot struct {
	cfg     Config
	tg      *telegram.Client
	q       *dbq.Queries
	capture *capture.Service
	stt     stt.Transcriber
	now     func() time.Time
}

func New(cfg Config, tg *telegram.Client, pool *pgxpool.Pool, c *capture.Service, t stt.Transcriber) *Bot {
	return &Bot{cfg: cfg, tg: tg, q: dbq.New(pool), capture: c, stt: t, now: time.Now}
}

// HandleUpdate processes one update. It is safe to call more than once for the same
// update: repeats are dropped.
func (b *Bot) HandleUpdate(ctx context.Context, upd telegram.Update) {
	var from *telegram.User
	var chat telegram.Chat
	kind := ""
	switch {
	case upd.Message != nil:
		from, chat = upd.Message.From, upd.Message.Chat
		switch {
		case upd.Message.Voice != nil:
			kind = "voice"
		case strings.HasPrefix(upd.Message.Text, "/"):
			kind = "command"
		case upd.Message.Text != "":
			kind = "text"
		}
	case upd.CallbackQuery != nil && upd.CallbackQuery.Message != nil:
		from, chat, kind = &upd.CallbackQuery.From, upd.CallbackQuery.Message.Chat, "callback"
	}
	if from == nil || from.IsBot || chat.Type != "private" || kind == "" {
		return
	}

	u, err := b.q.UpsertUser(ctx, dbq.UpsertUserParams{
		TelegramID: from.ID, FirstName: from.FirstName, Username: optional(from.Username), Language: languageFor(from.LanguageCode),
	})
	if err != nil {
		slog.Error("bot: upsert user", "err", err)
		return
	}
	if _, err := b.q.ClaimIngestion(ctx, dbq.ClaimIngestionParams{
		Source: "telegram", ExternalID: strconv.FormatInt(upd.UpdateID, 10), UserID: &u.ID, Kind: kind,
	}); err != nil {
		return // already handled (pgx.ErrNoRows) or the database is down; either way do nothing twice
	}

	switch kind {
	case "command":
		b.command(ctx, u, upd.Message)
	case "text":
		b.text(ctx, u, upd.Message, upd.Message.Text, "chat", "")
	case "voice":
		b.voice(ctx, u, upd.Message)
	case "callback":
		b.callback(ctx, u, upd.CallbackQuery)
	}
}

func optional(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

var languageNames = map[string]bool{"uz-Latn": true, "uz-Cyrl": true, "ru": true, "en": true}

func languageKeyboard() *telegram.InlineKeyboardMarkup {
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{
		{{Text: "O'zbekcha", CallbackData: "l:uz-Latn"}, {Text: "Ўзбекча", CallbackData: "l:uz-Cyrl"}},
		{{Text: "Русский", CallbackData: "l:ru"}, {Text: "English", CallbackData: "l:en"}},
	}}
}

func languageFor(code string) string {
	switch {
	case strings.HasPrefix(code, "ru"):
		return "ru"
	case strings.HasPrefix(code, "en"):
		return "en"
	}
	return i18n.Default
}

func (b *Bot) send(ctx context.Context, chatID int64, text string, kb *telegram.InlineKeyboardMarkup, replyTo int64) {
	m := telegram.SendMessage{ChatID: chatID, Text: text, ParseMode: "HTML", ReplyMarkup: kb}
	if replyTo != 0 {
		m.ReplyTo = &telegram.ReplyParameters{MessageID: replyTo}
	}
	if _, err := b.tg.SendMessage(ctx, m); err != nil {
		slog.Warn("bot: send", "err", err)
	}
}

func (b *Bot) appButton(lang, path string) *telegram.InlineKeyboardButton {
	if b.cfg.MiniAppURL == "" {
		return nil
	}
	return &telegram.InlineKeyboardButton{Text: i18n.T(lang, "start.app_button"), WebApp: &telegram.WebAppInfo{URL: b.cfg.MiniAppURL + "#" + path}}
}

// ---- commands ----

func (b *Bot) command(ctx context.Context, u dbq.User, m *telegram.Message) {
	name, arg, _ := strings.Cut(strings.TrimSpace(m.Text), " ")
	name, _, _ = strings.Cut(name, "@") // /start@MyBot in some clients
	lang := u.Language
	switch name {
	case "/start":
		if _, err := b.q.MarkStarted(ctx, u.ID); err == nil {
			now := b.now()
			u.StartedAt = &now
			b.capture.Event(ctx, u.ID, "bot_start", map[string]any{"payload": arg != ""})
		}
		text := i18n.T(lang, "start.greeting", html.EscapeString(u.FirstName)) + "\n\n" + i18n.T(lang, "start.try")
		var kb *telegram.InlineKeyboardMarkup
		if btn := b.appButton(lang, "/"); btn != nil {
			kb = &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{*btn}}}
		}
		b.send(ctx, m.Chat.ID, text, kb, 0)
	case "/help":
		b.send(ctx, m.Chat.ID, i18n.T(lang, "help"), nil, 0)
	case "/today":
		b.send(ctx, m.Chat.ID, b.todayText(ctx, u), nil, 0)
	case "/undo":
		tx, err := b.capture.UndoLast(ctx, u)
		switch {
		case errors.Is(err, ledger.ErrNotFound):
			b.send(ctx, m.Chat.ID, i18n.T(lang, "undo.nothing"), nil, 0)
		case err != nil:
			b.fail(ctx, m.Chat.ID, u, err)
		default:
			b.send(ctx, m.Chat.ID, i18n.T(lang, "undone", b.oneLine(ctx, u, tx)), nil, 0)
		}
	case "/app":
		if btn := b.appButton(lang, "/"); btn != nil {
			b.send(ctx, m.Chat.ID, "📊", &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{*btn}}}, 0)
		} else {
			b.send(ctx, m.Chat.ID, i18n.T(lang, "app.unavailable"), nil, 0)
		}
	case "/lang":
		b.send(ctx, m.Chat.ID, i18n.T(lang, "lang.choose"), languageKeyboard(), 0)
	case "/settings":
		b.settings(ctx, u, m.Chat.ID, strings.TrimSpace(arg))
	default:
		b.send(ctx, m.Chat.ID, i18n.T(lang, "help"), nil, 0)
	}
}

func (b *Bot) todayText(ctx context.Context, u dbq.User) string {
	rows, err := b.capture.Today(ctx, u)
	if err != nil || len(rows) == 0 {
		return i18n.T(u.Language, "today.none")
	}
	var n int64
	var totals []string
	for _, r := range rows {
		n += r.N
		totals = append(totals, i18n.Money(u.Language, r.Amount, r.Currency))
	}
	if n == 1 {
		return i18n.T(u.Language, "today.summary_one", strings.Join(totals, " + "))
	}
	return i18n.T(u.Language, "today.summary", n, strings.Join(totals, " + "))
}

func (b *Bot) settings(ctx context.Context, u dbq.User, chatID int64, arg string) {
	if arg != "" {
		t, err := time.Parse("15:04", arg)
		if err != nil {
			b.send(ctx, chatID, i18n.T(u.Language, "settings.bad_time"), nil, 0)
			return
		}
		on := true
		u, err = b.q.SetNotify(ctx, dbq.SetNotifyParams{Enabled: &on, NotifyTime: pgtype.Time{Microseconds: int64(t.Hour()*3600+t.Minute()*60) * 1e6, Valid: true}, ID: u.ID})
		if err != nil {
			b.fail(ctx, chatID, u, err)
			return
		}
	}
	b.send(ctx, chatID, b.settingsText(u), b.settingsKeyboard(u), 0)
}

func notifyClock(u dbq.User) string {
	min := u.NotifyTime.Microseconds / 60e6
	return fmt.Sprintf("%02d:%02d", min/60, min%60)
}

func (b *Bot) settingsText(u dbq.User) string {
	if !u.NotifyEnabled {
		return i18n.T(u.Language, "settings.off_now")
	}
	return i18n.T(u.Language, "settings.on", notifyClock(u)) + "\n" + i18n.T(u.Language, "settings.time")
}

func (b *Bot) settingsKeyboard(u dbq.User) *telegram.InlineKeyboardMarkup {
	btn := telegram.InlineKeyboardButton{Text: i18n.T(u.Language, "settings.turn_off"), CallbackData: "n:off"}
	if !u.NotifyEnabled {
		btn = telegram.InlineKeyboardButton{Text: i18n.T(u.Language, "settings.turn_on"), CallbackData: "n:on"}
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{btn}}}
}

// ---- logging ----

func (b *Bot) text(ctx context.Context, u dbq.User, m *telegram.Message, text, source, transcript string) {
	ref := fmt.Sprintf("tg:%d:%d", m.Chat.ID, m.MessageID)
	outs, err := b.capture.Capture(ctx, u, text, source, ref)
	if err != nil {
		b.fail(ctx, m.Chat.ID, u, err)
		return
	}
	if len(outs) == 0 {
		msg := i18n.T(u.Language, "not_understood")
		if transcript != "" {
			msg = i18n.T(u.Language, "card.heard", html.EscapeString(transcript)) + "\n" + msg
		}
		b.send(ctx, m.Chat.ID, msg, nil, m.MessageID)
		return
	}
	for _, o := range outs {
		text, kb := b.render(ctx, u, o)
		if transcript != "" {
			text += "\n" + i18n.T(u.Language, "card.heard", html.EscapeString(transcript))
		}
		b.send(ctx, m.Chat.ID, text, kb, m.MessageID)
		if o.FirstLog && u.NotifyEnabled {
			b.send(ctx, m.Chat.ID, i18n.T(u.Language, "first_log.notify", notifyClock(u)), nil, 0)
		}
	}
}

func (b *Bot) voice(ctx context.Context, u dbq.User, m *telegram.Message) {
	b.capture.Event(ctx, u.ID, "voice_received", map[string]any{"seconds": m.Voice.Duration})
	if m.Voice.Duration > maxVoiceSeconds {
		b.send(ctx, m.Chat.ID, i18n.T(u.Language, "voice.too_long"), nil, m.MessageID)
		return
	}
	if _, ok := b.stt.(stt.None); ok {
		b.send(ctx, m.Chat.ID, i18n.T(u.Language, "voice.unavailable"), nil, m.MessageID)
		return
	}
	// Audio is held in memory only and never stored.
	audio, err := b.tg.DownloadFile(ctx, m.Voice.FileID, maxVoiceBytes)
	if err != nil {
		slog.Warn("bot: download voice", "err", err)
		b.send(ctx, m.Chat.ID, i18n.T(u.Language, "voice.failed"), nil, m.MessageID)
		return
	}
	transcript, err := b.stt.Transcribe(ctx, audio, m.Voice.MimeType, u.Language)
	if err != nil || strings.TrimSpace(transcript) == "" {
		if err != nil {
			slog.Warn("bot: transcribe", "err", err)
		}
		b.capture.Event(ctx, u.ID, "voice_failed", nil)
		b.send(ctx, m.Chat.ID, i18n.T(u.Language, "voice.failed"), nil, m.MessageID)
		return
	}
	b.text(ctx, u, m, transcript, "voice", transcript)
}

func (b *Bot) fail(ctx context.Context, chatID int64, u dbq.User, err error) {
	slog.Error("bot", "err", err)
	b.send(ctx, chatID, i18n.T(u.Language, "error"), nil, 0)
}

// ---- callbacks ----

func (b *Bot) callback(ctx context.Context, u dbq.User, cq *telegram.CallbackQuery) {
	chatID, msgID := cq.Message.Chat.ID, cq.Message.MessageID
	parts := strings.Split(cq.Data, ":")
	arg := func(i int) int64 {
		if i >= len(parts) {
			return 0
		}
		n, _ := strconv.ParseInt(parts[i], 10, 64)
		return n
	}
	toast := ""
	defer func() {
		if err := b.tg.AnswerCallbackQuery(ctx, cq.ID, toast); err != nil {
			slog.Warn("bot: answer callback", "err", err)
		}
	}()
	edit := func(text string, kb *telegram.InlineKeyboardMarkup) {
		if err := b.tg.EditMessageText(ctx, telegram.EditMessageText{ChatID: chatID, MessageID: msgID, Text: text, ParseMode: "HTML", ReplyMarkup: kb}); err != nil {
			slog.Warn("bot: edit", "err", err)
		}
	}
	showOutcome := func(o capture.Outcome, err error) {
		switch {
		case errors.Is(err, capture.ErrDraftGone):
			edit(i18n.T(u.Language, "expired"), nil)
		case err != nil:
			slog.Error("bot: callback", "data", cq.Data, "err", err)
			toast = i18n.T(u.Language, "error")
		case o.Kind == capture.Merged:
			edit(i18n.T(u.Language, "merged"), nil)
		default:
			edit(b.render(ctx, u, o))
		}
	}
	showTx := func(tx ledger.Transaction, err error) {
		if err != nil {
			if !errors.Is(err, ledger.ErrNotFound) {
				slog.Error("bot: callback", "data", cq.Data, "err", err)
			}
			toast = i18n.T(u.Language, "error")
			return
		}
		edit(b.card(ctx, u, tx, ""), b.cardKeyboard(u, tx))
	}

	switch parts[0] {
	case "u": // undo
		tx, err := b.capture.Undo(ctx, u, arg(1))
		if err != nil {
			toast = i18n.T(u.Language, "undo.nothing")
			return
		}
		edit(i18n.T(u.Language, "undone", b.oneLine(ctx, u, tx)), nil)
	case "c": // category picker
		tx, err := b.capture.Ledger().Get(ctx, u.ID, arg(1))
		if err != nil {
			toast = i18n.T(u.Language, "undo.nothing")
			return
		}
		edit(b.card(ctx, u, tx, "")+"\n\n"+i18n.T(u.Language, "ask.category"), b.categoryKeyboard(ctx, u, tx))
	case "cs":
		showTx(b.capture.ChangeCategory(ctx, u, arg(1), arg(2)))
	case "a": // account picker
		tx, err := b.capture.Ledger().Get(ctx, u.ID, arg(1))
		if err != nil {
			toast = i18n.T(u.Language, "undo.nothing")
			return
		}
		edit(b.card(ctx, u, tx, ""), b.accountKeyboard(ctx, u, tx.Currency, func(id int64) string { return fmt.Sprintf("as:%d:%d", tx.ID, id) }, fmt.Sprintf("b:%d", tx.ID)))
	case "as":
		showTx(b.capture.ChangeAccount(ctx, u, arg(1), arg(2)))
	case "b": // back to the card
		showTx(b.capture.Ledger().Get(ctx, u.ID, arg(1)))
	case "dm":
		showOutcome(b.capture.AnswerAmount(ctx, u, arg(1), arg(2) == 1))
	case "da":
		showOutcome(b.capture.AnswerAccount(ctx, u, arg(1), arg(2)))
	case "dd":
		showOutcome(b.capture.AnswerDuplicate(ctx, u, arg(1), arg(2) == 1))
	case "dx":
		question := ""
		if len(parts) > 2 {
			question = parts[2]
		}
		if err := b.capture.CancelDraft(ctx, u, arg(1), question); err != nil && !errors.Is(err, capture.ErrDraftGone) {
			slog.Error("bot: cancel draft", "err", err)
		}
		edit(i18n.T(u.Language, "cancelled"), nil)
	case "l": // language
		if len(parts) < 2 || !languageNames[parts[1]] {
			return
		}
		lang := parts[1]
		if _, err := b.q.UpdateUserSettings(ctx, dbq.UpdateUserSettingsParams{Language: &lang, ID: u.ID}); err != nil {
			toast = i18n.T(u.Language, "error")
			return
		}
		edit(i18n.T(lang, "lang.set"), nil)
	case "n":
		on := len(parts) > 1 && parts[1] == "on"
		nu, err := b.q.SetNotify(ctx, dbq.SetNotifyParams{Enabled: &on, ID: u.ID})
		if err != nil {
			toast = i18n.T(u.Language, "error")
			return
		}
		b.capture.Event(ctx, u.ID, "notify_toggled", map[string]any{"on": on})
		edit(b.settingsText(nu), b.settingsKeyboard(nu))
	}
}
