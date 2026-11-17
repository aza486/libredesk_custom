package aiassist

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
	"unicode/utf8"
)

const MaxAudioBytes = 8 << 20

// AudioExtension validates the declared container against its signature. The
// configured gateway must decode the audio and enforce a 120-second duration cap.
func AudioExtension(data []byte, contentType string) string {
	typ, _, err := mime.ParseMediaType(contentType)
	if err != nil || len(data) < 12 || len(data) > MaxAudioBytes {
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
	ext := AudioExtension(data, contentType)
	if ext == "" {
		return "", 400, ErrInvalidResponse
	}
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
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
	res, err := c.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", 504, ErrTimeout
		}
		return "", 502, ErrUnavailable
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", 502, ErrUnavailable
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	var result struct {
		Text string `json:"text"`
	}
	if err != nil || len(raw) > 65536 || !utf8.Valid(raw) || json.Unmarshal(raw, &result) != nil || !ValidText(result.Text, 16000) {
		return "", 502, ErrInvalidResponse
	}
	return strings.TrimSpace(result.Text), 200, nil
}
