package autoassigner

import (
	"context"
	"strconv"
	"time"

	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/team"
	"github.com/abhinavxd/artemis/internal/user"
	"github.com/mr-karan/balance"
	"github.com/zerodha/logf"
)

const (
	roundRobinDefaultWeight = 1
)

// Engine handles the assignment of unassigned conversations to agents using a round-robin strategy.
type Engine struct {
	// Smooth Weighted Round Robin.
	teamRoundRobinBalancer map[int]*balance.Balance
	convMgr                *conversation.Manager
	userMgr                *user.Manager
	teamMgr                *team.Manager
	lo                     *logf.Logger
}

// New creates a new instance of the Engine.
func New(teamMgr *team.Manager, userMgr *user.Manager, convMgr *conversation.Manager, lo *logf.Logger) (*Engine, error) {
	// Get all teams and add users of each them to their respective round robin balancer.
	teams, err := teamMgr.GetAll()
	if err != nil {
		return nil, err
	}

	var balancer = make(map[int]*balance.Balance)
	for _, team := range teams {
		// Fetch all users in the team.
		users, err := teamMgr.GetTeamMembers(team.Name)
		if err != nil {
			return nil, err
		}

		// Now add the users to team balance map.
		for _, user := range users {
			if _, ok := balancer[team.ID]; !ok {
				balancer[team.ID] = balance.NewBalance()
			} else {
				balancer[team.ID].Add(strconv.Itoa(user.ID), roundRobinDefaultWeight)
			}
		}
	}
	return &Engine{
		teamRoundRobinBalancer: balancer,
		userMgr:                userMgr,
		teamMgr:                teamMgr,
		lo:                     lo,
	}, nil
}

// AssignConversations processes unassigned conversations and assigns them to agents.
func (e *Engine) AssignConversations() error {
	unassignedConv, err := e.convMgr.GetUnassigned()
	if err != nil {
		return err
	}

	for _, conv := range unassignedConv {
		// Fetch an agent from the team balancer pool and assign.
		pool, ok := e.teamRoundRobinBalancer[conv.AssignedTeamID.Int]
		if !ok {
			continue
		}
		userID := pool.Get()
		
		if userID == "" {
			continue
		}

		e.convMgr.UpdateAssignee(conv.UUID, []byte("88be466f-adf3-427e-af6a-88df2d3fbb01"), "agent")
	}
	return nil
}

func (e *Engine) Serve(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.AssignConversations()
		}
	}
}
