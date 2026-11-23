package aiassist

import (
	"context"
	"errors"
	"strings"
)

const MaxOpenAIAudioBytes = 25_000_000

var ErrTranscriptionConfiguration = errors.New("invalid transcription configuration: choose whisper or openai; openai requires OPENAI_API_KEY")

// TranscriptionConfig is server-owned. Whisper credentials are independent of
// OpenAIAPIKey, which the application reads exclusively from OPENAI_API_KEY.
type TranscriptionConfig struct {
	Provider     string
	Whisper      Config
	OpenAIAPIKey string
}

// Transcription selects exactly one provider at startup. There is no fallback.
type Transcription struct {
	client *Client
	openAI bool
}

func NewTranscription(cfg TranscriptionConfig) (*Transcription, error) {
	switch cfg.Provider {
	case "", "whisper":
		client, err := NewWithConfig(cfg.Whisper)
		if err != nil || client == nil {
			return nil, err
		}
		return &Transcription{client: client}, nil
	case "openai":
		key := cfg.OpenAIAPIKey
		if strings.TrimSpace(key) == "" || strings.ContainsAny(key, "\r\n") {
			return nil, ErrTranscriptionConfiguration
		}
		client, err := NewWithConfig(Config{
			Endpoint: "https://api.openai.com/v1/audio/transcriptions",
			Token:    key, DialControl: cfg.Whisper.DialControl,
		})
		if err != nil {
			return nil, ErrTranscriptionConfiguration
		}
		return &Transcription{client: client, openAI: true}, nil
	default:
		return nil, ErrTranscriptionConfiguration
	}
}

func (t *Transcription) MaxAudioBytes() int {
	if t != nil && t.openAI {
		return MaxOpenAIAudioBytes
	}
	return MaxAudioBytes
}

func (t *Transcription) ValidAudio(data []byte, contentType string) bool {
	return audioExtension(data, contentType, t.MaxAudioBytes()) != ""
}

func (t *Transcription) Transcribe(ctx context.Context, data []byte, contentType string) (string, int, error) {
	return t.client.transcribe(ctx, data, contentType, t.MaxAudioBytes(), t.openAI)
}
