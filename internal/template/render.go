package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/valyala/fasthttp"
)

const (
	// Built-in templates names stored in the database.
	TmplConversationAssigned = "Conversation assigned"
	TmplSLABreachWarning     = "SLA breach warning"
	TmplSLABreached          = "SLA breached"

	// Built-in templates fetched from memory stored in `static` directory.
	TmplResetPassword = "reset-password"
	TmplWelcome       = "welcome"

	// Template names for rendering.
	TmplBase    = "base"
	TmplContent = "content"
)

// RenderEmailWithTemplate renders content inside the default outgoing email template.
func (m *Manager) RenderEmailWithTemplate(data any, content string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	defaultTmpl, err := m.getDefaultOutgoingEmailTemplate()
	if err != nil {
		if err == ErrTemplateNotFound {
			m.lo.Warn("default outgoing email template not found, rendering content without any template")
			return content, nil
		}
		m.lo.Error("error fetching default outgoing email template", "error", err)
		return "", fmt.Errorf("fetching default outgoing email template: %w", err)
	}

	baseTemplate, err := template.New(TmplBase).Funcs(m.funcMap).Parse(defaultTmpl.Body)
	if err != nil {
		return "", fmt.Errorf("parsing base template: %w", err)
	}

	contentTemplate, err := template.New(TmplContent).Funcs(m.funcMap).Parse(content)
	if err != nil {
		return "", fmt.Errorf("parsing content template: %w", err)
	}

	baseTemplate, err = baseTemplate.AddParseTree(TmplContent, contentTemplate.Tree)
	if err != nil {
		return "", fmt.Errorf("adding content template: %w", err)
	}

	var rendered strings.Builder
	if err := baseTemplate.ExecuteTemplate(&rendered, TmplBase, data); err != nil {
		return "", fmt.Errorf("executing base template: %w", err)
	}

	return rendered.String(), nil
}

// RenderStoredEmailTemplate fetches and renders an email template from the database, including subject and body and returns the rendered content.
func (m *Manager) RenderStoredEmailTemplate(name string, data any) (string, string, error) {
	tmpl, err := m.getByName(name)
	if err != nil {
		if err == ErrTemplateNotFound {
			return "", "", fmt.Errorf("template %s not found", name)
		}
		return "", "", err
	}

	executeContentTemplate := func(tmplBody string) (string, error) {
		var sb strings.Builder
		t, err := template.New(name).Funcs(m.funcMap).Parse(tmplBody)
		if err != nil {
			return "", fmt.Errorf("parsing content template: %w", err)
		}
		if err := t.Execute(&sb, data); err != nil {
			return "", fmt.Errorf("executing content template: %w", err)
		}
		return sb.String(), nil
	}

	executeSubjectTemplate := func(subject string) (string, error) {
		var sb strings.Builder
		subjectTmpl, err := template.New("subject").Funcs(m.funcMap).Parse(subject)
		if err != nil {
			return "", fmt.Errorf("parsing subject template: %w", err)
		}
		if err := subjectTmpl.Execute(&sb, data); err != nil {
			return "", fmt.Errorf("executing subject template: %w", err)
		}
		return sb.String(), nil
	}

	defaultTmpl, err := m.getDefaultOutgoingEmailTemplate()
	if err != nil {
		if err == ErrTemplateNotFound {
			m.lo.Warn("default outgoing email template not found, rendering content any template")
			content, err := executeContentTemplate(tmpl.Body)
			if err != nil {
				return "", "", err
			}
			subject, err := executeSubjectTemplate(tmpl.Subject.String)
			if err != nil {
				return "", "", err
			}
			return content, subject, nil
		}
		return "", "", err
	}

	baseTemplate, err := template.New(TmplBase).Funcs(m.funcMap).Parse(defaultTmpl.Body)
	if err != nil {
		return "", "", fmt.Errorf("parsing base template: %w", err)
	}

	contentTemplate, err := template.New(TmplContent).Funcs(m.funcMap).Parse(tmpl.Body)
	if err != nil {
		return "", "", fmt.Errorf("parsing content template: %w", err)
	}

	baseTemplate, err = baseTemplate.AddParseTree(TmplContent, contentTemplate.Tree)
	if err != nil {
		return "", "", fmt.Errorf("adding content template: %w", err)
	}

	var rendered strings.Builder
	if err := baseTemplate.ExecuteTemplate(&rendered, TmplBase, data); err != nil {
		return "", "", fmt.Errorf("executing base template: %w", err)
	}

	subject, err := executeSubjectTemplate(tmpl.Subject.String)
	if err != nil {
		return "", "", err
	}

	return rendered.String(), subject, nil
}

// RenderInMemoryTemplate executes an in-memory template with data and returns the rendered content.
// This is for system emails like reset password and welcome email etc.
func (m *Manager) RenderInMemoryTemplate(name string, data interface{}) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	var buf bytes.Buffer
	if err := m.tpls.ExecuteTemplate(&buf, name, data); err != nil {
		return "", fmt.Errorf("executing in-memory template %q: %w", name, err)
	}
	return buf.String(), nil
}

// RenderWebPage renders a template to the http.ResponseWriter with data.
func (m *Manager) RenderWebPage(ctx *fasthttp.RequestCtx, tmplFile string, data map[string]interface{}) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetStatusCode(fasthttp.StatusOK)
	// Add no-cache headers
	ctx.Response.Header.Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	ctx.Response.Header.Set("Pragma", "no-cache")
	ctx.Response.Header.Set("Expires", "0")
	return m.webTpls.ExecuteTemplate(ctx, tmplFile, data)
}
