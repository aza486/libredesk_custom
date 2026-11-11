package user

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/volatiletech/null/v9"
)

// NormalizeWorkPhone preserves international formatting and extension labels.
// Only surrounding whitespace is normalized; controls are rejected even there.
func NormalizeWorkPhone(value string) (null.String, error) {
	invalid := func() (null.String, error) {
		return null.String{}, envelope.NewError(envelope.InputError, "account.workPhone.invalid", nil)
	}
	if !utf8.ValidString(value) {
		return invalid()
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return invalid()
		}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return null.String{}, nil
	}
	if utf8.RuneCountInString(value) > 64 {
		return invalid()
	}
	digits := 0
	for _, r := range value {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case unicode.IsLetter(r), r == ' ', strings.ContainsRune("+()-./#", r):
		default:
			return invalid()
		}
	}
	if digits < 3 {
		return invalid()
	}
	return null.StringFrom(value), nil
}

// UpdateWorkPhone changes only the agent's optional business phone.
func (u *Manager) UpdateWorkPhone(id int, value string) error {
	phone, err := NormalizeWorkPhone(value)
	if err != nil {
		return envelope.NewError(envelope.InputError, u.i18n.T("account.workPhone.invalid"), nil)
	}
	result, err := u.q.UpdateWorkPhone.Exec(id, phone)
	if err != nil {
		return envelope.NewError(envelope.GeneralError, u.i18n.T("globals.messages.somethingWentWrong"), nil)
	}
	n, err := result.RowsAffected()
	if err != nil || n != 1 {
		return envelope.NewError(envelope.InputError, u.i18n.T("account.workPhone.invalid"), nil)
	}
	u.InvalidateAgentCache(id)
	return nil
}
