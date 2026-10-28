package stt

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fake records the multipart request a provider sends and answers with reply.
func fake(t *testing.T, status int, reply string) (*httptest.Server, *http.Request, map[string]string) {
	t.Helper()
	var got http.Request
	fields := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("multipart: %v", err)
		}
		for k, v := range r.MultipartForm.Value {
			fields[k] = v[0]
		}
		for k, fh := range r.MultipartForm.File {
			f, _ := fh[0].Open()
			b, _ := io.ReadAll(f)
			fields["file:"+k] = fh[0].Filename + "=" + string(b)
		}
		got = *r
		w.WriteHeader(status)
		io.WriteString(w, reply)
	}))
	t.Cleanup(srv.Close)
	return srv, &got, fields
}

func TestAisha(t *testing.T) {
	srv, req, fields := fake(t, 200, `{"id":1,"duration":2.1,"transcript":"taksi yigirma besh ming"}`)
	text, err := Aisha{Key: "k1", URL: srv.URL}.Transcribe(context.Background(), []byte("OggS"), "audio/ogg", "uz-Cyrl")
	if err != nil || text != "taksi yigirma besh ming" {
		t.Fatalf("got %q, %v", text, err)
	}
	if req.Header.Get("X-Api-Key") != "k1" || fields["file:audio"] != "voice.ogg=OggS" || fields["language"] != "uz" {
		t.Fatalf("request: key %q, fields %v", req.Header.Get("X-Api-Key"), fields)
	}
}

func TestUzbekVoice(t *testing.T) {
	srv, req, fields := fake(t, 200, `{"result":{"text":"non sakkiz ming"}}`)
	text, err := UzbekVoice{Key: "k2", URL: srv.URL}.Transcribe(context.Background(), []byte("OggS"), "audio/ogg", "uz-Latn")
	if err != nil || text != "non sakkiz ming" {
		t.Fatalf("got %q, %v", text, err)
	}
	if req.Header.Get("Authorization") != "k2" || fields["file:file"] != "voice.ogg=OggS" || fields["language"] != "uz-ru" || fields["blocking"] != "true" {
		t.Fatalf("request: auth %q, fields %v", req.Header.Get("Authorization"), fields)
	}
}

func TestProviderErrors(t *testing.T) {
	srv, _, _ := fake(t, 402, `{"detail":"Insufficient balance"}`)
	_, err := Aisha{Key: "secret-key", URL: srv.URL}.Transcribe(context.Background(), []byte("x"), "", "ru")
	if err == nil || !strings.Contains(err.Error(), "402") || strings.Contains(err.Error(), "secret-key") {
		t.Fatalf("err = %v", err)
	}
	srv, _, _ = fake(t, 200, `{"result":{}}`)
	if _, err := (UzbekVoice{Key: "k", URL: srv.URL}).Transcribe(context.Background(), []byte("x"), "", "ru"); err == nil {
		t.Fatal("empty result accepted")
	}
}

func TestNewNeedsKey(t *testing.T) {
	t.Setenv("AISHA_API_KEY", "")
	if _, err := New("aisha"); err == nil {
		t.Fatal("aisha without key accepted")
	}
	t.Setenv("AISHA_API_KEY", "k")
	if tr, err := New("aisha"); err != nil || tr.(Aisha).Key != "k" {
		t.Fatalf("New(aisha) = %v, %v", tr, err)
	}
	if _, err := New("muxlisa"); err == nil {
		t.Fatal("unknown provider accepted")
	}
}
