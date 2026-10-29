package api_test

import (
	"math/rand/v2"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/api"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

func TestSavingsByMonth(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken}, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))
	c.do("PATCH", "/api/me", map[string]any{"timezone": "Asia/Tashkent"}, 200)

	card := num(c.do("POST", "/api/accounts", map[string]any{"name": "Humo", "type": "humo", "currency": "UZS", "opening_balance_minor": 10_000_000_00}, 201)["id"])
	// The opening balance of a savings account is not money saved this month.
	bank := num(c.do("POST", "/api/accounts", map[string]any{"name": "Omonat", "type": "savings", "currency": "UZS", "opening_balance_minor": 5_000_000_00, "include_in_total": false}, 201)["id"])

	now := time.Now().In(time.FixedZone("UZT", 5*3600))
	today := now.Format(time.DateOnly)
	lastMonth := time.Date(now.Year(), now.Month()-1, 10, 0, 0, 0, 0, time.UTC)
	transfer := func(day string, amount int64, from, to int64) {
		c.do("POST", "/api/transactions", map[string]any{"type": "transfer", "occurred_on": day, "amount_minor": amount, "account_id": from, "to_account_id": to}, 201)
	}
	transfer(lastMonth.Format(time.DateOnly), 1_000_000_00, card, bank)
	transfer(today, 2_000_000_00, card, bank)
	transfer(today, 300_000_00, bank, card)
	deleted := num(c.do("POST", "/api/transactions", map[string]any{"type": "transfer", "occurred_on": today, "amount_minor": 999_00, "account_id": card, "to_account_id": bank}, 201)["id"])
	c.do("DELETE", "/api/transactions/"+strconv.FormatInt(deleted, 10), nil, 204)

	out := c.do("GET", "/api/savings?months=3", nil, 200)
	if out["current_month"] != now.Format("2006-01") {
		t.Fatalf("current_month = %v", out["current_month"])
	}
	got := map[string][2]int64{}
	for _, m := range out["months"].([]any) {
		m := m.(map[string]any)
		got[m["month"].(string)] = [2]int64{num(m["put_in_minor"]), num(m["taken_out_minor"])}
	}
	want := map[string][2]int64{
		lastMonth.Format("2006-01"): {1_000_000_00, 0},
		now.Format("2006-01"):       {2_000_000_00, 300_000_00},
	}
	if len(got) != len(want) || got[lastMonth.Format("2006-01")] != want[lastMonth.Format("2006-01")] || got[now.Format("2006-01")] != want[now.Format("2006-01")] {
		t.Fatalf("months = %v, want %v", got, want)
	}

	if out := c.do("GET", "/api/savings?months=1", nil, 200); len(out["months"].([]any)) != 1 {
		t.Fatalf("months=1 = %v", out)
	}
	c.do("GET", "/api/savings?months=0", nil, 400)
	// Another user's savings are not visible.
	if out := newClient(t, srv, rand.Int64N(1<<50)).do("GET", "/api/savings", nil, 200); len(out["months"].([]any)) != 0 {
		t.Fatalf("other user sees %v", out)
	}
}
