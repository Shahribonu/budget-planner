package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/api"
	"github.com/Shahribonu/budget-planner/backend/internal/auth"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

const botToken = "123456:TEST-token"

type client struct {
	t    *testing.T
	srv  *httptest.Server
	auth string
}

func newClient(t *testing.T, srv *httptest.Server, telegramID int64) *client {
	v := url.Values{}
	v.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
	v.Set("user", fmt.Sprintf(`{"id":%d,"first_name":"Aziz","language_code":"ru"}`, telegramID))
	return &client{t, srv, "tma " + auth.SignInitData(v, botToken)}
}

func (c *client) do(method, path string, body any, wantStatus int) map[string]any {
	c.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, c.srv.URL+path, &buf)
	req.Header.Set("Authorization", c.auth)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	if res.StatusCode != wantStatus {
		c.t.Fatalf("%s %s = %d %v, want %d", method, path, res.StatusCode, out, wantStatus)
	}
	return out
}

func num(v any) int64 { return int64(v.(float64)) }

// Manual tracking end to end: onboarding, accounts, logging, editing, undo, summary.
func TestManualTrackingEndToEnd(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken}, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))

	me := c.do("GET", "/api/me", nil, 200)
	if me["onboarded"] != false || me["language"] != "ru" {
		t.Errorf("new user = %v", me)
	}
	c.do("PATCH", "/api/me", map[string]any{"language": "uz-Latn", "main_currency": "UZS"}, 200)
	if me := c.do("PATCH", "/api/me", map[string]any{"notify_enabled": false, "notify_time": "22:15"}, 200); me["notify_enabled"] != false || me["notify_time"] != "22:15" {
		t.Errorf("notify settings = %v", me)
	}

	cash := num(c.do("POST", "/api/accounts", map[string]any{"name": "Naqd", "type": "cash", "currency": "UZS", "opening_balance_minor": 1_000_000_00}, 201)["id"])
	humo := num(c.do("POST", "/api/accounts", map[string]any{"name": "Humo", "type": "humo", "currency": "UZS"}, 201)["id"])
	if c.do("GET", "/api/me", nil, 200)["onboarded"] != true {
		t.Error("not onboarded after creating accounts")
	}

	var taxi int64
	for _, cat := range c.do("GET", "/api/categories", nil, 200)["categories"].([]any) {
		if m := cat.(map[string]any); m["key"] == "taxi" {
			taxi = num(m["id"])
		}
	}
	today := time.Now().Format(time.DateOnly)
	expense := map[string]any{"type": "expense", "occurred_on": today, "amount_minor": 25_000_00, "account_id": cash, "category_id": taxi, "description": "taksi"}
	tx := c.do("POST", "/api/transactions", expense, 201)
	id := num(tx["id"])
	if tx["currency"] != "UZS" || num(tx["amount_minor"]) != 25_000_00 || tx["occurred_on"] != today {
		t.Errorf("created = %v", tx)
	}

	c.do("POST", "/api/transactions", map[string]any{"type": "transfer", "occurred_on": today, "amount_minor": 200_000_00, "account_id": cash, "to_account_id": humo}, 201)

	expense["amount_minor"] = 30_000_00
	c.do("PUT", fmt.Sprintf("/api/transactions/%d", id), expense, 200)

	sum := c.do("GET", "/api/summary", nil, 200)
	if got := num(sum["expense"].([]any)[0].(map[string]any)["amount_minor"]); got != 30_000_00 {
		t.Errorf("summary expense = %d", got)
	}
	if got := num(sum["balances"].([]any)[0].(map[string]any)["amount_minor"]); got != 970_000_00 {
		t.Errorf("summary balance = %d", got)
	}

	c.do("DELETE", fmt.Sprintf("/api/transactions/%d", id), nil, 204)
	c.do("GET", fmt.Sprintf("/api/transactions/%d", id), nil, 404)
	c.do("POST", fmt.Sprintf("/api/transactions/%d/restore", id), nil, 200)

	if n := len(c.do("GET", "/api/transactions?type=expense,transfer", nil, 200)["transactions"].([]any)); n != 2 {
		t.Errorf("two types = %d", n)
	}
	b := c.do("POST", "/api/budgets", map[string]any{"category_id": taxi, "currency": "UZS", "amount_minor": 100_000_00}, 201)
	if got := c.do("PATCH", fmt.Sprintf("/api/budgets/%d", num(b["id"])), map[string]any{"amount_minor": 200_000_00}, 200); num(got["amount_minor"]) != 200_000_00 || num(got["spent_minor"]) != 30_000_00 {
		t.Errorf("patched budget = %v", got)
	}
	list := c.do("GET", "/api/transactions?type=expense&q=taks", nil, 200)
	if n := len(list["transactions"].([]any)); n != 1 {
		t.Errorf("list = %d items", n)
	}
	accs := c.do("GET", "/api/accounts", nil, 200)["accounts"].([]any)
	if got := num(accs[0].(map[string]any)["balance_minor"]); got != 770_000_00 {
		t.Errorf("cash balance = %d, want 770 000 so'm", got/100)
	}

	// Another Telegram user cannot see or touch it.
	other := newClient(t, srv, rand.Int64N(1<<50))
	other.do("GET", fmt.Sprintf("/api/transactions/%d", id), nil, 404)
	other.do("DELETE", fmt.Sprintf("/api/transactions/%d", id), nil, 404)
}

