package autoassigner

import (
	"context"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/message"
	notifier "github.com/abhinavxd/artemis/internal/notification"
	"github.com/abhinavxd/artemis/internal/systeminfo"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/abhinavxd/artemis/internal/ws"
	"github.com/mr-karan/balance"
	"github.com/zerodha/logf"
)

const (
	roundRobinDefaultWeight = 1
)

// Engine handles the assignment of unassigned conversations to agents of a team using a round-robin strategy.
type Engine struct {
	teamRoundRobinBalancer map[int]*balance.Balance
	// Mutex to protect the balancer map
	mu sync.Mutex

	userIDMap map[string]int
	convMgr   *conversation.Manager
	teamMgr   *team.Manager
	userMgr   *user.Manager
	msgMgr    *message.Manager
	lo        *logf.Logger
	hub       *ws.Hub
	notifier  notifier.Notifier
}

// New creates a new instance of the Engine.
func New(teamMgr *team.Manager, userMgr *user.Manager, convMgr *conversation.Manager, msgMgr *message.Manager,
	notifier notifier.Notifier, hub *ws.Hub, lo *logf.Logger) (*Engine, error) {
	var e = Engine{
		notifier:  notifier,
		convMgr:   convMgr,
		teamMgr:   teamMgr,
		msgMgr:    msgMgr,
		userMgr:   userMgr,
		lo:        lo,
		hub:       hub,
		mu:        sync.Mutex{},
		userIDMap: map[string]int{},
	}
	balancer, err := e.populateBalancerPool()
	if err != nil {
		return nil, err
	}
	e.teamRoundRobinBalancer = balancer
	return &e, nil
}

func (e *Engine) Serve(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.assignConversations(); err != nil {
				e.lo.Error("Error assigning conversations", "error", err)
			}
		}
	}
}

func (e *Engine) RefreshBalancer() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	balancer, err := e.populateBalancerPool()
	if err != nil {
		e.lo.Error("Error updating team balancer pool", "error", err)
		return err
	}
	e.teamRoundRobinBalancer = balancer
	return nil
}

// populateBalancerPool populates the team balancer bool with the team members.
func (e *Engine) populateBalancerPool() (map[int]*balance.Balance, error) {
	var (
		balancer   = make(map[int]*balance.Balance)
		teams, err = e.teamMgr.GetAll()
	)

	if err != nil {
		return nil, err
	}

	for _, team := range teams {
		users, err := e.teamMgr.GetTeamMembers(team.Name)
		if err != nil {
			return nil, err
		}

		// Add the users to team balancer pool.
		for _, user := range users {
			if _, ok := balancer[team.ID]; !ok {
				balancer[team.ID] = balance.NewBalance()
			}
			// FIXME: Balancer only supports strings, using a map to store DB ids.
			balancer[team.ID].Add(user.UUID, roundRobinDefaultWeight)
			e.userIDMap[user.UUID] = user.ID
		}
	}
	return balancer, nil
}

// assignConversations fetches unassigned conversations and assigns them to users.
func (e *Engine) assignConversations() error {
	unassigned, err := e.convMgr.GetUnassigned()
	if err != nil {
		return err
	}

	if len(unassigned) > 0 {
		e.lo.Debug("found unassigned conversations", "count", len(unassigned))
	}

	// Get system user, all actions here are done on behalf of the system user.
	systemUser, err := e.userMgr.GetUser(0, systeminfo.SystemUserUUID)
	if err != nil {
		return err
	}

	for _, conversation := range unassigned {
		// Get user uuid from the pool.
		userUUID := e.getUser(conversation)
		if userUUID == "" {
			e.lo.Warn("user uuid not found for round robin assignment", "team_id", conversation.AssignedTeamID.Int)
			continue
		}

		// Get user ID from the map.
		// FIXME: Balance only supports strings.
		userID, ok := e.userIDMap[userUUID]
		if !ok {
			e.lo.Warn("user id not found for user uuid", "uuid", userUUID, "team_id", conversation.AssignedTeamID.Int)
			continue
		}

		// Update assignee and record the assigne change message.
		if err := e.convMgr.UpdateUserAssignee(conversation.UUID, userID, systemUser); err != nil {
			continue
		}

		// Send notification to the assignee.
		e.notifier.SendAssignedConversationNotification([]string{userUUID}, conversation.UUID)

	}
	return nil
}

// getUser returns user uuid from the team balancer pool.
func (e *Engine) getUser(conversation models.Conversation) string {
	pool, ok := e.teamRoundRobinBalancer[conversation.AssignedTeamID.Int]
	if !ok {
		e.lo.Warn("team not found in balancer", "id", conversation.AssignedTeamID.Int)
		return ""
	}
	return pool.Get()
}
