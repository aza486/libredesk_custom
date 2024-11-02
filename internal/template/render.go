package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/abhinavxd/artemis/internal/conversation/models"
)

const (
	TmplConversationAssigned = "conversation-assigned"
	TmplResetPassword        = "reset-password"
	TmplWelcome              = "welcome"
	TmplBase                 = "base"
	TmplContent              = "content"
)

// RenderEmail renders a message into the base email template. It combines the base template
// with message content and conversation data to produce the final email HTML.
// The base template must contain a {{ template "content" . }} block where the message
// will be inserted.
func (m *Manager) RenderEmail(conversation models.Conversation, messageContent string) (string, error) {
	tmpl, err := m.GetDefault()
	if err != nil {
		if err == ErrTemplateNotFound {
			m.lo.Warn("default email template not found, using message content as is")
			return messageContent, nil
		}
		return "", err
	}

	// Parse base template first
	baseTmpl, err := template.New(TmplBase).Parse(tmpl.Body)
	if err != nil {
		return "", fmt.Errorf("parsing base template: %w", err)
	}

	// Parse message template
	msgTmpl, err := template.New(TmplContent).Parse(messageContent)
	if err != nil {
		return "", fmt.Errorf("parsing message template: %w", err)
	}

	// Add message template to base
	baseTmpl, err = baseTmpl.AddParseTree(TmplContent, msgTmpl.Tree)
	if err != nil {
		return "", fmt.Errorf("adding content template to base: %w", err)
	}

	var out strings.Builder
	if err := baseTmpl.ExecuteTemplate(&out, TmplBase, conversation); err != nil {
		return "", fmt.Errorf("executing template: %w", err)
	}

	return out.String(), nil
}

// Render executes a named template with the provided data.
func (m *Manager) Render(name string, data interface{}) (string, error) {
	var rendered bytes.Buffer

	err := m.tpls.ExecuteTemplate(&rendered, name, data)
	if err != nil {
		return "", fmt.Errorf("executing template %s: %w", name, err)
	}

	return rendered.String(), nil
}
