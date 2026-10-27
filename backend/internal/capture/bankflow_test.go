package capture_test

import (
	"testing"

	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/bank"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
)

const (
	payClick = "💸 To'lov\n➖ 37.470,00 UZS\n📍 AAB CLICK QR OPLATA\n💳 HUMOCARD *5958\n🕓 18:38 23.10.2026\n💰 118.363,04 UZS"
	payJeti  = "💸 To'lov\n➖ 75.200,00 UZS\n📍 IP OOO JETI ASPAN>TA\n💳 HUMOCARD *5958\n🕓 12:03 26.10.2026\n💰 33.163,04 UZS"
	topUp    = "🎉 To'ldirish\n➕ 18.000,00 UZS\n📍 Alif uz u2h>Toshkent\n💳 HUMOCARD *5958\n🕓 15:48 26.10.2026\n💰 51.163,04 UZS"
)

func (f *fixture) bank(text string) capture.Outcome {
	f.t.Helper()
	n, ok := bank.Parse(text)
	if !ok {
		f.t.Fatalf("not a notification: %q", text)
	}
	o, err := f.svc.CaptureBank(ctx, f.user, n)
	if err != nil {
		f.t.Fatal(err)
	}
	return o
}

func TestBankFirstCardBecomesAccountMatchingTheBank(t *testing.T) {
	f := setup(t)
	o := f.bank(payClick)
	if o.Kind != capture.AskCard {
		t.Fatalf("kind = %v, want AskCard", o.Kind)
	}
	saved, err := f.svc.AnswerCard(ctx, f.user, o.DraftID, 0)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Kind != capture.Saved || saved.NewAccount != "Humo *5958" || saved.Tx.Source != "bank" {
		t.Fatalf("saved = %+v", saved)
	}
	if saved.Tx.AmountMinor != 3_747_000 || saved.Tx.Description != "Click QR" || *saved.Tx.CategoryID != f.cats["other_expense"] {
		t.Errorf("tx = %+v", saved.Tx)
	}
	// The new account starts so that, after this payment, it equals the bank's balance.
	if got := f.balance(saved.Tx.AccountID); got != 11_836_304 || saved.BankBalance != nil {
		t.Errorf("balance %d, mismatch %v", got, saved.BankBalance)
	}
	// "Click QR" says nothing about what was bought, so it is not learned.
	if _, err := f.q.GetMerchantRule(ctx, dbqRule(f.user.ID, "click qr")); err == nil {
		t.Error("generic merchant was learned")
	}

	// The card is known now; the same message forwarded again is one transaction.
	again := f.bank(payClick)
	if again.Kind != capture.Saved || again.Tx.ID != saved.Tx.ID {
		t.Errorf("forwarded twice = %+v", again)
	}

	// A later payment: the bank's balance differs because of payments the user didn't forward.
	j := f.bank(payJeti)
	if j.Kind != capture.Saved || j.Tx.Description != "Jeti Aspan" || j.BankBalance == nil || *j.BankBalance != 3_316_304 {
		t.Fatalf("second payment = %+v", j)
	}
	// One tap sets the account to the bank's figure.
	bal, cur, err := f.svc.ReconcileToBank(ctx, f.user, j.Tx.AccountID)
	if err != nil || bal != 3_316_304 || cur != "UZS" || f.balance(j.Tx.AccountID) != 3_316_304 {
		t.Errorf("reconcile = %d %s %v; balance %d", bal, cur, err, f.balance(j.Tx.AccountID))
	}
	if s, _ := f.led.Summary(ctx, f.user.ID, "2026-10"); total(s.Income) != 0 {
		t.Errorf("reconcile counted as income: %+v", s.Income)
	}
}

