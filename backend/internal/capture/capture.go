// Package capture turns chat and voice messages into ledger transactions.
//
// Each amount in a message becomes a pending item that passes three checks in order:
// amount (is "taksi 25" really 25 so'm?), account (which card or cash?) and duplicate
// (was this already logged a few minutes ago?). An item that clears all three is written
// to the ledger at once and shown with an Undo button. An item that fails one is stored as
// a draft with a single question; it touches balances only after the user answers.
package capture

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/capture/categorize"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/llm"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/normalizer"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/i18n"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

var ErrDraftGone = errors.New("draft expired or already answered")

type Kind int

const (
	Saved Kind = iota
	AskAmount
	AskAccount
	AskDuplicate
	Merged // the user said a possible duplicate was the same expense
	AskCard
	AskBankKind
	AskTransferFrom
)

type stage int

const (
	stageAmount stage = iota
	stageAccount
	stageDuplicate
	stageCommit
)

// Item is one captured amount on its way to the ledger. It is the draft payload.
type Item struct {
	Type           ledger.Type `json:"type"`
	AmountMinor    int64       `json:"amount_minor"`
	Currency       string      `json:"currency"`
	Description    string      `json:"description"`
	CategoryID     int64       `json:"category_id"`
	CategoryKey    string      `json:"category_key"`    // "" for custom categories
	CategoryOrigin string      `json:"category_origin"` // rule, user, ai, default
	AccountID      int64       `json:"account_id"`
	OccurredOn     string      `json:"occurred_on"`
	Pattern        string      `json:"pattern"`
	SourceRef      string      `json:"source_ref"`
	NeedsReview    bool        `json:"needs_review"`
	Bank           *BankInfo   `json:"bank,omitempty"` // set for forwarded bank notifications
}

type Outcome struct {
	Kind       Kind
	Item       Item
	Tx         ledger.Transaction // Saved
	DraftID    int64              // Ask*
	Accounts   []dbq.ListAccountsForCaptureRow
	Duplicate  dbq.FindPossibleDuplicateRow
	NewAccount string // an account was created for this item
	FirstLog   bool   // the user's first saved transaction ever
	// Bank notifications: the card's balance as the bank reported it and in the app,
	// set when they differ and this notification is the newest for the card.
	BankBalance *int64
	AppBalance  int64
}

type Service struct {
	pool   *pgxpool.Pool
	q      *dbq.Queries
	ledger *ledger.Service
	ai     *llm.Classifier // nil disables the fallback
	now    func() time.Time
}

func New(pool *pgxpool.Pool, l *ledger.Service, ai *llm.Classifier) *Service {
	return &Service{pool: pool, q: dbq.New(pool), ledger: l, ai: ai, now: time.Now}
}

func (s *Service) loc(u dbq.User) *time.Location {
	if loc, err := time.LoadLocation(u.Timezone); err == nil {
		return loc
	}
	return time.UTC
}

// Capture handles one message. ref identifies the message (e.g. "tg:123:456") and makes
// retries idempotent. source is "chat" or "voice". It returns no outcomes when the text
// holds no amount.
func (s *Service) Capture(ctx context.Context, u dbq.User, text, source, ref string) ([]Outcome, error) {
	now := s.now().In(s.loc(u))
	items := normalizer.Parse(text, normalizer.Options{DefaultCurrency: currencyOf(u), Now: now})
	if len(items) == 0 {
		return nil, nil
	}
	income := strings.HasPrefix(strings.TrimSpace(text), "+")
	cats, err := s.categories(ctx, u.ID)
	if err != nil {
		return nil, err
	}

	var out []Outcome
	for i, it := range items {
		item := Item{
			Type:        ledger.Expense,
			AmountMinor: it.Amount.Minor,
			Currency:    string(it.Amount.Currency),
			Description: it.Description,
			OccurredOn:  it.Date.Format(time.DateOnly),
			Pattern:     categorize.Pattern(it.Description),
			SourceRef:   fmt.Sprintf("%s:%d", ref, i),
			NeedsReview: it.NeedsReview,
		}
		accountHint := s.categorize(ctx, u, &item, cats, income)
		item.AccountID = accountHint
		o, err := s.advance(ctx, u, item, stageAmount, source, ref)
		if err != nil {
			return out, err
		}
		out = append(out, o)
	}
	return out, nil
}

