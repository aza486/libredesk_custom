// Package sla implements service-level agreement (SLA) calculations for conversations.
package sla

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	bmodels "github.com/abhinavxd/libredesk/internal/business_hours/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/envelope"
	models "github.com/abhinavxd/libredesk/internal/sla/models"
	tmodels "github.com/abhinavxd/libredesk/internal/team/models"
	"github.com/abhinavxd/libredesk/internal/workerpool"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs            embed.FS
	slaGracePeriod = 5 * time.Minute
)

const (
	SLATypeFirstResponse = "first_response"
	SLATypeResolution    = "resolution"
	SLATypeEveryResponse = "every_response"
)

// Manager provides SLA management and calculations.
type Manager struct {
	q                queries
	lo               *logf.Logger
	pool             *workerpool.Pool
	teamStore        teamStore
	appSettingsStore appSettingsStore
	businessHrsStore businessHrsStore
	opts             Opts
}

// Opts defines options for initializing Manager.
type Opts struct {
	DB              *sqlx.DB
	Lo              *logf.Logger
	ScannerInterval time.Duration
}

type teamStore interface {
	Get(id int) (tmodels.Team, error)
}

type appSettingsStore interface {
	GetByPrefix(prefix string) (types.JSONText, error)
}

type businessHrsStore interface {
	Get(id int) (bmodels.BusinessHours, error)
}

// queries holds prepared SQL statements.
type queries struct {
	GetSLA                *sqlx.Stmt `query:"get-sla-policy"`
	GetAllSLA             *sqlx.Stmt `query:"get-all-sla-policies"`
	InsertSLA             *sqlx.Stmt `query:"insert-sla-policy"`
	DeleteSLA             *sqlx.Stmt `query:"delete-sla-policy"`
	UpdateSLA             *sqlx.Stmt `query:"update-sla-policy"`
	GetUnbreachedSLAs     *sqlx.Stmt `query:"get-unbreached-slas"`
	UpdateBreachedAt      *sqlx.Stmt `query:"update-breached-at"`
	UpdateDueAt           *sqlx.Stmt `query:"update-due-at"`
	UpdateMetAt           *sqlx.Stmt `query:"update-met-at"`
	InsertConversationSLA *sqlx.Stmt `query:"insert-conversation-sla"`
}

// New returns a new Manager.
func New(opts Opts, pool *workerpool.Pool, teamStore teamStore, appSettingsStore appSettingsStore, businessHrsStore businessHrsStore) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:                q,
		lo:               opts.Lo,
		pool:             pool,
		teamStore:        teamStore,
		appSettingsStore: appSettingsStore,
		businessHrsStore: businessHrsStore,
		opts:             opts,
	}, nil
}

// Get retrieves an SLA by its ID.
func (m *Manager) Get(id int) (models.SLAPolicy, error) {
	var sla models.SLAPolicy
	if err := m.q.GetSLA.Get(&sla, id); err != nil {
		m.lo.Error("error fetching SLA", "error", err)
		return sla, envelope.NewError(envelope.GeneralError, "Error fetching SLA", nil)
	}
	return sla, nil
}

