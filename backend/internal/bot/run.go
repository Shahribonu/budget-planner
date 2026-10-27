package bot

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"html"
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
	b.budgetAlerts(ctx)
	b.planPrompts(ctx)
	users, err := b.q.UsersDueForSummary(ctx)
	if err != nil {
		slog.Warn("scheduler: due users", "err", err)
		return
	}
	for _, u := range users {
		b.sendSummary(ctx, u)
	}
}

// budgetAlerts warns once per budget and month at 80% and again at 100%. Checking every
// minute catches spending from the app and the chat alike.
func (b *Bot) budgetAlerts(ctx context.Context) {
	rows, err := b.q.BudgetsForAlerts(ctx)
	if err != nil {
		slog.Warn("scheduler: budgets", "err", err)
		return
	}
	for _, r := range rows {
		threshold := 0
		switch {
		case r.Spent >= r.Amount:
			threshold = 100
		case r.Spent*100 >= r.Amount*80:
			threshold = 80
		default:
			continue
		}
		if _, err := b.q.ClaimBudgetAlert(ctx, dbq.ClaimBudgetAlertParams{BudgetID: r.ID, Month: r.Month, Threshold: int32(threshold)}); err != nil {
			continue // already sent (pgx.ErrNoRows) or a database error; never send twice
		}
		if threshold == 100 { // no 80% warning after the budget is already used up
			_, _ = b.q.ClaimBudgetAlert(ctx, dbq.ClaimBudgetAlertParams{BudgetID: r.ID, Month: r.Month, Threshold: 80})
		}
		lang := r.Language
		name := i18n.T(lang, "budget.overall")
		if r.CategoryID != nil {
			if c, err := b.q.GetCategory(ctx, dbq.GetCategoryParams{ID: *r.CategoryID, UserID: &r.UserID}); err == nil {
				name = c.Icon + " " + categoryName(lang, c)
			}
		}
		spent, limit := i18n.Money(lang, r.Spent, r.Currency), i18n.Money(lang, r.Amount, r.Currency)
		text := i18n.T(lang, "budget.over", html.EscapeString(name), spent, limit)
		if threshold == 80 {
			text = i18n.T(lang, "budget.warn", html.EscapeString(name), r.Spent*100/r.Amount, spent, limit)
		}
		if _, err := b.tg.SendMessage(ctx, telegram.SendMessage{ChatID: r.TelegramID, Text: text, ParseMode: "HTML"}); err != nil {
			slog.Warn("scheduler: budget alert", "err", err)
			continue
		}
		b.capture.Event(ctx, r.UserID, "budget_alert", map[string]any{"threshold": threshold, "category": r.CategoryID != nil})
	}
}

// dueLines lists installment payments due tomorrow or earlier and debts due today or
// earlier, so the evening summary doubles as the payment reminder.
func (b *Bot) dueLines(ctx context.Context, u dbq.User, today time.Time) []string {
	lang := u.Language
	when := func(d time.Time) (string, bool) {
		switch {
		case d.Before(today):
			return i18n.T(lang, "due.overdue", d.Format("02.01")), true
		case d.Equal(today):
			return i18n.T(lang, "due.today"), true
		case d.Equal(today.AddDate(0, 0, 1)):
			return i18n.T(lang, "due.tomorrow"), true
		}
		return "", false
	}
	var lines []string
	plans, err := b.capture.Ledger().ListPlans(ctx, u.ID)
	if err != nil {
		slog.Warn("scheduler: plans", "err", err)
	}
	for _, p := range plans {
		if p.Next == nil {
			continue
		}
		if w, ok := when(p.Next.DueOn); ok {
			name := p.Title
			if p.Provider != "" {
				name = p.Provider + " · " + p.Title
			}
			lines = append(lines, i18n.T(lang, "due.installment", html.EscapeString(name), i18n.Money(lang, p.Next.Amount, p.Currency), w))
		}
	}
	debts, err := b.capture.Ledger().ListDebts(ctx, u.ID)
	if err != nil {
		slog.Warn("scheduler: debts", "err", err)
	}
	for _, d := range debts {
		if d.Remaining == 0 || d.DueOn == nil || d.DueOn.After(today) {
			continue
		}
		w, _ := when(*d.DueOn)
		key := "due.lent"
		if d.Direction == "borrowed" {
			key = "due.borrowed"
		}
		lines = append(lines, i18n.T(lang, key, html.EscapeString(d.Counterparty), i18n.Money(lang, d.Remaining, d.Currency), w))
	}
	return lines
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
	if due := b.dueLines(ctx, u, day); len(due) > 0 {
		text += "\n\n" + i18n.T(lang, "due.title") + "\n" + strings.Join(due, "\n")
	}
	if until, started, err := b.billing.MaybeStartTrial(ctx, u); err != nil {
		slog.Warn("scheduler: trial", "err", err)
	} else if started {
		text += "\n\n" + i18n.T(lang, "premium.trial", until.In(b.loc(u)).Format("02.01.2006"))
		b.capture.Event(ctx, u.ID, "trial_started", nil)
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
