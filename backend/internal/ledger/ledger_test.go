package ledger_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

var ctx = context.Background()

type fixture struct {
	t    *testing.T
	pool *pgxpool.Pool
	svc  *ledger.Service
	user int64
	cats map[string]int64
}

func setup(t *testing.T) *fixture {
	pool := testdb.Pool(t)
	u := testdb.User(t, pool)
	cs, err := dbq.New(pool).ListCategories(ctx, &u.ID)
	if err != nil {
		t.Fatal(err)
	}
	cats := map[string]int64{}
	for _, c := range cs {
		cats[*c.Key] = c.ID
	}
	return &fixture{t, pool, ledger.New(pool), u.ID, cats}
}

func (f *fixture) account(name, typ, cur string, opening int64) int64 {
	f.t.Helper()
	a, err := f.svc.CreateAccount(ctx, f.user, ledger.NewAccount{Name: name, Type: typ, Currency: cur, OpeningBalanceMinor: opening})
	if err != nil {
		f.t.Fatal(err)
	}
	return a.ID
}

func (f *fixture) record(in ledger.Input) ledger.Transaction {
	f.t.Helper()
	if in.OccurredOn.IsZero() {
		in.OccurredOn = time.Now().UTC().Truncate(24 * time.Hour)
	}
	tx, _, err := f.svc.Record(ctx, f.user, in)
	if err != nil {
		f.t.Fatalf("record %+v: %v", in, err)
	}
	return tx
}

func (f *fixture) balances() map[int64]int64 {
	f.t.Helper()
	accs, err := f.svc.ListAccounts(ctx, f.user, true)
	if err != nil {
		f.t.Fatal(err)
	}
	out := map[int64]int64{}
	for _, a := range accs {
		out[a.ID] = a.BalanceMinor
	}
	return out
}

func (f *fixture) summary() ledger.Summary {
	f.t.Helper()
	s, err := f.svc.Summary(ctx, f.user, "")
	if err != nil {
		f.t.Fatal(err)
	}
	return s
}

func total(ms []ledger.Money, cur string) int64 {
	for _, m := range ms {
		if m.Currency == cur {
			return m.AmountMinor
		}
	}
	return 0
}

func ptr[T any](v T) *T { return &v }

const sum = 100 // UZS minor units per so'm

// Product plan section 18: the balance invariant.
func TestBalanceInvariant(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 1_000_000*sum)
	humo := f.account("Humo", "humo", "UZS", 0)
	usd := f.account("Dollar", "cash", "USD", 100*100)

	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 100_000 * sum, AccountID: cash, CategoryID: ptr(f.cats["groceries"])})
	before := total(f.summary().Balances, "UZS")
	f.record(ledger.Input{Type: ledger.Transfer, AmountMinor: 200_000 * sum, AccountID: cash, ToAccountID: &humo})
	s := f.summary()

	if got := total(s.Balances, "UZS"); got != 900_000*sum || got != before {
		t.Errorf("UZS wealth = %d, want 900 000 so'm and unchanged by transfer (%d)", got/sum, before/sum)
	}
	b := f.balances()
	if b[cash] != 700_000*sum || b[humo] != 200_000*sum {
		t.Errorf("cash = %d, humo = %d; want 700 000 and 200 000", b[cash]/sum, b[humo]/sum)
	}
	if got := total(s.Expense, "UZS"); got != 100_000*sum {
		t.Errorf("expense = %d, want 100 000 (transfer must not count)", got/sum)
	}
	if got := total(s.Income, "UZS"); got != 0 {
		t.Errorf("income = %d, want 0 (opening balance and transfer must not count)", got/sum)
	}

	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 20 * 100, AccountID: usd, CategoryID: ptr(f.cats["cafe"])})
	s = f.summary()
	if got := total(s.Balances, "UZS"); got != 900_000*sum {
		t.Errorf("USD expense changed UZS total to %d", got/sum)
	}
	if got := total(s.Balances, "USD"); got != 80*100 {
		t.Errorf("USD balance = %d, want 80", got/100)
	}
	if got := total(s.Expense, "USD"); got != 20*100 {
		t.Errorf("USD expense = %d, want 20", got/100)
	}
}

