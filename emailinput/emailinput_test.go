package emailinput

import (
	"testing"

	"github.com/ows4444/tui"
)

func rk(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }

func typeString(t *testing.T, m *Model, s string) {
	t.Helper()
	for _, r := range s {
		next, _ := m.Update(rk(r))
		*m = next
	}
}

func focused() Model {
	m := New()
	m.Focus()
	return m
}

// Criterion #514: whitespace runes are rejected, everything else forwarded.
func TestRejectsWhitespaceRunes(t *testing.T) {
	m := focused()
	typeString(t, &m, "a b\tc\nd")
	if m.Value() != "abcd" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "abcd")
	}
}

func TestBehavesLikeTextinputForOtherOperations(t *testing.T) {
	m := focused()
	typeString(t, &m, "abc")
	next, _ := m.Update(tui.Key{Type: tui.KeyLeft})
	m = next
	next, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	m = next
	if m.Value() != "ac" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "ac")
	}
	if m.Cursor() != 1 {
		t.Fatalf("Cursor() = %d, want 1", m.Cursor())
	}

	next, _ = m.Update(tui.PasteEvent{Text: "x@y"})
	m = next
	if m.Value() != "ax@yc" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "ax@yc")
	}
}

// Criterion #515: Valid heuristic.
func TestValid(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"", false},
		{"nodomain", false},
		{"a@b", false},      // no '.' after '@'
		{"a@.com", false},   // empty part before '.'
		{"a@b.", false},     // empty part after '.'
		{"@b.com", false},   // empty local part
		{"a@@b.com", false}, // more than one '@'
		{"a@b.com", true},
		{"first.last@sub.example.com", true},
	}
	for _, c := range cases {
		m := New()
		m.SetValue(c.value)
		if got := m.Valid(); got != c.want {
			t.Errorf("Valid() for %q = %v, want %v", c.value, got, c.want)
		}
	}
}
