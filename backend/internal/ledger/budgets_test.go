package ledger_test

import (
	"testing"

	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
)

func TestFreePlanAllowsOneBudget(t *testing.T) {
	f := setup(t)
	if _, err := f.svc.CreateBudget(ctx, f.user, nil, "UZS", 100); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.CreateBudget(ctx, f.user, ptr(f.cats["cafe"]), "UZS", 100); !isValidation(err, "premium_required") {
		t.Errorf("second free budget: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `update users set premium_until = now() + interval '1 day' where id = $1`, f.user); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.CreateBudget(ctx, f.user, ptr(f.cats["cafe"]), "UZS", 100); err != nil {
		t.Errorf("premium second budget: %v", err)
	}
}

func TestBudgets(t *testing.T) {
	f := setup(t)
	if _, err := f.pool.Exec(ctx, `update users set premium_until = now() + interval '1 day' where id = $1`, f.user); err != nil {
		t.Fatal(err)
	}
	cash := f.account("Naqd", "cash", "UZS", 5_000_000*sum)
	humo := f.account("Humo", "humo", "UZS", 0)
	usd := f.account("Dollar", "cash", "USD", 0)

	overall, err := f.svc.CreateBudget(ctx, f.user, nil, "UZS", 1_000_000*sum)
	if err != nil {
		t.Fatal(err)
	}
	cafe, err := f.svc.CreateBudget(ctx, f.user, ptr(f.cats["cafe"]), "UZS", 300_000*sum)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.CreateBudget(ctx, f.user, ptr(f.cats["cafe"]), "UZS", 1); !isValidation(err, "budget_exists") {
		t.Errorf("duplicate budget: %v", err)
	}
	if _, err := f.svc.CreateBudget(ctx, f.user, ptr(f.cats["salary"]), "UZS", 1); !isValidation(err, "invalid_category") {
		t.Errorf("income category budget: %v", err)
	}

	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 250_000 * sum, AccountID: cash, CategoryID: ptr(f.cats["cafe"])})
	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 100_000 * sum, AccountID: cash, CategoryID: ptr(f.cats["taxi"])})
	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 10 * 100, AccountID: usd, CategoryID: ptr(f.cats["cafe"])})        // other currency
	f.record(ledger.Input{Type: ledger.Transfer, AmountMinor: 900_000 * sum, AccountID: cash, ToAccountID: &humo})              // not spending
	f.record(ledger.Input{Type: ledger.Income, AmountMinor: 900_000 * sum, AccountID: cash, CategoryID: ptr(f.cats["salary"])}) // not spending

	bs, month, daysLeft, err := f.svc.Budgets(ctx, f.user, "")
	if err != nil {
		t.Fatal(err)
	}
	spent := map[int64]int64{}
	for _, b := range bs {
		spent[b.ID] = b.Spent
	}
	if spent[overall.ID] != 350_000*sum || spent[cafe.ID] != 250_000*sum {
		t.Errorf("spent = overall %d, cafe %d", spent[overall.ID]/sum, spent[cafe.ID]/sum)
	}
	if month == "" || daysLeft < 0 || daysLeft > 30 {
		t.Errorf("month %q days left %d", month, daysLeft)
	}
	if bs, _, _, _ := f.svc.Budgets(ctx, f.user, "2020-01"); bs[0].Spent != 0 {
		t.Errorf("past month spent = %d", bs[0].Spent)
	}

	if err := f.svc.UpdateBudget(ctx, f.user, cafe.ID, 500_000*sum); err != nil {
		t.Fatal(err)
	}
	if err := f.svc.DeleteBudget(ctx, f.user, overall.ID); err != nil {
		t.Fatal(err)
	}
	other := setup(t)
	if err := other.svc.DeleteBudget(ctx, other.user, cafe.ID); err != ledger.ErrNotFound {
		t.Errorf("other user deletes budget: %v", err)
	}
	// The overall budget can be created again after deleting it.
	if _, err := f.svc.CreateBudget(ctx, f.user, nil, "UZS", 2_000_000*sum); err != nil {
		t.Errorf("recreate overall: %v", err)
	}
}
