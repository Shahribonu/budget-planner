package capture_test

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

var ctx = context.Background()

type fixture struct {
	t    *testing.T
	svc  *capture.Service
	led  *ledger.Service
	q    *dbq.Queries
	user dbq.User
	cats map[string]int64
	msg  int
}

func setup(t *testing.T) *fixture {
	pool := testdb.Pool(t)
	u := testdb.User(t, pool)
	l := ledger.New(pool)
	q := dbq.New(pool)
	cs, err := q.ListCategories(ctx, &u.ID)
	if err != nil {
		t.Fatal(err)
	}
	cats := map[string]int64{}
	for _, c := range cs {
		cats[*c.Key] = c.ID
	}
	return &fixture{t: t, svc: capture.New(pool, l, nil), led: l, q: q, user: u, cats: cats}
}

// say sends one chat message and returns the outcomes.
func (f *fixture) say(text string) []capture.Outcome {
	f.t.Helper()
	f.msg++
	outs, err := f.svc.Capture(ctx, f.user, text, "chat", fmt.Sprintf("test:%d:%d", rand.Int64(), f.msg))
	if err != nil {
		f.t.Fatalf("Capture(%q): %v", text, err)
	}
	return outs
}

func (f *fixture) one(text string) capture.Outcome {
	f.t.Helper()
	outs := f.say(text)
	if len(outs) != 1 {
		f.t.Fatalf("Capture(%q) = %d outcomes, want 1", text, len(outs))
	}
	return outs[0]
}

func (f *fixture) account(name, typ, cur string) int64 {
	f.t.Helper()
	a, err := f.led.CreateAccount(ctx, f.user.ID, ledger.NewAccount{Name: name, Type: typ, Currency: cur})
	if err != nil {
		f.t.Fatal(err)
	}
	return a.ID
}

func (f *fixture) balance(id int64) int64 {
	f.t.Helper()
	accs, err := f.led.ListAccounts(ctx, f.user.ID, false)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, a := range accs {
		if a.ID == id {
			return a.BalanceMinor
		}
	}
	f.t.Fatalf("account %d not found", id)
	return 0
}

const sum = 100

func TestFirstMessageCreatesCashAccount(t *testing.T) {
	f := setup(t)
	o := f.one("taksi 25 ming")
	if o.Kind != capture.Saved {
		t.Fatalf("kind = %v", o.Kind)
	}
	if o.NewAccount == "" || !o.FirstLog {
		t.Errorf("new account %q, first log %v", o.NewAccount, o.FirstLog)
	}
	if o.Tx.AmountMinor != 25_000*sum || *o.Tx.CategoryID != f.cats["taxi"] || o.Tx.Source != "chat" || o.Tx.Description != "taksi" {
		t.Errorf("tx = %+v", o.Tx)
	}
	if f.balance(o.Tx.AccountID) != -25_000*sum {
		t.Errorf("balance = %d", f.balance(o.Tx.AccountID))
	}
	// Second message reuses the account and is not a first log.
	o2 := f.one("non 8 ming")
	if o2.NewAccount != "" || o2.FirstLog || o2.Tx.AccountID != o.Tx.AccountID {
		t.Errorf("second = %+v", o2)
	}
}

func TestSeveralItemsInOneMessage(t *testing.T) {
	f := setup(t)
	outs := f.say("non 8 ming, sut 14 ming, taksi 20 ming")
	if len(outs) != 3 {
		t.Fatalf("%d outcomes", len(outs))
	}
	want := []struct {
		cat    string
		amount int64
	}{{"groceries", 8_000}, {"groceries", 14_000}, {"taxi", 20_000}}
	for i, w := range want {
		if outs[i].Kind != capture.Saved || *outs[i].Tx.CategoryID != f.cats[w.cat] || outs[i].Tx.AmountMinor != w.amount*sum {
			t.Errorf("item %d = %+v", i, outs[i].Tx)
		}
	}
}

func TestShorthandAmountAsks(t *testing.T) {
	f := setup(t)
	o := f.one("taksi 25")
	if o.Kind != capture.AskAmount {
		t.Fatalf("kind = %v, want AskAmount", o.Kind)
	}
	saved, err := f.svc.AnswerAmount(ctx, f.user, o.DraftID, true)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Kind != capture.Saved || saved.Tx.AmountMinor != 25_000*sum {
		t.Errorf("after answer = %+v", saved)
	}
	// Answering twice is refused.
	if _, err := f.svc.AnswerAmount(ctx, f.user, o.DraftID, true); err != capture.ErrDraftGone {
		t.Errorf("second answer err = %v", err)
	}
}

func TestAccountQuestionAndMemory(t *testing.T) {
	f := setup(t)
	humo := f.account("Humo", "humo", "UZS")
	f.account("Uzcard", "uzcard", "UZS")

	o := f.one("korzinka 120 ming")
	if o.Kind != capture.AskAccount || len(o.Accounts) != 2 {
		t.Fatalf("kind = %v, accounts = %d", o.Kind, len(o.Accounts))
	}
	if f.balance(humo) != 0 {
		t.Error("draft touched the balance")
	}
	saved, err := f.svc.AnswerAccount(ctx, f.user, o.DraftID, humo)
	if err != nil {
		t.Fatal(err)
	}
	if saved.Kind != capture.Saved || saved.Tx.AccountID != humo || f.balance(humo) != -120_000*sum {
		t.Errorf("after answer = %+v", saved)
	}
	// The same shop next time goes straight to Humo.
	if o := f.one("korzinka 50 ming"); o.Kind != capture.Saved || o.Tx.AccountID != humo {
		t.Errorf("remembered account: %+v", o)
	}
}