func currencyOf(u dbq.User) money.Currency { return money.Currency(u.MainCurrency) }

type categorySet struct {
	byKey map[string]dbq.Category
	byID  map[int64]dbq.Category
	llm   []llm.Category
}

func (s *Service) categories(ctx context.Context, userID int64) (categorySet, error) {
	cs, err := s.q.ListCategories(ctx, &userID)
	if err != nil {
		return categorySet{}, err
	}
	set := categorySet{byKey: map[string]dbq.Category{}, byID: map[int64]dbq.Category{}}
	for _, c := range cs {
		set.byID[c.ID] = c
		if c.Key != nil {
			set.byKey[*c.Key] = c
			set.llm = append(set.llm, llm.Category{Key: *c.Key, Kind: c.Kind})
		}
	}
	return set, nil
}

// categorize sets the item's category and type: the user's own past choice first, then
// keyword rules, then the model, then "Other". It returns the account remembered for
// this description, if any.
func (s *Service) categorize(ctx context.Context, u dbq.User, item *Item, cats categorySet, income bool) int64 {
	set := func(c dbq.Category, origin string) {
		item.CategoryID, item.CategoryOrigin, item.Type = c.ID, origin, ledger.Type(c.Kind)
		item.CategoryKey = ""
		if c.Key != nil {
			item.CategoryKey = *c.Key
		}
	}
	wantKind := func(kind string) bool { return !income || kind == "income" }

	if item.Pattern != "" {
		r, err := s.q.GetMerchantRule(ctx, dbq.GetMerchantRuleParams{UserID: u.ID, Pattern: item.Pattern})
		if err == nil {
			if c, ok := cats.byID[r.CategoryID]; ok && wantKind(c.Kind) {
				set(c, r.Origin)
				if r.AccountID != nil {
					return *r.AccountID
				}
				return 0
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			slog.Warn("merchant rule lookup", "err", err)
		}
		if m, ok := categorize.Guess(item.Description); ok && wantKind(string(m.Kind)) {
			if c, ok := cats.byKey[m.Key]; ok {
				set(c, "rule")
				return 0
			}
		}
		if s.ai != nil {
			kind := "expense"
			if income {
				kind = "income"
			}
			res, err := s.ai.Classify(ctx, item.Description, kind, cats.llm)
			s.event(ctx, u.ID, "ai_category", map[string]any{
				"hit": res.Key != "", "error": err != nil, "input_tokens": res.InputTokens, "output_tokens": res.OutputTokens,
			})
			if err != nil {
				slog.Warn("ai classify", "err", err)
			} else if c, ok := cats.byKey[res.Key]; ok && c.Kind == kind {
				set(c, "ai")
				_ = s.q.UpsertMerchantRule(ctx, dbq.UpsertMerchantRuleParams{UserID: u.ID, Pattern: item.Pattern, CategoryID: c.ID, Origin: "ai"})
				return 0
			}
		}
	}
	if income {
		set(cats.byKey["other_income"], "default")
	} else {
		set(cats.byKey["other_expense"], "default")
	}
	return 0
}

// advance runs the remaining checks from st and either saves the item or stores a draft
// with one question.
func (s *Service) advance(ctx context.Context, u dbq.User, item Item, st stage, source, ref string) (Outcome, error) {
	o := Outcome{Item: item}

	if st <= stageAmount && item.NeedsReview {
		return s.ask(ctx, u, o, "amount", AskAmount, source)
	}

	if st <= stageAccount {
		accs, err := s.q.ListAccountsForCapture(ctx, dbq.ListAccountsForCaptureParams{UserID: u.ID, Currency: item.Currency})
		if err != nil {
			return o, err
		}
		switch {
		case len(accs) == 0:
			name := i18n.T(u.Language, "acct.default_cash")
			if item.Currency == "USD" {
				name += " $"
			}
			a, err := s.ledger.CreateAccount(ctx, u.ID, ledger.NewAccount{Name: name, Type: "cash", Currency: item.Currency})
			if err != nil {
				return o, err
			}
			item.AccountID, o.NewAccount = a.ID, a.Name
		case containsAccount(accs, item.AccountID):
			// remembered for this description
		case len(accs) == 1:
			item.AccountID = accs[0].ID
		case accs[0].LastUsed.After(time.Unix(0, 0)):
			item.AccountID = accs[0].ID // most recently used; the card offers a switch
		default:
			o.Item, o.Accounts = item, accs
			return s.ask(ctx, u, o, "account", AskAccount, source)
		}
		o.Item = item
	}

	if st <= stageDuplicate && item.Type == ledger.Expense {
		d, err := time.Parse(time.DateOnly, item.OccurredOn)
		if err != nil {
			return o, err
		}
		dup, err := s.q.FindPossibleDuplicate(ctx, dbq.FindPossibleDuplicateParams{
			UserID: u.ID, Type: string(item.Type), OccurredOn: d, Currency: item.Currency,
			Amount: item.AmountMinor, SourcePrefix: ref,
		})
		if err == nil {
			o.Duplicate = dup
			return s.ask(ctx, u, o, "duplicate", AskDuplicate, source)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return o, err
		}
	}

	return s.commit(ctx, u, o, source)
}

func containsAccount(accs []dbq.ListAccountsForCaptureRow, id int64) bool {
	for _, a := range accs {
		if a.ID == id {
			return true
		}
	}
	return false
}

func (s *Service) ask(ctx context.Context, u dbq.User, o Outcome, question string, kind Kind, source string) (Outcome, error) {
	payload, err := json.Marshal(o.Item)
	if err != nil {
		return o, err
	}
	d, err := s.q.CreateDraft(ctx, dbq.CreateDraftParams{UserID: u.ID, Question: question, Payload: payload, Source: source, SourceRef: o.Item.SourceRef})
	if err != nil {
		return o, err
	}
	o.Kind, o.DraftID = kind, d.ID
	s.event(ctx, u.ID, "draft_created", map[string]any{"question": question, "source": source})
	return o, nil
}

func (s *Service) commit(ctx context.Context, u dbq.User, o Outcome, source string) (Outcome, error) {
	d, err := time.Parse(time.DateOnly, o.Item.OccurredOn)
	if err != nil {
		return o, err
	}
	ref := o.Item.SourceRef
	tx, created, err := s.ledger.Record(ctx, u.ID, ledger.Input{
		Type: o.Item.Type, OccurredOn: d, AmountMinor: o.Item.AmountMinor, AccountID: o.Item.AccountID,
		CategoryID: &o.Item.CategoryID, Description: o.Item.Description, Source: source, SourceRef: &ref,
	})
	if err != nil {
		return o, err
	}
	o.Kind, o.Tx = Saved, tx
	if created {
		s.event(ctx, u.ID, "log_saved", map[string]any{"source": source, "type": string(o.Item.Type), "category_origin": o.Item.CategoryOrigin})
		o.FirstLog = s.firstLog(ctx, u)
	}
	return o, nil
}

func (s *Service) firstLog(ctx context.Context, u dbq.User) bool {
	seen, err := s.q.HasEvent(ctx, dbq.HasEventParams{UserID: &u.ID, Name: "first_log"})
	if err != nil || seen {
		return false
	}
	props := map[string]any{}
	if u.StartedAt != nil {
		props["seconds_since_start"] = int(s.now().Sub(*u.StartedAt).Seconds())
	}
	s.event(ctx, u.ID, "first_log", props)
	return true
}

// event records a product analytics event. Failures are logged, never returned:
// analytics must not break capture.
func (s *Service) event(ctx context.Context, userID int64, name string, props map[string]any) {
	b, _ := json.Marshal(props)
	if err := s.q.InsertEvent(ctx, dbq.InsertEventParams{UserID: &userID, Name: name, Props: b}); err != nil {
		slog.Warn("event", "name", name, "err", err)
	}
}

// Event lets other packages record analytics through the same path.
func (s *Service) Event(ctx context.Context, userID int64, name string, props map[string]any) {
	s.event(ctx, userID, name, props)
}

// ---- answers to draft questions ----

// takeDraft locks a pending draft, marks it answered and returns its item.
func (s *Service) takeDraft(ctx context.Context, u dbq.User, id int64, question, status string) (Item, string, error) {
	var item Item
	var source string
	err := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		q := dbq.New(tx)
		d, err := q.GetPendingDraft(ctx, dbq.GetPendingDraftParams{ID: id, UserID: u.ID})
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && d.Question != question) {
			return ErrDraftGone
		}
		if err != nil {
			return err
		}
		if err := json.Unmarshal(d.Payload, &item); err != nil {
			return err
		}
		source = d.Source
		return q.ResolveDraft(ctx, dbq.ResolveDraftParams{ID: id, Status: status})
	})
	if err == nil {
		s.event(ctx, u.ID, "draft_answered", map[string]any{"question": question, "status": status})
	}
	return item, source, err
}

