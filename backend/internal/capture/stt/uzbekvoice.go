package stt

import (
	"context"
	"errors"
	"net/http"
)

// UzbekVoice is uzbekvoice.ai's speech-to-text. Blocking mode handles audio up to a
// minute, which matches the bot's voice note limit.
// Docs: https://uzbekvoice.ai/developers/api/stt
type UzbekVoice struct {
	Key, URL string
}

func (u UzbekVoice) Transcribe(ctx context.Context, audio []byte, mime, langHint string) (string, error) {
	url := u.URL
	if url == "" {
		url = "https://uzbekvoice.ai/api/v1/stt"
	}
	// Mixed Uzbek-Russian, since people switch languages mid-sentence (numbers especially);
	// there is no English model.
	lang := "uz-ru"
	if language(langHint) == "ru" {
		lang = "ru"
	}
	// The response shape isn't fully documented; accept the field names seen so far.
	var out struct {
		Result *struct {
			Text             string `json:"text"`
			ConversationText string `json:"conversation_text"`
		} `json:"result"`
		Text string `json:"text"`
	}
	err := upload(ctx, url, http.Header{"Authorization": {u.Key}}, "file", audio, mime, map[string]string{
		"language": lang, "model": "general", "blocking": "true", "return_offsets": "false", "run_diarization": "false",
	}, &out)
	if err != nil {
		return "", err
	}
	switch {
	case out.Result != nil && out.Result.Text != "":
		return out.Result.Text, nil
	case out.Result != nil && out.Result.ConversationText != "":
		return out.Result.ConversationText, nil
	case out.Text != "":
		return out.Text, nil
	}
	return "", errors.New("stt: uzbekvoice: no text in response")
}