func TestPossibleDuplicate(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS")
	// Logged in the app first...
	today := time.Now().In(time.FixedZone("UZT", 5*3600))
	_, _, err := f.led.Record(ctx, f.user.ID, ledger.Input{
		Type: ledger.Expense, OccurredOn: time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC),
		AmountMinor: 45_000 * sum, AccountID: cash, CategoryID: ptr(f.cats["cafe"]), Description: "tushlik", Source: "manual",
	})
	if err != nil {
		t.Fatal(err)
	}
	// ...then said in chat.
	o := f.one("obed 45 ming")
	if o.Kind != capture.AskDuplicate {
		t.Fatalf("kind = %v, want AskDuplicate", o.Kind)
	}
	merged, err := f.svc.AnswerDuplicate(ctx, f.user, o.DraftID, false)
	if err != nil || merged.Kind != capture.Merged {
		t.Fatalf("merge = %+v, %v", merged, err)
	}
	if f.balance(cash) != -45_000*sum {
		t.Errorf("balance after merge = %d", f.balance(cash)/sum)
	}

	o = f.one("obed 45 ming")
	kept, err := f.svc.AnswerDuplicate(ctx, f.user, o.DraftID, true)
	if err != nil || kept.Kind != capture.Saved {
		t.Fatalf("keep = %+v, %v", kept, err)
	}
	if f.balance(cash) != -90_000*sum {
		t.Errorf("balance after keep = %d", f.balance(cash)/sum)
	}
}

func TestSameMessageTwiceIsOneTransaction(t *testing.T) {
	f := setup(t)
	ref := fmt.Sprintf("test:%d:1", rand.Int64())
	a, err := f.svc.Capture(ctx, f.user, "taksi 25 ming", "chat", ref)
	if err != nil {
		t.Fatal(err)
	}
	b, err := f.svc.Capture(ctx, f.user, "taksi 25 ming", "chat", ref)
	if err != nil {
		t.Fatal(err)
	}
	if a[0].Tx.ID != b[0].Tx.ID {
		t.Errorf("replay created a second transaction: %d vs %d", a[0].Tx.ID, b[0].Tx.ID)
	}
	if f.balance(a[0].Tx.AccountID) != -25_000*sum {
		t.Errorf("balance = %d", f.balance(a[0].Tx.AccountID)/sum)
	}
}

func TestCategoryCorrectionIsLearned(t *testing.T) {
	f := setup(t)
	o := f.one("Ozbegim 120 ming") // a shop no keyword knows
	if *o.Tx.CategoryID != f.cats["other_expense"] || o.Item.CategoryOrigin != "default" {
		t.Fatalf("unknown shop = %+v", o.Item)
	}
	if _, err := f.svc.ChangeCategory(ctx, f.user, o.Tx.ID, f.cats["groceries"]); err != nil {
		t.Fatal(err)
	}
	o = f.one("ozbegim 50 ming")
	if *o.Tx.CategoryID != f.cats["groceries"] || o.Item.CategoryOrigin != "user" {
		t.Errorf("after correction = %+v", o.Item)
	}
}

func TestIncome(t *testing.T) {
	f := setup(t)
	o := f.one("oylik 8 mln")
	if o.Tx.Type != ledger.Income || *o.Tx.CategoryID != f.cats["salary"] || f.balance(o.Tx.AccountID) != 8_000_000*sum {
		t.Errorf("salary = %+v", o.Tx)
	}
	o = f.one("+500 ming")
	if o.Tx.Type != ledger.Income || *o.Tx.CategoryID != f.cats["other_income"] {
		t.Errorf("+500 = %+v", o.Tx)
	}
}

func TestDollarsGetTheirOwnAccount(t *testing.T) {
	f := setup(t)
	uzs := f.one("taksi 25 ming")
	usd := f.one("kofe 3$")
	if usd.Tx.Currency != "USD" || usd.Tx.AmountMinor != 300 || usd.NewAccount == "" || usd.Tx.AccountID == uzs.Tx.AccountID {
		t.Errorf("usd = %+v (new account %q)", usd.Tx, usd.NewAccount)
	}
}

func TestUndoAndChangeAccount(t *testing.T) {
	f := setup(t)
	o := f.one("taksi 25 ming")
	cash := o.Tx.AccountID
	humo := f.account("Humo", "humo", "UZS")

	moved, err := f.svc.ChangeAccount(ctx, f.user, o.Tx.ID, humo)
	if err != nil || moved.AccountID != humo {
		t.Fatalf("moved = %+v, %v", moved, err)
	}
	if f.balance(cash) != 0 || f.balance(humo) != -25_000*sum {
		t.Errorf("balances after move: cash %d humo %d", f.balance(cash), f.balance(humo))
	}
	if _, err := f.svc.UndoLast(ctx, f.user); err != nil {
		t.Fatal(err)
	}
	if f.balance(humo) != 0 {
		t.Errorf("balance after undo = %d", f.balance(humo))
	}
	if _, err := f.svc.UndoLast(ctx, f.user); err != ledger.ErrNotFound {
		t.Errorf("undo with nothing left: %v", err)
	}
}

func TestNoAmount(t *testing.T) {
	f := setup(t)
	if outs := f.say("salom"); len(outs) != 0 {
		t.Errorf("salom = %+v", outs)
	}
}

func ptr[T any](v T) *T { return &v }

func dbqRule(userID int64, pattern string) dbq.GetMerchantRuleParams {
	return dbq.GetMerchantRuleParams{UserID: userID, Pattern: pattern}
}

func mustDate(s string) time.Time {
	d, err := time.Parse(time.DateOnly, s)
	if err != nil {
		panic(err)
	}
	return d
}

func total(ms []ledger.Money) int64 {
	var t int64
	for _, m := range ms {
		t += m.AmountMinor
	}
	return t
}
