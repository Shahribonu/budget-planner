// Command loadtest simulates many Mini App users against a running server and reports
// latency per route. Each virtual user signs its own init data with the bot token the
// server was started with, sets up an account, then repeatedly opens Home, browses
// history and logs an expense, pausing between screens like a person would.
//
// Run the server with a throwaway token and database, never production's:
//
//	TELEGRAM_BOT_TOKEN=123:load BOT_MODE=off go run ./cmd/server
//	go run ./cmd/loadtest -token 123:load -users 500 -duration 2m
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/auth"
)

type sample struct {
	route  string
	status int
	took   time.Duration
}

type vu struct {
	base, authz string
	http        *http.Client
	out         chan<- sample
}

func (v *vu) call(route, method, path string, body any) (map[string]any, int) {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, v.base+path, &buf)
	req.Header.Set("Authorization", v.authz)
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	res, err := v.http.Do(req)
	if err != nil {
		v.out <- sample{route, 0, time.Since(start)}
		return nil, 0
	}
	defer res.Body.Close()
	var m map[string]any
	b, _ := io.ReadAll(res.Body)
	_ = json.Unmarshal(b, &m)
	v.out <- sample{route, res.StatusCode, time.Since(start)}
	return m, res.StatusCode
}

func main() {
	base := flag.String("url", "http://localhost:8080", "server base URL")
	token := flag.String("token", "", "bot token the server runs with (required)")
	users := flag.Int("users", 200, "concurrent virtual users")
	duration := flag.Duration("duration", time.Minute, "how long to run after setup")
	think := flag.Duration("think", 2*time.Second, "average pause between screens")
	flag.Parse()
	if *token == "" {
		fmt.Fprintln(os.Stderr, "-token is required")
		os.Exit(2)
	}

	samples := make(chan sample, 10_000)
	results := map[string][]sample{}
	done := make(chan struct{})
	go func() {
		for s := range samples {
			results[s.route] = append(results[s.route], s)
		}
		close(done)
	}()

	client := &http.Client{Timeout: 30 * time.Second, Transport: &http.Transport{MaxIdleConnsPerHost: *users}}
	runID := time.Now().Unix() % 1_000_000
	deadline := time.Now().Add(*duration)
	var wg sync.WaitGroup
	for i := range *users {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Synthetic Telegram ids far above real ones, unique per run.
			id := int64(9_000_000_000_000) + runID*100_000 + int64(i)
			vals := url.Values{}
			vals.Set("auth_date", strconv.FormatInt(time.Now().Unix(), 10))
			vals.Set("user", fmt.Sprintf(`{"id":%d,"first_name":"Load %d","language_code":"uz"}`, id, i))
			v := &vu{*base, "tma " + auth.SignInitData(vals, *token), client, samples}
			run(v, deadline, *think)
		}()
		time.Sleep(10 * time.Millisecond) // ramp up
	}
	wg.Wait()
	close(samples)
	<-done
	report(results, *duration)
}

func run(v *vu, deadline time.Time, think time.Duration) {
	v.call("GET /me", "GET", "/api/me", nil)
	acc, status := v.call("POST /accounts", "POST", "/api/accounts", map[string]any{
		"name": "Naqd", "type": "cash", "currency": "UZS", "opening_balance_minor": 5_000_000_00})
	if status != http.StatusCreated {
		return
	}
	accountID := acc["id"]
	cats, _ := v.call("GET /categories", "GET", "/api/categories", nil)
	var expense []any
	if list, ok := cats["categories"].([]any); ok {
		for _, c := range list {
			if m := c.(map[string]any); m["kind"] == "expense" {
				expense = append(expense, m["id"])
			}
		}
	}
	pause := func() { time.Sleep(time.Duration(rand.Int64N(int64(2 * think)))) }
	for time.Now().Before(deadline) {
		// Home: the app fires these together on open.
		var wg sync.WaitGroup
		for _, r := range [][2]string{{"GET /me", "/api/me"}, {"GET /accounts", "/api/accounts"}, {"GET /summary", "/api/summary"}, {"GET /transactions", "/api/transactions?limit=20"}} {
			wg.Add(1)
			go func() { defer wg.Done(); v.call(r[0], "GET", r[1], nil) }()
		}
		wg.Wait()
		pause()
		if len(expense) > 0 {
			v.call("POST /transactions", "POST", "/api/transactions", map[string]any{
				"type": "expense", "occurred_on": time.Now().Format(time.DateOnly), "account_id": accountID,
				"category_id": expense[rand.IntN(len(expense))], "amount_minor": (rand.Int64N(200) + 1) * 1_000_00, "description": "load",
			})
		}
		pause()
		switch rand.IntN(3) {
		case 0:
			v.call("GET /plan", "GET", "/api/plan?create=false", nil)
		case 1:
			v.call("GET /budgets", "GET", "/api/budgets", nil)
		default:
			v.call("GET /transactions", "GET", "/api/transactions?limit=50&type=expense", nil)
		}
		pause()
	}
}

func report(results map[string][]sample, d time.Duration) {
	routes := make([]string, 0, len(results))
	total, failed := 0, 0
	for r := range results {
		routes = append(routes, r)
	}
	slices.Sort(routes)
	fmt.Printf("%-20s %7s %6s %8s %8s %8s %8s\n", "route", "count", "errors", "p50", "p95", "p99", "max")
	for _, r := range routes {
		ss := results[r]
		lat := make([]time.Duration, len(ss))
		errs := 0
		for i, s := range ss {
			lat[i] = s.took
			// GET /plan?create=false answers 404 for a month without a plan, by design.
			if s.status == 0 || s.status >= 400 && !(s.status == http.StatusNotFound && r == "GET /plan") {
				errs++
			}
		}
		slices.Sort(lat)
		pct := func(p float64) time.Duration { return lat[min(len(lat)-1, int(p*float64(len(lat))))] }
		fmt.Printf("%-20s %7d %6d %8s %8s %8s %8s\n", r, len(ss), errs,
			pct(.50).Round(time.Millisecond), pct(.95).Round(time.Millisecond), pct(.99).Round(time.Millisecond), lat[len(lat)-1].Round(time.Millisecond))
		total += len(ss)
		failed += errs
	}
	fmt.Printf("\n%d requests, %d errors (%.2f%%), %.0f req/s\n", total, failed, 100*float64(failed)/float64(max(total, 1)), float64(total)/d.Seconds())
}
