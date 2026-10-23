package email

import (
	"regexp"
	"strings"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/emersion/go-message/mail"
	"github.com/jhillyerd/enmime/v2"
)

// Some gateways emit opaque IDs with multiple @ signs. Keep the existing
// Message-ID compatibility for these IDs, but never salvage nested brackets,
// prose or partial identifiers from a damaged field.
var opaqueIDList = regexp.MustCompile(`^(?:[\t\r\n ]*<[^<>\s]+>)+[\t\r\n ]*$`)
var opaqueID = regexp.MustCompile(`^[^<>\s]+@[^<>\s]+$`)
var bracketedID = regexp.MustCompile(`<([^<>\s]+)>`)

// Marks errors for which continuing the batch could overtake an unpersisted
// parent. MIME parsing errors retain the existing skip-and-continue behavior.
type incomingEmailPersistenceError struct{ error }

func (e incomingEmailPersistenceError) Unwrap() error { return e.error }

func parseThreadMessageIDs(value string) []string {
	var h mail.Header
	h.Set("References", value)
	if ids, err := h.MsgIDList("References"); err == nil {
		return ids
	}
	value = strings.TrimSpace(value)
	if opaqueIDList.MatchString(value) {
		var ids []string
		for _, match := range bracketedID.FindAllStringSubmatch(value, -1) {
			if !opaqueID.MatchString(match[1]) {
				return nil
			}
			ids = append(ids, match[1])
		}
		return ids
	}
	// Retain the pre-existing bare Message-ID fallback as an exact opaque ID.
	if opaqueID.MatchString(value) {
		return []string{value}
	}
	return nil
}

func setThreadHeaders(envelope *enmime.Envelope, incoming *models.IncomingMessage) {
	var parents []string
	for _, value := range envelope.GetHeaderValues("In-Reply-To") {
		parents = append(parents, parseThreadMessageIDs(value)...)
	}
	incoming.InReplyTo = ""
	incoming.References = nil
	if len(parents) == 1 {
		incoming.InReplyTo = parents[0]
	} else {
		incoming.References = append(incoming.References, parents...)
	}
	for _, value := range envelope.GetHeaderValues("References") {
		incoming.References = append(incoming.References, parseThreadMessageIDs(value)...)
	}
}
