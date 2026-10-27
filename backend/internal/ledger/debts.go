package ledger

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

const (
	DebtTx               Type = "debt"
	DebtPaymentTx        Type = "debt_payment"
	InstallmentTx        Type = "installment"
	InstallmentPaymentTx Type = "installment_payment"
)

// ---- debts (qarz) ----

type Debt struct {
	ID           int64
	Direction    string // lent or borrowed
	Counterparty string
	Currency     string
	Principal    int64 // total lent or borrowed
	Remaining    int64 // still owed, always >= 0
	DueOn        *time.Time
	Note         string
	CreatedAt    time.Time
	OccurredOn   time.Time // when the money was lent or borrowed
	AccountID    *int64    // the account it moved through; nil for a debt from before tracking
}

type NewDebt struct {
	Direction    string
	Counterparty string
	Currency     string
	AmountMinor  int64
	// AccountID is the asset account the money left (lent) or arrived in (borrowed).
	// Nil records a debt that already existed before the user started tracking.
	AccountID  *int64
	OccurredOn time.Time
	DueOn      *time.Time
	Note       string
}

func validName(field, s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" || utf8.RuneCountInString(s) > maxAccountName {
		return "", invalid("invalid_"+field, field, field+" must be 1–64 characters")
	}
	return s, nil
}

// CreateDebt records money lent to or borrowed from someone. Lending moves money from an
// asset into a receivable, so it is neither an expense nor income.
func (s *Service) CreateDebt(ctx context.Context, userID int64, in NewDebt) (Debt, error) {
	var err error
	if in.Direction != "lent" && in.Direction != "borrowed" {
		return Debt{}, invalid("invalid_direction", "direction", "direction must be lent or borrowed")
	}
	if in.Counterparty, err = validName("counterparty", in.Counterparty); err != nil {
		return Debt{}, err
	}
	if !currencies[in.Currency] {
		return Debt{}, invalid("invalid_currency", "currency", "currency must be UZS or USD")
	}
	if in.AmountMinor <= 0 || in.AmountMinor > maxAmount {
		return Debt{}, invalid("invalid_amount", "amount_minor", "amount must be positive")
	}
	if utf8.RuneCountInString(in.Note) > maxDescription {
		return Debt{}, invalid("invalid_note", "note", "note is too long")
	}
	if in.DueOn != nil && in.DueOn.Before(in.OccurredOn) {
		return Debt{}, invalid("invalid_due_date", "due_on", "due date is before the debt date")
	}

	var out Debt
	err = s.inTx(ctx, func(q *dbq.Queries) error {
		user, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		if err := s.checkDate(user, in.OccurredOn); err != nil {
			return err
		}
		var asset *account
		if in.AccountID != nil {
			a, err := s.ownAccount(ctx, q, userID, *in.AccountID, "account_id")
			if err != nil {
				return err
			}
			if a.currency != in.Currency {
				return invalid("currency_mismatch", "account_id", "account currency differs from the debt")
			}
			asset = &a
		}
		kind, sign := "receivable", int64(1)
		if in.Direction == "borrowed" {
			kind, sign = "payable", -1
		}
		acc, err := q.CreateLiabilityAccount(ctx, dbq.CreateLiabilityAccountParams{UserID: userID, Kind: kind, Name: in.Counterparty, Currency: in.Currency})
		if err != nil {
			return err
		}
		d, err := q.InsertDebt(ctx, dbq.InsertDebtParams{
			UserID: userID, Direction: in.Direction, Counterparty: in.Counterparty, AccountID: acc.ID,
			Currency: in.Currency, DueOn: in.DueOn, Note: strings.TrimSpace(in.Note),
		})
		if err != nil {
			return err
		}
		legs := []leg{{asset: acc.ID, amount: sign * in.AmountMinor, currency: in.Currency}}
		if asset != nil {
			legs = append(legs, leg{asset: asset.id, amount: -sign * in.AmountMinor, currency: in.Currency})
		} else {
			legs = append(legs, leg{system: "opening", amount: -sign * in.AmountMinor, currency: in.Currency})
		}
		if err := s.writeTx(ctx, q, userID, DebtTx, in.OccurredOn, nil, in.Counterparty, legs); err != nil {
			return err
		}
		out = toDebt(d.ID, d.Direction, d.Counterparty, d.Currency, d.DueOn, d.Note, d.CreatedAt, sign*in.AmountMinor, in.AmountMinor)
		out.OccurredOn, out.AccountID = in.OccurredOn, in.AccountID
		return audit(ctx, q, userID, "debt", d.ID, "create", nil, out)
	})
	return out, err
}

