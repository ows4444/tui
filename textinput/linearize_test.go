package textinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

var _ tui.Linearizer = Model{}

func TestLinearize(t *testing.T) {
	m := New()
	m.Prompt = "Name:  "
	m.Placeholder = "Ada Lovelace"
	if got := m.Linearize(); got != "Name, text field, empty, placeholder: Ada Lovelace" {
		t.Errorf("empty = %q", got)
	}
	m.SetValue("Grace")
	m.Focus()
	if got := m.Linearize(); got != "Name, text field, focused, value: Grace" {
		t.Errorf("focused with value = %q", got)
	}
	m.Blur()
	if got := m.Linearize(); got != "Name, text field, value: Grace" {
		t.Errorf("blurred = %q", got)
	}
	bare := New()
	if got := bare.Linearize(); got != "Text field, empty" {
		t.Errorf("no prompt or placeholder = %q", got)
	}
	if strings.ContainsAny(m.Linearize(), "\x1b") {
		t.Error("no escape sequences allowed")
	}
}

// Prompts that are only punctuation ("> ", "$ ") carry no wording to speak.
func TestLinearizeIgnoresPunctuationOnlyPrompts(t *testing.T) {
	for _, prompt := range []string{"> ", "$ ", "»", "  ", ""} {
		m := New()
		m.Prompt = prompt
		m.SetValue("x")
		if got := m.Linearize(); got != "Text field, value: x" {
			t.Errorf("prompt %q -> %q", prompt, got)
		}
	}
	m := New()
	m.Prompt = "  Full name (required): "
	if got := m.Linearize(); got != "Full name (required), text field, empty" {
		t.Errorf("inner punctuation should survive: %q", got)
	}
}
