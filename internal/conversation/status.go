package conversation

import (
	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
)

// GetAllStatuses retrieves all statuses.
func (t *Manager) GetAllStatuses() ([]models.Status, error) {
	var statuses []models.Status
	if err := t.q.GetAllStatuses.Select(&statuses); err != nil {
		t.lo.Error("error fetching statuses", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching statuses", nil)
	}
	return statuses, nil
}
