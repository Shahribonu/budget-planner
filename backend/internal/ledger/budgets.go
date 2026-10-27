package ledger

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

// FreeBudgets is how many budgets the free plan allows.
const FreeBudgets = 1

// Budget is a monthly limit, overall (CategoryID nil) or for one expense category.
type Budget struct {
	ID         int64
	CategoryID *int64
	Currency   string
	Amount     int64
	Spent      int64 // in the requested month
}

func (s *Service) CreateBudget(ctx context.Context, userID int64, categoryID *int64, currency string, amount int64) (Budget, error) {
	if !currencies[currency] {
		return Budget{}, invalid("invalid_currency", "currency", "currency must be UZS or USD")
	}
	if amount <= 0 || amount > maxAmount {
		return Budget{}, invalid("invalid_amount", "amount_minor", "amount must be positive")
	}
	q := dbq.New(s.pool)
	// Free plan: one budget. Premium: unlimited. Budgets made while Premium stay after it ends.
	u, err := q.GetUser(ctx, userID)
	if err != nil {
		return Budget{}, err
	}
	if u.PremiumUntil == nil || !u.PremiumUntil.After(s.now()) {
		n, err := q.CountActiveBudgets(ctx, userID)
		if err != nil {
			return Budget{}, err
		}
		if n >= FreeBudgets {
			return Budget{}, invalid("premium_required", "category_id", "more than one budget needs Premium")
		}
	}
	if categoryID != nil {
		c, err := q.GetCategory(ctx, dbq.GetCategoryParams{ID: *categoryID, UserID: &userID})
		if err != nil || c.Kind != "expense" {
			return Budget{}, invalid("invalid_category", "category_id", "unknown expense category")
		}
	}
	b, err := q.InsertBudget(ctx, dbq.InsertBudgetParams{UserID: userID, CategoryID: categoryID, Currency: currency, Amount: amount})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Budget{}, invalid("budget_exists", "category_id", "a budget for this category and currency already exists")
	}
	if err != nil {
		return Budget{}, err
	}
	return Budget{ID: b.ID, CategoryID: b.CategoryID, Currency: b.Currency, Amount: b.Amount}, nil
}

func (s *Service) UpdateBudget(ctx context.Context, userID, id, amount int64) error {
	if amount <= 0 || amount > maxAmount {
		return invalid("invalid_amount", "amount_minor", "amount must be positive")
	}
	_, err := dbq.New(s.pool).UpdateBudgetAmount(ctx, dbq.UpdateBudgetAmountParams{Amount: amount, ID: id, UserID: userID})
	return notFound(err)
}

func (s *Service) DeleteBudget(ctx context.Context, userID, id int64) error {
	n, err := dbq.New(s.pool).SoftDeleteBudget(ctx, dbq.SoftDeleteBudgetParams{ID: id, UserID: userID})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Budgets returns every budget with its spending in month ("YYYY-MM", empty for the
// current month in the user's time zone), the month used, and the days left in it
// (0 for a past month).
func (s *Service) Budgets(ctx context.Context, userID int64, month string) ([]Budget, string, int, error) {
	q := dbq.New(s.pool)
	user, err := q.GetUser(ctx, userID)
	if err != nil {
		return nil, "", 0, err
	}
	today := s.today(user)
	from := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC)
	if month != "" {
		if from, err = time.Parse("2006-01", month); err != nil {
			return nil, "", 0, invalid("invalid_month", "month", "month must be YYYY-MM")
		}
	}
	to := from.AddDate(0, 1, 0)
	daysLeft := 0
	if !today.Before(from) && today.Before(to) {
		daysLeft = int(to.Sub(today).Hours()/24) - 1
	}
	rows, err := q.BudgetProgress(ctx, dbq.BudgetProgressParams{UserID: userID, FromDate: from, ToDate: to})
	if err != nil {
		return nil, "", 0, err
	}
	out := make([]Budget, len(rows))
	for i, r := range rows {
		out[i] = Budget{ID: r.ID, CategoryID: r.CategoryID, Currency: r.Currency, Amount: r.Amount, Spent: r.Spent}
	}
	return out, from.Format("2006-01"), daysLeft, nil
}
