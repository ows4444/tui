package passwordinput

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

// Criterion #337: a non-empty value renders every character masked, while
// Value() still returns the real, unmasked text.
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
	if got := strings.Count(view, string(Mask)); got != len("hunter2") {
		t.Fatalf("View() contains %d mask runes, want %d: %q", got, len("hunter2"), view)
	}
}

// TestUpdatePreservesWrapperTypeAcrossTheOuterWrapper proves the bug this
// session's Component[T] sweep surfaced: before Model.Update existed,
// calling Update through the outer wrapper (the standard
// `m, cmd := m.Update(msg)` pattern every other widget in this library
// supports) resolved to the promoted textinput.Model.Update, whose return
// type is textinput.Model, not Model. That's observable here even though
// this wrapper adds no extra field: masking is Model's own View
// override, so if Update had silently downgraded the type, the result's
// View() would render the real, unmasked text — this proves it doesn't.
func TestUpdatePreservesWrapperTypeAcrossTheOuterWrapper(t *testing.T) {
	m := focused()

	next, _ := m.Update(rk('a'))
	if next.Value() != "a" {
		t.Fatalf("Value() after Update() = %q, want %q", next.Value(), "a")
	}
	if view := next.View(); strings.Contains(view, "a") {
		t.Fatalf("View() after Update() = %q, leaked the unmasked value — Update returned textinput.Model, not Model", view)
	}
}

// Criterion #338: an empty value renders Placeholder unmasked.
func TestEmptyValueRendersPlaceholderUnmasked(t *testing.T) {
	m := New()
	m.Placeholder = "Password"
	view := m.View()
	if !strings.Contains(view, "Password") {
		t.Fatalf("View() = %q, want it to contain placeholder %q", view, "Password")
	}
	if strings.Contains(view, string(Mask)) {
		t.Fatalf("View() = %q, want no mask runes for empty value", view)
	}
}
