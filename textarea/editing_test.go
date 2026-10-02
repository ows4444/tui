package textarea

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
)

func press(m *Model, k tui.Key) {
	next, _ := m.Update(k)
	*m = next
}

func ctrl(r rune) tui.Key { return tui.Key{Type: tui.KeyCtrl, Code: r} }

func TestSetValueTruncatesToCharLimitAndMovesCursorToEnd(t *testing.T) {
	m := New()
	m.CharLimit = 3
	m.SetValue("abcdef")
	if m.Value() != "abc" || m.Cursor() != 3 {
		t.Fatalf("Value=%q Cursor=%d, want %q and 3", m.Value(), m.Cursor(), "abc")
	}
}

func TestResetClearsValueAndCursor(t *testing.T) {
	m := New()
	m.SetValue("abc")
	m.Reset()
	if m.Value() != "" || m.Cursor() != 0 {
		t.Fatalf("Value=%q Cursor=%d after Reset", m.Value(), m.Cursor())
	}
}

func TestFocusBlurAndIgnoredWhenUnfocused(t *testing.T) {
	m := New()
	if m.Focused() {
		t.Fatal("new model should not be focused")
	}
	press(&m, rk('x'))
	if m.Value() != "" {
		t.Fatalf("unfocused model accepted input: %q", m.Value())
	}
	if cmd := m.Focus(); cmd == nil || !m.Focused() {
		t.Fatal("Focus should focus and return a blink Cmd")
	}
	m.Blur()
	if m.Focused() {
		t.Fatal("Blur should unfocus")
	}
}

func TestBlinkTogglesCursorAndReschedules(t *testing.T) {
	m := focused()
	next, cmd := m.Update(blinkMsg{id: m.blinkID})
	if next.cursorVisible == m.cursorVisible || cmd == nil {
		t.Fatalf("blink should toggle visibility and return a Cmd (visible %v -> %v)", m.cursorVisible, next.cursorVisible)
	}
}

func TestDeleteRemovesRuneAtCursor(t *testing.T) {
	m := focused()
	m.SetValue("abc")
	m.SetCursor(1)
	press(&m, tui.Key{Type: tui.KeyDelete})
	if m.Value() != "ac" || m.Cursor() != 1 {
		t.Fatalf("Value=%q Cursor=%d, want ac/1", m.Value(), m.Cursor())
	}
	m.SetCursor(2)
	press(&m, tui.Key{Type: tui.KeyDelete}) // at end: no-op
	if m.Value() != "ac" {
		t.Fatalf("Delete at end changed value: %q", m.Value())
	}
}

func TestLeftRightSpaceAndBounds(t *testing.T) {
	m := focused()
	press(&m, tui.Key{Type: tui.KeyLeft}) // at 0: stays
	press(&m, tui.Key{Type: tui.KeyRight})
	if m.Cursor() != 0 {
		t.Fatalf("cursor moved on empty value: %d", m.Cursor())
	}
	typeString(t, &m, "a")
	press(&m, tui.Key{Type: tui.KeySpace})
	typeString(t, &m, "b")
	if m.Value() != "a b" {
		t.Fatalf("Value=%q, want %q", m.Value(), "a b")
	}
	press(&m, tui.Key{Type: tui.KeyLeft})
	press(&m, tui.Key{Type: tui.KeyRight})
	press(&m, tui.Key{Type: tui.KeyRight}) // at end: stays
	if m.Cursor() != 3 {
		t.Fatalf("Cursor=%d, want 3", m.Cursor())
	}
}

func TestHomeEndAreLinewise(t *testing.T) {
	m := focused()
	m.SetValue("ab\ncde")
	m.SetCursor(4)
	press(&m, tui.Key{Type: tui.KeyHome})
	if m.Cursor() != 3 {
		t.Fatalf("Home: Cursor=%d, want 3", m.Cursor())
	}
	press(&m, tui.Key{Type: tui.KeyEnd})
	if m.Cursor() != 6 {
		t.Fatalf("End: Cursor=%d, want 6", m.Cursor())
	}
}

func TestCtrlEditingShortcuts(t *testing.T) {
	cases := []struct {
		name   string
		key    rune
		value  string
		cursor int
		want   string
		wantC  int
	}{
		{"ctrl+a line start", 'a', "ab\ncde", 5, "ab\ncde", 3},
		{"ctrl+e line end", 'e', "ab\ncde", 3, "ab\ncde", 6},
		{"ctrl+u kills to line start", 'u', "ab\ncde", 5, "ab\ne", 3},
		{"ctrl+k kills to line end", 'k', "ab\ncde", 4, "ab\nc", 4},
		{"ctrl+w kills word", 'w', "one two", 7, "one ", 4},
		{"ctrl+w skips trailing spaces", 'w', "one two  ", 9, "one ", 4},
		{"ctrl+w stops at line start", 'w', "a\nbc", 4, "a\n", 2},
	}
	for _, c := range cases {
		m := focused()
		m.SetValue(c.value)
		m.SetCursor(c.cursor)
		press(&m, ctrl(c.key))
		if m.Value() != c.want || m.Cursor() != c.wantC {
			t.Errorf("%s: Value=%q Cursor=%d, want %q/%d", c.name, m.Value(), m.Cursor(), c.want, c.wantC)
		}
	}
	m := focused()
	press(&m, tui.Key{Type: tui.KeyCtrl}) // no runes: ignored
}

func TestAltRunesIgnoredAndCharLimitEnforced(t *testing.T) {
	m := focused()
	m.CharLimit = 2
	press(&m, tui.Key{Type: tui.KeyRunes, Text: "x", Mod: input.ModAlt})
	if m.Value() != "" {
		t.Fatalf("Alt+x inserted text: %q", m.Value())
	}
	typeString(t, &m, "abc")
	if m.Value() != "ab" {
		t.Fatalf("Value=%q, want CharLimit-truncated %q", m.Value(), "ab")
	}
}

func TestPasteKeepsNewlinesDropsCR(t *testing.T) {
	m := focused()
	next, _ := m.Update(tui.PasteEvent{Text: "a\r\nb"})
	if next.Value() != "a\nb" {
		t.Fatalf("Value=%q, want %q", next.Value(), "a\nb")
	}
}
