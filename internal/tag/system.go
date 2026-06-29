package tag

const (
	SystemTagInternal = 8
	SystemTagCustomer = 9
)

var systemTagIDs = map[int]struct{}{
	SystemTagInternal: {},
	SystemTagCustomer: {},
}

var systemTagNames = map[string]struct{}{
	"🏢Intern":        {},
	"🦽Kundenticket": {},
}

func IsSystemTag(id int) bool {
	_, ok := systemTagIDs[id]
	return ok
}

func IsSystemTagName(name string) bool {
	_, ok := systemTagNames[name]
	return ok
}
