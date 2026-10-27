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

	"github.com/Shahribonu/budget-planner/backend/internal/bot"
	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/stt"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
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
	b := bot.New(bot.Config{MiniAppURL: "https://app.example"}, telegram.New("TOKEN").WithBaseURL(srv.URL), pool, capture.New(pool, l, nil), transcriber)
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
		var mine []call
		for _, m := range msgs {
			if int64(m.body["chat_id"].(float64)) == h.userID {
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
			if int64(m.body["chat_id"].(float64)) == h.userID {
				t.Error("summary sent twice")
			}
		}
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
