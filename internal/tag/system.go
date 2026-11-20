package tag

import (
	"strings"

	"golang.org/x/text/unicode/norm"
)

const SystemTagCustomerName = "🦽Kundenticket"

const (
	SystemTagInternal    = 8
	SystemTagCustomer    = 9
	SystemTagHuman       = 11
	SystemTagServiceMail = 14
)

var systemTagIDs = map[int]struct{}{
	SystemTagInternal:    {},
	SystemTagCustomer:    {},
	SystemTagHuman:       {},
	SystemTagServiceMail: {},
}

var systemTagNames = map[string]struct{}{
	"🏢Intern":              {},
	SystemTagCustomerName:  {},
	"😎Mensch erforderlich": {},
	"🧷Service-Mail":        {},
}

func IsSystemTag(id int) bool {
	_, ok := systemTagIDs[id]
	return ok
}

func IsSystemTagName(name string) bool {
	_, ok := systemTagNames[name]
	return ok
}

// Reserved identities include Spam definitions, whose conversation membership
// remains editable through the ordinary authorized Spam/tag actions.
func IsReservedDefinition(id int, name string) bool {
	for _, required := range requiredSystemTags {
		if id == required.ID || sameSystemName(name, required.Name) {
			return true
		}
	}
	return false
}

// IsClassificationTag protects membership by both identity and name, including
// legacy inconsistent definitions. Spam is deliberately excluded here.
func IsClassificationTag(id int, name string) bool {
	for _, required := range requiredSystemTags {
		if required.ID != 12 && (id == required.ID || sameSystemName(name, required.Name)) {
			return true
		}
	}
	return false
}

func sameSystemName(name, canonical string) bool {
	name = strings.Map(func(r rune) rune {
		if r == '\ufe0e' || r == '\ufe0f' {
			return -1
		}
		return r
	}, norm.NFC.String(strings.TrimSpace(name)))
	return strings.EqualFold(name, canonical)
}