func TestExchange(t *testing.T) {
	f := setup(t)
	usd := f.account("Dollar", "cash", "USD", 100*100)
	cash := f.account("Naqd", "cash", "UZS", 0)

	tx := f.record(ledger.Input{Type: ledger.Exchange, AmountMinor: 100 * 100, AccountID: usd, ToAccountID: &cash, ToAmountMinor: ptr(int64(1_265_000 * sum))})
	if tx.AccountID != usd || *tx.ToAccountID != cash || tx.AmountMinor != 100*100 || *tx.ToAmountMinor != 1_265_000*sum || *tx.ToCurrency != "UZS" {
		t.Errorf("exchange shape = %+v", tx)
	}
	b := f.balances()
	if b[usd] != 0 || b[cash] != 1_265_000*sum {
		t.Errorf("balances = %v", b)
	}
	s := f.summary()
	if len(s.Income) != 0 || len(s.Expense) != 0 {
		t.Errorf("exchange counted as income/expense: %+v %+v", s.Income, s.Expense)
	}
}

func TestIncomeAndCategories(t *testing.T) {
	f := setup(t)
	humo := f.account("Humo", "humo", "UZS", 0)
	f.record(ledger.Input{Type: ledger.Income, AmountMinor: 8_000_000 * sum, AccountID: humo, CategoryID: ptr(f.cats["salary"])})
	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 25_000 * sum, AccountID: humo, CategoryID: ptr(f.cats["taxi"])})
	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 15_000 * sum, AccountID: humo, CategoryID: ptr(f.cats["taxi"])})
	f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 90_000 * sum, AccountID: humo, CategoryID: ptr(f.cats["cafe"])})

	s := f.summary()
	if got := total(s.Income, "UZS"); got != 8_000_000*sum {
		t.Errorf("income = %d", got/sum)
	}
	if len(s.ByCategory) != 2 || s.ByCategory[0].CategoryID != f.cats["cafe"] || s.ByCategory[1].AmountMinor != 40_000*sum {
		t.Errorf("by category = %+v", s.ByCategory)
	}
	if len(s.Recent) != 4 {
		t.Errorf("recent = %d items", len(s.Recent))
	}
}

func TestEditDeleteRestore(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 500_000*sum)
	humo := f.account("Humo", "humo", "UZS", 0)
	tx := f.record(ledger.Input{Type: ledger.Expense, AmountMinor: 45_000 * sum, AccountID: cash, CategoryID: ptr(f.cats["cafe"]), Description: "tushlik"})

	// Edit amount and account.
	edited, err := f.svc.Update(ctx, f.user, tx.ID, ledger.Input{
		Type: ledger.Expense, OccurredOn: tx.OccurredOn, AmountMinor: 50_000 * sum, AccountID: humo,
		CategoryID: ptr(f.cats["cafe"]), Description: "tushlik", Source: "manual",
	})
	if err != nil {
		t.Fatal(err)
	}
	if edited.AmountMinor != 50_000*sum || edited.AccountID != humo {
		t.Errorf("edited = %+v", edited)
	}
	if b := f.balances(); b[cash] != 500_000*sum || b[humo] != -50_000*sum {
		t.Errorf("after edit balances = %v", b)
	}

	// Change type: expense → transfer.
	if _, err := f.svc.Update(ctx, f.user, tx.ID, ledger.Input{
		Type: ledger.Transfer, OccurredOn: tx.OccurredOn, AmountMinor: 50_000 * sum, AccountID: cash, ToAccountID: &humo, Source: "manual",
	}); err != nil {
		t.Fatal(err)
	}
	if s := f.summary(); total(s.Expense, "UZS") != 0 {
		t.Errorf("expense after turning into transfer = %d", total(s.Expense, "UZS"))
	}

	if err := f.svc.Delete(ctx, f.user, tx.ID); err != nil {
		t.Fatal(err)
	}
	if b := f.balances(); b[cash] != 500_000*sum || b[humo] != 0 {
		t.Errorf("after delete balances = %v", b)
	}
	if _, err := f.svc.Get(ctx, f.user, tx.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Errorf("deleted Get err = %v", err)
	}
	if err := f.svc.Delete(ctx, f.user, tx.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Errorf("second delete err = %v", err)
	}
	if _, err := f.svc.Restore(ctx, f.user, tx.ID); err != nil {
		t.Fatal(err)
	}
	if b := f.balances(); b[cash] != 450_000*sum || b[humo] != 50_000*sum {
		t.Errorf("after restore balances = %v", b)
	}
}

