package textinput_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/textinput"
)

// Criterion #517: New returns a Model preconfigured with a distinct
// placeholder for file-path entry, behaving identically to
// textinput.Model for everything else.
func TestPathNewHasPathPlaceholder(t *testing.T) {
	m := textinput.NewPath()
	if m.Placeholder == "" {
		t.Fatalf("textinput.NewPath() has no Placeholder")
	}
	view := m.View()
	if !strings.Contains(view, m.Placeholder) {
		t.Fatalf("View() = %q, want it to contain placeholder %q", view, m.Placeholder)
	}
}

func TestPathBehavesIdenticallyToTextinputOtherwise(t *testing.T) {
	m := textinput.NewPath()
	m.Focus()
	next, _ := m.Update(rk('/'))
	m = next
	next, _ = m.Update(rk('a'))
	m = next
	if m.Value() != "/a" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "/a")
	}

	plain := textinput.New()
	plain.Focus()
	pn, _ := plain.Update(rk('/'))
	plain = pn
	pn, _ = plain.Update(rk('a'))
	plain = pn
	if plain.Value() != m.Value() {
		t.Fatalf("pathinput diverged from textinput: got %q, want %q", m.Value(), plain.Value())
	}
}