// writeTx inserts a transaction with the given legs.
func (s *Service) writeTx(ctx context.Context, q *dbq.Queries, userID int64, t Type, on time.Time, categoryID *int64, description string, legs []leg) error {
	tx, err := q.InsertTransaction(ctx, dbq.InsertTransactionParams{
		UserID: userID, Type: string(t), OccurredOn: on, CategoryID: categoryID, Description: description, Source: "manual",
	})
	if err != nil {
		return err
	}
	return writeLegs(ctx, q, userID, tx.ID, legs)
}

func (s *Service) checkDate(u dbq.User, d time.Time) error {
	if d.Year() < 2000 || d.After(s.today(u).AddDate(0, 0, 1)) {
		return invalid("invalid_date", "occurred_on", "date is out of range")
	}
	return nil
}

func toDebt(id int64, dir, who, cur string, due *time.Time, note string, created time.Time, balance, principal int64) Debt {
	remaining := balance
	if dir == "borrowed" {
		remaining = -balance
	}
	return Debt{ID: id, Direction: dir, Counterparty: who, Currency: cur, Principal: principal, Remaining: max(remaining, 0), DueOn: due, Note: note, CreatedAt: created}
}

func (s *Service) ListDebts(ctx context.Context, userID int64) ([]Debt, error) {
	q := dbq.New(s.pool)
	rows, err := q.ListDebts(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Debt, 0, len(rows))
	for _, r := range rows {
		acc := r.AccountID
		history, _, err := s.listAll(ctx, q, userID, Filter{AccountID: &acc, Type: ptrTo(string(DebtTx))})
		if err != nil {
			return nil, err
		}
		principal, on, account := debtOrigin(history)
		d := toDebt(r.ID, r.Direction, r.Counterparty, r.Currency, r.DueOn, r.Note, r.CreatedAt, r.Balance, principal)
		d.OccurredOn, d.AccountID = on, account
		out = append(out, d)
	}
	return out, nil
}

// debtOrigin sums the debt-creating transactions on a debt account and returns the first
// one's date and account.
func debtOrigin(history []Transaction) (principal int64, on time.Time, account *int64) {
	for _, t := range history { // newest first
		if t.Type != DebtTx {
			continue
		}
		principal += t.AmountMinor
		on, account = t.OccurredOn, nil
		if t.AccountID != 0 {
			a := t.AccountID
			account = &a
		}
	}
	return principal, on, account
}

func ptrTo[T any](v T) *T { return &v }

func (s *Service) GetDebt(ctx context.Context, userID, id int64) (Debt, []Transaction, error) {
	q := dbq.New(s.pool)
	r, err := q.GetDebt(ctx, dbq.GetDebtParams{ID: id, UserID: userID})
	if err != nil {
		return Debt{}, nil, notFound(err)
	}
	history, _, err := s.listAll(ctx, q, userID, Filter{AccountID: &r.AccountID})
	if err != nil {
		return Debt{}, nil, err
	}
	principal, on, account := debtOrigin(history)
	d := toDebt(r.ID, r.Direction, r.Counterparty, r.Currency, r.DueOn, r.Note, r.CreatedAt, r.Balance, principal)
	d.OccurredOn, d.AccountID = on, account
	return d, history, nil
}

