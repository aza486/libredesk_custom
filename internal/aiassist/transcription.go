package aiassist

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"unicode/utf8"
)

const MaxAudioBytes = 8 << 20

// AudioExtension validates the declared container against its signature. The
// configured gateway must decode the audio and enforce a 120-second duration cap.
func AudioExtension(data []byte, contentType string) string {
	return audioExtension(data, contentType, MaxAudioBytes)
}

func audioExtension(data []byte, contentType string, limit int) string {
	typ, _, err := mime.ParseMediaType(contentType)
	if err != nil || len(data) < 12 || len(data) > limit {
		return ""
	}
	switch typ {
	case "audio/webm":
		if bytes.HasPrefix(data, []byte{0x1a, 0x45, 0xdf, 0xa3}) {
			return "webm"
		}
	case "audio/ogg":
		if bytes.HasPrefix(data, []byte("OggS")) {
			return "ogg"
		}
	case "audio/mp4":
		if string(data[4:8]) == "ftyp" {
			return "mp4"
		}
	}
	return ""
}

// Transcribe sends only ephemeral audio to the explicitly configured gateway.
// No retries, file storage, upstream error bodies or request logging.
func (c *Client) Transcribe(ctx context.Context, data []byte, contentType string) (string, int, error) {
	return c.transcribe(ctx, data, contentType, MaxAudioBytes, false)
}

func (c *Client) transcribe(ctx context.Context, data []byte, contentType string, limit int, openAI bool) (string, int, error) {
	if len(data) > limit {
		return "", 413, ErrInvalidResponse
	}
	ext := audioExtension(data, contentType, limit)
	if ext == "" {
		return "", 400, ErrInvalidResponse
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if openAI {
		for _, field := range []struct{ name, value string }{{"model", "gpt-transcribe"}, {"response_format", "json"}, {"languages[]", "de"}} {
			if err := w.WriteField(field.name, field.value); err != nil {
				return "", 502, ErrUnavailable
			}
		}
	}
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="file"; filename="recording.`+ext+`"`)
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	if err != nil {
		return "", 502, ErrUnavailable
	}
	if _, err = part.Write(data); err != nil {
		return "", 502, ErrUnavailable
	}
	if err = w.Close(); err != nil {
		return "", 502, ErrUnavailable
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, &body)
	if err != nil {
		return "", 503, ErrUnavailable
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		var timeout net.Error
		if ctx.Err() != nil || (errors.As(err, &timeout) && timeout.Timeout()) {
			return "", 504, ErrTimeout
		}
		return "", 502, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		if openAI {
			switch res.StatusCode {
			case 413:
				return "", 413, ErrInvalidResponse
			case 429:
				return "", 503, ErrUnavailable
			case 408, 504:
				return "", 504, ErrTimeout
			}
		}
		return "", 502, ErrUnavailable
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil {
		var timeout net.Error
		if ctx.Err() != nil || (errors.As(err, &timeout) && timeout.Timeout()) {
			return "", 504, ErrTimeout
		}
		return "", 502, ErrInvalidResponse
	}
	var result struct {
		Text string `json:"text"`
	}
	if len(raw) > 65536 || !utf8.Valid(raw) || json.Unmarshal(raw, &result) != nil || !ValidText(result.Text, 16000) {
		return "", 502, ErrInvalidResponse
	}
	return strings.TrimSpace(result.Text), 200, nil
}
