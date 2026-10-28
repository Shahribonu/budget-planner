package api

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/Shahribonu/budget-planner/backend/internal/capture/stt"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

const (
	maxVoiceSeconds = 60
	// 60 s of 16 kHz mono 16-bit WAV is 1.92 MB; compressed formats are far smaller.
	maxVoiceBytes = 2 << 20
)

// voiceTypes are the recordings accepted, by the Content-Type of the audio part.
var voiceTypes = map[string]bool{"audio/wav": true, "audio/ogg": true, "audio/mp4": true, "audio/mpeg": true}

func (s *Server) voiceEnabled() bool {
	_, none := s.cfg.Voice.(stt.None)
	return s.cfg.Voice != nil && s.cfg.Capture != nil && !none
}

// transcribeVoice turns a recording from the Mini App into suggested transactions. It
// saves nothing; the app opens them in the form for the user to confirm. Audio is held
// in memory only and never stored.
func (s *Server) transcribeVoice(w http.ResponseWriter, r *http.Request) {
	if !s.voiceEnabled() {
		writeError(w, http.StatusServiceUnavailable, "voice_unavailable", "", "voice notes are not available yet")
		return
	}
	uid := userID(r)
	if !s.voiceLimit.allow(strconv.FormatInt(uid, 10)) {
		tooMany(w)
		return
	}
	f, fh, err := r.FormFile("audio")
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid", "audio", "audio file required, at most 2 MB")
		return
	}
	defer f.Close()
	mime, _, _ := strings.Cut(fh.Header.Get("Content-Type"), ";")
	if !voiceTypes[mime] {
		writeError(w, http.StatusBadRequest, "invalid", "audio", "unsupported audio type")
		return
	}
	audio, err := io.ReadAll(io.LimitReader(f, maxVoiceBytes+1))
	if err != nil || len(audio) > maxVoiceBytes {
		writeError(w, http.StatusBadRequest, "invalid", "audio", "audio file required, at most 2 MB")
		return
	}
	if mime == "audio/wav" {
		if secs, ok := wavSeconds(audio); !ok || secs > maxVoiceSeconds+1 {
			writeError(w, http.StatusBadRequest, "invalid", "audio", "recording must be a WAV of at most 60 seconds")
			return
		}
	}
	u, err := s.q.GetUser(r.Context(), uid)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	props, _ := json.Marshal(map[string]any{"source": "app", "bytes": len(audio)})
	event := func(name string) {
		_ = s.q.InsertEvent(r.Context(), dbq.InsertEventParams{UserID: &uid, Name: name, Props: props})
	}
	event("voice_received")
	text, err := s.cfg.Voice.Transcribe(r.Context(), audio, mime, u.Language)
	text = strings.TrimSpace(text)
	if err != nil || text == "" {
		if err != nil {
			slog.Warn("api: transcribe", "err", err)
		}
		event("voice_failed")
		writeError(w, http.StatusBadGateway, "voice_failed", "", "could not recognize the recording")
		return
	}
	items, err := s.cfg.Capture.Suggest(r.Context(), u, text)
	if err != nil {
		s.internal(w, r, err)
		return
	}
	type itemJSON struct {
		Type        string `json:"type"`
		AmountMinor int64  `json:"amount_minor"`
		Currency    string `json:"currency"`
		Description string `json:"description"`
		CategoryID  int64  `json:"category_id"`
		AccountID   *int64 `json:"account_id"`
		OccurredOn  string `json:"occurred_on"`
		NeedsReview bool   `json:"needs_review"`
	}
	out := make([]itemJSON, len(items))
	for i, it := range items {
		out[i] = itemJSON{Type: string(it.Type), AmountMinor: it.AmountMinor, Currency: it.Currency, Description: it.Description,
			CategoryID: it.CategoryID, OccurredOn: it.OccurredOn, NeedsReview: it.NeedsReview}
		if it.AccountID != 0 {
			out[i].AccountID = &it.AccountID
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"text": text, "items": out})
}

// wavSeconds reads the duration from a canonical PCM WAV header.
func wavSeconds(b []byte) (float64, bool) {
	if len(b) < 44 || string(b[0:4]) != "RIFF" || string(b[8:12]) != "WAVE" {
		return 0, false
	}
	byteRate := binary.LittleEndian.Uint32(b[28:32])
	if byteRate == 0 {
		return 0, false
	}
	return float64(len(b)-44) / float64(byteRate), true
}