// GetAll fetches all SLA policies.
func (m *Manager) GetAll() ([]models.SLAPolicy, error) {
	var slas = make([]models.SLAPolicy, 0)
	if err := m.q.GetAllSLA.Select(&slas); err != nil {
		m.lo.Error("error fetching SLAs", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching SLAs", nil)
	}
	return slas, nil
}

// Create adds a new SLA policy.
func (m *Manager) Create(name, description, firstResponseDuration, resolutionDuration string) error {
	if _, err := m.q.InsertSLA.Exec(name, description, firstResponseDuration, resolutionDuration); err != nil {
		m.lo.Error("error inserting SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error creating SLA", nil)
	}
	return nil
}

// Delete removes an SLA policy by its ID.
func (m *Manager) Delete(id int) error {
	if _, err := m.q.DeleteSLA.Exec(id); err != nil {
		m.lo.Error("error deleting SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error deleting SLA", nil)
	}
	return nil
}

// Update modifies an SLA policy by its ID.
func (m *Manager) Update(id int, name, description, firstResponseDuration, resolutionDuration string) error {
	if _, err := m.q.UpdateSLA.Exec(id, name, description, firstResponseDuration, resolutionDuration); err != nil {
		m.lo.Error("error updating SLA", "error", err)
		return envelope.NewError(envelope.GeneralError, "Error updating SLA", nil)
	}
	return nil
}

// ApplySLA associates an SLA policy with a conversation.
func (m *Manager) ApplySLA(conversationID, slaPolicyID int) (models.SLAPolicy, error) {
	sla, err := m.Get(slaPolicyID)
	if err != nil {
		return sla, err
	}
	for _, t := range []string{SLATypeFirstResponse, SLATypeResolution} {
		if t == SLATypeFirstResponse && sla.FirstResponseTime == "" {
			continue
		}
		if t == SLATypeResolution && sla.ResolutionTime == "" {
			continue
		}
		if _, err := m.q.InsertConversationSLA.Exec(conversationID, slaPolicyID, t); err != nil && !dbutil.IsUniqueViolationError(err) {
			m.lo.Error("error applying SLA to conversation", "error", err)
			return sla, envelope.NewError(envelope.GeneralError, "Error applying SLA to conversation", nil)
		}
	}
	return sla, nil
}

// Run starts the SLA worker pool and periodically processes unbreached SLAs (blocking).
func (m *Manager) Run(ctx context.Context) {
	ticker := time.NewTicker(m.opts.ScannerInterval)
	defer ticker.Stop()
	m.pool.Run()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.processUnbreachedSLAs(); err != nil {
				m.lo.Error("error during SLA periodic check", "error", err)
			}
		}
	}
}

// Close shuts down the SLA worker pool.
func (m *Manager) Close() error {
	m.pool.Close()
	return nil
}

// CalculateConversationDeadlines calculates deadlines for SLA policies attached to a conversation.
func (m *Manager) CalculateConversationDeadlines(conversationCreatedAt time.Time, assignedTeamID, slaPolicyID int) (time.Time, time.Time, error) {
	var (
		businessHrsID, timezone                   = 0, ""
		firstResponseDeadline, resolutionDeadline = time.Time{}, time.Time{}
	)

	// Fetch SLA policy.
	slaPolicy, err := m.Get(slaPolicyID)
	if err != nil {
		return firstResponseDeadline, resolutionDeadline, err
	}

	// First fetch business hours and timezone from assigned team if available.
	if assignedTeamID != 0 {
		team, err := m.teamStore.Get(assignedTeamID)
		if err != nil {
			return firstResponseDeadline, resolutionDeadline, err
		}
		businessHrsID = team.BusinessHoursID.Int
		timezone = team.Timezone
	}

	// If not found in team, fetch from app settings.
	if businessHrsID == 0 || timezone == "" {
		settingsJ, err := m.appSettingsStore.GetByPrefix("app")
		if err != nil {
			return firstResponseDeadline, resolutionDeadline, err
		}

		var out map[string]interface{}
		if err := json.Unmarshal([]byte(settingsJ), &out); err != nil {
			m.lo.Error("error parsing settings", "error", err)
			return firstResponseDeadline, resolutionDeadline, envelope.NewError(envelope.GeneralError, "Error parsing settings", nil)
		}

		businessHrsIDStr, _ := out["app.business_hours_id"].(string)
		businessHrsID, _ = strconv.Atoi(businessHrsIDStr)
		timezone, _ = out["app.timezone"].(string)
	}

	// Not set, skip SLA calculation.
	if businessHrsID == 0 || timezone == "" {
		m.lo.Warn("default business hours or timezone not set, skipping SLA calculation")
		return firstResponseDeadline, resolutionDeadline, nil
	}

	bh, err := m.businessHrsStore.Get(businessHrsID)
	if err != nil {
		m.lo.Error("error fetching business hours", "error", err)
		return firstResponseDeadline, resolutionDeadline, err
	}

	calculateDeadline := func(durationStr string) (time.Time, error) {
		if durationStr == "" {
			return time.Time{}, nil
		}
		dur, parseErr := time.ParseDuration(durationStr)
		if parseErr != nil {
			return time.Time{}, fmt.Errorf("parsing duration: %v", parseErr)
		}
		deadline, err := m.CalculateDeadline(
			conversationCreatedAt,
			int(dur.Minutes()),
			bh,
			timezone,
		)
		if err != nil {
			return time.Time{}, err
		}
		return deadline.Add(slaGracePeriod), nil
	}
	firstResponseDeadline, err = calculateDeadline(slaPolicy.FirstResponseTime)
	if err != nil {
		return firstResponseDeadline, resolutionDeadline, err
	}
	resolutionDeadline, err = calculateDeadline(slaPolicy.ResolutionTime)
	if err != nil {
		return firstResponseDeadline, resolutionDeadline, err
	}
	return firstResponseDeadline, resolutionDeadline, nil
}

