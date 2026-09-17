package tag

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
	"🦽Kundenticket":        {},
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
