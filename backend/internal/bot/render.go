package bot

import (
	"context"
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/i18n"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
)

type kb = telegram.InlineKeyboardMarkup
type btn = telegram.InlineKeyboardButton

// render turns a capture outcome into a message: a confirmation card or one question.
func (b *Bot) render(ctx context.Context, u dbq.User, o capture.Outcome) (string, *kb) {
	lang := u.Language
	it := o.Item
	label := b.itemLabel(ctx, u, it)
	switch o.Kind {
	case capture.Saved:
		text := b.card(ctx, u, o.Tx, o.NewAccount)
		kbd := b.cardKeyboard(u, o.Tx)
		if o.BankBalance != nil {
			text += "\n" + i18n.T(lang, "bank.mismatch", i18n.Money(lang, *o.BankBalance, it.Currency), i18n.Money(lang, o.AppBalance, it.Currency))
			kbd.InlineKeyboard = append(kbd.InlineKeyboard, []btn{{Text: i18n.T(lang, "btn.reconcile"), CallbackData: fmt.Sprintf("rc:%d", it.AccountID)}})
		}
		return text, kbd

	case capture.AskCard:
		rows := accountRows(lang, o.Accounts, func(id int64) string { return fmt.Sprintf("bc:%d:%d", o.DraftID, id) })
		rows = append(rows, []btn{{Text: i18n.T(lang, "btn.new_card", it.Bank.CardMask), CallbackData: fmt.Sprintf("bc:%d:0", o.DraftID)}})
		rows = append(rows, []btn{{Text: i18n.T(lang, "btn.cancel"), CallbackData: fmt.Sprintf("dx:%d:card", o.DraftID)}})
		return i18n.T(lang, "ask.card", it.Bank.CardMask), &kb{InlineKeyboard: rows}

	case capture.AskBankKind:
		amount := "<b>" + i18n.Money(lang, it.AmountMinor, it.Currency) + "</b>"
		if it.Bank.Direction == "credit" {
			return i18n.T(lang, "ask.bank_in", amount, label), &kb{InlineKeyboard: [][]btn{
				{{Text: i18n.T(lang, "btn.income"), CallbackData: fmt.Sprintf("bk:%d:income", o.DraftID)}, {Text: i18n.T(lang, "btn.own_in"), CallbackData: fmt.Sprintf("bk:%d:own", o.DraftID)}},
				{{Text: i18n.T(lang, "btn.cancel"), CallbackData: fmt.Sprintf("dx:%d:bank_kind", o.DraftID)}},
			}}
		}
		return i18n.T(lang, "ask.bank_out", amount, label), &kb{InlineKeyboard: [][]btn{
			{{Text: i18n.T(lang, "btn.expense"), CallbackData: fmt.Sprintf("bk:%d:expense", o.DraftID)}, {Text: i18n.T(lang, "btn.own_out"), CallbackData: fmt.Sprintf("bk:%d:own", o.DraftID)}},
			{{Text: i18n.T(lang, "btn.cancel"), CallbackData: fmt.Sprintf("dx:%d:bank_kind", o.DraftID)}},
		}}

	case capture.AskTransferFrom:
		q := "ask.from"
		if it.Bank.Direction == "debit" {
			q = "ask.to"
		}
		rows := accountRows(lang, o.Accounts, func(id int64) string { return fmt.Sprintf("bf:%d:%d", o.DraftID, id) })
		rows = append(rows, []btn{{Text: i18n.T(lang, "btn.untracked"), CallbackData: fmt.Sprintf("bf:%d:0", o.DraftID)}})
		return i18n.T(lang, q), &kb{InlineKeyboard: rows}

	case capture.AskAmount:
		as := i18n.Money(lang, it.AmountMinor, it.Currency)
		thousands := i18n.Money(lang, it.AmountMinor*1000, it.Currency)
		return i18n.T(lang, "ask.amount", label, "<b>"+thousands+"</b>", as), &kb{InlineKeyboard: [][]btn{
			{{Text: thousands, CallbackData: fmt.Sprintf("dm:%d:1", o.DraftID)}, {Text: as, CallbackData: fmt.Sprintf("dm:%d:0", o.DraftID)}},
			{{Text: i18n.T(lang, "btn.cancel"), CallbackData: fmt.Sprintf("dx:%d:amount", o.DraftID)}},
		}}

	case capture.AskAccount:
		rows := accountRows(lang, o.Accounts, func(id int64) string { return fmt.Sprintf("da:%d:%d", o.DraftID, id) })
		rows = append(rows, []btn{{Text: i18n.T(lang, "btn.cancel"), CallbackData: fmt.Sprintf("dx:%d:account", o.DraftID)}})
		return i18n.T(lang, "ask.account", label, "<b>"+i18n.Money(lang, it.AmountMinor, it.Currency)+"</b>"), &kb{InlineKeyboard: rows}

	case capture.AskDuplicate:
		amount := "<b>" + i18n.Money(lang, it.AmountMinor, it.Currency) + "</b>"
		at := o.Duplicate.CreatedAt.In(b.loc(u)).Format("15:04")
		prev := html.EscapeString(o.Duplicate.Description)
		if prev == "" {
			prev = "—"
		}
		return i18n.T(lang, "ask.duplicate", amount, at, prev), &kb{InlineKeyboard: [][]btn{{
			{Text: i18n.T(lang, "btn.merge"), CallbackData: fmt.Sprintf("dd:%d:0", o.DraftID)},
			{Text: i18n.T(lang, "btn.keep"), CallbackData: fmt.Sprintf("dd:%d:1", o.DraftID)},
		}}}
	}
	return i18n.T(lang, "error"), nil
}