// AnswerAmount confirms "taksi 25" as written (thousands=false) or as 25 000 (thousands=true).
func (s *Service) AnswerAmount(ctx context.Context, u dbq.User, draftID int64, thousands bool) (Outcome, error) {
	item, source, err := s.takeDraft(ctx, u, draftID, "amount", "committed")
	if err != nil {
		return Outcome{}, err
	}
	if thousands {
		item.AmountMinor *= 1000
	}
	item.NeedsReview = false
	return s.advance(ctx, u, item, stageAccount, source, refOf(item))
}

func (s *Service) AnswerAccount(ctx context.Context, u dbq.User, draftID, accountID int64) (Outcome, error) {
	item, source, err := s.takeDraft(ctx, u, draftID, "account", "committed")
	if err != nil {
		return Outcome{}, err
	}
	item.AccountID = accountID
	if item.Pattern != "" {
		_ = s.q.UpsertMerchantRule(ctx, dbq.UpsertMerchantRuleParams{UserID: u.ID, Pattern: item.Pattern, CategoryID: item.CategoryID, AccountID: &accountID, Origin: originForRule(item)})
	}
	return s.advance(ctx, u, item, stageDuplicate, source, refOf(item))
}

// AnswerDuplicate records the item anyway (keep=true) or drops it.
func (s *Service) AnswerDuplicate(ctx context.Context, u dbq.User, draftID int64, keep bool) (Outcome, error) {
	status := "cancelled"
	if keep {
		status = "committed"
	}
	item, source, err := s.takeDraft(ctx, u, draftID, "duplicate", status)
	if err != nil {
		return Outcome{}, err
	}
	if !keep {
		if item.Bank != nil && item.Bank.DuplicateOf != 0 {
			// The bank message confirms an expense logged by hand: keep that one, on the card.
			if _, err := s.ChangeAccount(ctx, u, item.Bank.DuplicateOf, item.AccountID); err != nil {
				return Outcome{}, err
			}
		}
		return Outcome{Kind: Merged, Item: item}, nil
	}
	if item.Bank != nil {
		return s.advanceBank(ctx, u, item, bankCommit)
	}
	return s.commit(ctx, u, Outcome{Item: item}, source)
}

