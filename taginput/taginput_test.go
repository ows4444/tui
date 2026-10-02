package taginput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/widgets"
)

func rk(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func typeString(m *Model, s string) {
	for _, r := range s {
		next, _ := m.Update(rk(r))
		*m = next
	}
}

func focused() Model {
	m := New()
	m.Input.Focus()
	return m
}

// Criterion #534: Enter on a non-empty input appends it as a new tag and
// clears the input field.
func TestEnterAddsTagAndClearsInput(t *testing.T) {
	m := focused()
	typeString(&m, "urgent")

	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})

	if len(m.Tags) != 1 || m.Tags[0] != "urgent" {
		t.Fatalf("Tags = %v, want [urgent]", m.Tags)
	}
	if m.Input.Value() != "" {
		t.Fatalf("Input.Value() = %q, want empty after Enter", m.Input.Value())
	}
}

// Criterion #535: Backspace while the input is empty removes the last tag
// rather than affecting the (already-empty) input.
func TestBackspaceOnEmptyInputPopsLastTag(t *testing.T) {
	m := focused()
	typeString(&m, "one")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	typeString(&m, "two")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})

	if len(m.Tags) != 2 {
		t.Fatalf("Tags = %v, want 2 entries before Backspace", m.Tags)
	}

	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})

	if len(m.Tags) != 1 || m.Tags[0] != "one" {
		t.Fatalf("Tags = %v, want [one] after Backspace on empty input", m.Tags)
	}
	if m.Input.Value() != "" {
		t.Fatalf("Input.Value() = %q, want unaffected empty value", m.Input.Value())
	}
}

// Backspace with a non-empty input value is forwarded to Input rather than
// popping a tag.
func TestBackspaceOnNonEmptyInputEditsInput(t *testing.T) {
	m := focused()
	typeString(&m, "one")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	typeString(&m, "tw")

	m, _ = m.Update(tui.Key{Type: tui.KeyBackspace})

	if len(m.Tags) != 1 || m.Tags[0] != "one" {
		t.Fatalf("Tags = %v, want unchanged [one]", m.Tags)
	}
	if m.Input.Value() != "t" {
		t.Fatalf("Input.Value() = %q, want %q", m.Input.Value(), "t")
	}
}

// Criterion #536: a positive MaxTags already reached makes Enter a no-op
// beyond the cap.
func TestMaxTagsCapsEnter(t *testing.T) {
	m := focused()
	m.MaxTags = 1
	typeString(&m, "one")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	typeString(&m, "two")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})

	if len(m.Tags) != 1 || m.Tags[0] != "one" {
		t.Fatalf("Tags = %v, want capped at [one]", m.Tags)
	}
	if m.Input.Value() != "two" {
		t.Fatalf("Input.Value() = %q, want %q (input left as-is when capped)", m.Input.Value(), "two")
	}
}

// Enter on an empty input is a no-op (no empty tag is ever added).
func TestEnterOnEmptyInputIsNoop(t *testing.T) {
	m := focused()
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})

	if len(m.Tags) != 0 {
		t.Fatalf("Tags = %v, want none added from an empty Enter", m.Tags)
	}
}

// Criterion #537: View renders each tag via widgets.Tag followed by the
// input field, not a reimplementation of chip styling.
func TestViewRendersTagsThenInput(t *testing.T) {
	m := focused()
	typeString(&m, "one")
	m, _ = m.Update(tui.Key{Type: tui.KeyEnter})
	typeString(&m, "rest")

	got := m.View()

	wantTag := widgets.Tag("one", widgets.TagSolid, m.Variant, m.Theme)
	if !strings.Contains(got, wantTag) {
		t.Fatalf("View() = %q, want it to contain widgets.Tag chip %q", got, wantTag)
	}
	if !strings.Contains(got, m.Input.View()) {
		t.Fatalf("View() = %q, want it to contain Input.View() %q", got, m.Input.View())
	}
	if strings.Index(got, wantTag) > strings.Index(got, m.Input.View()) {
		t.Fatalf("View() = %q, want tag chip before input field", got)
	}
}
