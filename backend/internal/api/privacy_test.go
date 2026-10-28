package api_test

import (
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/api"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

// fakeDocs records sendDocument uploads; blocked makes them fail like a blocked bot.
type fakeDocs struct {
	mu      sync.Mutex
	files   map[string]string
	blocked bool
}

func (f *fakeDocs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.blocked {
		_, _ = io.WriteString(w, `{"ok":false,"error_code":403,"description":"Forbidden: bot was blocked by the user"}`)
		return
	}
	file, hdr, err := r.FormFile("document")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	b, _ := io.ReadAll(file)
	f.files[hdr.Filename] = string(b)
	_, _ = io.WriteString(w, `{"ok":true,"result":{}}`)
}

func TestTermsExportAndDelete(t *testing.T) {
	pool := testdb.Pool(t)
	docs := &fakeDocs{files: map[string]string{}}
	tgSrv := httptest.NewServer(docs)
	defer tgSrv.Close()
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken, Telegram: telegram.New(botToken).WithBaseURL(tgSrv.URL)}, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))

	if me := c.do("GET", "/api/me", nil, 200); me["terms_accepted"] != false {
		t.Errorf("new user terms_accepted = %v", me["terms_accepted"])
	}
	if me := c.do("PATCH", "/api/me", map[string]any{"accept_terms": true}, 200); me["terms_accepted"] != true {
		t.Errorf("after accept = %v", me["terms_accepted"])
	}

	cash := num(c.do("POST", "/api/accounts", map[string]any{"name": "Naqd", "type": "cash", "currency": "UZS", "opening_balance_minor": 500_000_00}, 201)["id"])
	var food int64
	for _, cat := range c.do("GET", "/api/categories", nil, 200)["categories"].([]any) {
		if m := cat.(map[string]any); m["key"] == "cafe" {
			food = num(m["id"])
		}
	}
	c.do("POST", "/api/transactions", map[string]any{"type": "expense", "amount_minor": 45_000_00, "category_id": food,
		"account_id": cash, "description": "tushlik, \"Rayhon\"", "occurred_on": "2026-10-20"}, 201)

	// A blocked bot releases the slot so the user can retry right after pressing Start.
	docs.setBlocked(true)
	c.do("POST", "/api/me/export", nil, 409)
	docs.setBlocked(false)
	c.do("POST", "/api/me/export", nil, 202)
	c.do("POST", "/api/me/export", nil, 429)

	var csvFile, jsonFile string
	for name, body := range docs.files {
		switch {
		case strings.HasSuffix(name, ".csv"):
			csvFile = body
		case strings.HasSuffix(name, ".json"):
			jsonFile = body
		}
	}
	if !strings.Contains(csvFile, `2026-10-20,expense,45000.00,UZS,Naqd,,,,Кафе`) || !strings.Contains(csvFile, `"tushlik, ""Rayhon"""`) {
		t.Errorf("csv = %q", csvFile)
	}
	if !strings.Contains(jsonFile, `"transactions"`) || !strings.Contains(jsonFile, `"Naqd"`) {
		t.Errorf("json = %.300s", jsonFile)
	}

	// Everything that links to accounts must go too.
	today := time.Now().Format(time.DateOnly)
	c.do("POST", "/api/debts", map[string]any{"direction": "lent", "counterparty": "Akam", "currency": "UZS", "amount_minor": 100_000_00, "occurred_on": today, "account_id": cash}, 201)
	c.do("POST", "/api/installments", map[string]any{"title": "Telefon", "provider": "Uzum Nasiya", "category_id": food, "currency": "UZS",
		"total_minor": 3_000_000_00, "months": 3, "first_due_on": today, "pay_from_account_id": cash, "occurred_on": today}, 201)
	c.do("POST", "/api/budgets", map[string]any{"category_id": food, "currency": "UZS", "amount_minor": 100_000_00}, 201)
	c.do("POST", "/api/recurring", map[string]any{"kind": "income", "title": "Oylik", "currency": "UZS", "amount_minor": 8_000_000_00, "day_of_month": 10}, 201)
	c.do("GET", "/api/plan", nil, 200)

	c.do("DELETE", "/api/me", map[string]any{}, 422)
	c.do("DELETE", "/api/me", map[string]any{"confirm": "delete"}, 204)
	// The next open starts from scratch.
	me := c.do("GET", "/api/me", nil, 200)
	if me["onboarded"] != false || me["terms_accepted"] != false {
		t.Errorf("after delete = %v", me)
	}
	if accs := c.do("GET", "/api/accounts", nil, 200)["accounts"].([]any); len(accs) != 0 {
		t.Errorf("accounts survived delete: %v", accs)
	}
}

func TestRateLimit(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken}, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))
	limited := 0
	for range 60 {
		req, _ := http.NewRequest("GET", srv.URL+"/api/categories", nil)
		req.Header.Set("Authorization", c.auth)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode == http.StatusTooManyRequests {
			limited++
			if res.Header.Get("Retry-After") == "" {
				t.Error("429 without Retry-After")
			}
		}
	}
	if limited == 0 {
		t.Error("60 requests in a burst were never limited")
	}
}

func (f *fakeDocs) setBlocked(b bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blocked = b
}

func TestFailedLoginsLimitedByIP(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken}, pool))
	defer srv.Close()
	status := func(auth string) int {
		req, _ := http.NewRequest("GET", srv.URL+"/api/me", nil)
		req.Header.Set("Authorization", auth)
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		return res.StatusCode
	}
	for i := range 20 {
		if got := status("tma auth_date=1&hash=00"); got != http.StatusUnauthorized {
			t.Fatalf("bad login %d = %d, want 401", i, got)
		}
	}
	if got := status("tma auth_date=1&hash=00"); got != http.StatusTooManyRequests {
		t.Errorf("21st bad login = %d, want 429", got)
	}
	// Valid users behind the same address are never locked out.
	if got := status(newClient(t, srv, rand.Int64N(1<<50)).auth); got != http.StatusOK {
		t.Errorf("valid login after failures = %d", got)
	}
}