func (s *Service) CancelDraft(ctx context.Context, u dbq.User, draftID int64, question string) error {
	_, _, err := s.takeDraft(ctx, u, draftID, question, "cancelled")
	return err
}

// refOf recovers the message ref ("tg:1:2") from an item ref ("tg:1:2:0").
func refOf(item Item) string {
	if i := strings.LastIndex(item.SourceRef, ":"); i > 0 {
		return item.SourceRef[:i]
	}
	return item.SourceRef
}

func originForRule(item Item) string {
	if item.CategoryOrigin == "ai" {
		return "ai"
	}
	return "user"
}

// ---- fixes to saved transactions ----

// ChangeCategory recategorizes a saved transaction and remembers the choice for its description.
func (s *Service) ChangeCategory(ctx context.Context, u dbq.User, txID, categoryID int64) (ledger.Transaction, error) {
	tx, err := s.ledger.Get(ctx, u.ID, txID)
	if err != nil {
		return tx, err
	}
	cats, err := s.categories(ctx, u.ID)
	if err != nil {
		return tx, err
	}
	c, ok := cats.byID[categoryID]
	if !ok || (tx.Type != ledger.Expense && tx.Type != ledger.Income) {
		return tx, ledger.ErrNotFound
	}
	updated, err := s.ledger.Update(ctx, u.ID, txID, ledger.Input{
		Type: ledger.Type(c.Kind), OccurredOn: tx.OccurredOn, AmountMinor: tx.AmountMinor, AccountID: tx.AccountID,
		CategoryID: &categoryID, Description: tx.Description, Source: tx.Source,
	})
	if err != nil {
		return updated, err
	}
	if p := categorize.Pattern(tx.Description); p != "" {
		acc := tx.AccountID
		if err := s.q.UpsertMerchantRule(ctx, dbq.UpsertMerchantRuleParams{UserID: u.ID, Pattern: p, CategoryID: categoryID, AccountID: &acc, Origin: "user"}); err != nil {
			slog.Warn("learn rule", "err", err)
		}
	}
	s.event(ctx, u.ID, "category_changed", nil)
	return updated, nil
}

