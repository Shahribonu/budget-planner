package ledger

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

type Filter struct {
	From, To   *time.Time
	Type       *string
	CategoryID *int64
	AccountID  *int64
	Currency   *string
	Q          *string
	Cursor     string
	Limit      int
}

// List returns transactions newest first and a cursor for the next page ("" at the end).
func (s *Service) List(ctx context.Context, userID int64, f Filter) ([]Transaction, string, error) {
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	p := dbq.ListTransactionsParams{
		UserID: userID, FromDate: f.From, ToDate: f.To, Type: f.Type, CategoryID: f.CategoryID,
		AccountID: f.AccountID, Currency: f.Currency, Lim: int32(f.Limit + 1),
	}
	if f.Q != nil && strings.TrimSpace(*f.Q) != "" {
		q := escapeLike(strings.TrimSpace(*f.Q))
		p.Q = &q
	}
	if f.Cursor != "" {
		d, id, err := decodeCursor(f.Cursor)
		if err != nil {
			return nil, "", invalid("invalid_cursor", "cursor", "invalid cursor")
		}
		p.CursorDate, p.CursorID = &d, &id
	}

	q := dbq.New(s.pool)
	rows, err := q.ListTransactions(ctx, p)
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(rows) > f.Limit {
		rows = rows[:f.Limit]
		last := rows[len(rows)-1]
		next = encodeCursor(last.OccurredOn, last.ID)
	}
	if len(rows) == 0 {
		return []Transaction{}, "", nil
	}
	out, err := loadMany(ctx, q, rows)
	return out, next, err
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func encodeCursor(d time.Time, id int64) string {
	return base64.RawURLEncoding.EncodeToString(fmt.Appendf(nil, "%s|%d", d.Format(time.DateOnly), id))
}

func decodeCursor(c string) (time.Time, int64, error) {
	b, err := base64.RawURLEncoding.DecodeString(c)
	if err != nil {
		return time.Time{}, 0, err
	}
	ds, idStr, ok := strings.Cut(string(b), "|")
	if !ok {
		return time.Time{}, 0, fmt.Errorf("bad cursor")
	}
	d, err := time.Parse(time.DateOnly, ds)
	if err != nil {
		return time.Time{}, 0, err
	}
	var id int64
	if _, err := fmt.Sscan(idStr, &id); err != nil {
		return time.Time{}, 0, err
	}
	return d, id, nil
}

type Money struct {
	AmountMinor int64
	Currency    string
}

type CategoryTotal struct {
	CategoryID  int64
	AmountMinor int64
	Currency    string
}

type Summary struct {
	Month      string
	Balances   []Money
	Income     []Money
	Expense    []Money
	ByCategory []CategoryTotal
	Recent     []Transaction
	OwedToMe   []Money // open debts and anything else receivable
	IOwe       []Money // open debts and installment plans, as positive amounts
}

// Summary returns Home screen data. month is "YYYY-MM"; empty means the current month
// in the user's time zone. All totals are per currency.
func (s *Service) Summary(ctx context.Context, userID int64, month string) (Summary, error) {
	q := dbq.New(s.pool)
	user, err := q.GetUser(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	var from time.Time
	if month == "" {
		t := s.today(user)
		from = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	} else if from, err = time.Parse("2006-01", month); err != nil {
		return Summary{}, invalid("invalid_month", "month", "month must be YYYY-MM")
	}
	to := from.AddDate(0, 1, 0)
	out := Summary{Month: from.Format("2006-01"), Balances: []Money{}, Income: []Money{}, Expense: []Money{}, ByCategory: []CategoryTotal{}, OwedToMe: []Money{}, IOwe: []Money{}}

	bal, err := q.AssetTotals(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	for _, b := range bal {
		out.Balances = append(out.Balances, Money{b.Amount, b.Currency})
	}

	flows, err := q.FlowTotals(ctx, dbq.FlowTotalsParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return Summary{}, err
	}
	for _, f := range flows {
		if f.Flow == "income" {
			out.Income = append(out.Income, Money{-f.Amount, f.Currency})
		} else {
			out.Expense = append(out.Expense, Money{f.Amount, f.Currency})
		}
	}

	cats, err := q.ExpenseByCategory(ctx, dbq.ExpenseByCategoryParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return Summary{}, err
	}
	for _, c := range cats {
		out.ByCategory = append(out.ByCategory, CategoryTotal{c.CategoryID, c.Amount, c.Currency})
	}

	owed, err := q.LiabilityTotals(ctx, userID)
	if err != nil {
		return Summary{}, err
	}
	for _, o := range owed {
		if o.Kind == "receivable" {
			out.OwedToMe = append(out.OwedToMe, Money{o.Amount, o.Currency})
		} else {
			out.IOwe = append(out.IOwe, Money{-o.Amount, o.Currency})
		}
	}

	out.Recent, _, err = s.List(ctx, userID, Filter{Limit: 10})
	return out, err
}
