package capture

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Shahribonu/budget-planner/backend/internal/capture/bank"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/categorize"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/i18n"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
)

// BankInfo carries a bank notification through the draft questions.
type BankInfo struct {
	Bank      string `json:"bank"`
	Kind      string `json:"kind"`
	Direction string `json:"direction"`
	CardMask  string `json:"card_mask"`
	Merchant  string `json:"merchant"`
	Balance   *int64 `json:"balance,omitempty"`
	At        string `json:"at,omitempty"` // RFC 3339
	// Choice: expense, income, own (transfer with one of the user's accounts), untracked
	// (from or to an account the user doesn't track).
	Choice      string `json:"choice,omitempty"`
	OtherID     int64  `json:"other_id,omitempty"` // the user's other account in an own transfer
	DuplicateOf int64  `json:"duplicate_of,omitempty"`
}

type bankStage int

const (
	bankCard bankStage = iota
	bankKind
	bankFrom
	bankDuplicate
	bankCommit
)

// Payment services whose name says nothing about what was bought: never learned as a rule.
var genericMerchants = map[string]bool{"click": true, "click qr": true, "payme": true, "apelsin": true, "uzum": true, "upay": true, "oson": true, "alif u2h": true}

// CaptureBank handles a forwarded bank notification.
func (s *Service) CaptureBank(ctx context.Context, u dbq.User, n bank.Notification) (Outcome, error) {
	at := n.At
	if at.IsZero() {
		at = s.now()
	}
	y, m, d := at.In(s.loc(u)).Date()
	name := bank.MerchantName(n.Merchant)
	item := Item{
		Type:        ledger.Expense,
		AmountMinor: n.Amount.Minor,
		Currency:    string(n.Amount.Currency),
		Description: name,
		OccurredOn:  time.Date(y, m, d, 0, 0, 0, 0, time.UTC).Format(time.DateOnly),
		SourceRef:   fmt.Sprintf("%s:%s:%d:%d:%s", n.Bank, n.CardMask, at.Unix(), n.Amount.Minor, n.Direction),
		Bank: &BankInfo{
			Bank: n.Bank, Kind: string(n.Kind), Direction: string(n.Direction), CardMask: n.CardMask,
			Merchant: n.Merchant, At: at.Format(time.RFC3339),
		},
	}
	if !genericMerchants[categorize.Pattern(name)] {
		item.Pattern = categorize.Pattern(name)
	}
	if n.Balance != nil && n.Balance.Currency == n.Amount.Currency {
		b := n.Balance.Minor
		item.Bank.Balance = &b
	}
	switch n.Kind {
	case bank.Payment, bank.Fee:
		item.Bank.Choice = "expense"
	case bank.Refund:
		item.Bank.Choice = "income"
	}
	s.event(ctx, u.ID, "bank_received", map[string]any{"bank": n.Bank, "kind": string(n.Kind)})
	return s.advanceBank(ctx, u, item, bankCard)
}

