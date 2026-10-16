package main

import "github.com/google/uuid"

type responseProvenance struct {
	Source     string `json:"response_source"`
	SourceUUID string `json:"source_message_uuid"`
	NoteUUID   string `json:"note_uuid,omitempty"`
}

func validProvenance(p responseProvenance, private bool, sender, channel, method, feedbackSource string) bool {
	if _, err := uuid.Parse(p.SourceUUID); err != nil {
		return false
	}
	if p.Source != "human" {
		if _, err := uuid.Parse(p.NoteUUID); err != nil {
			return false
		}
	}
	if private || sender != "agent" || channel != "email" || method != authMethodSession || p.SourceUUID == "" {
		return false
	}
	switch p.Source {
	case "ai", "ai_adapted":
		return p.NoteUUID != "" && feedbackSource == ""
	case "human":
		return feedbackSource == p.SourceUUID
	default:
		return false
	}
}