func TestBankTopUpFromUntrackedCardIsNotIncome(t *testing.T) {
	f := setup(t)
	humo := f.account("Humo", "humo", "UZS")
	if err := f.led.AssignCard(ctx, f.user.ID, humo, "5958"); err != nil {
		t.Fatal(err)
	}
	o := f.bank(topUp)
	if o.Kind != capture.AskBankKind {
		t.Fatalf("kind = %v, want AskBankKind", o.Kind)
	}
	o, err := f.svc.AnswerBankKind(ctx, f.user, o.DraftID, "own")
	if err != nil || o.Kind != capture.AskTransferFrom || len(o.Accounts) != 0 {
		t.Fatalf("own = %+v, %v", o, err)
	}
	o, err = f.svc.AnswerTransferFrom(ctx, f.user, o.DraftID, 0)
	if err != nil || o.Kind != capture.Saved {
		t.Fatalf("untracked = %+v, %v", o, err)
	}
	if f.balance(humo) != 1_800_000 {
		t.Errorf("balance = %d", f.balance(humo))
	}
	if s, _ := f.led.Summary(ctx, f.user.ID, "2026-10"); total(s.Income) != 0 {
		t.Errorf("own money counted as income: %+v", s.Income)
	}
}

func TestBankTopUpAsIncomeAndOwnTransfer(t *testing.T) {
	f := setup(t)
	humo := f.account("Humo", "humo", "UZS")
	alif := f.account("Alif", "uzcard", "UZS")
	_ = f.led.AssignCard(ctx, f.user.ID, humo, "5958")

	o := f.bank(topUp)
	o, err := f.svc.AnswerBankKind(ctx, f.user, o.DraftID, "own")
	if err != nil || len(o.Accounts) != 1 || o.Accounts[0].ID != alif {
		t.Fatalf("own choices = %+v, %v", o.Accounts, err)
	}
	o, err = f.svc.AnswerTransferFrom(ctx, f.user, o.DraftID, alif)
	if err != nil || o.Tx.Type != ledger.Transfer || o.Tx.AccountID != alif || *o.Tx.ToAccountID != humo {
		t.Fatalf("transfer = %+v, %v", o.Tx, err)
	}
	if f.balance(humo) != 1_800_000 || f.balance(alif) != -1_800_000 {
		t.Errorf("balances humo %d alif %d", f.balance(humo), f.balance(alif))
	}

	// A different top-up recorded as income.
	o = f.bank("🎉 To'ldirish\n➕ 5.000.000,00 UZS\n📍 OOO EMPLOYER\n💳 HUMOCARD *5958\n🕓 10:00 25.10.2026")
	o, err = f.svc.AnswerBankKind(ctx, f.user, o.DraftID, "income")
	if err != nil || o.Tx.Type != ledger.Income || o.Tx.AmountMinor != 500_000_000 {
		t.Errorf("income = %+v, %v", o.Tx, err)
	}
}

func TestBankConfirmsHandLoggedExpense(t *testing.T) {
	f := setup(t)
	humo := f.account("Humo", "humo", "UZS")
	_ = f.led.AssignCard(ctx, f.user.ID, humo, "5958")
	cash := f.account("Naqd", "cash", "UZS")
	// Logged by hand on the same day, wrongly from cash.
	d := mustDate("2026-10-26")
	logged, _, err := f.led.Record(ctx, f.user.ID, ledger.Input{Type: ledger.Expense, OccurredOn: d, AmountMinor: 7_520_000,
		AccountID: cash, CategoryID: ptr(f.cats["cafe"]), Description: "tushlik", Source: "chat"})
	if err != nil {
		t.Fatal(err)
	}
	o := f.bank(payJeti)
	if o.Kind != capture.AskDuplicate || o.Duplicate.ID != logged.ID {
		t.Fatalf("kind = %v dup %+v", o.Kind, o.Duplicate)
	}
	merged, err := f.svc.AnswerDuplicate(ctx, f.user, o.DraftID, false)
	if err != nil || merged.Kind != capture.Merged {
		t.Fatalf("merge = %+v, %v", merged, err)
	}
	// One expense, now on the card.
	if f.balance(cash) != 0 || f.balance(humo) != -7_520_000 {
		t.Errorf("after merge: cash %d humo %d", f.balance(cash), f.balance(humo))
	}
}