// Product plan section 18: replay the same update 5 times, expect one transaction.
func TestIdempotentSourceRef(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 0)
	in := ledger.Input{Type: ledger.Expense, AmountMinor: 25_000 * sum, AccountID: cash, CategoryID: ptr(f.cats["taxi"]),
		Source: "chat", SourceRef: ptr("update:777"), OccurredOn: time.Now().UTC().Truncate(24 * time.Hour)}
	var first ledger.Transaction
	for i := range 5 {
		tx, created, err := f.svc.Record(ctx, f.user, in)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			first = tx
			if !created {
				t.Error("first record not created")
			}
		} else if created || tx.ID != first.ID {
			t.Errorf("replay %d: created=%v id=%d, want existing %d", i, created, tx.ID, first.ID)
		}
	}
	if b := f.balances(); b[cash] != -25_000*sum {
		t.Errorf("balance = %d, want one expense", b[cash]/sum)
	}
}

func TestOwnership(t *testing.T) {
	a, b := setup(t), setup(t)
	aCash := a.account("Naqd", "cash", "UZS", 100_000*sum)
	bCash := b.account("Naqd", "cash", "UZS", 0)
	aTx := a.record(ledger.Input{Type: ledger.Expense, AmountMinor: 1000 * sum, AccountID: aCash, CategoryID: ptr(a.cats["taxi"])})

	today := time.Now().UTC().Truncate(24 * time.Hour)
	if _, _, err := b.svc.Record(ctx, b.user, ledger.Input{Type: ledger.Expense, OccurredOn: today, AmountMinor: 1, AccountID: aCash, CategoryID: ptr(b.cats["taxi"]), Source: "manual"}); !isValidation(err, "invalid_account") {
		t.Errorf("spending from another user's account: %v", err)
	}
	if _, _, err := b.svc.Record(ctx, b.user, ledger.Input{Type: ledger.Transfer, OccurredOn: today, AmountMinor: 1, AccountID: bCash, ToAccountID: &aCash, Source: "manual"}); !isValidation(err, "invalid_account") {
		t.Errorf("transfer into another user's account: %v", err)
	}
	if _, err := b.svc.Get(ctx, b.user, aTx.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Errorf("reading another user's transaction: %v", err)
	}
	if err := b.svc.Delete(ctx, b.user, aTx.ID); !errors.Is(err, ledger.ErrNotFound) {
		t.Errorf("deleting another user's transaction: %v", err)
	}
	if _, err := b.svc.UpdateAccount(ctx, b.user, aCash, ledger.AccountPatch{Name: ptr("x")}); !errors.Is(err, ledger.ErrNotFound) {
		t.Errorf("renaming another user's account: %v", err)
	}
	if ts, _, _ := b.svc.List(ctx, b.user, ledger.Filter{}); len(ts) != 0 {
		t.Errorf("b sees %d transactions", len(ts))
	}
}

func TestValidation(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 0)
	humo := f.account("Humo", "humo", "UZS", 0)
	usd := f.account("Dollar", "cash", "USD", 0)
	archived := f.account("Old", "uzcard", "UZS", 0)
	if _, err := f.svc.UpdateAccount(ctx, f.user, archived, ledger.AccountPatch{Archived: ptr(true)}); err != nil {
		t.Fatal(err)
	}
	today := time.Now().UTC().Truncate(24 * time.Hour)
	base := ledger.Input{Type: ledger.Expense, OccurredOn: today, AmountMinor: 100, AccountID: cash, CategoryID: ptr(f.cats["taxi"]), Source: "manual"}
	with := func(mod func(*ledger.Input)) ledger.Input { in := base; mod(&in); return in }

	tests := []struct {
		name string
		in   ledger.Input
		code string
	}{
		{"zero amount", with(func(in *ledger.Input) { in.AmountMinor = 0 }), "invalid_amount"},
		{"negative amount", with(func(in *ledger.Input) { in.AmountMinor = -5 }), "invalid_amount"},
		{"no category", with(func(in *ledger.Input) { in.CategoryID = nil }), "category_required"},
		{"income category on expense", with(func(in *ledger.Input) { in.CategoryID = ptr(f.cats["salary"]) }), "invalid_category"},
		{"far future", with(func(in *ledger.Input) { in.OccurredOn = today.AddDate(0, 0, 3) }), "invalid_date"},
		{"archived account", with(func(in *ledger.Input) { in.AccountID = archived }), "archived_account"},
		{"opening via API", with(func(in *ledger.Input) { in.Type = ledger.Opening }), "invalid_type"},
		{"transfer to self", with(func(in *ledger.Input) { in.Type, in.ToAccountID = ledger.Transfer, &cash }), "same_account"},
		{"transfer across currencies", with(func(in *ledger.Input) { in.Type, in.ToAccountID = ledger.Transfer, &usd }), "currency_mismatch"},
		{"exchange same currency", with(func(in *ledger.Input) {
			in.Type, in.ToAccountID, in.ToAmountMinor = ledger.Exchange, &humo, ptr(int64(1))
		}), "same_currency"},
		{"exchange without amount", with(func(in *ledger.Input) { in.Type, in.ToAccountID = ledger.Exchange, &usd }), "invalid_amount"},
		{"transfer without target", with(func(in *ledger.Input) { in.Type = ledger.Transfer }), "to_account_required"},
	}
	for _, tt := range tests {
		if _, _, err := f.svc.Record(ctx, f.user, tt.in); !isValidation(err, tt.code) {
			t.Errorf("%s: err = %v, want %s", tt.name, err, tt.code)
		}
	}
}

