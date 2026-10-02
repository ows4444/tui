package numberinput

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

// Criterion #335: a non-digit, non-leading-minus rune is rejected.
func TestRejectsNonDigitNonLeadingMinus(t *testing.T) {
	m := focused()
	typeString(t, &m, "12a3")
	if m.Value() != "123" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "123")
	}

	m2 := focused()
	typeString(t, &m2, "1-2")
	if m2.Value() != "12" {
		t.Fatalf("Value() = %q, want %q (mid-string '-' rejected)", m2.Value(), "12")
	}
}

// Criterion #336: digit runes are inserted, and a single leading '-' is
// accepted only as the very first character.
func TestAcceptsDigitsAndLeadingMinus(t *testing.T) {
	m := focused()
	typeString(t, &m, "-42")
	if m.Value() != "-42" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "-42")
	}

	// A second '-' typed at the start (before the existing '-') should be
	// rejected since the value would already start with '-'.
	m.SetCursor(0)
	next, _ := m.Update(rk('-'))
	m = next
	if m.Value() != "-42" {
		t.Fatalf("Value() = %q, want %q (extra leading '-' rejected)", m.Value(), "-42")
	}
}

// Criterion #336: behaves like textinput.Model for navigation, deletion
// and paste of digits.
func TestBehavesLikeTextinputForOtherOperations(t *testing.T) {
	m := focused()
	typeString(t, &m, "123")
	next, _ := m.Update(tui.Key{Type: tui.KeyLeft})
	m = next
	next, _ = m.Update(tui.Key{Type: tui.KeyBackspace})
	m = next
	if m.Value() != "13" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "13")
	}
	if m.Cursor() != 1 {
		t.Fatalf("Cursor() = %d, want 1", m.Cursor())
	}

	next, _ = m.Update(tui.PasteEvent{Text: "45"})
	m = next
	if m.Value() != "1453" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "1453")
	}
}
