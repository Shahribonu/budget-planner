// Package ledger is the only writer of money data. Every input source (Mini App, chat,
// voice, bank messages) ends up here; it validates, builds balanced postings and commits
// them in one database transaction.
package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

type Type string

const (
	Expense  Type = "expense"
	Income   Type = "income"
	Transfer Type = "transfer"
	Exchange Type = "exchange"
	Opening  Type = "opening"
)

const (
	maxAmount      = 1_000_000_000_000_000 // minor units; far above any real balance
	maxDescription = 200
	maxAccountName = 64
)

var ErrNotFound = errors.New("not found")

// ValidationError is a user-fixable problem with the input.
type ValidationError struct {
	Code    string
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func invalid(code, field, msg string) error {
	return &ValidationError{Code: code, Field: field, Message: msg}
}

var (
	accountTypes = map[string]bool{"cash": true, "humo": true, "uzcard": true, "visa": true, "mastercard": true, "savings": true, "other": true}
	currencies   = map[string]bool{"UZS": true, "USD": true}
	sources      = map[string]bool{"manual": true, "chat": true, "voice": true, "bank": true, "receipt": true}
)

type Service struct {
	pool *pgxpool.Pool
	now  func() time.Time
}

func New(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool, now: time.Now}
}

func (s *Service) inTx(ctx context.Context, fn func(q *dbq.Queries) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error { return fn(dbq.New(tx)) })
}

// today is the current date in the user's time zone.
func (s *Service) today(u dbq.User) time.Time {
	loc, err := time.LoadLocation(u.Timezone)
	if err != nil {
		loc = time.UTC
	}
	y, m, d := s.now().In(loc).Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// ---- accounts ----

type Account struct {
	ID           int64
	Name         string
	Type         string
	Currency     string
	BalanceMinor int64
	Archived     bool
}

type NewAccount struct {
	Name                string
	Type                string
	Currency            string
	OpeningBalanceMinor int64
}

func (s *Service) CreateAccount(ctx context.Context, userID int64, in NewAccount) (Account, error) {
	in.Name = strings.TrimSpace(in.Name)
	switch {
	case in.Name == "" || utf8.RuneCountInString(in.Name) > maxAccountName:
		return Account{}, invalid("invalid_name", "name", "name must be 1–64 characters")
	case !accountTypes[in.Type]:
		return Account{}, invalid("invalid_type", "type", "unknown account type")
	case !currencies[in.Currency]:
		return Account{}, invalid("invalid_currency", "currency", "currency must be UZS or USD")
	case abs(in.OpeningBalanceMinor) > maxAmount:
		return Account{}, invalid("invalid_amount", "opening_balance_minor", "amount too large")
	}

	var out Account
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		user, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		a, err := q.CreateAssetAccount(ctx, dbq.CreateAssetAccountParams{UserID: userID, Type: &in.Type, Name: in.Name, Currency: in.Currency})
		if err != nil {
			return err
		}
		if in.OpeningBalanceMinor != 0 {
			tx, err := q.InsertTransaction(ctx, dbq.InsertTransactionParams{
				UserID: userID, Type: string(Opening), OccurredOn: s.today(user), Source: "manual",
			})
			if err != nil {
				return err
			}
			legs := legsFor(Opening, in.OpeningBalanceMinor, account{a.ID, a.Currency}, nil, 0)
			if err := writeLegs(ctx, q, userID, tx.ID, legs); err != nil {
				return err
			}
		}
		out = Account{ID: a.ID, Name: a.Name, Type: in.Type, Currency: a.Currency, BalanceMinor: in.OpeningBalanceMinor}
		return audit(ctx, q, userID, "account", a.ID, "create", nil, out)
	})
	return out, err
}

func (s *Service) ListAccounts(ctx context.Context, userID int64, includeArchived bool) ([]Account, error) {
	rows, err := dbq.New(s.pool).ListAssetAccounts(ctx, dbq.ListAssetAccountsParams{UserID: userID, IncludeArchived: includeArchived})
	if err != nil {
		return nil, err
	}
	out := make([]Account, len(rows))
	for i, r := range rows {
		out[i] = Account{ID: r.ID, Name: r.Name, Type: deref(r.Type), Currency: r.Currency, BalanceMinor: r.Balance, Archived: r.ArchivedAt != nil}
	}
	return out, nil
}

type AccountPatch struct {
	Name     *string
	Archived *bool
}

