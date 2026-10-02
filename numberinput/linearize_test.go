package numberinput

import (
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeNamesTheFieldAsNumber(t *testing.T) {
	m := New()
	m.Prompt = "Age: "
	m.SetValue("42")
	if got := m.Linearize(); got != "Age, number field, value: 42" {
		t.Errorf("Linearize = %q", got)
	}
	bare := New()
	if got := bare.Linearize(); got != "Number field, empty" {
		t.Errorf("no prompt = %q", got)
	}
}
