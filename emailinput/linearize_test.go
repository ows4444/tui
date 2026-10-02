package emailinput

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeNamesTheFieldAsEmail(t *testing.T) {
	m := New()
	m.Prompt = "Email: "
	m.SetValue("ada@example.com")
	if got := m.Linearize(); got != "Email, email field, value: ada@example.com" {
		t.Errorf("Linearize = %q", got)
	}
	bare := New()
	if got := bare.Linearize(); got != "Email field, empty" {
		t.Errorf("no prompt = %q", got)
	}
}