// listAll pages through every matching transaction (debt histories are short).
func (s *Service) listAll(ctx context.Context, q *dbq.Queries, userID int64, f Filter) ([]Transaction, string, error) {
	f.Limit = 200
	var all []Transaction
	for {
		page, next, err := s.List(ctx, userID, f)
		if err != nil {
			return nil, "", err
		}
		all = append(all, page...)
		if next == "" {
			return all, "", nil
		}
		f.Cursor = next
	}
}

type DebtPatch struct {
	Counterparty *string
	Note         *string
	SetDue       bool
	DueOn        *time.Time
}

func (s *Service) UpdateDebt(ctx context.Context, userID, id int64, p DebtPatch) (Debt, error) {
	if p.Counterparty != nil {
		n, err := validName("counterparty", *p.Counterparty)
		if err != nil {
			return Debt{}, err
		}
		p.Counterparty = &n
	}
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		d, err := q.UpdateDebt(ctx, dbq.UpdateDebtParams{Counterparty: p.Counterparty, Note: p.Note, SetDue: p.SetDue, DueOn: p.DueOn, ID: id, UserID: userID})
		if err != nil {
			return notFound(err)
		}
		if p.Counterparty != nil {
			if err := q.RenameAccount(ctx, dbq.RenameAccountParams{Name: *p.Counterparty, ID: d.AccountID, UserID: userID}); err != nil {
				return err
			}
		}
		return audit(ctx, q, userID, "debt", id, "update", nil, p)
	})
	if err != nil {
		return Debt{}, err
	}
	d, _, err := s.GetDebt(ctx, userID, id)
	return d, err
}

// RecordDebtPayment records money coming back (lent) or paid back (borrowed).
// Paying more than is owed is refused.
func (s *Service) RecordDebtPayment(ctx context.Context, userID, debtID, amount, accountID int64, on time.Time) (Transaction, error) {
	if amount <= 0 || amount > maxAmount {
		return Transaction{}, invalid("invalid_amount", "amount_minor", "amount must be positive")
	}
	var out Transaction
	err := s.inTx(ctx, func(q *dbq.Queries) error {
		user, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		if err := s.checkDate(user, on); err != nil {
			return err
		}
		d, err := q.GetDebt(ctx, dbq.GetDebtParams{ID: debtID, UserID: userID})
		if err != nil {
			return notFound(err)
		}
		remaining, sign := d.Balance, int64(1)
		if d.Direction == "borrowed" {
			remaining, sign = -d.Balance, -1
		}
		if amount > remaining {
			return invalid("overpayment", "amount_minor", "amount is more than what is owed")
		}
		a, err := s.ownAccount(ctx, q, userID, accountID, "account_id")
		if err != nil {
			return err
		}
		if a.currency != d.Currency {
			return invalid("currency_mismatch", "account_id", "account currency differs from the debt")
		}
		tx, err := q.InsertTransaction(ctx, dbq.InsertTransactionParams{
			UserID: userID, Type: string(DebtPaymentTx), OccurredOn: on, Description: d.Counterparty, Source: "manual",
		})
		if err != nil {
			return err
		}
		legs := []leg{
			{asset: a.id, amount: sign * amount, currency: d.Currency},
			{asset: d.AccountID, amount: -sign * amount, currency: d.Currency},
		}
		if err := writeLegs(ctx, q, userID, tx.ID, legs); err != nil {
			return err
		}
		if out, err = load(ctx, q, tx); err != nil {
			return err
		}
		return audit(ctx, q, userID, "debt", debtID, "payment", nil, out)
	})
	return out, err
}

