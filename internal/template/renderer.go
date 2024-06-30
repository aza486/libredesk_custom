package template

import (
	"bytes"
	"text/template"
)

// RenderDefault renders the system default template with the data.
func (m *Manager) RenderDefault(data interface{}) (string, string, error) {
	templ, err := m.GetDefaultTemplate()
	if err != nil {
		return "", "", err
	}

	tmpl, err := template.New("").Parse(templ.Body)
	if err != nil {
		return "", "", err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", "", err
	}

	return rendered.String(), templ.Subject, nil
}
