package bot

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/i18n"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/planning"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
)

const planModeFor = 30 * time.Minute

func monthName(lang string, m time.Time) string {
	return i18n.T(lang, fmt.Sprintf("month.%d", int(m.Month())))
}

// startPlan asks for the month's plan; the next message is read as plan items.
func (b *Bot) startPlan(ctx context.Context, u dbq.User, chatID int64) {
	month, err := b.plan.MonthFor(u, "")
	if err != nil {
		b.fail(ctx, chatID, u, err)
		return
	}
	mode, until := "plan", b.now().Add(planModeFor)
	if err := b.q.SetChatMode(ctx, dbq.SetChatModeParams{Mode: &mode, Until: &until, ID: u.ID}); err != nil {
		b.fail(ctx, chatID, u, err)
		return
	}
	b.send(ctx, chatID, i18n.T(u.Language, "plan.prompt", monthName(u.Language, month)), nil, 0)
}

// planText handles a message sent while the user is writing their plan. It reports
// whether the message was taken.
func (b *Bot) planText(ctx context.Context, u dbq.User, m *telegram.Message, text string) bool {
	if u.ChatMode == nil || *u.ChatMode != "plan" || u.ChatModeUntil == nil || b.now().After(*u.ChatModeUntil) {
		return false
	}
	month, err := b.plan.MonthFor(u, "")
	if err != nil {
		b.fail(ctx, m.Chat.ID, u, err)
		return true
	}
	items, err := b.plan.ParseItems(ctx, u.ID, u.MainCurrency, text)
	if err != nil {
		b.fail(ctx, m.Chat.ID, u, err)
		return true
	}
	if len(items) == 0 {
		b.send(ctx, m.Chat.ID, i18n.T(u.Language, "plan.none"), nil, m.MessageID)
		return true // stay in plan mode for another try
	}
	if err := b.q.SetChatMode(ctx, dbq.SetChatModeParams{ID: u.ID}); err != nil {
		slog.Warn("bot: clear chat mode", "err", err)
	}
	v, err := b.plan.AddItems(ctx, u.ID, month, u.MainCurrency, items)
	if err != nil {
		b.fail(ctx, m.Chat.ID, u, err)
		return true
	}
	b.capture.Event(ctx, u.ID, "plan_items_added", map[string]any{"count": len(items), "source": "chat"})
	var kb *telegram.InlineKeyboardMarkup
	if app := b.appButton(u.Language, "/plan"); app != nil {
		app.Text = i18n.T(u.Language, "plan.open_btn")
		kb = &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{*app}}}
	}
	b.send(ctx, m.Chat.ID, b.planSummary(u, v), kb, m.MessageID)
	return true
}

// planSummary renders the plan's recommendations with the figures behind them.
func (b *Bot) planSummary(u dbq.User, v planning.View) string {
	lang, cur := u.Language, v.Plan.Currency
	money := func(x int64) string { return i18n.Money(lang, x, cur) }
	var sb strings.Builder
	sb.WriteString("<b>" + i18n.T(lang, "plan.title", monthName(lang, v.Plan.Month)) + "</b>\n")
	in := v.Inputs
	if in.IncomeSource == "none" {
		sb.WriteString(i18n.T(lang, "plan.no_income") + "\n")
	}
	sb.WriteString(i18n.T(lang, "plan.inputs", money(in.Income), money(in.Reserved), money(in.Savings), money(in.Buffer)) + "\n")
	sb.WriteString(i18n.T(lang, "plan.free", money(v.Result.Free)) + "\n")

	groups := map[planning.Group][]string{}
	for _, it := range v.Items {
		d := it.Decision
		if d == nil {
			continue
		}
		amount := money(it.AmountMax)
		if it.AmountMin != it.AmountMax {
			amount = money(it.AmountMin) + "–" + amount
		}
		line := "• " + html.EscapeString(it.Title) + " — " + amount
		var why []string
		switch d.Reason {
		case planning.ReasonNotEnough:
			if !d.SuggestMonth.IsZero() && d.MonthlySaving == 0 {
				why = append(why, i18n.T(lang, "plan.r.next_month"))
			}
			if d.FromSavings > 0 {
				why = append(why, i18n.T(lang, "plan.r.savings", money(d.FromSavings)))
			}
		case planning.ReasonBeforeSalary:
			why = append(why, i18n.T(lang, "plan.r.salary"))
		case planning.ReasonShortfall:
			why = append(why, i18n.T(lang, "plan.r.shortfall"))
		}
		if d.MonthlySaving > 0 {
			why = append(why, i18n.T(lang, "plan.r.saving", money(d.MonthlySaving), d.Months))
		}
		if len(why) > 0 {
			line += " (" + strings.Join(why, "; ") + ")"
		}
		groups[d.Group] = append(groups[d.Group], line)
	}
	for _, g := range []struct {
		group planning.Group
		key   string
	}{{planning.Fits, "plan.fits"}, {planning.Later, "plan.later"}, {planning.Reconsider, "plan.reconsider"}} {
		if lines := groups[g.group]; len(lines) > 0 {
			sb.WriteString("\n" + i18n.T(lang, g.key) + "\n" + strings.Join(lines, "\n") + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// offerMatch asks whether a new expense is a planned purchase.
func (b *Bot) offerMatch(ctx context.Context, u dbq.User, chatID int64, tx ledger.Transaction) {
	item, err := b.plan.MatchPurchase(ctx, u.ID, tx)
	if err != nil {
		slog.Warn("bot: plan match", "err", err)
		return
	}
	if item == nil {
		return
	}
	kb := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{
		{Text: i18n.T(u.Language, "match.yes"), CallbackData: fmt.Sprintf("pm:%d:%d", item.ID, tx.ID)},
		{Text: i18n.T(u.Language, "match.no"), CallbackData: "pn"},
	}}}
	b.send(ctx, chatID, i18n.T(u.Language, "match.ask", html.EscapeString(item.Title)), kb, 0)
}

// planPrompts invites users to plan next month, once, from the 25th.
func (b *Bot) planPrompts(ctx context.Context) {
	users, err := b.q.UsersDueForPlanPrompt(ctx)
	if err != nil {
		slog.Warn("scheduler: plan prompts", "err", err)
		return
	}
	for _, u := range users {
		y, m, _ := b.now().In(b.loc(u)).Date()
		next := time.Date(y, m, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
		if err := b.q.MarkPlanPrompted(ctx, dbq.MarkPlanPromptedParams{Month: &next, ID: u.ID}); err != nil {
			continue
		}
		kb := &telegram.InlineKeyboardMarkup{InlineKeyboard: [][]telegram.InlineKeyboardButton{{{Text: i18n.T(u.Language, "plan.start_btn"), CallbackData: "pl"}}}}
		if _, err := b.tg.SendMessage(ctx, telegram.SendMessage{ChatID: u.TelegramID, Text: i18n.T(u.Language, "plan.monthly", monthName(u.Language, next)), ParseMode: "HTML", ReplyMarkup: kb}); err != nil {
			slog.Warn("scheduler: plan prompt", "err", err)
			continue
		}
		b.capture.Event(ctx, u.ID, "plan_prompted", nil)
	}
}
