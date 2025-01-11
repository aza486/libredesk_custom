package businesshours

import (
	"github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// GetAllHolidays retrieves all holidays.
func (m *Manager) GetAllHolidays() ([]models.Holiday, error) {
	var holidays []models.Holiday
	if err := m.q.GetAllHolidays.Select(&holidays); err != nil {
		m.lo.Error("error getting holidays", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error getting holidays", nil)
	}
	return holidays, nil
}

// AddHoliday adds a new holiday.
func (m *Manager) AddHoliday(name string, date string) error {
	if _, err := m.q.InsertHoliday.Exec(name, date); err != nil {
		m.lo.Error("error inserting holiday", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error adding holiday", nil)
	}
	return nil
}

// RemoveHoliday removes a holiday by name.
func (m *Manager) RemoveHoliday(name string) error {
	if _, err := m.q.DeleteHoliday.Exec(name); err != nil {
		m.lo.Error("error deleting holiday", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error removing holiday", nil)
	}
	return nil
}
