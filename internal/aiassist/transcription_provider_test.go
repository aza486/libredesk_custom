package aiassist

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTranscriptionProviderConfiguration(t *testing.T) {
	whisper := Config{Endpoint: "http://172.18.0.1:18081/transcribe", Token: "whisper-secret", InternalHTTPEndpoint: "http://172.18.0.1:18081/transcribe", InternalHTTPIP: "172.18.0.1"}
	for _, provider := range []string{"", "whisper", "openai"} {
		c, err := NewTranscription(TranscriptionConfig{Provider: provider, Whisper: whisper, OpenAIAPIKey: "openai-secret"})
		if err != nil || c == nil {
			t.Fatal(provider, err)
		}
		if c.client.http.Timeout != 45*time.Second || c.client.http.CheckRedirect(nil, nil) != http.ErrUseLastResponse || c.client.http.Transport.(*http.Transport).Proxy != nil {
			t.Fatal("timeout, redirect or proxy policy lost")
		}
		if provider == "openai" {
			if c.client.endpoint != "https://api.openai.com/v1/audio/transcriptions" || c.client.token != "openai-secret" || c.MaxAudioBytes() != 25_000_000 {
				t.Fatal("OpenAI configuration")
			}
		} else if c.client.endpoint != whisper.Endpoint || c.client.token != whisper.Token || c.MaxAudioBytes() != MaxAudioBytes {
			t.Fatal("Whisper configuration changed")
		}
	}
	for _, cfg := range []TranscriptionConfig{
		{Provider: "unknown-secret"}, {Provider: "openai", Whisper: whisper},
		{Provider: "openai", OpenAIAPIKey: " "}, {Provider: "openai", OpenAIAPIKey: "secret\n"},
		{Provider: "whisper", Whisper: Config{Endpoint: "http://unsafe", Token: "secret"}, OpenAIAPIKey: "unused"},
	} {
		c, err := NewTranscription(cfg)
		if err == nil || c != nil || strings.Contains(err.Error(), "secret") {
			t.Fatal("invalid configuration accepted or exposed")
		}
	}
	if c, err := NewTranscription(TranscriptionConfig{OpenAIAPIKey: "unused"}); c != nil || err != nil {
		t.Fatal("API key implicitly enabled OpenAI")
	}
	// An inactive Whisper configuration is retained without being used or validated.
	if c, err := NewTranscription(TranscriptionConfig{Provider: "openai", OpenAIAPIKey: "key", Whisper: Config{Endpoint: "unused"}}); c == nil || err != nil {
		t.Fatal("inactive provider blocked OpenAI")
	}
}

func TestProvidersMultipartResponseCompatibility(t *testing.T) {
	for _, provider := range []string{"whisper", "openai"} {
		for _, tc := range []struct {
			name, body     string
			upstream, want int
		}{
			{"success", `{"text":"  Hallo 😀  ","languages":[{"code":"de"}]}`, 200, 200},
			{"empty", `{"text":" "}`, 200, 502}, {"missing", `{}`, 200, 502},
			{"malformed", `not-json`, 200, 502}, {"wrong type", `{"text":123}`, 200, 502},
			{"NUL", `{"text":"bad\u0000"}`, 200, 502},
			{"long", `{"text":"` + strings.Repeat("x", 16001) + `"}`, 200, 502},
			{"oversized response", strings.Repeat("x", 65537), 200, 502},
			{"invalid UTF8", string([]byte{0xff}), 200, 502},
			{"unauthorized", "UPSTREAM_SECRET", 401, 502},
			{"forbidden", "UPSTREAM_SECRET", 403, 502},
			{"bad request", "UPSTREAM_SECRET", 400, 502},
			{"server error", "UPSTREAM_SECRET", 500, 502},
			{"redirect", "UPSTREAM_SECRET", 307, 502},
			{"rate limit", "UPSTREAM_SECRET", 429, 503},
			{"too large", "UPSTREAM_SECRET", 413, 413},
			{"timeout", "UPSTREAM_SECRET", 504, 504},
			{"request timeout", "UPSTREAM_SECRET", 408, 504},
		} {
			t.Run(provider+"/"+tc.name, func(t *testing.T) {
				c, err := NewTranscription(TranscriptionConfig{Provider: provider, OpenAIAPIKey: "synthetic-key", Whisper: Config{Endpoint: "https://whisper.example/transcribe", Token: "synthetic-token"}})
				if err != nil {
					t.Fatal(err)
				}
				audio := []byte("OggStestdata1234")
				calls := 0
				c.client.http.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
					calls++
					key, endpoint := "synthetic-token", "https://whisper.example/transcribe"
					if provider == "openai" {
						key, endpoint = "synthetic-key", "https://api.openai.com/v1/audio/transcriptions"
					}
					if r.Method != "POST" || r.URL.String() != endpoint || r.Header.Get("Authorization") != "Bearer "+key || r.Header.Get("Accept") != "application/json" {
						t.Fatal("request contract")
					}
					if err := r.ParseMultipartForm(MaxOpenAIAudioBytes); err != nil {
						t.Fatal(err)
					}
					defer r.MultipartForm.RemoveAll()
					f, h, err := r.FormFile("file")
					if err != nil {
						t.Fatal(err)
					}
					defer f.Close()
					data, _ := io.ReadAll(f)
					if !bytes.Equal(data, audio) || h.Filename != "recording.ogg" || h.Header.Get("Content-Type") != "audio/ogg;codecs=opus" {
						t.Fatal("audio changed")
					}
					if provider == "openai" {
						if r.FormValue("model") != "gpt-transcribe" || r.FormValue("response_format") != "json" || r.FormValue("languages[]") != "de" || len(r.MultipartForm.Value) != 3 {
							t.Fatal("OpenAI fields")
						}
					} else if len(r.MultipartForm.Value) != 0 {
						t.Fatal("Whisper payload changed")
					}
					return &http.Response{StatusCode: tc.upstream, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
				})
				text, status, err := c.Transcribe(context.Background(), audio, "audio/ogg;codecs=opus")
				want := tc.want
				if provider == "whisper" && tc.upstream != 200 {
					want = 502
				}
				if status != want || (err == nil) != (want == 200) || calls != 1 {
					t.Fatalf("status=%d error=%v calls=%d", status, err, calls)
				}
				if want == 200 && text != "Hallo 😀" {
					t.Fatal("response compatibility")
				}
				if err != nil && (text != "" || strings.Contains(err.Error(), "SECRET")) {
					t.Fatal("upstream data exposed")
				}
			})
		}
	}
}

