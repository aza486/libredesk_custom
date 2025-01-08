// Package ai handles management of LLM providers.
package ai

import (
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
	efs           embed.FS
	systemPrompts = map[string]string{
		"make_friendly":             "Modify the text to make it more friendly and approachable.",
		"make_professional":         "Rephrase the text to make it sound more formal and professional.",
		"make_concise":              "Simplify the text to make it more concise and to the point.",
		"add_empathy":               "Add empathy to the text while retaining the original meaning.",
		"adjust_positive_tone":      "Adjust the tone of the text to make it sound more positive and reassuring.",
		"provide_clear_explanation": "Rewrite the text to provide a clearer explanation of the issue or solution.",
		"add_urgency":               "Modify the text to convey a sense of urgency without being rude.",
		"make_actionable":           "Rephrase the text to clearly specify the next steps for the customer.",
		"adjust_neutral_tone":       "Adjust the tone to make it neutral and unbiased.",
	}
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
	GetProvider     *sqlx.Stmt `query:"get-provider"`
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

// SendPromptToProvider sends a prompt to the specified provider and returns the response.
func (m *Manager) SendPromptToProvider(provider, k string, prompt string) (string, error) {

	// Fetch the system prompt.
	systemPrompt, ok := systemPrompts[k]
	if !ok {
		m.lo.Error("invalid system prompt key", "key", k)
		return "", envelope.NewError(envelope.InputError, "Invalid system prompt key", nil)
	}

	client, err := m.getProviderClient(provider)
	if err != nil {
		m.lo.Error("error getting provider client", "provider", provider, "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error getting provider client", nil)
	}

	payload := PromptPayload{
		SystemPrompt: systemPrompt,
		UserPrompt:   prompt,
	}

	response, err := client.SendPrompt(payload)
	if err != nil {
		m.lo.Error("error sending prompt to provider", "provider", provider, "error", err)
		return "", envelope.NewError(envelope.GeneralError, "Error sending prompt to provider", nil)
	}

	return response, nil
}

// getProviderClient retrieves a ProviderClient for the specified provider.
func (m *Manager) getProviderClient(providerName string) (ProviderClient, error) {
	var p models.Provider

	if err := m.q.GetProvider.Get(&p, providerName); err != nil {
		m.lo.Error("error fetching provider details", "provider", providerName, "error", err)
		return nil, envelope.NewError(envelope.GeneralError, "Error fetching provider details", nil)
	}

	switch ProviderType(p.Provider) {
	case ProviderOpenAI:
		config := struct {
			APIKey string `json:"api_key"`
		}{}
		if err := json.Unmarshal([]byte(p.Config), &config); err != nil {
			m.lo.Error("error parsing provider config", "provider", providerName, "error", err)
			return nil, envelope.NewError(envelope.GeneralError, "Error parsing provider config", nil)
		}
		return NewOpenAIClient(config.APIKey), nil
	default:
		m.lo.Error("unsupported provider type", "provider", p.Provider)
		return nil, envelope.NewError(envelope.GeneralError, "Unsupported provider type", nil)
	}
}
