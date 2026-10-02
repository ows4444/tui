package textinput_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/textinput"
)

func rk(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

// Criterion #339: New returns a Model preconfigured with a distinct
// search prompt and/or placeholder, behaving identically to
// textinput.Model for everything else.
func TestSearchNewHasDistinctPromptOrPlaceholder(t *testing.T) {
	m := textinput.NewSearch()
	if m.Prompt == "" && m.Placeholder == "" {
		t.Fatalf("textinput.NewSearch() has neither a distinct Prompt nor Placeholder")
	}
	view := m.View()
	if !strings.Contains(view, m.Prompt) && !strings.Contains(view, m.Placeholder) {
		t.Fatalf("View() = %q, want it to reflect the configured prompt/placeholder", view)
	}
}

func TestSearchBehavesIdenticallyToTextinputOtherwise(t *testing.T) {
	m := textinput.NewSearch()
	m.Focus()
	next, _ := m.Update(rk('a'))
	m = next
	next, _ = m.Update(rk('b'))
	m = next
	if m.Value() != "ab" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "ab")
	}

	// Same behavior as a plain textinput.Model given the same key sequence.
	plain := textinput.New()
	plain.Focus()
	pn, _ := plain.Update(rk('a'))
	plain = pn
	pn, _ = plain.Update(rk('b'))
	plain = pn
	if plain.Value() != m.Value() {
		t.Fatalf("searchinput diverged from textinput: got %q, want %q", m.Value(), plain.Value())
	}
}
