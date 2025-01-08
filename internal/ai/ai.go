// Package ai handles management of LLM providers.
package ai

import (
	"database/sql"
	"embed"
	"encoding/json"

	"github.com/abhinavxd/artemis/internal/ai/models"
	"github.com/abhinavxd/artemis/internal/dbutil"
	"github.com/abhinavxd/artemis/internal/envelope"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

var (
	//go:embed queries.sql
	efs embed.FS
)

// Manager manages LLM providers.
type Manager struct {
	q  queries
	lo *logf.Logger
}

// Opts contains options for initializing the Manager.
type Opts struct {
	DB *sqlx.DB
	Lo *logf.Logger
}

// queries contains prepared SQL queries.
type queries struct {
	GetDefaultProvider *sqlx.Stmt `query:"get-default-provider"`
	GetPrompt          *sqlx.Stmt `query:"get-prompt"`
	GetPrompts         *sqlx.Stmt `query:"get-prompts"`
}

// New creates and returns a new instance of the Manager.
func New(opts Opts) (*Manager, error) {
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, opts.DB, efs); err != nil {
		return nil, err
	}
	return &Manager{
		q:  q,
		lo: opts.Lo,
	}, nil
}

// SendPrompt sends a prompt to the default provider and returns the response.
func (m *Manager) SendPrompt(k string, prompt string) (string, error) {
	// return "Hello Abhinav,\n\nI wanted to let you know that we have successfully added a top-up of Rs 20,000 to your account. You can expect to receive the funds by the end of the day.\n\nIf you have any concerns or questions, please don't hesitate to update this ticket. We are always happy to assist you in any way we can.\n\nPlease keep in mind that if we do not hear back from you within 24 hours, the ticket will be marked as <i>closed</i>. However, you can easily reopen it if you need further assistance. Additionally, you can always reach out to us through our support portal for any help you may need.\n\nThank you for choosing our services. We are here to make your experience as smooth and hassle-free as possible.\n\nBest regards, \n[Your Name]", nil
	systemPrompt, err := m.getPrompt(k)
	if err != nil {
		return "", err
	}

	client, err := m.getDefaultProviderClient()
	if err != nil {
		m.lo.Error("error getting provider client", "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error getting provider client", nil)
	}

	payload := PromptPayload{
		SystemPrompt: systemPrompt,
		UserPrompt:   prompt,
	}

	response, err := client.SendPrompt(payload)
	if err != nil {
		m.lo.Error("error sending prompt to provider", "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error sending prompt to provider", nil)
	}

	return response, nil
}

// GetPrompts returns a list of prompts from the database.
func (m *Manager) GetPrompts() ([]models.Prompt, error) {
	var prompts = make([]models.Prompt, 0)
	if err := m.q.GetPrompts.Select(&prompts); err != nil {
		m.lo.Error("error fetching prompts", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching prompts", nil)
	}
	return prompts, nil
}

// getPrompt returns a prompt from the database.
func (m *Manager) getPrompt(k string) (string, error) {
	var p models.Prompt
	if err := m.q.GetPrompt.Get(&p, k); err != nil {
		if err == sql.ErrNoRows {
			m.lo.Error("error prompt not found", "key", k)
			return "", envelope.NewError(envelope.InputError, "Prompt not found", nil)
		}
		m.lo.Error("error fetching prompt", "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error fetching prompt", nil)
	}
	return p.Content, nil
}

// getDefaultProviderClient returns a ProviderClient for the default provider.
func (m *Manager) getDefaultProviderClient() (ProviderClient, error) {
	var p models.Provider

	if err := m.q.GetDefaultProvider.Get(&p); err != nil {
		m.lo.Error("error fetching provider details", "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching provider details", nil)
	}

	switch ProviderType(p.Provider) {
	case ProviderOpenAI:
		config := struct {
			APIKey string `json:"api_key"`
		}{}
		if err := json.Unmarshal([]byte(p.Config), &config); err != nil {
			m.lo.Error("error parsing provider config", "error", err)
			return nil, envelope.NewError(envelope.GeneralError, "Error parsing provider config", nil)
		}
		return NewOpenAIClient(config.APIKey), nil
	default:
		m.lo.Error("unsupported provider type", "provider", p.Provider)
		return nil, envelope.NewError(envelope.GeneralError, "Unsupported provider type", nil)
	}
}
