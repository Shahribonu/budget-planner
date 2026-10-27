package ledger_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
)

func today() time.Time { return time.Now().UTC().Truncate(24 * time.Hour) }

// Product plan section 18: "Lend 50 000 to a friend: spending does not change, cash drops,
// receivables rise by 50 000."
func TestLendingIsNotSpending(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 1_000_000*sum)
	before := f.summary()

	d, err := f.svc.CreateDebt(ctx, f.user, ledger.NewDebt{
		Direction: "lent", Counterparty: "Aziz", Currency: "UZS", AmountMinor: 50_000 * sum, AccountID: &cash, OccurredOn: today(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.Remaining != 50_000*sum || d.Principal != 50_000*sum {
		t.Errorf("debt = %+v", d)
	}
	s := f.summary()
	if total(s.Expense, "UZS") != total(before.Expense, "UZS") || total(s.Income, "UZS") != 0 {
		t.Errorf("lending changed spending: %+v income %+v", s.Expense, s.Income)
	}
	if f.balances()[cash] != 950_000*sum {
		t.Errorf("cash = %d", f.balances()[cash]/sum)
	}
	if total(s.OwedToMe, "UZS") != 50_000*sum {
		t.Errorf("owed to me = %+v", s.OwedToMe)
	}

	// Partial repayment into another account, then the rest.
	humo := f.account("Humo", "humo", "UZS", 0)
	if _, err := f.svc.RecordDebtPayment(ctx, f.user, d.ID, 20_000*sum, humo, today()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.RecordDebtPayment(ctx, f.user, d.ID, 40_000*sum, cash, today()); !isValidation(err, "overpayment") {
		t.Errorf("overpayment: %v", err)
	}
	pay, err := f.svc.RecordDebtPayment(ctx, f.user, d.ID, 30_000*sum, cash, today())
	if err != nil {
		t.Fatal(err)
	}
	if pay.DebtID == nil || *pay.DebtID != d.ID || pay.AccountID != cash || pay.AmountMinor != 30_000*sum {
		t.Errorf("payment tx = %+v", pay)
	}
	got, history, err := f.svc.GetDebt(ctx, f.user, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Remaining != 0 || len(history) != 3 {
		t.Errorf("after repayment: %+v, %d history rows", got, len(history))
	}
	s = f.summary()
	if total(s.OwedToMe, "UZS") != 0 || total(s.Expense, "UZS") != 0 || total(s.Income, "UZS") != 0 {
		t.Errorf("after repayment summary = %+v", s)
	}

	// Undoing a payment reopens the debt.
	if err := f.svc.Delete(ctx, f.user, pay.ID); err != nil {
		t.Fatal(err)
	}
	if got, _, _ := f.svc.GetDebt(ctx, f.user, d.ID); got.Remaining != 30_000*sum {
		t.Errorf("after undoing payment remaining = %d", got.Remaining/sum)
	}
	// The debt itself can't be deleted as a plain transaction.
	if err := f.svc.Delete(ctx, f.user, history[len(history)-1].ID); !isValidation(err, "managed_elsewhere") {
		t.Errorf("deleting the debt tx: %v", err)
	}
}

func TestBorrowedAndPreexistingDebts(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 0)
	b, err := f.svc.CreateDebt(ctx, f.user, ledger.NewDebt{
		Direction: "borrowed", Counterparty: "Akam", Currency: "UZS", AmountMinor: 300_000 * sum, AccountID: &cash, OccurredOn: today(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if f.balances()[cash] != 300_000*sum || b.Remaining != 300_000*sum {
		t.Errorf("borrowed: cash %d, remaining %d", f.balances()[cash]/sum, b.Remaining/sum)
	}
	// An old debt from before tracking: no account touched.
	old, err := f.svc.CreateDebt(ctx, f.user, ledger.NewDebt{
		Direction: "lent", Counterparty: "Dilnoza", Currency: "USD", AmountMinor: 100 * 100, OccurredOn: today(),
	})
	if err != nil {
		t.Fatal(err)
	}
	s := f.summary()
	if total(s.IOwe, "UZS") != 300_000*sum || total(s.OwedToMe, "USD") != 100*100 || total(s.Balances, "USD") != 0 {
		t.Errorf("summary = owe %+v, owed %+v, balances %+v", s.IOwe, s.OwedToMe, s.Balances)
	}
	if _, err := f.svc.RecordDebtPayment(ctx, f.user, b.ID, 100_000*sum, cash, today()); err != nil {
		t.Fatal(err)
	}
	if f.balances()[cash] != 200_000*sum {
		t.Errorf("after paying back: cash %d", f.balances()[cash]/sum)
	}
	if _, err := f.svc.RecordDebtPayment(ctx, f.user, old.ID, 10*100, cash, today()); !isValidation(err, "currency_mismatch") {
		t.Errorf("USD debt paid into UZS cash: %v", err)
	}
	if err := f.svc.DeleteDebt(ctx, f.user, b.ID); err != nil {
		t.Fatal(err)
	}
	if f.balances()[cash] != 0 || total(f.summary().IOwe, "UZS") != 0 {
		t.Errorf("after deleting debt: cash %d, owe %+v", f.balances()[cash], f.summary().IOwe)
	}
	ds, _ := f.svc.ListDebts(ctx, f.user)
	if len(ds) != 1 || ds[0].ID != old.ID {
		t.Errorf("debts = %+v", ds)
	}
}

func TestDebtOwnershipAndValidation(t *testing.T) {
	a, b := setup(t), setup(t)
	cash := a.account("Naqd", "cash", "UZS", 0)
	d, err := a.svc.CreateDebt(ctx, a.user, ledger.NewDebt{Direction: "lent", Counterparty: "Aziz", Currency: "UZS", AmountMinor: 100, AccountID: &cash, OccurredOn: today()})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := b.svc.GetDebt(ctx, b.user, d.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Errorf("other user reads debt: %v", err)
	}
	bCash := b.account("Naqd", "cash", "UZS", 0)
	if _, err := b.svc.RecordDebtPayment(ctx, b.user, d.ID, 1, bCash, today()); !errors.Is(err, ledger.ErrNotFound) {
		t.Errorf("other user pays debt: %v", err)
	}
	if _, err := b.svc.CreateDebt(ctx, b.user, ledger.NewDebt{Direction: "lent", Counterparty: "X", Currency: "UZS", AmountMinor: 1, AccountID: &cash, OccurredOn: today()}); !isValidation(err, "invalid_account") {
		t.Errorf("lending from another user's account: %v", err)
	}
	for name, in := range map[string]ledger.NewDebt{
		"invalid_direction":    {Direction: "gift", Counterparty: "X", Currency: "UZS", AmountMinor: 1, OccurredOn: today()},
		"invalid_counterparty": {Direction: "lent", Counterparty: " ", Currency: "UZS", AmountMinor: 1, OccurredOn: today()},
		"invalid_amount":       {Direction: "lent", Counterparty: "X", Currency: "UZS", AmountMinor: 0, OccurredOn: today()},
	} {
		if _, err := a.svc.CreateDebt(ctx, a.user, in); !isValidation(err, name) {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestInstallmentPlan(t *testing.T) {
	f := setup(t)
	humo := f.account("Humo", "humo", "UZS", 10_000_000*sum)
	first := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	jan := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	janSummary := func() ledger.Summary {
		s, err := f.svc.Summary(ctx, f.user, "2026-01")
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	p, err := f.svc.CreatePlan(ctx, f.user, ledger.NewPlan{
		Title: "iPhone", Provider: "Alif", CategoryID: f.cats["home"], Currency: "UZS",
		TotalMinor: 10_000_000 * sum, Months: 3, FirstDueOn: first, PayFromAccountID: &humo,
		OccurredOn: jan, RecordPurchase: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// 10 000 000 / 3 = 3 333 333 ×2 + 3 333 334; Jan 31 → Feb 28 → Mar 31.
	wantDue := []string{"2026-01-31", "2026-02-28", "2026-03-31"}
	wantAmt := []int64{3_333_333 * sum, 3_333_333 * sum, 3_333_334 * sum}
	for i, r := range p.Schedule {
		if r.DueOn.Format(time.DateOnly) != wantDue[i] || r.Amount != wantAmt[i] || r.Paid {
			t.Errorf("row %d = %+v", i, r)
		}
	}
	// The purchase is this month's spending, once.
	s := janSummary()
	if total(s.Expense, "UZS") != 10_000_000*sum || total(s.Installments, "UZS") != 10_000_000*sum {
		t.Errorf("after purchase: expense %+v, owe %+v", s.Expense, s.IOwe)
	}
	if f.balances()[humo] != 10_000_000*sum {
		t.Error("buying on installment moved money")
	}

	// Pay the next installment (amount 0 = the next due amount).
	pay, err := f.svc.PayInstallment(ctx, f.user, p.ID, 0, humo, today())
	if err != nil {
		t.Fatal(err)
	}
	if pay.AmountMinor != 3_333_333*sum || pay.PlanID == nil {
		t.Errorf("payment = %+v", pay)
	}
	// A partial payment covers part of the second row.
	if _, err := f.svc.PayInstallment(ctx, f.user, p.ID, 1_000_000*sum, humo, today()); err != nil {
		t.Fatal(err)
	}
	p, _ = f.svc.GetPlan(ctx, f.user, p.ID)
	if !p.Schedule[0].Paid || p.Schedule[1].Paid || p.Next == nil || p.Next.N != 2 || p.Next.Amount != 2_333_333*sum {
		t.Errorf("after partial: schedule %+v next %+v", p.Schedule, p.Next)
	}
	s = janSummary()
	if total(s.Expense, "UZS") != 10_000_000*sum {
		t.Errorf("payments counted as spending again: %+v", s.Expense)
	}
	if total(s.Installments, "UZS") != p.Remaining || p.Remaining != (10_000_000-3_333_333-1_000_000)*sum {
		t.Errorf("remaining %d, owe %+v", p.Remaining/sum, s.IOwe)
	}
	if _, err := f.svc.PayInstallment(ctx, f.user, p.ID, p.Remaining+1, humo, today()); !isValidation(err, "overpayment") {
		t.Errorf("overpaying: %v", err)
	}
	if _, err := f.svc.PayInstallment(ctx, f.user, p.ID, p.Remaining, humo, today()); err != nil {
		t.Fatal(err)
	}
	p, _ = f.svc.GetPlan(ctx, f.user, p.ID)
	if p.Remaining != 0 || p.Next != nil || !p.Schedule[2].Paid {
		t.Errorf("fully paid = %+v", p)
	}
	if f.balances()[humo] != 0 {
		t.Errorf("humo = %d", f.balances()[humo]/sum)
	}

	if err := f.svc.DeletePlan(ctx, f.user, p.ID); err != nil {
		t.Fatal(err)
	}
	if f.balances()[humo] != 10_000_000*sum || total(janSummary().Expense, "UZS") != 0 {
		t.Error("deleting the plan did not remove its transactions")
	}
}

func TestPlanStartedBeforeTracking(t *testing.T) {
	f := setup(t)
	p, err := f.svc.CreatePlan(ctx, f.user, ledger.NewPlan{
		Title: "Muzlatkich", CategoryID: f.cats["home"], Currency: "UZS", TotalMinor: 6_000_000 * sum,
		Months: 6, FirstDueOn: today(), OccurredOn: today(), RecordPurchase: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := f.summary()
	if total(s.Expense, "UZS") != 0 || total(s.Installments, "UZS") != 6_000_000*sum || len(s.IOwe) != 0 || p.Next.Amount != 1_000_000*sum {
		t.Errorf("expense %+v, installments %+v, owe %+v, next %+v", s.Expense, s.Installments, s.IOwe, p.Next)
	}
}
