package planning_test

import (
	"context"
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/planning"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

var ctx = context.Background()

const sum = 100

type fx struct {
	t    *testing.T
	svc  *planning.Service
	led  *ledger.Service
	user int64
	cats map[string]int64
}

func setup(t *testing.T) *fx {
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
	l := ledger.New(pool)
	return &fx{t, planning.NewService(pool, l), l, u.ID, cats}
}

func ptr[T any](v T) *T { return &v }

var nov = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)

func (f *fx) decisions(v planning.View) map[string]planning.Decision {
	out := map[string]planning.Decision{}
	for _, it := range v.Items {
		if it.Decision != nil {
			out[it.Title] = *it.Decision
		}
	}
	return out
}

// Product plan section 7.4, through the database: recurring salary and bills, overrides
// for savings and buffer, and the four items.
func TestSection74ThroughDatabase(t *testing.T) {
	f := setup(t)
	for _, r := range []planning.NewRecurring{
		{Kind: "income", Title: "Oylik", Currency: "UZS", Amount: 8_000_000 * sum, Day: 1},
		{Kind: "expense", Title: "Ijara", CategoryID: ptr(f.cats["rent"]), Currency: "UZS", Amount: 3_000_000 * sum, Day: 5},
		{Kind: "expense", Title: "Kommunal", CategoryID: ptr(f.cats["utilities"]), Currency: "UZS", Amount: 700_000 * sum, Day: 10},
		{Kind: "expense", Title: "Nasiya", Currency: "UZS", Amount: 500_000 * sum, Day: 15},
	} {
		if _, err := f.svc.CreateRecurring(ctx, f.user, r); err != nil {
			t.Fatal(err)
		}
	}
	cash, err := f.led.CreateAccount(ctx, f.user, ledger.NewAccount{Name: "Humo", Type: "humo", Currency: "UZS", OpeningBalanceMinor: 8_000_000 * sum})
	if err != nil {
		t.Fatal(err)
	}
	_ = cash
	v, err := f.svc.UpdateSettings(ctx, f.user, nov, "UZS", planning.Settings{Savings: ptr(int64(800_000 * sum)), SetBuffer: true, BufferOverride: ptr(int64(2_100_000 * sum))})
	if err != nil {
		t.Fatal(err)
	}
	if v.Inputs.Income != 8_000_000*sum || v.Inputs.IncomeSource != "recurring" || v.Inputs.Reserved != 4_200_000*sum || v.Result.Free != 900_000*sum {
		t.Fatalf("inputs = %+v, free %d", v.Inputs, v.Result.Free/sum)
	}
	v, err = f.svc.AddItems(ctx, f.user, nov, "UZS", []planning.NewItem{
		{Title: "Dentist", CategoryID: ptr(f.cats["health"]), AmountMin: 500_000 * sum, Priority: "must"},
		{Title: "Winter jacket", CategoryID: ptr(f.cats["clothing"]), AmountMin: 800_000 * sum, Priority: "should"},
		{Title: "Wedding gift", CategoryID: ptr(f.cats["gifts"]), AmountMin: 300_000 * sum, Priority: "should", DueOn: ptr(time.Date(2026, 11, 18, 0, 0, 0, 0, time.UTC))},
		{Title: "AirPods", AmountMin: 2_500_000 * sum, Priority: "nice"},
	})
	if err != nil {
		t.Fatal(err)
	}
	d := f.decisions(v)
	if d["Dentist"].Group != planning.Fits || d["Wedding gift"].Group != planning.Fits || d["Winter jacket"].Group != planning.Later || d["AirPods"].Group != planning.Reconsider {
		t.Errorf("decisions = %+v", d)
	}
	if v.Result.Left != 100_000*sum {
		t.Errorf("left = %d", v.Result.Left/sum)
	}
}

func TestParseItems(t *testing.T) {
	f := setup(t)
	items, err := f.svc.ParseItems(ctx, f.user, "UZS", "ijara 3 mln, tish shifokori taxminan 500 ming, qishki kurtka 700-900 ming, AirPods 2.5 mln agar bo'lsa")
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		title    string
		min, max int64
		prio     string
		cat      string
	}{
		{"ijara", 3_000_000, 3_000_000, "must", "rent"},
		{"tish shifokori", 500_000, 500_000, "must", "health"},
		{"qishki kurtka", 700_000, 900_000, "should", "clothing"},
		{"AirPods", 2_500_000, 2_500_000, "nice", ""},
	}
	if len(items) != len(want) {
		t.Fatalf("items = %+v", items)
	}
	for i, w := range want {
		it := items[i]
		cat := ""
		for k, id := range f.cats {
			if it.CategoryID != nil && *it.CategoryID == id {
				cat = k
			}
		}
		if it.Title != w.title || it.AmountMin != w.min*sum || it.AmountMax != w.max*sum || it.Priority != w.prio || cat != w.cat {
			t.Errorf("item %d = %+v (cat %q), want %+v", i, it, cat, w)
		}
	}
}

