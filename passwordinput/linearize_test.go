package passwordinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeNeverRevealsThePassword(t *testing.T) {
	m := New()
	m.Prompt = "Password: "
	m.SetValue("hunter2-secret")
	m.Focus()
	got := m.Linearize()
	if got != "Password, password field, focused, 14 characters entered" {
		t.Errorf("Linearize = %q", got)
	}
	for _, leak := range []string{"hunter", "secret"} {
		if strings.Contains(got, leak) {
			t.Errorf("Linearize leaked %q: %q", leak, got)
		}
	}
	// The embedded textinput's own method would speak the value; the override
	// is what protects it, so pin that the two really differ.
	if strings.Contains(m.Model.Linearize(), "hunter2-secret") == false {
		t.Fatal("test premise changed: the embedded Linearize no longer speaks the value")
	}
	if strings.Contains(tuiLinearize(m), "hunter") {
		t.Error("a tui.Linearizer view of the Model leaked the password")
	}
}

// tuiLinearize calls Linearize the way Program does: through the interface.
func tuiLinearize(l tui.Linearizer) string { return l.Linearize() }

func TestLinearizeEmptyAndSingleCharacter(t *testing.T) {
	m := New()
	if got := m.Linearize(); got != "Password field, empty" {
		t.Errorf("empty = %q", got)
	}
	m.SetValue("x")
	if got := m.Linearize(); got != "Password field, 1 character entered" {
		t.Errorf("one char = %q", got)
	}
}
