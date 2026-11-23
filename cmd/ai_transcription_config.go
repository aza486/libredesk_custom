package main

import (
	"os"

	"github.com/abhinavxd/libredesk/internal/aiassist"
	"github.com/abhinavxd/libredesk/internal/ssrf"
	"github.com/knadh/koanf/v2"
)

func transcriptionConfig(k *koanf.Koanf, control ssrf.Control) aiassist.TranscriptionConfig {
	provider := k.String("ai_transcription.provider")
	// Optional unprefixed alias; the standard LIBREDESK_ override is loaded by initConfig.
	if value := os.Getenv("TRANSCRIPTION_PROVIDER"); value != "" {
		provider = value
	}
	return aiassist.TranscriptionConfig{
		Provider: provider, OpenAIAPIKey: os.Getenv("OPENAI_API_KEY"),
		Whisper: aiassist.Config{
			Endpoint:             k.String("ai_transcription.endpoint"),
			Token:                k.String("ai_transcription.token"),
			InternalHTTPEndpoint: k.String("ai_transcription.internal_http_endpoint"),
			InternalHTTPIP:       k.String("ai_transcription.internal_http_ip"),
			DialControl:          control,
		},
	}
}
