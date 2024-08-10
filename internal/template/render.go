package template

import (
	"bytes"
	"text/template"
)

// RenderDefault renders the system default template with the provided data.
func (m *Manager) RenderDefault(data map[string]string) (string, error) {
	templ, err := m.GetDefault()
	if err != nil {
		// Template not found, return the content as is.
		if err == ErrTemplateNotFound {
			return data["Content"], nil
		}
		return "", err
	}

	var rendered bytes.Buffer
	if err := template.Must(template.New("default").Parse(templ.Body)).Execute(&rendered, data); err != nil {
		return "", err
	}

	return rendered.String(), nil
}
