package attachment

import (
	"encoding/json"
	"fmt"
	"net/textproto"
)

const (
	DispositionInline     = "inline"
	DispositionAttachment = "attachment"
)

// Attachment represents a file or blob attachment that can be sent or received on a message.
type Attachment struct {
	Name        string               `json:"name"`
	Size        int                  `json:"size"`
	Content     []byte               `json:"content"`
	ContentID   string               `json:"content_id"`
	ContentType string               `json:"content_type"`
	Disposition string               `json:"disposition"`
	URL         string               `json:"url"`
	Header      textproto.MIMEHeader `json:"-"`
}

type Attachments []Attachment

func (a *Attachments) Scan(value interface{}) error {
	if value == nil {
		*a = make(Attachments, 0)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("Attachments.Scan: type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, a)
}

func MakeHeader(contentType, fileName, encoding string) textproto.MIMEHeader {
	if encoding == "" {
		encoding = "base64"
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", "attachment; filename="+fileName)
	h.Set("Content-Type", fmt.Sprintf("%s; name=\""+fileName+"\"", contentType))
	h.Set("Content-Transfer-Encoding", encoding)
	return h
}
