package numberinput

import "strings"

// Linearize is the embedded textinput's line with the field named "number
// field".
func (m Model) Linearize() string {
	s := m.Model.Linearize()
	if strings.HasPrefix(s, "Text field") {
		return "Number field" + strings.TrimPrefix(s, "Text field")
	}
	return strings.Replace(s, ", text field", ", number field", 1)
}