func (b *Bot) loc(u dbq.User) *time.Location {
	if l, err := time.LoadLocation(u.Timezone); err == nil {
		return l
	}
	return time.UTC
}

// itemLabel is how an item is named in a question: its description, else its category.
func (b *Bot) itemLabel(ctx context.Context, u dbq.User, it capture.Item) string {
	if it.Description != "" {
		return html.EscapeString(it.Description)
	}
	if it.CategoryKey != "" {
		return i18n.Category(u.Language, it.CategoryKey)
	}
	return "—"
}

// card is the confirmation shown after saving:
//
//	🚕 Taksi · 25 000 so'm
//	💳 Humo · bugun
//	taksiga
func (b *Bot) card(ctx context.Context, u dbq.User, tx ledger.Transaction, newAccount string) string {
	lang := u.Language
	icon, name := "📦", ""
	if tx.CategoryID != nil {
		if c, ok := b.category(ctx, u, *tx.CategoryID); ok {
			icon = c.Icon
			name = categoryName(lang, c)
		}
	}
	amount := i18n.Money(lang, tx.AmountMinor, tx.Currency)
	if tx.Type == ledger.Income {
		amount = "+" + amount
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s <b>%s</b> · <b>%s</b>\n", icon, html.EscapeString(name), amount)
	fmt.Fprintf(&sb, "💳 %s · %s", html.EscapeString(b.accountName(ctx, u, tx.AccountID, tx.Currency)), b.dayLabel(u, tx.OccurredOn))
	if tx.Description != "" && !strings.EqualFold(tx.Description, name) {
		fmt.Fprintf(&sb, "\n<i>%s</i>", html.EscapeString(tx.Description))
	}
	if newAccount != "" {
		sb.WriteString("\n" + i18n.T(lang, "card.new_account", html.EscapeString(newAccount)))
	}
	return sb.String()
}

// oneLine is a short reference to a transaction: "Taksi · 25 000 so'm".
func (b *Bot) oneLine(ctx context.Context, u dbq.User, tx ledger.Transaction) string {
	name := tx.Description
	if tx.CategoryID != nil {
		if c, ok := b.category(ctx, u, *tx.CategoryID); ok {
			name = categoryName(u.Language, c)
		}
	}
	return html.EscapeString(name) + " · " + i18n.Money(u.Language, tx.AmountMinor, tx.Currency)
}

func (b *Bot) dayLabel(u dbq.User, d time.Time) string {
	y, m, day := b.now().In(b.loc(u)).Date()
	today := time.Date(y, m, day, 0, 0, 0, 0, time.UTC)
	switch {
	case d.Equal(today):
		return i18n.T(u.Language, "card.today")
	case d.Equal(today.AddDate(0, 0, -1)):
		return i18n.T(u.Language, "card.yesterday")
	}
	return d.Format("02.01.2006")
}

func (b *Bot) cardKeyboard(u dbq.User, tx ledger.Transaction) *kb {
	lang := u.Language
	rows := [][]btn{{
		{Text: i18n.T(lang, "btn.category"), CallbackData: fmt.Sprintf("c:%d", tx.ID)},
		{Text: i18n.T(lang, "btn.account"), CallbackData: fmt.Sprintf("a:%d", tx.ID)},
	}}
	last := []btn{{Text: i18n.T(lang, "btn.undo"), CallbackData: fmt.Sprintf("u:%d", tx.ID)}}
	if app := b.appButton(lang, fmt.Sprintf("/tx/%d", tx.ID)); app != nil {
		app.Text = i18n.T(lang, "btn.edit")
		last = append([]btn{*app}, last...)
	}
	return &kb{InlineKeyboard: append(rows, last)}
}

func (b *Bot) categoryKeyboard(ctx context.Context, u dbq.User, tx ledger.Transaction) *kb {
	cats, err := b.capture.Categories(ctx, u, string(tx.Type))
	if err != nil {
		return nil
	}
	var rows [][]btn
	var row []btn
	for _, c := range cats {
		row = append(row, btn{Text: c.Icon + " " + categoryName(u.Language, c), CallbackData: fmt.Sprintf("cs:%d:%d", tx.ID, c.ID)})
		if len(row) == 2 {
			rows, row = append(rows, row), nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	rows = append(rows, []btn{{Text: i18n.T(u.Language, "btn.back"), CallbackData: fmt.Sprintf("b:%d", tx.ID)}})
	return &kb{InlineKeyboard: rows}
}

func (b *Bot) accountKeyboard(ctx context.Context, u dbq.User, currency string, data func(int64) string, back string) *kb {
	accs, err := b.capture.Accounts(ctx, u, currency)
	if err != nil {
		return nil
	}
	rows := accountRows(u.Language, accs, data)
	rows = append(rows, []btn{{Text: i18n.T(u.Language, "btn.back"), CallbackData: back}})
	return &kb{InlineKeyboard: rows}
}

func accountRows(lang string, accs []dbq.ListAccountsForCaptureRow, data func(int64) string) [][]btn {
	var rows [][]btn
	var row []btn
	for _, a := range accs {
		row = append(row, btn{Text: a.Name, CallbackData: data(a.ID)})
		if len(row) == 3 {
			rows, row = append(rows, row), nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	return rows
}

func categoryName(lang string, c dbq.Category) string {
	if c.Name != nil {
		return *c.Name
	}
	if c.Key != nil {
		return i18n.Category(lang, *c.Key)
	}
	return ""
}

func (b *Bot) category(ctx context.Context, u dbq.User, id int64) (dbq.Category, bool) {
	c, err := b.q.GetCategory(ctx, dbq.GetCategoryParams{ID: id, UserID: &u.ID})
	return c, err == nil
}

func (b *Bot) accountName(ctx context.Context, u dbq.User, id int64, currency string) string {
	accs, err := b.capture.Accounts(ctx, u, currency)
	if err == nil {
		for _, a := range accs {
			if a.ID == id {
				return a.Name
			}
		}
	}
	return "—"
}
