package bot_test

import (
	"context"
	"encoding/json"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/bot"
	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/stt"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/planning"
	"github.com/Shahribonu/budget-planner/backend/internal/telegram"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

var ctx = context.Background()

// fakeTelegram records Bot API calls.
type fakeTelegram struct {
	mu    sync.Mutex
	calls []call
}

type call struct {
	method string
	body   map[string]any
}

func (f *fakeTelegram) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/file/") {
		_, _ = w.Write([]byte("OggS fake audio"))
		return
	}
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	var body map[string]any
	b, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(b, &body)
	f.mu.Lock()
	f.calls = append(f.calls, call{method, body})
	f.mu.Unlock()
	result := any(true)
	if method == "createInvoiceLink" {
		result = "https://t.me/$invoice"
	}
	if method == "getFile" {
		result = map[string]any{"file_id": "f", "file_path": "voice/1.oga", "file_size": 15}
	}
	if method == "sendMessage" {
		result = map[string]any{"message_id": 1, "chat": map[string]any{"id": 1, "type": "private"}, "date": 0}
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": result})
}

func (f *fakeTelegram) take() []call {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := f.calls
	f.calls = nil
	return c
}

type harness struct {
	t      *testing.T
	b      *bot.Bot
	tg     *fakeTelegram
	userID int64
	update int64
	msg    int64
}

func setup(t *testing.T, transcriber stt.Transcriber) *harness {
	pool := testdb.Pool(t)
	fake := &fakeTelegram{}
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)
	l := ledger.New(pool)
	tg := telegram.New("TOKEN").WithBaseURL(srv.URL)
	b := bot.New(bot.Config{MiniAppURL: "https://app.example"}, tg, pool, capture.New(pool, l, nil), planning.NewService(pool, l),
		transcriber)
	return &harness{t: t, b: b, tg: fake, userID: rand.Int64N(1 << 50), update: rand.Int64N(1 << 40)}
}

func (h *harness) message(text string) telegram.Update {
	h.update++
	h.msg++
	return telegram.Update{UpdateID: h.update, Message: &telegram.Message{
		MessageID: h.msg, Text: text, Chat: telegram.Chat{ID: h.userID, Type: "private"},
		From: &telegram.User{ID: h.userID, FirstName: "Aziz", LanguageCode: "uz"},
	}}
}

func (h *harness) callback(data string) telegram.Update {
	h.update++
	return telegram.Update{UpdateID: h.update, CallbackQuery: &telegram.CallbackQuery{
		ID: "cb", Data: data, From: telegram.User{ID: h.userID, FirstName: "Aziz"},
		Message: &telegram.Message{MessageID: 99, Chat: telegram.Chat{ID: h.userID, Type: "private"}},
	}}
}

func sent(calls []call, method string) []call {
	var out []call
	for _, c := range calls {
		if c.method == method {
			out = append(out, c)
		}
	}
	return out
}

// buttonData collects callback_data values of an inline keyboard.
func buttonData(c call) []string {
	var out []string
	kb, _ := c.body["reply_markup"].(map[string]any)
	rows, _ := kb["inline_keyboard"].([]any)
	for _, r := range rows {
		for _, b := range r.([]any) {
			if d, ok := b.(map[string]any)["callback_data"].(string); ok {
				out = append(out, d)
			}
		}
	}
	return out
}

