package api_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/Shahribonu/budget-planner/backend/internal/api"
	"github.com/Shahribonu/budget-planner/backend/internal/capture"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/stt"
	"github.com/Shahribonu/budget-planner/backend/internal/ledger"
	"github.com/Shahribonu/budget-planner/backend/internal/testdb"
)

type fakeSTT struct{ text, mime, lang string }

func (f *fakeSTT) Transcribe(_ context.Context, _ []byte, mime, lang string) (string, error) {
	f.mime, f.lang = mime, lang
	return f.text, nil
}

// wav builds a silent 16 kHz mono 16-bit recording of the given length.
func wav(seconds int) []byte {
	data := make([]byte, 32000*seconds)
	h := make([]byte, 44)
	copy(h, "RIFF")
	binary.LittleEndian.PutUint32(h[4:], uint32(36+len(data)))
	copy(h[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(h[16:], 16)
	binary.LittleEndian.PutUint16(h[20:], 1)
	binary.LittleEndian.PutUint16(h[22:], 1)
	binary.LittleEndian.PutUint32(h[24:], 16000)
	binary.LittleEndian.PutUint32(h[28:], 32000)
	binary.LittleEndian.PutUint16(h[32:], 2)
	binary.LittleEndian.PutUint16(h[34:], 16)
	copy(h[36:], "data")
	binary.LittleEndian.PutUint32(h[40:], uint32(len(data)))
	return append(h, data...)
}

func postVoice(t *testing.T, c *client, audio []byte, mime string) (int, map[string]any) {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="audio"; filename="voice.wav"`)
	hdr.Set("Content-Type", mime)
	part, _ := mw.CreatePart(hdr)
	part.Write(audio)
	mw.Close()
	req, _ := http.NewRequest("POST", c.srv.URL+"/api/voice", &body)
	req.Header.Set("Authorization", c.auth)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	return res.StatusCode, out
}

func TestVoice(t *testing.T) {
	pool := testdb.Pool(t)
	fake := &fakeSTT{text: "taksi yigirma besh ming"}
	cfg := api.Config{BotToken: botToken, Voice: fake, Capture: capture.New(pool, ledger.New(pool), nil)}
	srv := httptest.NewServer(api.New(cfg, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))

	if me := c.do("GET", "/api/me", nil, 200); me["voice_enabled"] != true {
		t.Fatalf("voice_enabled = %v", me["voice_enabled"])
	}
	status, out := postVoice(t, c, wav(2), "audio/wav")
	if status != 200 {
		t.Fatalf("POST /api/voice = %d %v", status, out)
	}
	items := out["items"].([]any)
	if out["text"] != "taksi yigirma besh ming" || len(items) != 1 {
		t.Fatalf("out = %v", out)
	}
	it := items[0].(map[string]any)
	if num(it["amount_minor"]) != 25_000_00 || it["currency"] != "UZS" || it["type"] != "expense" || num(it["category_id"]) == 0 {
		t.Fatalf("item = %v", it)
	}
	if fake.mime != "audio/wav" || fake.lang != "ru" {
		t.Fatalf("transcriber got mime %q lang %q", fake.mime, fake.lang)
	}
	// Nothing is saved until the user confirms in the form.
	if txs := c.do("GET", "/api/transactions", nil, 200); len(txs["transactions"].([]any)) != 0 {
		t.Fatalf("voice saved a transaction: %v", txs)
	}

	if status, _ := postVoice(t, c, wav(61+1), "audio/wav"); status != 400 {
		t.Errorf("62 s recording = %d, want 400", status)
	}
	if status, _ := postVoice(t, c, []byte("<html>"), "text/html"); status != 400 {
		t.Errorf("text/html = %d, want 400", status)
	}
	fake.text = "salom"
	if status, out := postVoice(t, c, wav(1), "audio/wav"); status != 200 || len(out["items"].([]any)) != 0 {
		t.Errorf("no amount = %d %v, want 200 with no items", status, out)
	}
	limited := false
	for range 10 {
		if status, _ := postVoice(t, c, wav(1), "audio/wav"); status == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Error("voice was never rate limited")
	}
}

func TestVoiceUnavailable(t *testing.T) {
	pool := testdb.Pool(t)
	srv := httptest.NewServer(api.New(api.Config{BotToken: botToken, Voice: stt.None{}}, pool))
	defer srv.Close()
	c := newClient(t, srv, rand.Int64N(1<<50))
	if me := c.do("GET", "/api/me", nil, 200); me["voice_enabled"] != false {
		t.Fatalf("voice_enabled = %v", me["voice_enabled"])
	}
	if status, out := postVoice(t, c, wav(1), "audio/wav"); status != 503 {
		t.Fatalf("POST /api/voice = %d %v, want 503", status, out)
	}
}
