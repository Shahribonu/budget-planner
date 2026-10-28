package stt

import (
	"context"
	"net/http"
)

// Aisha is aisha.group's synchronous speech-to-text, billed per minute in UZS.
// Docs: https://aisha.group/en/api-documentation/speech-to-text
type Aisha struct {
	Key, URL string
}

func (a Aisha) Transcribe(ctx context.Context, audio []byte, mime, langHint string) (string, error) {
	url := a.URL
	if url == "" {
		url = "https://back.aisha.group/api/v1/stt/post/"
	}
	var out struct {
		Transcript string `json:"transcript"`
	}
	err := upload(ctx, url, http.Header{"X-Api-Key": {a.Key}}, "audio", audio, mime,
		map[string]string{"language": language(langHint), "has_diarization": "false", "has_offset": "false"}, &out)
	return out.Transcript, err
}
