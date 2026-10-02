package textarea

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
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

// Criterion #329: Enter inserts a newline at the cursor instead of being
// ignored.
func TestEnterInsertsNewline(t *testing.T) {
	m := focused()
	typeString(t, &m, "ab")
	next, _ := m.Update(tui.Key{Type: tui.KeyEnter})
	m = next
	typeString(t, &m, "cd")
	if m.Value() != "ab\ncd" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "ab\ncd")
	}
}

// Criterion #330: a value containing newlines renders each line on its
// own row in View.
func TestViewRendersEachLineOnOwnRow(t *testing.T) {
	m := New()
	m.SetValue("one\ntwo\nthree")
	lines := splitLines(m.View())
	if len(lines) != 3 {
		t.Fatalf("View() produced %d lines, want 3: %v", len(lines), lines)
	}
	if lines[0] != "one" || lines[1] != "two" || lines[2] != "three" {
		t.Fatalf("View() lines = %v, want [one two three]", lines)
	}
}

// Criterion #331: Backspace at the start of a line (immediately after a
// newline) deletes that newline, joining it with the previous line.
func TestBackspaceAtLineStartJoinsLines(t *testing.T) {
	m := focused()
	m.SetValue("ab\ncd")
	m.SetCursor(3) // right after the newline, before 'c'
	next, _ := m.Update(tui.Key{Type: tui.KeyBackspace})
	m = next
	if m.Value() != "abcd" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "abcd")
	}
	if m.Cursor() != 2 {
		t.Fatalf("Cursor() = %d, want 2", m.Cursor())
	}
}

// Criterion #332: Up/Down move the cursor to the adjacent line, clamped
// to that line's own length if shorter.
func TestUpDownMovesToAdjacentLineClamped(t *testing.T) {
	m := focused()
	m.SetValue("hello\nhi\nworld")
	// Cursor at end of "hello" (col 5, line 0).
	m.SetCursor(5)

	next, _ := m.Update(tui.Key{Type: tui.KeyDown})
	m = next
	// Line 1 is "hi" (len 2), so column clamps to 2: offset 6 ("hello\n"+2).
	if m.Cursor() != 8 {
		t.Fatalf("after Down, Cursor() = %d, want 8", m.Cursor())
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyDown})
	m = next
	// Line 2 is "world" (len 5); column was clamped to 2 on line 1, so
	// moving down again keeps column 2: offset of line 2 start (9) + 2 = 11.
	if m.Cursor() != 11 {
		t.Fatalf("after second Down, Cursor() = %d, want 11", m.Cursor())
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyUp})
	m = next
	if m.Cursor() != 8 {
		t.Fatalf("after Up, Cursor() = %d, want 8", m.Cursor())
	}
}

// Criterion #333: when Width is set and a line is longer than Width, that
// line's rendered ansi.Width stays within Width, independent of other
// lines.
func TestWidthScrollsLongLineIndependently(t *testing.T) {
	m := focused()
	m.Width = 5
	m.SetValue("a-very-long-first-line\nshort")
	m.SetCursor(20) // somewhere in the middle of the long first line

	lines := splitLines(m.View())
	if len(lines) != 2 {
		t.Fatalf("View() produced %d lines, want 2: %v", len(lines), lines)
	}
	if w := ansi.Width(lines[0]); w > m.Width {
		t.Fatalf("first line ansi.Width = %d, want <= %d", w, m.Width)
	}
	if w := ansi.Width(lines[1]); w > m.Width {
		t.Fatalf("second line ansi.Width = %d, want <= %d", w, m.Width)
	}
}

// Criterion #334: an empty value renders Placeholder, matching
// textinput.Model's empty-value behavior.
func TestEmptyValueRendersPlaceholder(t *testing.T) {
	m := New()
	m.Placeholder = "Type here..."
	view := m.View()
	if !contains(view, "Type here...") {
		t.Fatalf("View() = %q, want it to contain placeholder %q", view, "Type here...")
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i, r := range s {
		if r == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func contains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
