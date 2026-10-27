// Package stt is the speech-to-text boundary. Providers are chosen by benchmark
// (see cmd/sttbench) and swapped by configuration, never by changing callers.
package stt

import (
	"context"
	"errors"
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

// New returns the provider named by STT_PROVIDER. Add providers here after they win the benchmark.
func New(name string) (Transcriber, error) {
	switch name {
	case "", "none":
		return None{}, nil
	}
	return nil, errors.New("unknown STT_PROVIDER " + name)
}