func TestCarryOverAndMatch(t *testing.T) {
	f := setup(t)
	oct := nov.AddDate(0, -1, 0)
	v, err := f.svc.AddItems(ctx, f.user, oct, "UZS", []planning.NewItem{
		{Title: "Kurtka", CategoryID: ptr(f.cats["clothing"]), AmountMin: 700_000 * sum, AmountMax: 900_000 * sum, Priority: "should"},
		{Title: "Kitob", CategoryID: ptr(f.cats["education"]), AmountMin: 100_000 * sum, Priority: "should"},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Buy the book in October.
	cash, _ := f.led.CreateAccount(ctx, f.user, ledger.NewAccount{Name: "Naqd", Type: "cash", Currency: "UZS", OpeningBalanceMinor: 5_000_000 * sum})
	tx, _, err := f.led.Record(ctx, f.user, ledger.Input{Type: ledger.Expense, OccurredOn: time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC),
		AmountMinor: 110_000 * sum, AccountID: cash.ID, CategoryID: ptr(f.cats["education"]), Description: "kitob", Source: "manual"})
	if err != nil {
		t.Fatal(err)
	}
	match, err := f.svc.MatchPurchase(ctx, f.user, tx)
	if err != nil || match == nil || match.Title != "Kitob" {
		t.Fatalf("match = %+v, %v", match, err)
	}
	if title, err := f.svc.MarkBought(ctx, f.user, match.ID, tx.ID); err != nil || title != "Kitob" {
		t.Fatalf("mark bought = %q, %v", title, err)
	}
	// A far cheaper purchase in the category is not the jacket.
	small, _, _ := f.led.Record(ctx, f.user, ledger.Input{Type: ledger.Expense, OccurredOn: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
		AmountMinor: 50_000 * sum, AccountID: cash.ID, CategoryID: ptr(f.cats["clothing"]), Source: "manual"})
	if m, _ := f.svc.MatchPurchase(ctx, f.user, small); m != nil {
		t.Errorf("matched %+v", m)
	}

	// November: the jacket carries over, the book does not.
	nv, err := f.svc.Get(ctx, f.user, nov, "UZS")
	if err != nil {
		t.Fatal(err)
	}
	if len(nv.Items) != 1 || nv.Items[0].Title != "Kurtka" || nv.Items[0].PostponedCount != 1 || nv.Items[0].AmountMax != 900_000*sum {
		t.Errorf("november items = %+v", nv.Items)
	}
	ov, _ := f.svc.Get(ctx, f.user, oct, "UZS")
	statuses := map[string]string{}
	for _, it := range ov.Items {
		statuses[it.Title] = it.Status
	}
	if statuses["Kurtka"] != "postponed" || statuses["Kitob"] != "bought" {
		t.Errorf("october statuses = %v", statuses)
	}
	// Opening November again doesn't carry it twice.
	if nv, _ = f.svc.Get(ctx, f.user, nov, "UZS"); len(nv.Items) != 1 {
		t.Errorf("carried twice: %d items", len(nv.Items))
	}
	_ = v
}

func TestInputsFromHistoryAndObligations(t *testing.T) {
	f := setup(t)
	acc, _ := f.led.CreateAccount(ctx, f.user, ledger.NewAccount{Name: "Humo", Type: "humo", Currency: "UZS"})
	// An installment payment and a borrowed debt due in November are obligations.
	if _, err := f.led.CreatePlan(ctx, f.user, ledger.NewPlan{Title: "Telefon", CategoryID: f.cats["home"], Currency: "UZS",
		TotalMinor: 3_000_000 * sum, Months: 3, FirstDueOn: time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC),
		OccurredOn: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), RecordPurchase: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.led.CreateDebt(ctx, f.user, ledger.NewDebt{Direction: "borrowed", Counterparty: "Akam", Currency: "UZS",
		AmountMinor: 400_000 * sum, AccountID: &acc.ID, OccurredOn: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		DueOn: ptr(time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC))}); err != nil {
		t.Fatal(err)
	}
	v, err := f.svc.Get(ctx, f.user, nov, "UZS")
	if err != nil {
		t.Fatal(err)
	}
	if v.Inputs.Reserved != 1_400_000*sum || len(v.Inputs.Lines) != 2 {
		t.Errorf("reserved = %d, lines %+v", v.Inputs.Reserved/sum, v.Inputs.Lines)
	}
	if v.Inputs.IncomeSource != "none" || v.Inputs.BufferSource != "none" {
		t.Errorf("sources = %s / %s (no history yet)", v.Inputs.IncomeSource, v.Inputs.BufferSource)
	}
}