func TestChatLoggingFlow(t *testing.T) {
	h := setup(t, stt.None{})

	h.b.HandleUpdate(ctx, h.message("/start"))
	msgs := sent(h.tg.take(), "sendMessage")
	if len(msgs) != 1 || !strings.Contains(msgs[0].body["text"].(string), "Aziz") {
		t.Fatalf("/start = %+v", msgs)
	}

	upd := h.message("taksi 25 ming")
	h.b.HandleUpdate(ctx, upd)
	msgs = sent(h.tg.take(), "sendMessage")
	if len(msgs) != 2 { // the card, then the one-time evening summary note
		t.Fatalf("log replies = %d: %+v", len(msgs), msgs)
	}
	card := msgs[0].body["text"].(string)
	if !strings.Contains(card, "Taksi") || !strings.Contains(card, "25 000 so'm") || !strings.Contains(card, "bugun") {
		t.Errorf("card = %q", card)
	}
	var undo string
	for _, d := range buttonData(msgs[0]) {
		if strings.HasPrefix(d, "u:") {
			undo = d
		}
	}
	if undo == "" {
		t.Fatalf("no undo button: %v", buttonData(msgs[0]))
	}

	// Telegram redelivers the same update: nothing happens twice.
	h.b.HandleUpdate(ctx, upd)
	if c := h.tg.take(); len(c) != 0 {
		t.Errorf("replayed update produced %d calls", len(c))
	}

	h.b.HandleUpdate(ctx, h.callback(undo))
	calls := h.tg.take()
	edits := sent(calls, "editMessageText")
	if len(edits) != 1 || !strings.Contains(edits[0].body["text"].(string), "Bekor qilindi") {
		t.Errorf("undo edit = %+v", edits)
	}
	if len(sent(calls, "answerCallbackQuery")) != 1 {
		t.Error("callback not answered")
	}

	h.b.HandleUpdate(ctx, h.message("/undo"))
	if msgs := sent(h.tg.take(), "sendMessage"); len(msgs) != 1 || !strings.Contains(msgs[0].body["text"].(string), "yo'q") {
		t.Errorf("/undo with nothing left = %+v", msgs)
	}
}

func TestLanguageSwitch(t *testing.T) {
	h := setup(t, stt.None{})
	h.b.HandleUpdate(ctx, h.message("/lang"))
	msgs := sent(h.tg.take(), "sendMessage")
	if len(msgs) != 1 || len(buttonData(msgs[0])) != 4 {
		t.Fatalf("/lang = %+v", msgs)
	}
	h.b.HandleUpdate(ctx, h.callback("l:ru"))
	h.tg.take()
	h.b.HandleUpdate(ctx, h.message("такси 25 тысяч"))
	msgs = sent(h.tg.take(), "sendMessage")
	if len(msgs) == 0 || !strings.Contains(msgs[0].body["text"].(string), "сегодня") {
		t.Errorf("after switching to Russian = %+v", msgs)
	}
	h.b.HandleUpdate(ctx, h.message("/today"))
	msgs = sent(h.tg.take(), "sendMessage")
	if len(msgs) != 1 || !strings.Contains(msgs[0].body["text"].(string), "1 расход") {
		t.Errorf("/today = %+v", msgs)
	}
}

func TestShorthandQuestionFlow(t *testing.T) {
	h := setup(t, stt.None{})
	h.b.HandleUpdate(ctx, h.message("taksi 25"))
	msgs := sent(h.tg.take(), "sendMessage")
	if len(msgs) != 1 {
		t.Fatalf("replies = %+v", msgs)
	}
	var thousands string
	for _, d := range buttonData(msgs[0]) {
		if strings.HasPrefix(d, "dm:") && strings.HasSuffix(d, ":1") {
			thousands = d
		}
	}
	h.b.HandleUpdate(ctx, h.callback(thousands))
	edits := sent(h.tg.take(), "editMessageText")
	if len(edits) != 1 || !strings.Contains(edits[0].body["text"].(string), "25 000 so'm") {
		t.Errorf("after answer = %+v", edits)
	}
	// Pressing the same button again: the question has expired.
	h.b.HandleUpdate(ctx, h.callback(thousands))
	edits = sent(h.tg.take(), "editMessageText")
	if len(edits) != 1 || !strings.Contains(edits[0].body["text"].(string), "eskirgan") {
		t.Errorf("second press = %+v", edits)
	}
}

func TestVoiceWithoutProvider(t *testing.T) {
	h := setup(t, stt.None{})
	upd := h.message("")
	upd.Message.Voice = &telegram.Voice{FileID: "f", Duration: 3}
	h.b.HandleUpdate(ctx, upd)
	msgs := sent(h.tg.take(), "sendMessage")
	if len(msgs) != 1 || !strings.Contains(msgs[0].body["text"].(string), "tez orada") {
		t.Errorf("voice reply = %+v", msgs)
	}
}

type fakeSTT struct{ text string }

func (f fakeSTT) Transcribe(context.Context, []byte, string, string) (string, error) {
	return f.text, nil
}