func TestAPIRejects(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken}, pool))
	defer srv.Close()

	bad := &client{t, srv, "tma auth_date=1&hash=00&user=%7B%22id%22%3A1%7D"}
	bad.do("GET", "/api/me", nil, 401)
	dev := &client{t, srv, "dev 42"} // dev auth is off unless configured
	dev.do("GET", "/api/me", nil, 401)

	c := newClient(t, srv, rand.Int64N(1<<50))
	c.do("POST", "/api/accounts", map[string]any{"name": "X", "type": "bank", "currency": "UZS"}, 422)
	c.do("POST", "/api/accounts", map[string]any{"name": "X", "type": "cash", "currency": "EUR"}, 422)
	c.do("POST", "/api/accounts", map[string]any{"name": "X", "type": "cash", "currency": "UZS", "balance": 1}, 400)
	c.do("POST", "/api/transactions", map[string]any{"type": "expense", "occurred_on": "26.10.2026"}, 400)
	c.do("PATCH", "/api/me", map[string]any{"timezone": "Mars/Base"}, 422)
	c.do("PATCH", "/api/me", map[string]any{"notify_time": "25:00"}, 422)
	c.do("GET", "/api/summary?month=2026-13", nil, 422)
}

func TestDebtsAndInstallmentsEndToEnd(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken}, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))
	today := time.Now().Format(time.DateOnly)

	cash := num(c.do("POST", "/api/accounts", map[string]any{"name": "Naqd", "type": "cash", "currency": "UZS", "opening_balance_minor": 1_000_000_00}, 201)["id"])
	d := c.do("POST", "/api/debts", map[string]any{
		"direction": "lent", "counterparty": "Aziz", "currency": "UZS", "amount_minor": 200_000_00,
		"account_id": cash, "occurred_on": today, "due_on": "2026-12-01", "note": "telefon uchun",
	}, 201)
	id := num(d["id"])
	if num(d["remaining_minor"]) != 200_000_00 || d["due_on"] != "2026-12-01" || d["closed"] != false {
		t.Errorf("created debt = %v", d)
	}
	c.do("POST", fmt.Sprintf("/api/debts/%d/payments", id), map[string]any{"amount_minor": 500_000_00, "account_id": cash, "occurred_on": today}, 422)
	pay := c.do("POST", fmt.Sprintf("/api/debts/%d/payments", id), map[string]any{"amount_minor": 50_000_00, "account_id": cash, "occurred_on": today}, 201)
	if pay["type"] != "debt_payment" || num(pay["debt_id"]) != id || pay["debt_direction"] != "lent" || pay["counterparty"] != "Aziz" {
		t.Errorf("payment = %v", pay)
	}
	if got := c.do("GET", "/api/debts", nil, 200)["debts"].([]any)[0].(map[string]any); got["occurred_on"] != today || num(got["account_id"]) != cash {
		t.Errorf("debt origin = %v", got)
	}
	// Undo a repayment, pay the rest, then try to restore the first one: that would overpay.
	c.do("DELETE", fmt.Sprintf("/api/transactions/%d", num(pay["id"])), nil, 204)
	rest := c.do("POST", fmt.Sprintf("/api/debts/%d/payments", id), map[string]any{"amount_minor": 200_000_00, "account_id": cash, "occurred_on": today}, 201)
	c.do("POST", fmt.Sprintf("/api/transactions/%d/restore", num(pay["id"])), nil, 422)
	c.do("DELETE", fmt.Sprintf("/api/transactions/%d", num(rest["id"])), nil, 204)
	c.do("POST", fmt.Sprintf("/api/transactions/%d/restore", num(pay["id"])), nil, 200)
	if n := len(c.do("GET", fmt.Sprintf("/api/transactions?debt_id=%d", id), nil, 200)["transactions"].([]any)); n != 2 {
		t.Errorf("transactions by debt = %d", n)
	}
	c.do("POST", "/api/debts", map[string]any{"direction": "lent", "counterparty": "X", "currency": "UZS", "amount_minor": 1, "occurred_on": today, "due_on": "2020-01-01"}, 422)
	// due_on: null clears it; omitting it keeps it.
	if got := c.do("PATCH", fmt.Sprintf("/api/debts/%d", id), map[string]any{"note": "x"}, 200); got["due_on"] != "2026-12-01" {
		t.Errorf("due kept = %v", got["due_on"])
	}
	if got := c.do("PATCH", fmt.Sprintf("/api/debts/%d", id), map[string]any{"due_on": nil}, 200); got["due_on"] != nil {
		t.Errorf("due cleared = %v", got["due_on"])
	}
	detail := c.do("GET", fmt.Sprintf("/api/debts/%d", id), nil, 200)
	if num(detail["debt"].(map[string]any)["remaining_minor"]) != 150_000_00 || len(detail["transactions"].([]any)) != 2 {
		t.Errorf("detail = %v", detail)
	}

	// An old debt touches no account: account_id is null.
	old := num(c.do("POST", "/api/debts", map[string]any{"direction": "borrowed", "counterparty": "Akam", "currency": "UZS", "amount_minor": 1_000_00, "occurred_on": today}, 201)["id"])
	oldTx := c.do("GET", fmt.Sprintf("/api/debts/%d", old), nil, 200)["transactions"].([]any)[0].(map[string]any)
	if oldTx["account_id"] != nil || oldTx["type"] != "debt" {
		t.Errorf("old debt tx = %v", oldTx)
	}

	var home int64
	for _, cat := range c.do("GET", "/api/categories", nil, 200)["categories"].([]any) {
		if m := cat.(map[string]any); m["key"] == "home" {
			home = num(m["id"])
		}
	}
	plan := c.do("POST", "/api/installments", map[string]any{
		"title": "Muzlatkich", "provider": "Uzum Nasiya", "category_id": home, "currency": "UZS",
		"total_minor": 3_000_000_00, "months": 3, "first_due_on": today, "pay_from_account_id": cash, "occurred_on": today,
	}, 201)
	pid := num(plan["id"])
	if len(plan["schedule"].([]any)) != 3 || num(plan["next"].(map[string]any)["amount_minor"]) != 1_000_000_00 {
		t.Errorf("plan = %v", plan)
	}
	c.do("POST", fmt.Sprintf("/api/installments/%d/payments", pid), map[string]any{"account_id": cash, "occurred_on": today}, 201)
	if n := len(c.do("GET", fmt.Sprintf("/api/transactions?installment_id=%d", pid), nil, 200)["transactions"].([]any)); n != 2 {
		t.Errorf("transactions by plan = %d", n)
	}
	c.do("GET", "/api/transactions?installment_id=999999999", nil, 404)
	if p := c.do("GET", fmt.Sprintf("/api/installments/%d", pid), nil, 200); num(p["remaining_minor"]) != 2_000_000_00 {
		t.Errorf("after payment = %v", p)
	}

	sum := c.do("GET", "/api/summary", nil, 200)
	owed := sum["owed_to_me"].([]any)[0].(map[string]any)
	owe := sum["i_owe"].([]any)[0].(map[string]any)
	inst := sum["installments"].([]any)[0].(map[string]any)
	if num(owed["amount_minor"]) != 150_000_00 || num(owe["amount_minor"]) != 1_000_00 || num(inst["amount_minor"]) != 2_000_000_00 {
		t.Errorf("summary owed %v, owe %v, installments %v", owed, owe, inst)
	}

	other := newClient(t, srv, rand.Int64N(1<<50))
	other.do("GET", fmt.Sprintf("/api/debts/%d", id), nil, 404)
	other.do("GET", fmt.Sprintf("/api/installments/%d", pid), nil, 404)
	c.do("DELETE", fmt.Sprintf("/api/installments/%d", pid), nil, 204)
	c.do("DELETE", fmt.Sprintf("/api/debts/%d", id), nil, 204)
	if n := len(c.do("GET", "/api/debts", nil, 200)["debts"].([]any)); n != 1 {
		t.Errorf("debts after delete = %d", n)
	}
}

