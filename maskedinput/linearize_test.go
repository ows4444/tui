package maskedinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearizeNeverRevealsTheValue(t *testing.T) {
	m := New()
	m.Prompt = "Card: "
	m.SetValue("4111111111111111")
	got := m.Linearize()
	if got != "Card, masked field, 16 characters entered" {
		t.Errorf("Linearize = %q", got)
	}
	if strings.Contains(got, "4111") {
		t.Errorf("leaked the value: %q", got)
	}
	if !strings.Contains(m.Model.Linearize(), "4111111111111111") {
		t.Fatal("test premise changed: the embedded Linearize no longer speaks the value")
	}
}

func TestLinearizeEmpty(t *testing.T) {
	if got := New().Linearize(); got != "Masked field, empty" {
		t.Errorf("empty = %q", got)
	}
}