// processUnbreachedSLAs fetches unbreached SLAs and pushes them to the worker pool for processing.
func (m *Manager) processUnbreachedSLAs() error {
	var unbreachedSLAs []models.ConversationSLA
	if err := m.q.GetUnbreachedSLAs.Select(&unbreachedSLAs); err != nil {
		m.lo.Error("error fetching unbreached SLAs", "error", err)
		return err
	}
	m.lo.Debug("processing unbreached SLAs", "count", len(unbreachedSLAs))
	for _, u := range unbreachedSLAs {
		slaData := u
		m.pool.Push(func() {
			if err := m.evaluateSLA(slaData); err != nil {
				m.lo.Error("error processing SLA", "error", err)
			}
		})
	}
	return nil
}

// evaluateSLA checks if an SLA has been breached or met and updates the database accordingly.
func (m *Manager) evaluateSLA(cSLA models.ConversationSLA) error {
	var deadline, compareTime time.Time

	// Calculate deadlines using the `created_at` which is the time SLA was applied to the conversation.
	// This will take care of the case where SLA is changed for a conversation.
	m.lo.Info("calculating SLA deadlines", "start_time", cSLA.CreatedAt, "conversation_id", cSLA.ConversationID, "sla_policy_id", cSLA.SLAPolicyID)
	firstResponseDeadline, resolutionDeadline, err := m.CalculateConversationDeadlines(cSLA.CreatedAt, cSLA.ConversationAssignedTeamID.Int, cSLA.SLAPolicyID)
	if err != nil {
		return err
	}

	switch cSLA.SLAType {
	case SLATypeFirstResponse:
		deadline = firstResponseDeadline
		compareTime = cSLA.ConversationFirstReplyAt.Time
	case SLATypeResolution:
		deadline = resolutionDeadline
		compareTime = cSLA.ConversationResolvedAt.Time
	default:
		return fmt.Errorf("unknown SLA type: %s", cSLA.SLAType)
	}

	if deadline.IsZero() {
		m.lo.Warn("could not calculate SLA deadline", "conversation_id", cSLA.ConversationID, "sla_policy_id", cSLA.SLAPolicyID)
		return nil
	}

	// Save deadline in DB.
	if _, err := m.q.UpdateDueAt.Exec(cSLA.ID, deadline); err != nil {
		m.lo.Error("error updating SLA due_at", "error", err)
		return fmt.Errorf("updating SLA due_at: %v", err)
	}

	if !compareTime.IsZero() {
		if compareTime.After(deadline) {
			return m.markSLABreached(cSLA.ID)
		}
		return m.markSLAMet(cSLA.ID, compareTime)
	}

	if time.Now().After(deadline) {
		return m.markSLABreached(cSLA.ID)
	}

	return nil
}

// markSLABreached updates the breach time for a conversation SLA.
func (m *Manager) markSLABreached(id int) error {
	if _, err := m.q.UpdateBreachedAt.Exec(id); err != nil {
		m.lo.Error("error updating SLA breach time", "error", err)
		return fmt.Errorf("updating SLA breach time: %v", err)
	}
	return nil
}

// markSLAMet updates the met time for a conversation SLA.
func (m *Manager) markSLAMet(id int, t time.Time) error {
	if _, err := m.q.UpdateMetAt.Exec(id, t); err != nil {
		m.lo.Error("error updating SLA met time", "error", err)
		return fmt.Errorf("updating SLA met time: %v", err)
	}
	return nil
}
