// Package autoassigner automatically assigning unassigned conversations to team agents in a round-robin fashion.
// Continuously assigns conversations at regular intervals.
package autoassigner

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/abhinavxd/artemis/internal/conversation"
	"github.com/abhinavxd/artemis/internal/conversation/models"
	"github.com/abhinavxd/artemis/internal/team"
	umodels "github.com/abhinavxd/artemis/internal/user/models"
	"github.com/mr-karan/balance"
	"github.com/zerodha/logf"
)

var (
	ErrTeamNotFound = errors.New("team not found")
)

// Engine represents a manager for assigning unassigned conversations
// to team agents in a round-robin pattern.
type Engine struct {
	roundRobinBalancer map[int]*balance.Balance
	// Mutex to protect the balancer map
	mu sync.Mutex

	systemUser          umodels.User
	conversationManager *conversation.Manager
	teamManager         *team.Manager
	lo                  *logf.Logger
}

// New initializes a new Engine instance, set up with the provided team manager,
// conversation manager, and logger.
func New(teamManager *team.Manager, conversationManager *conversation.Manager, systemUser umodels.User, lo *logf.Logger) (*Engine, error) {
	var e = Engine{
		conversationManager: conversationManager,
		teamManager:         teamManager,
		systemUser:          systemUser,
		lo:                  lo,
		mu:                  sync.Mutex{},
	}
	balancer, err := e.populateTeamBalancer()
	if err != nil {
		return nil, err
	}
	e.roundRobinBalancer = balancer
	return &e, nil
}

// Run initiates the conversation assignment process and is to be invoked as a goroutine.
// This function continuously assigns unassigned conversations to agents at regular intervals.
func (e *Engine) Run(ctx context.Context, interval time.Duration) {
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

// RefreshBalancer updates the round-robin balancer with the latest user and team data.
func (e *Engine) RefreshBalancer() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	balancer, err := e.populateTeamBalancer()
	if err != nil {
		e.lo.Error("Error updating team balancer pool", "error", err)
		return err
	}
	e.roundRobinBalancer = balancer
	return nil
}

// populateTeamBalancer populates the team balancer pool with the team members.
func (e *Engine) populateTeamBalancer() (map[int]*balance.Balance, error) {
	var (
		balancer = make(map[int]*balance.Balance)
	)

	teams, err := e.teamManager.GetAll()
	if err != nil {
		return nil, err
	}

	for _, team := range teams {
		if !team.AutoAssignConversations {
			continue
		}

		users, err := e.teamManager.GetTeamMembers(team.Name)
		if err != nil {
			return nil, err
		}

		// Add the users to team balancer pool.
		for _, user := range users {
			if _, ok := balancer[team.ID]; !ok {
				balancer[team.ID] = balance.NewBalance()
			}
			balancer[team.ID].Add(strconv.Itoa(user.ID), 1)
		}
	}
	return balancer, nil
}

// assignConversations function fetches conversations that have been assigned to teams but not to any individual user,
// and then proceeds to assign them to team members based on a round-robin strategy.
func (e *Engine) assignConversations() error {
	unassigned, err := e.conversationManager.GetUnassignedConversations()
	if err != nil {
		return err
	}

	if len(unassigned) > 0 {
		e.lo.Debug("found unassigned conversations", "count", len(unassigned))
	}

	for _, conversation := range unassigned {
		// Get user.
		uid, err := e.getUserFromPool(conversation)
		if err != nil {
			e.lo.Error("error fetching user from balancer pool", "error", err)
			continue
		}

		// Convert to int.
		userID, err := strconv.Atoi(uid)
		if err != nil {
			e.lo.Error("error converting user id from string to int", "error", err)
		}

		// Assign conversation.
		e.conversationManager.UpdateConversationUserAssignee(conversation.UUID, userID, e.systemUser)
	}
	return nil
}

// getUserFromPool returns user ID from the team balancer pool.
func (e *Engine) getUserFromPool(conversation models.Conversation) (string, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	pool, ok := e.roundRobinBalancer[conversation.AssignedTeamID.Int]
	if !ok {
		e.lo.Warn("team not found in balancer", "id", conversation.AssignedTeamID.Int)
		return "", ErrTeamNotFound
	}
	return pool.Get(), nil
}