func TestProviderAudioLimitsAndMIME(t *testing.T) {
	for _, provider := range []string{"whisper", "openai"} {
		c, _ := NewTranscription(TranscriptionConfig{Provider: provider, OpenAIAPIKey: "key", Whisper: Config{Endpoint: "https://whisper.example/transcribe", Token: "token"}})
		calls := 0
		c.client.http.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"text":"OK"}`))}, nil
		})
		data := make([]byte, c.MaxAudioBytes()+1)
		copy(data, "OggS")
		if c.ValidAudio(data, "audio/ogg") {
			t.Fatal("oversize valid")
		}
		if _, status, err := c.Transcribe(context.Background(), data, "audio/ogg"); status != 413 || err == nil || calls != 0 {
			t.Fatal("oversize uploaded")
		}
		for _, typ := range []string{"text/html", "audio/wav", "audio/ogg\r\nX: bad"} {
			if _, status, err := c.Transcribe(context.Background(), data[:12], typ); status != 400 || err == nil || calls != 0 {
				t.Fatal("bad MIME uploaded")
			}
		}
		if _, status, err := c.Transcribe(context.Background(), []byte("<html>test</html>"), "audio/ogg"); status != 400 || err == nil || calls != 0 {
			t.Fatal("bad signature uploaded")
		}
		for _, tc := range []struct{ typ, data string }{{"audio/webm;codecs=opus", "\x1a\x45\xdf\xa3testdata1234"}, {"audio/ogg", "OggStestdata1234"}, {"audio/mp4", "0000ftypisom0000"}} {
			if !c.ValidAudio([]byte(tc.data), tc.typ) {
				t.Fatal("browser format rejected")
			}
		}
		if _, status, err := c.Transcribe(context.Background(), data[:c.MaxAudioBytes()], "audio/ogg"); status != 200 || err != nil || calls != 1 {
			t.Fatal("exact limit rejected", status, err)
		}
	}
}

func TestProviderNetworkAndTimeoutErrors(t *testing.T) {
	for _, provider := range []string{"whisper", "openai"} {
		c, _ := NewTranscription(TranscriptionConfig{Provider: provider, OpenAIAPIKey: "key", Whisper: Config{Endpoint: "https://whisper.example", Token: "token"}})
		calls := 0
		c.client.http.Transport = roundTrip(func(*http.Request) (*http.Response, error) { calls++; return nil, errors.New("SECRET network details") })
		if text, status, err := c.Transcribe(context.Background(), []byte("OggStestdata1234"), "audio/ogg"); text != "" || status != 502 || err == nil || strings.Contains(err.Error(), "SECRET") || calls != 1 {
			t.Fatal("unsafe network error or retry")
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, status, err := c.Transcribe(ctx, []byte("OggStestdata1234"), "audio/ogg"); status != 504 || !errors.Is(err, ErrTimeout) {
			t.Fatal("cancellation lost")
		}
		c.client.http.Transport = roundTrip(func(*http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })
		if _, status, err := c.Transcribe(context.Background(), []byte("OggStestdata1234"), "audio/ogg"); status != 504 || !errors.Is(err, ErrTimeout) {
			t.Fatal("HTTP client timeout lost")
		}
		c.client.http.Transport = roundTrip(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: 200, Body: io.NopCloser(transcriptionReadError{})}, nil
		})
		if text, status, err := c.Transcribe(context.Background(), []byte("OggStestdata1234"), "audio/ogg"); text != "" || status != 504 || !errors.Is(err, ErrTimeout) {
			t.Fatal("response-body timeout lost")
		}
	}
}

type transcriptionReadError struct{}

func (transcriptionReadError) Read([]byte) (int, error) { return 0, context.DeadlineExceeded }
