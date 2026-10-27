package bot

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/i18n"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
)

// Poll receives updates with long polling, for local development without a public URL.
// It removes any webhook first, since Telegram allows only one delivery method.
func (b *Bot) Poll(ctx context.Context) error {
	if err := b.tg.DeleteWebhook(ctx); err != nil {
		return err
	}
	slog.Info("bot: polling for updates")
	var offset int64
	for ctx.Err() == nil {
		upds, err := b.tg.GetUpdates(ctx, offset, 50)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			slog.Warn("bot: getUpdates", "err", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, u := range upds {
			b.HandleUpdate(ctx, u)
			offset = u.UpdateID + 1
		}
	}
	return nil
}

// Webhook serves Telegram's update POSTs. Requests without the secret token set via
// setWebhook are rejected.
func (b *Bot) Webhook(secret string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("X-Telegram-Bot-Api-Secret-Token")
		if secret == "" || subtle.ConstantTimeCompare([]byte(got), []byte(secret)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var upd telegram.Update
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&upd); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		b.HandleUpdate(r.Context(), upd)
		w.WriteHeader(http.StatusOK)
	})
}

// SetCommands registers the command menu in every supported language.
func (b *Bot) SetCommands(ctx context.Context) {
	codes := map[string]string{"uz-Latn": "", "ru": "ru", "en": "en"}
	for lang, code := range codes {
		var cmds []telegram.BotCommand
		for _, line := range strings.Split(i18n.T(lang, "help"), "\n") {
			name, desc, ok := strings.Cut(line, " — ")
			if ok && strings.HasPrefix(name, "/") {
				cmds = append(cmds, telegram.BotCommand{Command: strings.TrimPrefix(name, "/"), Description: desc})
			}
		}
		if err := b.tg.SetMyCommands(ctx, cmds, code); err != nil {
			slog.Warn("bot: setMyCommands", "lang", lang, "err", err)
		}
	}
}

// RunScheduler sends evening summaries and expires old drafts, once a minute.
func (b *Bot) RunScheduler(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		b.tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// Tick runs one scheduler pass. Exposed for tests.
func (b *Bot) Tick(ctx context.Context) { b.tick(ctx) }

func (b *Bot) tick(ctx context.Context) {
	if _, err := b.q.ExpireDrafts(ctx); err != nil {
		slog.Warn("scheduler: expire drafts", "err", err)
	}
	users, err := b.q.UsersDueForSummary(ctx)
	if err != nil {
		slog.Warn("scheduler: due users", "err", err)
		return
	}
	for _, u := range users {
		b.sendSummary(ctx, u)
	}
}

func (b *Bot) sendSummary(ctx context.Context, u dbq.User) {
	y, m, d := b.now().In(b.loc(u)).Date()
	// Mark first: a failed send must not turn into a message every minute.
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	if err := b.q.MarkSummarySent(ctx, dbq.MarkSummarySentParams{ID: u.ID, Day: &day}); err != nil {
		slog.Warn("scheduler: mark sent", "err", err)
		return
	}
	lang := u.Language
	text := "<b>" + i18n.T(lang, "summary.title") + "</b>\n"
	rows, err := b.capture.Today(ctx, u)
	if err == nil && len(rows) > 0 {
		text += b.todayText(ctx, u) + "\n\n" + i18n.T(lang, "summary.prompt")
	} else {
		text += i18n.T(lang, "summary.none")
	}
	kb := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: i18n.T(lang, "summary.off"), CallbackData: "n:off"}}}}
	_, err = b.tg.SendMessage(ctx, telegram.SendMessage{ChatID: u.TelegramID, Text: text, ParseMode: "HTML", ReplyMarkup: kb})
	if telegram.IsBlocked(err) {
		off := false
		_, _ = b.q.SetNotify(ctx, dbq.SetNotifyParams{Enabled: &off, ID: u.ID})
		return
	}
	if err != nil {
		slog.Warn("scheduler: send summary", "err", err)
		return
	}
	b.capture.Event(ctx, u.ID, "summary_sent", map[string]any{"logged_today": len(rows) > 0})
}
