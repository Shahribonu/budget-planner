// Package stt is the speech-to-text boundary. Providers are chosen by benchmark
// (see cmd/sttbench) and swapped by configuration, never by changing callers.
package stt

import (
	"context"
	"errors"
	"os"
)

var ErrNotConfigured = errors.New("speech-to-text provider not configured")

type Transcriber interface {
	// Transcribe returns the text of a voice note. Audio is Telegram's OGG/Opus.
	// langHint is the user's language (uz-Latn, uz-Cyrl, ru, en); providers may ignore it.
	Transcribe(ctx context.Context, audio []byte, mime, langHint string) (string, error)
}

// None is used until a provider is chosen.
type None struct{}

func (None) Transcribe(context.Context, []byte, string, string) (string, error) {
	return "", ErrNotConfigured
}

// New returns the provider named by STT_PROVIDER, with its key from the environment.
// Candidates live here until cmd/sttbench picks one.
func New(name string) (Transcriber, error) {
	key := func(env string) (string, error) {
		if k := os.Getenv(env); k != "" {
			return k, nil
		}
		return "", errors.New(env + " is required for STT_PROVIDER " + name)
	}
	switch name {
	case "", "none":
		return None{}, nil
	case "aisha":
		k, err := key("AISHA_API_KEY")
		if err != nil {
			return nil, err
		}
		return Aisha{Key: k}, nil
	case "uzbekvoice":
		k, err := key("UZBEKVOICE_API_KEY")
		if err != nil {
			return nil, err
		}
		return UzbekVoice{Key: k}, nil
	}
	return nil, errors.New("unknown STT_PROVIDER " + name)
}
