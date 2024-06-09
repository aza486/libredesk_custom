package models

import (
	"encoding/json"
	"fmt"
	"net/textproto"
)

type Attachment struct {
	Filename           string               `json:"filename" db:"filename"`
	ContentType        string               `json:"content_type" db:"content_type"`
	UUID               string               `json:"uuid" db:"uuid"`
	URL                string               `json:"url" db:"url"`
	Size               string               `json:"size" db:"size"`
	ContentDisposition string               `json:"content_disposition" db:"content_disposition"`
	Header             textproto.MIMEHeader `json:"-"`
	Content            []byte               `json:"-"`
	ContentID          string               `json:"-"`
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