// DeleteDebt removes a debt and all its transactions from balances.
func (s *Service) DeleteDebt(ctx context.Context, userID, id int64) error {
	return s.inTx(ctx, func(q *dbq.Queries) error {
		d, err := q.GetDebt(ctx, dbq.GetDebtParams{ID: id, UserID: userID})
		if err != nil {
			return notFound(err)
		}
		if err := q.SoftDeleteAccountTransactions(ctx, dbq.SoftDeleteAccountTransactionsParams{UserID: userID, AccountID: d.AccountID}); err != nil {
			return err
		}
		if err := q.SoftDeleteDebt(ctx, dbq.SoftDeleteDebtParams{ID: id, UserID: userID}); err != nil {
			return err
		}
		return audit(ctx, q, userID, "debt", id, "delete", nil, nil)
	})
}

// ---- installment plans (nasiya) ----

type ScheduleRow struct {
	N      int32
	DueOn  time.Time
	Amount int64
	Paid   bool
}

type Plan struct {
	ID               int64
	Title            string
	Provider         string
	CategoryID       int64
	Currency         string
	Total            int64
	Paid             int64
	Remaining        int64
	Months           int32
	PayFromAccountID *int64
	Schedule         []ScheduleRow
	Next             *ScheduleRow // first unpaid row
	CreatedAt        time.Time
}

type NewPlan struct {
	Title            string
	Provider         string
	CategoryID       int64
	Currency         string
	TotalMinor       int64
	Months           int
	FirstDueOn       time.Time
	PayFromAccountID *int64
	OccurredOn       time.Time
	// RecordPurchase counts the full price as an expense on OccurredOn. False is for a
	// plan that started before the user began tracking: the debt is recorded, the past
	// purchase is not counted as this month's spending.
	RecordPurchase bool
}

// schedule splits total into months equal payments in whole currency units (no tiyin or
// cents); the last payment absorbs the remainder.
func schedule(total int64, months int, first time.Time) []ScheduleRow {
	const unit = 100 // minor units per so'm or dollar
	rows := make([]ScheduleRow, months)
	each := total / int64(months) / unit * unit
	for i := range rows {
		rows[i] = ScheduleRow{N: int32(i + 1), DueOn: addMonthsClamped(first, i), Amount: each}
	}
	rows[months-1].Amount = total - each*int64(months-1)
	return rows
}

// addMonthsClamped keeps the day of month where possible: Jan 31 + 1 month = Feb 28.
func addMonthsClamped(d time.Time, n int) time.Time {
	y, m, day := d.Date()
	first := time.Date(y, m+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(day, last), 0, 0, 0, 0, time.UTC)
}

// markPaid flags rows covered by the amount paid so far, in order, and returns the first
// unpaid row with its amount reduced by any partial payment.
func markPaid(rows []ScheduleRow, paid int64) (*ScheduleRow, []ScheduleRow) {
	var next *ScheduleRow
	left := paid
	for i := range rows {
		covered := min(left, rows[i].Amount)
		left -= covered
		rows[i].Paid = covered == rows[i].Amount
		if !rows[i].Paid && next == nil {
			r := rows[i]
			r.Amount -= covered
			next = &r
		}
	}
	return next, rows
}

