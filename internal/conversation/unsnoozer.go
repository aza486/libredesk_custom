package conversation

import (
	"context"
	"time"
)

// RunUnsnoozer runs the unsnoozer.
func (c *Manager) RunUnsnoozer(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.unsnoozeAll(ctx)
		}
	}
}

// unsnoozeAll unsnoozes all snoozed conversations.
func (c *Manager) unsnoozeAll(ctx context.Context) {
	if _, err := c.q.UnsnoozeAll.ExecContext(ctx); err != nil {
		c.lo.Error("error unsnoozing all conversations", err)
	}
}