func (s *Service) UpdateAccount(ctx context.Context, userID, id int64, p AccountPatch) (Account, error) {
	if p.Name != nil {
		n := strings.TrimSpace(*p.Name)
		if n == "" || utf8.RuneCountInString(n) > maxAccountName {
			return Account{}, invalid("invalid_name", "name", "name must be 1–64 characters")
		}
		p.Name = &n
	}
	var out Account
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		a, err := q.UpdateAssetAccount(ctx, dbq.UpdateAssetAccountParams{Name: p.Name, Archived: p.Archived, ID: id, UserID: userID})
		if err != nil {
			return notFound(err)
		}
		bal, err := q.AccountBalance(ctx, a.ID)
		if err != nil {
			return err
		}
		out = Account{ID: a.ID, Name: a.Name, Type: deref(a.Type), Currency: a.Currency, BalanceMinor: bal, Archived: a.ArchivedAt != nil}
		return audit(ctx, q, userID, "account", a.ID, "update", nil, out)
	})
	return out, err
}

// ---- transactions ----

type Input struct {
	Type          Type
	OccurredOn    time.Time
	AmountMinor   int64
	AccountID     int64
	ToAccountID   *int64
	ToAmountMinor *int64
	CategoryID    *int64
	Description   string
	Source        string
	SourceRef     *string
}

type Transaction struct {
	ID            int64
	Type          Type
	OccurredOn    time.Time
	AmountMinor   int64
	Currency      string
	AccountID     int64
	ToAccountID   *int64
	ToAmountMinor *int64
	ToCurrency    *string
	CategoryID    *int64
	Description   string
	Source        string
	CreatedAt     time.Time
	DebtID        *int64 // debt and debt_payment
	PlanID        *int64 // installment and installment_payment
}

// Record writes a new transaction. If in.SourceRef was already recorded for this source,
// the existing transaction is returned with created=false, so retries are safe.
func (s *Service) Record(ctx context.Context, userID int64, in Input) (tx Transaction, created bool, err error) {
	if in.Source == "" {
		in.Source = "manual"
	}
	err = s.inTx(ctx, func(q *dbq.Queries) error {
		user, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		legs, err := s.validate(ctx, q, user, &in)
		if err != nil {
			return err
		}
		row, err := q.InsertTransaction(ctx, dbq.InsertTransactionParams{
			UserID: userID, Type: string(in.Type), OccurredOn: in.OccurredOn, CategoryID: in.CategoryID,
			Description: in.Description, Source: in.Source, SourceRef: in.SourceRef,
		})
		if errors.Is(err, pgx.ErrNoRows) { // source_ref seen before
			row, err = q.GetTransactionBySourceRef(ctx, dbq.GetTransactionBySourceRefParams{UserID: userID, Source: in.Source, SourceRef: in.SourceRef})
			if err != nil {
				return err
			}
			tx, err = load(ctx, q, row)
			return err
		}
		if err != nil {
			return err
		}
		if err := writeLegs(ctx, q, userID, row.ID, legs); err != nil {
			return err
		}
		if tx, err = load(ctx, q, row); err != nil {
			return err
		}
		created = true
		return audit(ctx, q, userID, "transaction", row.ID, "create", nil, tx)
	})
	return tx, created, err
}

// Update replaces a transaction's details and postings.
func (s *Service) Update(ctx context.Context, userID, id int64, in Input) (Transaction, error) {
	var out Transaction
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		row, err := q.GetTransaction(ctx, dbq.GetTransactionParams{ID: id, UserID: userID})
		if err != nil {
			return notFound(err)
		}
		switch Type(row.Type) {
		case Opening:
			return invalid("opening_readonly", "type", "opening balances cannot be edited")
		case DebtTx, DebtPaymentTx, InstallmentTx, InstallmentPaymentTx:
			return invalid("managed_elsewhere", "type", "edit this in its debt or installment plan")
		}
		before, err := load(ctx, q, row)
		if err != nil {
			return err
		}
		user, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		legs, err := s.validate(ctx, q, user, &in)
		if err != nil {
			return err
		}
		row, err = q.UpdateTransaction(ctx, dbq.UpdateTransactionParams{
			Type: string(in.Type), OccurredOn: in.OccurredOn, CategoryID: in.CategoryID,
			Description: in.Description, ID: id, UserID: userID,
		})
		if err != nil {
			return err
		}
		if err := q.DeletePostings(ctx, id); err != nil {
			return err
		}
		if err := writeLegs(ctx, q, userID, id, legs); err != nil {
			return err
		}
		if out, err = load(ctx, q, row); err != nil {
			return err
		}
		return audit(ctx, q, userID, "transaction", id, "update", before, out)
	})
	return out, err
}