func (s *Service) CreatePlan(ctx context.Context, userID int64, in NewPlan) (Plan, error) {
	var err error
	if in.Title, err = validName("title", in.Title); err != nil {
		return Plan{}, err
	}
	in.Provider = strings.TrimSpace(in.Provider)
	if utf8.RuneCountInString(in.Provider) > maxAccountName {
		return Plan{}, invalid("invalid_provider", "provider", "provider is too long")
	}
	if !currencies[in.Currency] {
		return Plan{}, invalid("invalid_currency", "currency", "currency must be UZS or USD")
	}
	if in.TotalMinor <= 0 || in.TotalMinor > maxAmount {
		return Plan{}, invalid("invalid_amount", "total_minor", "total must be positive")
	}
	if in.Months < 1 || in.Months > 60 {
		return Plan{}, invalid("invalid_months", "months", "months must be 1–60")
	}
	if in.TotalMinor < int64(in.Months)*100 {
		return Plan{}, invalid("invalid_amount", "total_minor", "total is too small for that many months")
	}
	if in.FirstDueOn.Before(in.OccurredOn) || in.FirstDueOn.After(in.OccurredOn.AddDate(1, 0, 0)) {
		return Plan{}, invalid("invalid_date", "first_due_on", "first payment must be within a year after the purchase")
	}

	var id int64
	err = s.inTx(ctx, func(q *dbq.Queries) error {
		user, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		if err := s.checkDate(user, in.OccurredOn); err != nil {
			return err
		}
		c, err := q.GetCategory(ctx, dbq.GetCategoryParams{ID: in.CategoryID, UserID: &userID})
		if err != nil || c.Kind != "expense" {
			return invalid("invalid_category", "category_id", "unknown expense category")
		}
		if in.PayFromAccountID != nil {
			a, err := s.ownAccount(ctx, q, userID, *in.PayFromAccountID, "pay_from_account_id")
			if err != nil {
				return err
			}
			if a.currency != in.Currency {
				return invalid("currency_mismatch", "pay_from_account_id", "account currency differs from the plan")
			}
		}
		name := in.Title
		if in.Provider != "" {
			name = in.Provider + " · " + in.Title
		}
		acc, err := q.CreateLiabilityAccount(ctx, dbq.CreateLiabilityAccountParams{UserID: userID, Kind: "payable", Name: truncate(name, maxAccountName), Currency: in.Currency})
		if err != nil {
			return err
		}
		p, err := q.InsertPlan(ctx, dbq.InsertPlanParams{
			UserID: userID, Title: in.Title, Provider: in.Provider, CategoryID: in.CategoryID, AccountID: acc.ID,
			Currency: in.Currency, Total: in.TotalMinor, Months: int32(in.Months), PayFromAccountID: in.PayFromAccountID,
		})
		if err != nil {
			return err
		}
		for _, r := range schedule(in.TotalMinor, in.Months, in.FirstDueOn) {
			if err := q.InsertScheduleRow(ctx, dbq.InsertScheduleRowParams{PlanID: p.ID, N: r.N, DueOn: r.DueOn, Amount: r.Amount}); err != nil {
				return err
			}
		}
		counter := leg{system: "opening", amount: in.TotalMinor, currency: in.Currency}
		if in.RecordPurchase {
			counter.system = "expense"
		}
		legs := []leg{{asset: acc.ID, amount: -in.TotalMinor, currency: in.Currency}, counter}
		if err := s.writeTx(ctx, q, userID, InstallmentTx, in.OccurredOn, &in.CategoryID, in.Title, legs); err != nil {
			return err
		}
		id = p.ID
		return audit(ctx, q, userID, "installment", p.ID, "create", nil, in)
	})
	if err != nil {
		return Plan{}, err
	}
	return s.GetPlan(ctx, userID, id)
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}