func (s *Service) advanceBank(ctx context.Context, u dbq.User, item Item, st bankStage) (Outcome, error) {
	o := Outcome{Item: item}
	b := item.Bank
	debit := b.Direction == string(bank.Debit)

	if st <= bankCard && item.AccountID == 0 {
		a, err := s.q.GetAccountByCard(ctx, dbq.GetAccountByCardParams{UserID: u.ID, CardMask: &b.CardMask})
		switch {
		case err == nil && a.Currency == item.Currency:
			item.AccountID = a.ID
		case err == nil || errors.Is(err, pgx.ErrNoRows):
			o.Accounts, err = s.Accounts(ctx, u, item.Currency)
			if err != nil {
				return o, err
			}
			return s.ask(ctx, u, o, "card", AskCard, "bank")
		default:
			return o, err
		}
	}

	if st <= bankKind && b.Choice == "" {
		if b.Kind == string(bank.Withdrawal) {
			cash, name, err := s.cashAccount(ctx, u, item.Currency)
			if err != nil {
				return o, err
			}
			b.Choice, b.OtherID, o.NewAccount = "own", cash, name
		} else {
			o.Item = item
			return s.ask(ctx, u, o, "bank_kind", AskBankKind, "bank")
		}
	}

	if st <= bankFrom && b.Choice == "own" && b.OtherID == 0 {
		accs, err := s.Accounts(ctx, u, item.Currency)
		if err != nil {
			return o, err
		}
		for _, a := range accs {
			if a.ID != item.AccountID {
				o.Accounts = append(o.Accounts, a)
			}
		}
		o.Item = item
		return s.ask(ctx, u, o, "transfer_from", AskTransferFrom, "bank")
	}

	if st <= bankDuplicate && b.Choice == "expense" {
		d, err := time.Parse(time.DateOnly, item.OccurredOn)
		if err != nil {
			return o, err
		}
		dup, err := s.q.FindSameDayDuplicate(ctx, dbq.FindSameDayDuplicateParams{UserID: u.ID, OccurredOn: d, Currency: item.Currency, Amount: item.AmountMinor})
		if err == nil {
			b.DuplicateOf = dup.ID
			o.Item = item
			o.Duplicate = dbq.FindPossibleDuplicateRow{ID: dup.ID, CreatedAt: dup.CreatedAt, Description: dup.Description}
			return s.ask(ctx, u, o, "duplicate", AskDuplicate, "bank")
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return o, err
		}
	}

	// commit
	d, err := time.Parse(time.DateOnly, item.OccurredOn)
	if err != nil {
		return o, err
	}
	ref := item.SourceRef
	var tx ledger.Transaction
	var created bool
	switch b.Choice {
	case "expense", "income":
		cats, err := s.categories(ctx, u.ID)
		if err != nil {
			return o, err
		}
		s.categorize(ctx, u, &item, cats, b.Choice == "income")
		tx, created, err = s.ledger.Record(ctx, u.ID, ledger.Input{
			Type: item.Type, OccurredOn: d, AmountMinor: item.AmountMinor, AccountID: item.AccountID,
			CategoryID: &item.CategoryID, Description: item.Description, Source: "bank", SourceRef: &ref,
		})
		if err != nil {
			return o, err
		}
	case "own":
		from, to := b.OtherID, item.AccountID
		if debit {
			from, to = item.AccountID, b.OtherID
		}
		tx, created, err = s.ledger.Record(ctx, u.ID, ledger.Input{
			Type: ledger.Transfer, OccurredOn: d, AmountMinor: item.AmountMinor, AccountID: from, ToAccountID: &to,
			Description: item.Description, Source: "bank", SourceRef: &ref,
		})
		if err != nil {
			return o, err
		}
	case "untracked":
		delta := item.AmountMinor
		if debit {
			delta = -delta
		}
		if tx, err = s.ledger.Adjust(ctx, u.ID, item.AccountID, delta, d, item.Description); err != nil {
			return o, err
		}
		created = true
	default:
		return o, fmt.Errorf("bank item without a choice")
	}
	o.Kind, o.Tx, o.Item = Saved, tx, item
	if created {
		s.event(ctx, u.ID, "log_saved", map[string]any{"source": "bank", "type": string(tx.Type), "category_origin": item.CategoryOrigin})
		o.FirstLog = s.firstLog(ctx, u)
	}
	s.checkBankBalance(ctx, u, &o)
	return o, nil
}

// checkBankBalance stores the bank's balance when this notification is the newest for the
// card, and reports a difference from the app's balance.
func (s *Service) checkBankBalance(ctx context.Context, u dbq.User, o *Outcome) {
	b := o.Item.Bank
	if b.Balance == nil {
		return
	}
	at, err := time.Parse(time.RFC3339, b.At)
	if err != nil {
		return
	}
	if _, err := s.q.RecordBankBalance(ctx, dbq.RecordBankBalanceParams{Balance: b.Balance, At: &at, ID: o.Item.AccountID, UserID: u.ID}); err != nil {
		return // an older notification
	}
	app, err := s.ledger.Balance(ctx, o.Item.AccountID)
	if err == nil && app != *b.Balance {
		o.BankBalance, o.AppBalance = b.Balance, app
	}
}