// Delete hides a transaction from balances and reports; Restore undoes it.
func (s *Service) Delete(ctx context.Context, userID, id int64) error {
	return s.setDeleted(ctx, userID, id, true)
}

func (s *Service) Restore(ctx context.Context, userID, id int64) (Transaction, error) {
	if err := s.setDeleted(ctx, userID, id, false); err != nil {
		return Transaction{}, err
	}
	return s.Get(ctx, userID, id)
}

func (s *Service) setDeleted(ctx context.Context, userID, id int64, deleted bool) error {
	return s.inTx(ctx, func(q *dbq.Queries) error {
		row, err := q.GetTransaction(ctx, dbq.GetTransactionParams{ID: id, UserID: userID, WithDeleted: true})
		if err != nil {
			return notFound(err)
		}
		if t := Type(row.Type); deleted && (t == DebtTx || t == InstallmentTx) {
			return invalid("managed_elsewhere", "type", "delete the debt or installment plan instead")
		}
		if (row.DeletedAt != nil) == deleted {
			if deleted {
				return ErrNotFound
			}
			return nil
		}
		if _, err := q.SetTransactionDeleted(ctx, dbq.SetTransactionDeletedParams{Deleted: deleted, ID: id, UserID: userID}); err != nil {
			return err
		}
		action := "delete"
		if !deleted {
			action = "restore"
		}
		return audit(ctx, q, userID, "transaction", id, action, nil, nil)
	})
}

func (s *Service) Get(ctx context.Context, userID, id int64) (Transaction, error) {
	q := dbq.New(s.pool)
	row, err := q.GetTransaction(ctx, dbq.GetTransactionParams{ID: id, UserID: userID})
	if err != nil {
		return Transaction{}, notFound(err)
	}
	return load(ctx, q, row)
}

// validate checks ownership and consistency and returns the postings to write.
// It normalizes in: trims the description, clears fields the type does not use.
func (s *Service) validate(ctx context.Context, q *dbq.Queries, user dbq.User, in *Input) ([]leg, error) {
	switch in.Type {
	case Expense, Income, Transfer, Exchange:
	default:
		return nil, invalid("invalid_type", "type", "type must be expense, income, transfer or exchange")
	}
	if in.AmountMinor <= 0 || in.AmountMinor > maxAmount {
		return nil, invalid("invalid_amount", "amount_minor", "amount must be positive")
	}
	if !sources[in.Source] {
		return nil, invalid("invalid_source", "source", "unknown source")
	}
	if in.SourceRef != nil && *in.SourceRef == "" {
		in.SourceRef = nil
	}
	in.Description = strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(in.Description) > maxDescription {
		return nil, invalid("invalid_description", "description", "description is too long")
	}
	today := s.today(user)
	if in.OccurredOn.Year() < 2000 || in.OccurredOn.After(today.AddDate(0, 0, 1)) {
		return nil, invalid("invalid_date", "occurred_on", "date is out of range")
	}

	from, err := s.ownAccount(ctx, q, user.ID, in.AccountID, "account_id")
	if err != nil {
		return nil, err
	}

	var to *account
	var toAmount int64
	switch in.Type {
	case Expense, Income:
		in.ToAccountID, in.ToAmountMinor = nil, nil
		if in.CategoryID == nil {
			return nil, invalid("category_required", "category_id", "category is required")
		}
		c, err := q.GetCategory(ctx, dbq.GetCategoryParams{ID: *in.CategoryID, UserID: &user.ID})
		if err != nil || c.Kind != string(in.Type) {
			return nil, invalid("invalid_category", "category_id", "unknown category for this type")
		}
	case Transfer, Exchange:
		in.CategoryID = nil
		if in.ToAccountID == nil {
			return nil, invalid("to_account_required", "to_account_id", "destination account is required")
		}
		if *in.ToAccountID == in.AccountID {
			return nil, invalid("same_account", "to_account_id", "choose two different accounts")
		}
		t, err := s.ownAccount(ctx, q, user.ID, *in.ToAccountID, "to_account_id")
		if err != nil {
			return nil, err
		}
		to = &t
		if in.Type == Transfer {
			in.ToAmountMinor = nil
			if to.currency != from.currency {
				return nil, invalid("currency_mismatch", "to_account_id", "accounts have different currencies; use exchange")
			}
		} else {
			if to.currency == from.currency {
				return nil, invalid("same_currency", "to_account_id", "accounts have the same currency; use transfer")
			}
			if in.ToAmountMinor == nil || *in.ToAmountMinor <= 0 || *in.ToAmountMinor > maxAmount {
				return nil, invalid("invalid_amount", "to_amount_minor", "received amount must be positive")
			}
			toAmount = *in.ToAmountMinor
		}
	}
	return legsFor(in.Type, in.AmountMinor, from, to, toAmount), nil
}

