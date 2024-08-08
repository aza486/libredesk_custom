package template

import (
	"bytes"
	"text/template"
)

// RenderDefault renders the system default template with the provided data.
func (m *Manager) RenderDefault(data interface{}) (string, error) {
	templ, err := m.GetDefault()
	if err != nil {
		return "", err
	}

	tmpl, err := template.New("default").Parse(templ.Body)
	if err != nil {
		return "", err
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", err
	}

	return rendered.String(), nil
}
