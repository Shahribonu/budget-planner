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