func TestVoiceTranscriptGoesThroughCapture(t *testing.T) {
	h := setup(t, fakeSTT{"tushlik qirq besh ming"})
	upd := h.message("")
	upd.Message.Voice = &telegram.Voice{FileID: "f", Duration: 3}
	h.b.HandleUpdate(ctx, upd)
	calls := h.tg.take()
	msgs := sent(calls, "sendMessage")
	if len(msgs) == 0 {
		t.Fatalf("no reply; calls = %+v", calls)
	}
	card := msgs[0].body["text"].(string)
	if !strings.Contains(card, "45 000 so'm") || !strings.Contains(card, "tushlik qirq besh ming") {
		t.Errorf("voice card = %q", card)
	}
}

func TestIgnoresGroupsAndBots(t *testing.T) {
	h := setup(t, stt.None{})
	upd := h.message("taksi 25 ming")
	upd.Message.Chat.Type = "group"
	h.b.HandleUpdate(ctx, upd)
	upd = h.message("taksi 25 ming")
	upd.Message.From.IsBot = true
	h.b.HandleUpdate(ctx, upd)
	if c := h.tg.take(); len(c) != 0 {
		t.Errorf("replied in group or to a bot: %+v", c)
	}
}

func TestEveningSummary(t *testing.T) {
	pool := testdb.Pool(t)
	h := setup(t, stt.None{})
	h.b.HandleUpdate(ctx, h.message("/start"))
	h.b.HandleUpdate(ctx, h.message("taksi 25 ming"))
	h.tg.take()

	for _, minutesAgo := range []int{1, 23*60 + 30} { // also a time that wraps past midnight
		if _, err := pool.Exec(ctx, `update users set last_summary_on = null,
			notify_time = (now() at time zone timezone)::time - make_interval(mins => $2) where telegram_id = $1`, h.userID, minutesAgo); err != nil {
			t.Fatal(err)
		}
		h.b.Tick(ctx)
		msgs := sent(h.tg.take(), "sendMessage")
		var mine []call // summaries only: from the 25th a plan nudge may arrive too
		for _, m := range msgs {
			if int64(m.body["chat_id"].(float64)) == h.userID && strings.Contains(m.body["text"].(string), "xulosa") {
				mine = append(mine, m)
			}
		}
		due := minutesAgo < 120
		if due && (len(mine) != 1 || !strings.Contains(mine[0].body["text"].(string), "25\u202f000 so'm")) {
			t.Errorf("%d min after notify time: summary = %+v", minutesAgo, mine)
		}
		if !due && len(mine) != 0 {
			t.Errorf("%d min after notify time: sent %+v", minutesAgo, mine)
		}
		// Never twice a day.
		h.b.Tick(ctx)
		for _, m := range sent(h.tg.take(), "sendMessage") {
			if int64(m.body["chat_id"].(float64)) == h.userID && strings.Contains(m.body["text"].(string), "xulosa") {
				t.Error("summary sent twice")
			}
		}
	}
}

