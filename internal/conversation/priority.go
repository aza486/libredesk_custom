package conversation

import (
	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/envelope"
)

// GetAllPriorities retrieves all priorities.
func (t *Manager) GetAllPriorities() ([]models.Priority, error) {
	var priorities []models.Priority
	if err := t.q.GetAllPriorities.Select(&priorities); err != nil {
		t.lo.Error("error fetching priorities", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching priorities", nil)
	}
	return priorities, nil
}
