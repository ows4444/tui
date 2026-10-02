package emailinput

import "strings"

// Linearize is the embedded textinput's line with the field named "email
// field". The value is spoken, as an email address is not a secret.
func (m Model) Linearize() string {
	s := m.Model.Linearize()
	if strings.HasPrefix(s, "Text field") {
		return "Email field" + strings.TrimPrefix(s, "Text field")
	}
	return strings.Replace(s, ", text field", ", email field", 1)
}