func (s *Service) ownAccount(ctx context.Context, q *dbq.Queries, userID, id int64, field string) (account, error) {
	a, err := q.GetAssetAccount(ctx, dbq.GetAssetAccountParams{ID: id, UserID: userID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return account{}, invalid("invalid_account", field, "unknown account")
		}
		return account{}, err
	}
	if a.ArchivedAt != nil {
		return account{}, invalid("archived_account", field, "account is archived")
	}
	return account{a.ID, a.Currency}, nil
}

func writeLegs(ctx context.Context, q *dbq.Queries, userID, txID int64, legs []leg) error {
	for _, l := range legs {
		id := l.asset
		if id == 0 {
			var err error
			id, err = q.EnsureSystemAccount(ctx, dbq.EnsureSystemAccountParams{
				UserID: userID, Kind: systemKind[l.system], SystemKey: &l.system, Currency: l.currency,
			})
			if err != nil {
				return err
			}
		}
		if err := q.InsertPosting(ctx, dbq.InsertPostingParams{TransactionID: txID, AccountID: id, Amount: l.amount, Currency: l.currency}); err != nil {
			return err
		}
	}
	return nil
}

// load reads postings and shapes a transaction for callers.
func load(ctx context.Context, q *dbq.Queries, row dbq.Transaction) (Transaction, error) {
	out, err := loadMany(ctx, q, []dbq.Transaction{row})
	if err != nil {
		return Transaction{}, err
	}
	return out[0], nil
}

func loadMany(ctx context.Context, q *dbq.Queries, rows []dbq.Transaction) ([]Transaction, error) {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	ps, err := q.ListPostings(ctx, ids)
	if err != nil {
		return nil, err
	}
	byTx := map[int64][]dbq.ListPostingsRow{}
	liability := map[int64]dbq.ListPostingsRow{} // the debt or plan leg, by transaction
	for _, p := range ps {
		switch {
		case p.DebtID != nil || p.PlanID != nil:
			liability[p.TransactionID] = p
		case p.Kind == "asset":
			byTx[p.TransactionID] = append(byTx[p.TransactionID], p)
		}
	}
	out := make([]Transaction, len(rows))
	for i, r := range rows {
		t := Transaction{
			ID: r.ID, Type: Type(r.Type), OccurredOn: r.OccurredOn, CategoryID: r.CategoryID,
			Description: r.Description, Source: r.Source, CreatedAt: r.CreatedAt,
		}
		assets := byTx[r.ID]
		if l, ok := liability[r.ID]; ok {
			// Debts and installments: amount is the change in what is owed; the asset
			// leg, when there is one, is the account money moved through.
			t.DebtID, t.PlanID = l.DebtID, l.PlanID
			t.AmountMinor, t.Currency = abs(l.Amount), l.Currency
			if len(assets) == 1 {
				t.AccountID = assets[0].AccountID
			}
			out[i] = t
			continue
		}
		switch t.Type {
		case Transfer, Exchange:
			for _, p := range assets {
				if p.Amount < 0 {
					t.AccountID, t.AmountMinor, t.Currency = p.AccountID, -p.Amount, p.Currency
				} else {
					t.ToAccountID, t.ToAmountMinor, t.ToCurrency = &p.AccountID, &p.Amount, &p.Currency
				}
			}
			if t.Type == Transfer {
				t.ToAmountMinor, t.ToCurrency = nil, nil
			}
		default:
			if len(assets) != 1 {
				return nil, fmt.Errorf("transaction %d: expected one asset posting, got %d", r.ID, len(assets))
			}
			p := assets[0]
			t.AccountID, t.AmountMinor, t.Currency = p.AccountID, p.Amount, p.Currency
			if t.Type == Expense {
				t.AmountMinor = -p.Amount
			}
		}
		out[i] = t
	}
	return out, nil
}

func audit(ctx context.Context, q *dbq.Queries, userID int64, entity string, id int64, action string, before, after any) error {
	var b, a []byte
	if before != nil {
		b, _ = json.Marshal(before)
	}
	if after != nil {
		a, _ = json.Marshal(after)
	}
	return q.InsertAudit(ctx, dbq.InsertAuditParams{UserID: userID, Entity: entity, EntityID: id, Action: action, Before: b, After: a})
}

func notFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func abs(n int64) int64 {
	if n < 0 {
		return -n
	}
	return n
}
