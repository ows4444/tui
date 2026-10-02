package maskedinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
)

func rk(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func typeString(t *testing.T, m *Model, s string) {
	t.Helper()
	for _, r := range s {
		next, _ := m.Model.Update(rk(r))
		m.Model = next
	}
}

func focused() Model {
	m := New()
	m.Focus()
	return m
}

// Criterion #516: a non-empty value renders every character masked with
// the configurable Mask rune, while Value() still returns the real,
// unmasked text.
func TestNonEmptyValueIsMaskedInViewButNotInValue(t *testing.T) {
	m := focused()
	typeString(t, &m, "hunter2")

	if m.Value() != "hunter2" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "hunter2")
	}
	view := m.View()
	if strings.Contains(view, "hunter2") {
		t.Fatalf("View() = %q, leaked the real value", view)
	}
	if got := strings.Count(view, string(defaultMask)); got != len("hunter2") {
		t.Fatalf("View() contains %d default mask runes, want %d: %q", got, len("hunter2"), view)
	}
}

// Criterion #516: Mask is configurable.
func TestMaskIsConfigurable(t *testing.T) {
	m := focused()
	m.Mask = '*'
	typeString(t, &m, "secret")

	view := m.View()
	if strings.Contains(view, string(defaultMask)) {
		t.Fatalf("View() = %q, used default mask instead of configured Mask", view)
	}
	if got := strings.Count(view, "*"); got != len("secret") {
		t.Fatalf("View() contains %d '*' runes, want %d: %q", got, len("secret"), view)
	}
}

// TestUpdatePreservesMaskAcrossTheOuterWrapper proves the bug this
// session's Component[T] sweep surfaced: before Model.Update existed,
// calling Update through the outer wrapper (the standard
// `m, cmd := m.Update(msg)` pattern every other widget in this library
// supports) resolved to the promoted textinput.Model.Update, whose return
// type is textinput.Model — silently discarding Mask on the very first
// call. This proves the explicit override fixes it: Mask survives, and
// the returned value's type is Model (proven by the assignment itself
// compiling, since := with a textinput.Model on the right wouldn't infer
// a variable usable as Model further down).
func TestUpdatePreservesMaskAcrossTheOuterWrapper(t *testing.T) {
	m := focused()
	m.Mask = '*'

	next, _ := m.Update(rk('a'))
	if next.Mask != '*' {
		t.Fatalf("Mask after Update() = %q, want %q — Update silently dropped the wrapper's own field", string(next.Mask), "*")
	}
	if next.Value() != "a" {
		t.Fatalf("Value() after Update() = %q, want %q", next.Value(), "a")
	}
}

func TestEmptyValueRendersPlaceholderUnmasked(t *testing.T) {
	m := New()
	m.Placeholder = "PIN"
	view := m.View()
	if !strings.Contains(view, "PIN") {
		t.Fatalf("View() = %q, want it to contain placeholder %q", view, "PIN")
	}
	if strings.Contains(view, string(defaultMask)) {
		t.Fatalf("View() = %q, want no mask runes for empty value", view)
	}
}