func TestSummaryRemindsDuePayments(t *testing.T) {
	pool := testdb.Pool(t)
	h := setup(t, stt.None{})
	h.b.HandleUpdate(ctx, h.message("/start"))
	h.tg.take()
	var userID, home int64
	if err := pool.QueryRow(ctx, `select id from users where telegram_id = $1`, h.userID).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `select id from categories where key = 'home'`).Scan(&home); err != nil {
		t.Fatal(err)
	}
	l := ledger.New(pool)
	loc, _ := time.LoadLocation("Asia/Tashkent")
	y, m, d := time.Now().In(loc).Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	if _, err := l.CreatePlan(ctx, userID, ledger.NewPlan{Title: "Muzlatkich", Provider: "Uzum Nasiya", CategoryID: home, Currency: "UZS",
		TotalMinor: 3_000_000_00, Months: 3, FirstDueOn: today.AddDate(0, 0, 1), OccurredOn: today, RecordPurchase: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.CreateDebt(ctx, userID, ledger.NewDebt{Direction: "lent", Counterparty: "Aziz", Currency: "UZS",
		AmountMinor: 200_000_00, OccurredOn: today.AddDate(0, 0, -10), DueOn: ptr(today.AddDate(0, 0, -2))}); err != nil {
		t.Fatal(err)
	}
	if _, err := l.CreateDebt(ctx, userID, ledger.NewDebt{Direction: "lent", Counterparty: "Later", Currency: "UZS",
		AmountMinor: 1_00, OccurredOn: today, DueOn: ptr(today.AddDate(0, 0, 10))}); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `update users set notify_time = (now() at time zone timezone)::time - interval '1 minute' where id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	h.b.Tick(ctx)
	var text string
	for _, m := range sent(h.tg.take(), "sendMessage") {
		if int64(m.body["chat_id"].(float64)) == h.userID {
			text = m.body["text"].(string)
		}
	}
	for _, want := range []string{"Uzum Nasiya · Muzlatkich", "1\u202f000\u202f000 so'm", "ertaga", "Aziz", "200\u202f000 so'm", "muddati o'tgan"} {
		if !strings.Contains(text, want) {
			t.Errorf("summary lacks %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "Later") {
		t.Errorf("debt due in 10 days was included:\n%s", text)
	}
}

func ptr[T any](v T) *T { return &v }

func TestBudgetAlertsOncePerThreshold(t *testing.T) {
	pool := testdb.Pool(t)
	h := setup(t, stt.None{})
	h.b.HandleUpdate(ctx, h.message("/start"))
	h.b.HandleUpdate(ctx, h.message("tushlik 50 ming")) // creates the cash account
	h.tg.take()
	var userID, cafe int64
	_ = pool.QueryRow(ctx, `select id from users where telegram_id = $1`, h.userID).Scan(&userID)
	_ = pool.QueryRow(ctx, `select id from categories where key = 'cafe'`).Scan(&cafe)
	l := ledger.New(pool)
	if _, err := l.CreateBudget(ctx, userID, &cafe, "UZS", 100_000_00); err != nil {
		t.Fatal(err)
	}
	// Keep the evening summary out of the way.
	_, _ = pool.Exec(ctx, `update users set notify_enabled = false where id = $1`, userID)

	alerts := func() []string {
		h.b.Tick(ctx)
		var out []string
		for _, m := range sent(h.tg.take(), "sendMessage") {
			if int64(m.body["chat_id"].(float64)) == h.userID {
				out = append(out, m.body["text"].(string))
			}
		}
		return out
	}
	if a := alerts(); len(a) != 0 {
		t.Errorf("at 50%%: %v", a)
	}
	h.b.HandleUpdate(ctx, h.message("kofe 35 ming")) // 85%
	h.tg.take()
	if a := alerts(); len(a) != 1 || !strings.Contains(a[0], "85%") {
		t.Errorf("at 85%%: %v", a)
	}
	if a := alerts(); len(a) != 0 {
		t.Errorf("80%% alert repeated: %v", a)
	}
	h.b.HandleUpdate(ctx, h.message("kafe 30 ming")) // 115%
	h.tg.take()
	if a := alerts(); len(a) != 1 || !strings.Contains(a[0], "tugadi") {
		t.Errorf("at 115%%: %v", a)
	}
	if a := alerts(); len(a) != 0 {
		t.Errorf("100%% alert repeated: %v", a)
	}
}

func TestForwardedBankMessage(t *testing.T) {
	h := setup(t, stt.None{})
	h.b.HandleUpdate(ctx, h.message("💸 To'lov\n➖ 37.470,00 UZS\n📍 AAB CLICK QR OPLATA\n💳 HUMOCARD *5958\n🕓 18:38 23.10.2026\n💰 118.363,04 UZS"))
	msgs := sent(h.tg.take(), "sendMessage")
	if len(msgs) != 1 || !strings.Contains(msgs[0].body["text"].(string), "*5958") {
		t.Fatalf("card question = %+v", msgs)
	}
	var create string
	for _, d := range buttonData(msgs[0]) {
		if strings.HasPrefix(d, "bc:") && strings.HasSuffix(d, ":0") {
			create = d
		}
	}
	h.b.HandleUpdate(ctx, h.callback(create))
	edits := sent(h.tg.take(), "editMessageText")
	if len(edits) != 1 {
		t.Fatalf("edits = %+v", edits)
	}
	card := edits[0].body["text"].(string)
	for _, want := range []string{"37\u202f470 so'm", "Humo *5958", "Click QR"} {
		if !strings.Contains(card, want) {
			t.Errorf("card lacks %q: %q", want, card)
		}
	}
	if strings.Contains(card, "🏦") {
		t.Errorf("new account should match the bank: %q", card)
	}
}

func TestPlanFlowAndPurchaseMatch(t *testing.T) {
	pool := testdb.Pool(t)
	h := setup(t, stt.None{})
	h.b.HandleUpdate(ctx, h.message("/plan"))
	if msgs := sent(h.tg.take(), "sendMessage"); len(msgs) != 1 || !strings.Contains(msgs[0].body["text"].(string), "rejalashtiryapsiz") {
		t.Fatalf("/plan = %+v", msgs)
	}
	h.b.HandleUpdate(ctx, h.message("qishki kurtka 800 ming, AirPods 2.5 mln agar bo'lsa"))
	msgs := sent(h.tg.take(), "sendMessage")
	if len(msgs) != 1 {
		t.Fatalf("plan reply = %+v", msgs)
	}
	summary := msgs[0].body["text"].(string)
	for _, want := range []string{"rejasi", "qishki kurtka", "AirPods", "Daromad hali noma'lum"} {
		if !strings.Contains(summary, want) {
			t.Errorf("summary lacks %q:\n%s", want, summary)
		}
	}
	// Plan mode is over: the next message is an expense again.
	h.b.HandleUpdate(ctx, h.message("taksi 25 ming"))
	if msgs := sent(h.tg.take(), "sendMessage"); len(msgs) == 0 || !strings.Contains(msgs[0].body["text"].(string), "Taksi") {
		t.Fatalf("after plan = %+v", msgs)
	}

	// A planned item in the current month, then a purchase that matches it.
	var userID, clothing int64
	_ = pool.QueryRow(ctx, `select id from users where telegram_id = $1`, h.userID).Scan(&userID)
	_ = pool.QueryRow(ctx, `select id from categories where key = 'clothing'`).Scan(&clothing)
	loc, _ := time.LoadLocation("Asia/Tashkent")
	now := time.Now().In(loc)
	month := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	ps := planning.NewService(pool, ledger.New(pool))
	if _, err := ps.AddItems(ctx, userID, month, "UZS", []planning.NewItem{{Title: "Krossovka", CategoryID: &clothing, AmountMin: 600_000_00, AmountMax: 700_000_00, Priority: "should"}}); err != nil {
		t.Fatal(err)
	}
	h.b.HandleUpdate(ctx, h.message("krossovka 650 ming"))
	msgs = sent(h.tg.take(), "sendMessage")
	var yes string
	for _, m := range msgs {
		for _, d := range buttonData(m) {
			if strings.HasPrefix(d, "pm:") {
				yes = d
			}
		}
	}
	if yes == "" {
		t.Fatalf("no match question: %+v", msgs)
	}
	h.b.HandleUpdate(ctx, h.callback(yes))
	if edits := sent(h.tg.take(), "editMessageText"); len(edits) != 1 || !strings.Contains(edits[0].body["text"].(string), "Krossovka") {
		t.Errorf("match answer = %+v", edits)
	}
	v, _ := ps.Get(ctx, userID, month, "UZS")
	for _, it := range v.Items {
		if it.Title == "Krossovka" && (it.Status != "bought" || it.TransactionID == nil) {
			t.Errorf("item = %+v", it)
		}
	}
}

// Payments are turned off: old invoice links in chats must not take money.
func TestPaymentsRefused(t *testing.T) {
	h := setup(t, stt.None{})
	h.update++
	h.b.HandleUpdate(ctx, telegram.Update{UpdateID: h.update, PreCheckoutQuery: &telegram.PreCheckoutQuery{
		ID: "pc", From: telegram.User{ID: h.userID}, Currency: "XTR", TotalAmount: 150, InvoicePayload: "old"}})
	ans := sent(h.tg.take(), "answerPreCheckoutQuery")
	if len(ans) != 1 || ans[0].body["ok"] != false {
		t.Errorf("pre-checkout answers = %+v", ans)
	}
	h.b.HandleUpdate(ctx, h.message("/premium"))
	if calls := sent(h.tg.take(), "createInvoiceLink"); len(calls) != 0 {
		t.Errorf("/premium created invoices: %d", len(calls))
	}
}

func TestWebhookRejectsWrongSecret(t *testing.T) {
	h := setup(t, stt.None{})
	srv := httptest.NewServer(h.b.Webhook("s3cret"))
	defer srv.Close()
	req, _ := http.NewRequest("POST", srv.URL, strings.NewReader(`{"update_id":1}`))
	req.Header.Set("X-Telegram-Bot-Api-Secret-Token", "wrong")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d", res.StatusCode)
	}
}
