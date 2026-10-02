package commandpalette

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New(Command{Name: "open", Description: "Open a file"}, Command{Name: "save"}, Command{Name: "quit", Description: "Exit"})
	m.Input.Prompt = "Command: "
	m.Input.SetValue("o")
	got := m.Linearize()
	for _, want := range []string{"Command, text field, focused, value: o", "open, Open a file, option 1 of", ", selected"} {
		if !contains(got, want) {
			t.Errorf("Linearize lacks %q:\n%s", want, got)
		}
	}
	m.Input.SetValue("sav")
	if got := m.Linearize(); !contains(got, "1 command") || !contains(got, "save, option 1 of 1, selected") {
		t.Errorf("single command = %q", got)
	}
	m.Input.SetValue("qqqq")
	if got := m.Linearize(); contains(got, "command") && contains(got, "option") {
		t.Errorf("no match should list nothing: %q", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
