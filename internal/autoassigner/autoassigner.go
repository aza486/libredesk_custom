package autoassigner

import (
	"context"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/message"
	"github.com/abhinavxd/artemis/internal/systeminfo"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/mr-karan/balance"
	"github.com/zerodha/logf"
)

const (
	roundRobinDefaultWeight = 1
	strategyRoundRobin      = "round_robin"
	strategyLoadBalances    = "load_balanced"
)

// Engine handles the assignment of unassigned conversations to agents using a round-robin strategy.
type Engine struct {
	teamRoundRobinBalancer map[int]*balance.Balance
	mu                     sync.Mutex // Mutex to protect the balancer map
	convMgr                *conversation.Manager
	teamMgr                *team.Manager
	msgMgr                 *message.Manager
	lo                     *logf.Logger
	strategy               string
}

// New creates a new instance of the Engine.
func New(teamMgr *team.Manager, convMgr *conversation.Manager, msgMgr *message.Manager, lo *logf.Logger) (*Engine, error) {
	balance, err := populateBalancerPool(teamMgr)
	if err != nil {
		return nil, err
	}
	return &Engine{
		teamRoundRobinBalancer: balance,
		strategy:               strategyRoundRobin,
		convMgr:                convMgr,
		teamMgr:                teamMgr,
		msgMgr:                 msgMgr,
		lo:                     lo,
		mu:                     sync.Mutex{},
	}, nil
}

func populateBalancerPool(teamMgr *team.Manager) (map[int]*balance.Balance, error) {
	var (
		balancer   = make(map[int]*balance.Balance)
		teams, err = teamMgr.GetAll()
	)

	if err != nil {
		return nil, err
	}

	for _, team := range teams {
		users, err := teamMgr.GetTeamMembers(team.Name)
		if err != nil {
			return nil, err
		}

		// Now add the users to team balance map.
		for _, user := range users {
			if _, ok := balancer[team.ID]; !ok {
				balancer[team.ID] = balance.NewBalance()
			}
			balancer[team.ID].Add(user.UUID, roundRobinDefaultWeight)
		}
	}
	return balancer, nil
}

func (e *Engine) Serve(ctx context.Context, interval time.Duration) {
	// Start updating the balancer pool periodically in a separate goroutine
	go e.refreshBalancerPeriodically(ctx, 1*time.Minute)

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

// assignConversations fetches unassigned conversations and assigns them.
func (e *Engine) assignConversations() error {
	unassignedConv, err := e.convMgr.GetUnassigned()
	if err != nil {
		return err
	}

	if len(unassignedConv) > 0 {
		e.lo.Debug("found unassigned conversations", "count", len(unassignedConv))
	}

	for _, conv := range unassignedConv {
		if e.strategy == strategyRoundRobin {
			e.roundRobin(conv)
		}
	}
	return nil
}

// roundRobin fetches an user from the team balancer pool and assigns the conversation to that user.
func (e *Engine) roundRobin(conv models.Conversation) {
	pool, ok := e.teamRoundRobinBalancer[conv.AssignedTeamID.Int]
	if !ok {
		e.lo.Warn("team not found in balancer", "id", conv.AssignedTeamID.Int)
	}
	userUUID := pool.Get()
	e.lo.Debug("fetched user from rr pool for assignment", "user_uuid", userUUID)
	if userUUID == "" {
		e.lo.Warn("empty user returned from rr pool")
		return
	}
	if err := e.convMgr.UpdateAssignee(conv.UUID, []byte(userUUID), "agent"); err != nil {
		e.lo.Error("error updating conversation assignee", "error", err, "conv_uuid", conv.UUID, "user_uuid", userUUID)
		return
	}
	if err := e.msgMgr.RecordAssigneeUserChange(userUUID, conv.UUID, systeminfo.SystemUserUUID); err != nil {
		e.lo.Error("error recording conversation user change msg", "error", err, "conv_uuid", conv.UUID, "user_uuid", userUUID)
	}
}

func (e *Engine) refreshBalancerPeriodically(ctx context.Context, updateInterval time.Duration) {
	ticker := time.NewTicker(updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.refreshBalancer(); err != nil {
				e.lo.Error("Error updating team balancer pool", "error", err)
			}
		}
	}
}

func (e *Engine) refreshBalancer() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	balancer, err := populateBalancerPool(e.teamMgr)
	if err != nil {
		e.lo.Error("Error updating team balancer pool", "error", err)
		return err
	}
	e.teamRoundRobinBalancer = balancer
	return nil
}