func TestPlanEndToEnd(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken}, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))
	const q = "?month=2026-11&currency=UZS"

	c.do("POST", "/api/recurring", map[string]any{"kind": "income", "title": "Oylik", "currency": "UZS", "amount_minor": 8_000_000_00, "day_of_month": 10}, 201)
	rent := c.do("POST", "/api/recurring", map[string]any{"kind": "expense", "title": "Ijara", "currency": "UZS", "amount_minor": 3_000_000_00, "day_of_month": 1}, 201)
	c.do("POST", "/api/recurring", map[string]any{"kind": "expense", "title": "X", "currency": "UZS", "amount_minor": 1, "day_of_month": 32}, 422)
	var rentCat, salaryCat int64
	for _, cat := range c.do("GET", "/api/categories", nil, 200)["categories"].([]any) {
		m := cat.(map[string]any)
		switch m["key"] {
		case "rent":
			rentCat = num(m["id"])
		case "salary":
			salaryCat = num(m["id"])
		}
	}
	if got := c.do("PATCH", fmt.Sprintf("/api/recurring/%d", num(rent["id"])), map[string]any{"category_id": rentCat}, 200); num(got["category_id"]) != rentCat {
		t.Errorf("recurring category = %v", got)
	}
	c.do("PATCH", fmt.Sprintf("/api/recurring/%d", num(rent["id"])), map[string]any{"category_id": salaryCat}, 422)
	if got := c.do("PATCH", fmt.Sprintf("/api/recurring/%d", num(rent["id"])), map[string]any{"category_id": nil}, 200); got["category_id"] != nil {
		t.Errorf("category not cleared = %v", got)
	}

	preview := c.do("POST", "/api/plan/parse"+q, map[string]any{"text": "qishki kurtka 700-900 ming, AirPods 2.5 mln agar bo'lsa"}, 200)["items"].([]any)
	if len(preview) != 2 || preview[1].(map[string]any)["priority"] != "nice" {
		t.Fatalf("preview = %v", preview)
	}
	plan := c.do("POST", "/api/plan/items"+q, map[string]any{"items": preview}, 201)
	in := plan["inputs"].(map[string]any)
	if num(in["income_minor"]) != 8_000_000_00 || in["income_source"] != "recurring" || num(in["income_day"]) != 10 || num(in["reserved_minor"]) != 3_000_000_00 {
		t.Errorf("inputs = %v", in)
	}
	items := plan["items"].([]any)
	if len(items) != 2 || items[0].(map[string]any)["decision"] == nil {
		t.Fatalf("items = %v", items)
	}
	if line := in["lines"].([]any)[0].(map[string]any); num(line["ref_id"]) != num(rent["id"]) {
		t.Errorf("line ref = %v", line)
	}

	// Savings and an income override; then clear the override with null.
	plan = c.do("PATCH", "/api/plan"+q, map[string]any{"savings_minor": 1_000_000_00, "income_override": 9_000_000_00}, 200)
	if num(plan["inputs"].(map[string]any)["income_minor"]) != 9_000_000_00 || num(plan["result"].(map[string]any)["free_minor"]) != 5_000_000_00 {
		t.Errorf("after override = %v", plan["inputs"])
	}
	plan = c.do("PATCH", "/api/plan"+q, map[string]any{"income_override": nil}, 200)
	if plan["inputs"].(map[string]any)["income_source"] != "recurring" {
		t.Errorf("override not cleared = %v", plan["inputs"])
	}

	id := num(items[0].(map[string]any)["id"])
	plan = c.do("PATCH", fmt.Sprintf("/api/plan/items/%d", id), map[string]any{"status": "dropped"}, 200)
	for _, it := range plan["items"].([]any) {
		if m := it.(map[string]any); num(m["id"]) == id && (m["status"] != "dropped" || m["decision"] != nil) {
			t.Errorf("dropped item = %v", m)
		}
	}
	c.do("PATCH", fmt.Sprintf("/api/plan/items/%d", id), map[string]any{"priority": "urgent"}, 422)

	other := newClient(t, srv, rand.Int64N(1<<50))
	other.do("PATCH", fmt.Sprintf("/api/plan/items/%d", id), map[string]any{"status": "bought"}, 404)
	other.do("DELETE", fmt.Sprintf("/api/recurring/%d", num(rent["id"])), nil, 404)

	if left := len(c.do("DELETE", fmt.Sprintf("/api/plan/items/%d", id), nil, 200)["items"].([]any)); left != 1 {
		t.Errorf("items after delete = %d", left)
	}
	c.do("GET", "/api/plan?month=2030-01&currency=UZS&create=false", nil, 404)
	if got := c.do("GET", "/api/plan"+q+"&create=false", nil, 200); got["month"] != "2026-11" {
		t.Errorf("peek existing = %v", got["month"])
	}
	c.do("DELETE", fmt.Sprintf("/api/recurring/%d", num(rent["id"])), nil, 204)
	if n := len(c.do("GET", "/api/recurring", nil, 200)["recurring"].([]any)); n != 1 {
		t.Errorf("recurring left = %d", n)
	}
}