// The database itself refuses unbalanced postings, even if the service had a bug.
func TestDatabaseRejectsUnbalancedPostings(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 0)
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var id int64
	if err := tx.QueryRow(ctx, `insert into transactions (user_id, type, occurred_on, category_id) values ($1, 'expense', current_date, $2) returning id`, f.user, f.cats["taxi"]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `insert into postings (transaction_id, account_id, amount, currency) values ($1, $2, -100, 'UZS')`, id, cash); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err == nil {
		t.Fatal("unbalanced transaction committed")
	}

	// A posting in a currency other than its account's is refused too.
	_, err = f.pool.Exec(ctx, `insert into postings (transaction_id, account_id, amount, currency) values ($1, $2, 100, 'USD')`, id, cash)
	if err == nil {
		t.Fatal("currency mismatch accepted")
	}
}

func TestListFiltersAndPaging(t *testing.T) {
	f := setup(t)
	cash := f.account("Naqd", "cash", "UZS", 0)
	usd := f.account("Dollar", "cash", "USD", 0)
	day := func(d int) time.Time { return time.Date(2026, 9, d, 0, 0, 0, 0, time.UTC) }
	for d := 1; d <= 5; d++ {
		f.record(ledger.Input{Type: ledger.Expense, OccurredOn: day(d), AmountMinor: int64(d) * 1000 * sum, AccountID: cash, CategoryID: ptr(f.cats["taxi"]), Description: "taksi 100%_"})
	}
	f.record(ledger.Input{Type: ledger.Expense, OccurredOn: day(3), AmountMinor: 500, AccountID: usd, CategoryID: ptr(f.cats["cafe"]), Description: "coffee"})

	var all []ledger.Transaction
	cursor := ""
	for {
		page, next, err := f.svc.List(ctx, f.user, ledger.Filter{Limit: 2, Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, page...)
		if next == "" {
			break
		}
		cursor = next
	}
	if len(all) != 6 {
		t.Fatalf("paged %d transactions, want 6", len(all))
	}
	for i := 1; i < len(all); i++ {
		if all[i].OccurredOn.After(all[i-1].OccurredOn) {
			t.Errorf("not newest first at %d", i)
		}
	}

	count := func(fl ledger.Filter) int {
		ts, _, err := f.svc.List(ctx, f.user, fl)
		if err != nil {
			t.Fatal(err)
		}
		return len(ts)
	}
	if n := count(ledger.Filter{Currency: ptr("USD")}); n != 1 {
		t.Errorf("currency filter = %d", n)
	}
	if n := count(ledger.Filter{AccountID: &cash}); n != 5 {
		t.Errorf("account filter = %d", n)
	}
	if n := count(ledger.Filter{CategoryID: ptr(f.cats["cafe"])}); n != 1 {
		t.Errorf("category filter = %d", n)
	}
	if n := count(ledger.Filter{From: ptr(day(2)), To: ptr(day(3))}); n != 3 {
		t.Errorf("date filter = %d", n)
	}
	if n := count(ledger.Filter{Q: ptr("COFF")}); n != 1 {
		t.Errorf("search = %d", n)
	}
	if n := count(ledger.Filter{Q: ptr("%")}); n != 5 { // literal %, not a wildcard
		t.Errorf("search for %% = %d", n)
	}
}

func isValidation(err error, code string) bool {
	var ve *ledger.ValidationError
	return errors.As(err, &ve) && ve.Code == code
}
