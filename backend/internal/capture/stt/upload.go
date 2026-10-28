package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

// Under the API server's 30 s write timeout, so a slow provider still gets an answer out.
var client = &http.Client{Timeout: 20 * time.Second}

// upload posts audio as multipart form data and decodes the JSON reply into out.
// The file name tells the provider the format.
func upload(ctx context.Context, url string, header http.Header, fileField string, audio []byte, mime string, fields map[string]string, out any) error {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			return err
		}
	}
	fw, err := mw.CreateFormFile(fileField, fileName(mime))
	if err != nil {
		return err
	}
	if _, err := fw.Write(audio); err != nil {
		return err
	}
	if err := mw.Close(); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return err
	}
	req.Header = header.Clone()
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode/100 != 2 {
		// The body may echo the request but never the API key, which is a header.
		return fmt.Errorf("stt: %s: %s", res.Status, strings.TrimSpace(string(b[:min(len(b), 200)])))
	}
	return json.Unmarshal(b, out)
}

// language maps the user's app language to the "uz", "ru" or "en" codes providers take.
func language(hint string) string {
	switch hint {
	case "ru", "en":
		return hint
	}
	return "uz"
}

// fileName names the upload by format: Telegram voice notes are OGG/Opus, Mini App
// recordings WAV.
func fileName(mime string) string {
	switch mime {
	case "audio/wav":
		return "voice.wav"
	case "audio/mp4":
		return "voice.m4a"
	case "audio/mpeg":
		return "voice.mp3"
	}
	return "voice.ogg"
}