// ChangeAccount moves a saved transaction to another account in the same currency.
func (s *Service) ChangeAccount(ctx context.Context, u dbq.User, txID, accountID int64) (ledger.Transaction, error) {
	tx, err := s.ledger.Get(ctx, u.ID, txID)
	if err != nil {
		return tx, err
	}
	if tx.Type != ledger.Expense && tx.Type != ledger.Income {
		return tx, ledger.ErrNotFound
	}
	updated, err := s.ledger.Update(ctx, u.ID, txID, ledger.Input{
		Type: tx.Type, OccurredOn: tx.OccurredOn, AmountMinor: tx.AmountMinor, AccountID: accountID,
		CategoryID: tx.CategoryID, Description: tx.Description, Source: tx.Source,
	})
	if err != nil {
		return updated, err
	}
	if p := categorize.Pattern(tx.Description); p != "" && tx.CategoryID != nil {
		_ = s.q.UpsertMerchantRule(ctx, dbq.UpsertMerchantRuleParams{UserID: u.ID, Pattern: p, CategoryID: *tx.CategoryID, AccountID: &accountID, Origin: "user"})
	}
	s.event(ctx, u.ID, "account_changed", nil)
	return updated, nil
}

func (s *Service) Undo(ctx context.Context, u dbq.User, txID int64) (ledger.Transaction, error) {
	tx, err := s.ledger.Get(ctx, u.ID, txID)
	if err != nil {
		return tx, err
	}
	if err := s.ledger.Delete(ctx, u.ID, txID); err != nil {
		return tx, err
	}
	s.event(ctx, u.ID, "undo", nil)
	return tx, nil
}

// UndoLast deletes the most recent chat or voice transaction.
func (s *Service) UndoLast(ctx context.Context, u dbq.User) (ledger.Transaction, error) {
	last, err := s.q.LastChatTransaction(ctx, u.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ledger.Transaction{}, ledger.ErrNotFound
	}
	if err != nil {
		return ledger.Transaction{}, err
	}
	return s.Undo(ctx, u, last.ID)
}

// Today returns today's expense count and total per currency in the user's time zone.
func (s *Service) Today(ctx context.Context, u dbq.User) ([]dbq.DaySpendingRow, error) {
	y, m, d := s.now().In(s.loc(u)).Date()
	return s.q.DaySpending(ctx, dbq.DaySpendingParams{UserID: u.ID, Day: time.Date(y, m, d, 0, 0, 0, 0, time.UTC)})
}

// Accounts lists active accounts in one currency, most recently used first.
func (s *Service) Accounts(ctx context.Context, u dbq.User, currency string) ([]dbq.ListAccountsForCaptureRow, error) {
	return s.q.ListAccountsForCapture(ctx, dbq.ListAccountsForCaptureParams{UserID: u.ID, Currency: currency})
}

// Categories lists the user's categories of one kind.
func (s *Service) Categories(ctx context.Context, u dbq.User, kind string) ([]dbq.Category, error) {
	cs, err := s.q.ListCategories(ctx, &u.ID)
	if err != nil {
		return nil, err
	}
	var out []dbq.Category
	for _, c := range cs {
		if c.Kind == kind {
			out = append(out, c)
		}
	}
	return out, nil
}

// Ledger exposes the ledger for read-only lookups by the bot.
func (s *Service) Ledger() *ledger.Service { return s.ledger }
