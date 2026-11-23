package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/abhinavxd/libredesk/internal/aiassist"
	"github.com/knadh/koanf/v2"
	"github.com/valyala/fasthttp"
	"github.com/zerodha/fastglue"
)

func TestTranscriptionTOMLAndEnvironment(t *testing.T) {
	for _, tc := range []struct{ name, toml, prefixed, alias, want string }{
		{"default", "", "", "", ""},
		{"toml", `provider="openai"`, "", "", "openai"},
		{"prefixed overrides TOML", `provider="whisper"`, "openai", "", "openai"},
		{"alias overrides prefixed", `provider="openai"`, "openai", "whisper", "whisper"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("LIBREDESK_AI_TRANSCRIPTION__PROVIDER", tc.prefixed)
			if tc.prefixed == "" {
				if err := os.Unsetenv("LIBREDESK_AI_TRANSCRIPTION__PROVIDER"); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("TRANSCRIPTION_PROVIDER", tc.alias)
			t.Setenv("OPENAI_API_KEY", "synthetic-env-key")
			file := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(file, []byte("[ai_transcription]\n"+tc.toml+"\napi_key=\"ignored-toml-key\"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			k := koanf.New(".")
			k.Set("config", []string{file})
			initConfig(k)
			cfg := transcriptionConfig(k, nil)
			if cfg.Provider != tc.want || cfg.OpenAIAPIKey != "synthetic-env-key" {
				t.Fatal("configuration precedence")
			}
			t.Setenv("OPENAI_API_KEY", "")
			if cfg = transcriptionConfig(k, nil); cfg.OpenAIAPIKey != "" {
				t.Fatal("key read from TOML")
			}
			if cfg.Provider == "openai" {
				if c, err := aiassist.NewTranscription(cfg); c != nil || err == nil {
					t.Fatal("missing key accepted")
				}
			}
		})
	}
}

func TestTranscriptionPublicConfigContract(t *testing.T) {
	for _, provider := range []string{"", "whisper", "openai"} {
		cfg := aiassist.TranscriptionConfig{Provider: provider, OpenAIAPIKey: "private-key"}
		if provider == "whisper" {
			cfg.Whisper = aiassist.Config{Endpoint: "https://whisper.example/transcribe", Token: "private-token"}
		}
		client, err := aiassist.NewTranscription(cfg)
		if err != nil {
			t.Fatal(err)
		}
		ctx := &fasthttp.RequestCtx{}
		if err := handleAITranscriptionConfig(&fastglue.Request{RequestCtx: ctx, Context: &App{aiTranscription: client}}); err != nil {
			t.Fatal(err)
		}
		var result struct {
			Data map[string]bool `json:"data"`
		}
		if err := json.Unmarshal(ctx.Response.Body(), &result); err != nil || len(result.Data) != 1 || result.Data["enabled"] != (provider != "") {
			t.Fatal("public configuration must contain only enabled", err)
		}
	}
}