func (s *Service) plans(ctx context.Context, q *dbq.Queries, rows []dbq.ListPlansRow) ([]Plan, error) {
	ids := make([]int64, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	sched, err := q.ListSchedule(ctx, ids)
	if err != nil {
		return nil, err
	}
	byPlan := map[int64][]ScheduleRow{}
	for _, r := range sched {
		byPlan[r.PlanID] = append(byPlan[r.PlanID], ScheduleRow{N: r.N, DueOn: r.DueOn, Amount: r.Amount})
	}
	out := make([]Plan, len(rows))
	for i, r := range rows {
		next, sch := markPaid(byPlan[r.ID], r.Paid)
		out[i] = Plan{
			ID: r.ID, Title: r.Title, Provider: r.Provider, CategoryID: r.CategoryID, Currency: r.Currency,
			Total: r.Total, Paid: r.Paid, Remaining: r.Total - r.Paid, Months: r.Months,
			PayFromAccountID: r.PayFromAccountID, Schedule: sch, Next: next, CreatedAt: r.CreatedAt,
		}
	}
	return out, nil
}

func (s *Service) ListPlans(ctx context.Context, userID int64) ([]Plan, error) {
	q := dbq.New(s.pool)
	rows, err := q.ListPlans(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.plans(ctx, q, rows)
}

func (s *Service) GetPlan(ctx context.Context, userID, id int64) (Plan, error) {
	q := dbq.New(s.pool)
	r, err := q.GetPlan(ctx, dbq.GetPlanParams{ID: id, UserID: userID})
	if err != nil {
		return Plan{}, notFound(err)
	}
	ps, err := s.plans(ctx, q, []dbq.ListPlansRow{dbq.ListPlansRow(r)})
	if err != nil {
		return Plan{}, err
	}
	return ps[0], nil
}

// PayInstallment pays amount (or, when zero, the rest of the next due payment) from an
// asset account. Payments reduce the debt; they are not spending again.
func (s *Service) PayInstallment(ctx context.Context, userID, planID, amount, accountID int64, on time.Time) (Transaction, error) {
	plan, err := s.GetPlan(ctx, userID, planID)
	if err != nil {
		return Transaction{}, err
	}
	if amount == 0 && plan.Next != nil {
		amount = plan.Next.Amount
	}
	if amount <= 0 || amount > maxAmount {
		return Transaction{}, invalid("invalid_amount", "amount_minor", "amount must be positive")
	}
	if amount > plan.Remaining {
		return Transaction{}, invalid("overpayment", "amount_minor", "amount is more than what is left")
	}
	var out Transaction
	err = s.inTx(ctx, func(q *dbq.Queries) error {
		user, err := q.GetUser(ctx, userID)
		if err != nil {
			return err
		}
		if err := s.checkDate(user, on); err != nil {
			return err
		}
		p, err := q.GetPlan(ctx, dbq.GetPlanParams{ID: planID, UserID: userID})
		if err != nil {
			return notFound(err)
		}
		if amount > p.Total-p.Paid { // re-check inside the transaction
			return invalid("overpayment", "amount_minor", "amount is more than what is left")
		}
		a, err := s.ownAccount(ctx, q, userID, accountID, "account_id")
		if err != nil {
			return err
		}
		if a.currency != p.Currency {
			return invalid("currency_mismatch", "account_id", "account currency differs from the plan")
		}
		tx, err := q.InsertTransaction(ctx, dbq.InsertTransactionParams{
			UserID: userID, Type: string(InstallmentPaymentTx), OccurredOn: on, Description: p.Title, Source: "manual",
		})
		if err != nil {
			return err
		}
		legs := []leg{
			{asset: a.id, amount: -amount, currency: p.Currency},
			{asset: p.AccountID, amount: amount, currency: p.Currency},
		}
		if err := writeLegs(ctx, q, userID, tx.ID, legs); err != nil {
			return err
		}
		if out, err = load(ctx, q, tx); err != nil {
			return err
		}
		return audit(ctx, q, userID, "installment", planID, "payment", nil, out)
	})
	return out, err
}

func (s *Service) DeletePlan(ctx context.Context, userID, id int64) error {
	return s.inTx(ctx, func(q *dbq.Queries) error {
		p, err := q.GetPlan(ctx, dbq.GetPlanParams{ID: id, UserID: userID})
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if err := q.SoftDeleteAccountTransactions(ctx, dbq.SoftDeleteAccountTransactionsParams{UserID: userID, AccountID: p.AccountID}); err != nil {
			return err
		}
		if err := q.SoftDeletePlan(ctx, dbq.SoftDeletePlanParams{ID: id, UserID: userID}); err != nil {
			return err
		}
		return audit(ctx, q, userID, "installment", id, "delete", nil, nil)
	})
}
