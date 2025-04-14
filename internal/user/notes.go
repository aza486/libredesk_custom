package user

import (
	"github.com/abhinavxd/libredesk/internal/envelope"
	"github.com/abhinavxd/libredesk/internal/user/models"
)

// GetNotes returns all notes for a user.
func (u *Manager) GetNotes(id int) ([]models.Note, error) {
	var notes = make([]models.Note, 0)
	if err := u.q.GetNotes.Select(&notes, id); err != nil {
		u.lo.Error("error fetching user notes", "error", err)
		return notes, envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorFetching", "name", u.i18n.P("globals.terms.note")), nil)
	}
	return notes, nil
}

// CreateNote creates a new note for a user.
func (u *Manager) CreateNote(userID, authorID int, note string) error {
	if _, err := u.q.InsertNote.Exec(userID, authorID, note); err != nil {
		u.lo.Error("error creating user note", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorCreating", "name", u.i18n.P("globals.terms.note")), nil)
	}
	return nil
}

// DeleteNote deletes a note for a user.
func (u *Manager) DeleteNote(noteID int, contactID int) error {
	if _, err := u.q.DeleteNote.Exec(noteID, contactID); err != nil {
		u.lo.Error("error deleting user note", "error", err)
		return envelope.NewError(envelope.GeneralError, u.i18n.Ts("globals.messages.errorDeleting", "name", u.i18n.P("globals.terms.note")), nil)
	}
	return nil
}
