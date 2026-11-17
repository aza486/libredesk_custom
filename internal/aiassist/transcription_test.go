package aiassist

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAudioContainerValidation(t *testing.T) {
	for _, tc := range []struct{ typ, data, ext string }{
		{"audio/webm;codecs=opus", "\x1a\x45\xdf\xa3testdata1234", "webm"},
		{"audio/ogg;codecs=opus", "OggStestdata1234", "ogg"},
		{"audio/mp4", "0000ftypisom0000", "mp4"},
		{"text/html", "<html>test</html>", ""},
		{"audio/ogg", "<html>test</html>", ""},
		{"audio/ogg", "OggS", ""},
		{"audio/ogg", "OggS" + strings.Repeat("x", MaxAudioBytes), ""},
	} {
		if got := AudioExtension([]byte(tc.data), tc.typ); got != tc.ext {
			t.Errorf("type %q got %q", tc.typ, got)
		}
	}
}

func TestTranscriptionMultipartAndErrors(t *testing.T) {
	audio := []byte("OggStestdata1234")
	for _, tc := range []struct {
		name, response string
		status         int
		ok             bool
	}{
		{"success", `{"text":"  Hallo 😀  "}`, 200, true},
		{"empty", `{"text":" "}`, 200, false},
		{"bad JSON", `not-json`, 200, false},
		{"NUL", `{"text":"bad\u0000"}`, 200, false},
		{"long", `{"text":"` + strings.Repeat("x", 16001) + `"}`, 200, false},
		{"oversized", strings.Repeat("x", 65537), 200, false},
		{"upstream error", `secret provider failure`, 500, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer synthetic-token" {
					t.Error("missing auth")
				}
				if err := r.ParseMultipartForm(MaxAudioBytes); err != nil {
					t.Error(err)
					return
				}
				defer r.MultipartForm.RemoveAll()
				f, h, err := r.FormFile("file")
				if err != nil {
					t.Error(err)
					return
				}
				defer f.Close()
				raw, _ := io.ReadAll(f)
				if string(raw) != string(audio) || h.Filename != "recording.ogg" || h.Header.Get("Content-Type") != "audio/ogg;codecs=opus" {
					t.Error("audio changed")
				}
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.response)
			}))
			defer s.Close()
			c := &Client{endpoint: s.URL, token: "synthetic-token", http: s.Client()}
			text, status, err := c.Transcribe(context.Background(), audio, "audio/ogg;codecs=opus")
			if (err == nil) != tc.ok {
				t.Fatalf("status %d error %v", status, err)
			}
			if tc.ok && text != "Hallo 😀" {
				t.Fatal(text)
			}
			if !tc.ok && text != "" {
				t.Error("upstream body leaked")
			}
		})
	}
}
func TestTranscriptionTimeout(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { time.Sleep(25 * time.Millisecond) }))
	defer s.Close()
	c := &Client{endpoint: s.URL, http: s.Client()}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	_, status, err := c.Transcribe(ctx, []byte("OggStestdata1234"), "audio/ogg")
	if err == nil || status != 504 {
		t.Fatalf("%d %v", status, err)
	}
}