// cashAccount finds the user's cash account in a currency, creating one if needed.
func (s *Service) cashAccount(ctx context.Context, u dbq.User, currency string) (int64, string, error) {
	accs, err := s.Accounts(ctx, u, currency)
	if err != nil {
		return 0, "", err
	}
	for _, a := range accs {
		if a.Type != nil && *a.Type == "cash" {
			return a.ID, "", nil
		}
	}
	name := i18n.T(u.Language, "acct.default_cash")
	if currency == "USD" {
		name += " $"
	}
	a, err := s.ledger.CreateAccount(ctx, u.ID, ledger.NewAccount{Name: name, Type: "cash", Currency: currency})
	return a.ID, a.Name, err
}

// AnswerCard says which account a card is (0 creates a new one with the bank's balance).
func (s *Service) AnswerCard(ctx context.Context, u dbq.User, draftID, accountID int64) (Outcome, error) {
	item, _, err := s.takeDraft(ctx, u, draftID, "card", "committed")
	if err != nil {
		return Outcome{}, err
	}
	b := item.Bank
	o := Outcome{}
	if accountID == 0 {
		// Opening balance such that, after this transaction, the app matches the bank.
		var opening int64
		if b.Balance != nil {
			opening = *b.Balance + item.AmountMinor
			if b.Direction == string(bank.Credit) {
				opening = *b.Balance - item.AmountMinor
			}
		}
		a, err := s.ledger.CreateAccount(ctx, u.ID, ledger.NewAccount{
			Name: "Humo *" + b.CardMask, Type: "humo", Currency: item.Currency, OpeningBalanceMinor: opening,
		})
		if err != nil {
			return o, err
		}
		accountID, o.NewAccount = a.ID, a.Name
	}
	if err := s.ledger.AssignCard(ctx, u.ID, accountID, b.CardMask); err != nil {
		return o, err
	}
	item.AccountID = accountID
	out, err := s.advanceBank(ctx, u, item, bankKind)
	if out.NewAccount == "" {
		out.NewAccount = o.NewAccount
	}
	return out, err
}

// AnswerBankKind says what money in (or a transfer out) was: expense, income, own or untracked.
func (s *Service) AnswerBankKind(ctx context.Context, u dbq.User, draftID int64, choice string) (Outcome, error) {
	switch choice {
	case "expense", "income", "own":
	default:
		return Outcome{}, ErrDraftGone
	}
	item, _, err := s.takeDraft(ctx, u, draftID, "bank_kind", "committed")
	if err != nil {
		return Outcome{}, err
	}
	item.Bank.Choice = choice
	return s.advanceBank(ctx, u, item, bankFrom)
}

// AnswerTransferFrom names the user's other account in an own transfer; 0 means an
// account the user doesn't track.
func (s *Service) AnswerTransferFrom(ctx context.Context, u dbq.User, draftID, accountID int64) (Outcome, error) {
	item, _, err := s.takeDraft(ctx, u, draftID, "transfer_from", "committed")
	if err != nil {
		return Outcome{}, err
	}
	if accountID == 0 {
		item.Bank.Choice = "untracked"
	} else {
		item.Bank.OtherID = accountID
	}
	return s.advanceBank(ctx, u, item, bankDuplicate)
}

// ReconcileToBank sets an account to the newest balance its bank reported and returns it.
func (s *Service) ReconcileToBank(ctx context.Context, u dbq.User, accountID int64) (int64, string, error) {
	a, err := s.q.GetAssetAccount(ctx, dbq.GetAssetAccountParams{ID: accountID, UserID: u.ID})
	if err != nil || a.BankBalance == nil {
		return 0, "", ledger.ErrNotFound
	}
	y, m, d := s.now().In(s.loc(u)).Date()
	if _, err := s.ledger.Reconcile(ctx, u.ID, accountID, *a.BankBalance, time.Date(y, m, d, 0, 0, 0, 0, time.UTC), "reconcile"); err != nil {
		var ve *ledger.ValidationError
		if errors.As(err, &ve) && ve.Code == "invalid_amount" {
			return *a.BankBalance, a.Currency, nil // already equal
		}
		return 0, "", err
	}
	s.event(ctx, u.ID, "reconciled", nil)
	return *a.BankBalance, a.Currency, nil
}
