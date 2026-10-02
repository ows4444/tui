package textinput

import (
	"context"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func rk(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string([]rune{r})} }
func ck(r rune) tui.Key { return tui.Key{Type: tui.KeyCtrl, Code: r} }

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

func TestTypingInsertsAtCursor(t *testing.T) {
	m := focused()
	typeString(t, &m, "hello")
	if m.Value() != "hello" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "hello")
	}
	if m.Cursor() != 5 {
		t.Fatalf("Cursor() = %d, want 5", m.Cursor())
	}
}

func TestInsertAtMiddle(t *testing.T) {
	m := focused()
	typeString(t, &m, "helo") // deliberately missing the second 'l'
	m.SetCursor(3)            // between 'e' and the trailing "lo": "hel|o"... actually h-e-l-o, cursor 3 is before 'o'
	next, _ := m.Update(rk('l'))
	m = next
	if m.Value() != "hello" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "hello")
	}
	if m.Cursor() != 4 {
		t.Fatalf("Cursor() = %d, want 4", m.Cursor())
	}
}

func TestSpaceKey(t *testing.T) {
	m := focused()
	typeString(t, &m, "ab")
	next, _ := m.Update(tui.Key{Type: tui.KeySpace})
	m = next
	typeString(t, &m, "cd")
	if m.Value() != "ab cd" {
		t.Fatalf("Value() = %q, want %q", m.Value(), "ab cd")
	}
}

func TestBackspace(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		cursor     int
		wantValue  string
		wantCursor int
	}{
		{"middle", "hello", 3, "helo", 2},
		{"at start is a no-op", "hello", 0, "hello", 0},
		{"at end", "hello", 5, "hell", 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := focused()
			m.SetValue(tt.value)
			m.SetCursor(tt.cursor)
			next, _ := m.Update(tui.Key{Type: tui.KeyBackspace})
			m = next
			if m.Value() != tt.wantValue || m.Cursor() != tt.wantCursor {
				t.Errorf("got (%q, %d), want (%q, %d)", m.Value(), m.Cursor(), tt.wantValue, tt.wantCursor)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		cursor     int
		wantValue  string
		wantCursor int
	}{
		{"middle", "hello", 2, "helo", 2},
		{"at end is a no-op", "hello", 5, "hello", 5},
		{"at start", "hello", 0, "ello", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := focused()
			m.SetValue(tt.value)
			m.SetCursor(tt.cursor)
			next, _ := m.Update(tui.Key{Type: tui.KeyDelete})
			m = next
			if m.Value() != tt.wantValue || m.Cursor() != tt.wantCursor {
				t.Errorf("got (%q, %d), want (%q, %d)", m.Value(), m.Cursor(), tt.wantValue, tt.wantCursor)
			}
		})
	}
}

func TestLeftRightHomeEnd(t *testing.T) {
	m := focused()
	m.SetValue("hello")
	m.SetCursor(2)

	next, _ := m.Update(tui.Key{Type: tui.KeyLeft})
	m = next
	if m.Cursor() != 1 {
		t.Fatalf("after Left: Cursor() = %d, want 1", m.Cursor())
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyRight})
	m = next
	next, _ = m.Update(tui.Key{Type: tui.KeyRight})
	m = next
	if m.Cursor() != 3 {
		t.Fatalf("after 2x Right: Cursor() = %d, want 3", m.Cursor())
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyHome})
	m = next
	if m.Cursor() != 0 {
		t.Fatalf("after Home: Cursor() = %d, want 0", m.Cursor())
	}

	next, _ = m.Update(tui.Key{Type: tui.KeyEnd})
	m = next
	if m.Cursor() != 5 {
		t.Fatalf("after End: Cursor() = %d, want 5", m.Cursor())
	}

	// Boundary no-ops.
	m.SetCursor(0)
	next, _ = m.Update(tui.Key{Type: tui.KeyLeft})
	m = next
	if m.Cursor() != 0 {
		t.Errorf("Left at cursor 0 should be a no-op, got Cursor() = %d", m.Cursor())
	}
	m.SetCursor(5)
	next, _ = m.Update(tui.Key{Type: tui.KeyRight})
	m = next
	if m.Cursor() != 5 {
		t.Errorf("Right at end should be a no-op, got Cursor() = %d", m.Cursor())
	}
}

func TestCtrlAE(t *testing.T) {
	m := focused()
	m.SetValue("hello")
	m.SetCursor(2)

	next, _ := m.Update(ck('a'))
	m = next
	if m.Cursor() != 0 {
		t.Errorf("ctrl+a: Cursor() = %d, want 0", m.Cursor())
	}
	next, _ = m.Update(ck('e'))
	m = next
	if m.Cursor() != 5 {
		t.Errorf("ctrl+e: Cursor() = %d, want 5", m.Cursor())
	}
}

func TestCtrlUK(t *testing.T) {
	m := focused()
	m.SetValue("hello world")
	m.SetCursor(5) // "hello| world"

	next, _ := m.Update(ck('k'))
	m = next
	if m.Value() != "hello" {
		t.Fatalf("ctrl+k: Value() = %q, want %q", m.Value(), "hello")
	}

	m2 := focused()
	m2.SetValue("hello world")
	m2.SetCursor(6) // "hello |world"
	next, _ = m2.Update(ck('u'))
	m2 = next
	if m2.Value() != "world" || m2.Cursor() != 0 {
		t.Fatalf("ctrl+u: got (%q, %d), want (%q, 0)", m2.Value(), m2.Cursor(), "world")
	}
}

func TestCtrlWDeletesWordBeforeCursor(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		cursor     int
		wantValue  string
		wantCursor int
	}{
		{"end of single word", "hello", 5, "", 0},
		{"end of two words", "hello world", 11, "hello ", 6},
		{"skips trailing spaces first", "hello   ", 8, "", 0},
		{"mid-word", "hello world", 8, "hello rld", 6},
		{"at start is a no-op", "hello", 0, "hello", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := focused()
			m.SetValue(tt.value)
			m.SetCursor(tt.cursor)
			next, _ := m.Update(ck('w'))
			m = next
			if m.Value() != tt.wantValue || m.Cursor() != tt.wantCursor {
				t.Errorf("got (%q, %d), want (%q, %d)", m.Value(), m.Cursor(), tt.wantValue, tt.wantCursor)
			}
		})
	}
}

func TestCharLimit(t *testing.T) {
	m := focused()
	m.CharLimit = 3
	typeString(t, &m, "hello")
	if m.Value() != "hel" {
		t.Fatalf("Value() = %q, want %q (typing past CharLimit should be dropped)", m.Value(), "hel")
	}

	m2 := New()
	m2.CharLimit = 3
	m2.SetValue("hello")
	if m2.Value() != "hel" {
		t.Fatalf("SetValue() = %q, want %q (should truncate to CharLimit)", m2.Value(), "hel")
	}
}

func TestUpdateIgnoredWhenUnfocused(t *testing.T) {
	m := New() // not focused
	next, cmd := m.Update(rk('a'))
	if next.Value() != "" {
		t.Errorf("unfocused Update mutated value: %q", next.Value())
	}
	if cmd != nil {
		t.Errorf("unfocused Update returned a non-nil Cmd")
	}
}

func TestPasteInsertsAtCursor(t *testing.T) {
	m := focused()
	m.SetValue("ac")
	m.SetCursor(1) // "a|c"
	next, cmd := m.Update(tui.PasteEvent{Text: "b"})
	m = next
	if m.Value() != "abc" {
		t.Errorf("Value() = %q, want %q", m.Value(), "abc")
	}
	if m.Cursor() != 2 {
		t.Errorf("Cursor() = %d, want 2", m.Cursor())
	}
	if cmd != nil {
		t.Errorf("paste should not return a Cmd, got %v", cmd)
	}
}

func TestPasteStripsNewlines(t *testing.T) {
	m := focused()
	next, _ := m.Update(tui.PasteEvent{Text: "line1\nline2\r\nline3"})
	m = next
	if m.Value() != "line1line2line3" {
		t.Errorf("Value() = %q, want %q (newlines should be dropped in a single-line field)", m.Value(), "line1line2line3")
	}
}

func TestPasteRespectsCharLimit(t *testing.T) {
	m := focused()
	m.CharLimit = 3
	next, _ := m.Update(tui.PasteEvent{Text: "hello"})
	m = next
	if m.Value() != "hel" {
		t.Errorf("Value() = %q, want %q", m.Value(), "hel")
	}
}

func TestPasteIgnoredWhenUnfocused(t *testing.T) {
	m := New()
	next, _ := m.Update(tui.PasteEvent{Text: "hi"})
	if next.Value() != "" {
		t.Errorf("unfocused paste mutated value: %q", next.Value())
	}
}

func TestBlinkTogglesAndReschedulesWhileFocused(t *testing.T) {
	m := focused()
	if !m.cursorVisible {
		t.Fatal("Focus() should leave the cursor visible")
	}

	next, cmd := m.Update(blinkMsg{id: m.blinkID})
	m = next
	if m.cursorVisible {
		t.Error("cursorVisible should toggle to false on the first blink")
	}
	if cmd == nil {
		t.Fatal("blink should reschedule itself with a non-nil Cmd while focused")
	}

	// Running the returned Cmd should itself produce another blinkMsg.
	if _, ok := tui.RunCmd(context.Background(), cmd).(blinkMsg); !ok {
		t.Error("blink Cmd did not produce a blinkMsg")
	}
}

func TestBlinkIsNoOpAfterBlur(t *testing.T) {
	m := focused()
	m.Blur()
	next, cmd := m.Update(blinkMsg{})
	if next.cursorVisible {
		t.Error("blinkMsg after Blur should not make the cursor visible")
	}
	if cmd != nil {
		t.Error("blinkMsg after Blur should not reschedule (cmd should be nil), or the blink loop never stops")
	}
}

func TestViewCursorInMiddle(t *testing.T) {
	m := focused()
	m.SetValue("hi")
	m.SetCursor(1)
	got := m.View()
	want := "h" + ansi.NewStyle().Reverse().Render("i")
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestViewCursorAtEnd(t *testing.T) {
	m := focused()
	m.SetValue("hi")
	got := m.View()
	want := "hi" + ansi.NewStyle().Reverse().Render(" ")
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestViewNoCursorWhenUnfocused(t *testing.T) {
	m := New()
	m.SetValue("hi")
	got := m.View()
	if got != "hi" {
		t.Errorf("View() = %q, want %q (no cursor styling when unfocused)", got, "hi")
	}
}

func TestViewPlaceholder(t *testing.T) {
	m := focused()
	m.Placeholder = "type here"
	got := m.View()
	want := ansi.NewStyle().Reverse().Render(" ") + ansi.NewStyle().Faint().Render("type here")
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestViewPrompt(t *testing.T) {
	m := focused()
	m.Prompt = "> "
	m.SetValue("hi")
	got := m.View()
	want := "> hi" + ansi.NewStyle().Reverse().Render(" ")
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestVisibleWindow(t *testing.T) {
	tests := []struct {
		name               string
		total, pos, w      int
		wantStart, wantEnd int
	}{
		{"cursor at start", 20, 0, 5, 0, 5},
		{"cursor at end", 20, 20, 5, 15, 20},
		{"cursor centered mid-string", 20, 10, 5, 8, 13},
		{"width equals total is a no-op window", 5, 3, 5, 0, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleWindow(tt.total, tt.pos, tt.w)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("visibleWindow(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.total, tt.pos, tt.w, start, end, tt.wantStart, tt.wantEnd)
			}
			if end-start != tt.w {
				t.Errorf("window width = %d, want %d", end-start, tt.w)
			}
			if tt.pos < tt.total && (tt.pos < start || tt.pos >= end) {
				t.Errorf("cursor pos %d not in window [%d, %d)", tt.pos, start, end)
			}
		})
	}
}

func TestViewScrollsToKeepCursorVisible(t *testing.T) {
	m := focused()
	m.Width = 5
	m.SetValue("abcdefghij") // 10 chars
	m.SetCursor(9)           // near the end

	got := m.View()
	// Window should be the last 5 chars: "fghij", cursor on the final 'j'.
	want := "fghi" + ansi.NewStyle().Reverse().Render("j")
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestBlinkSingleLoopAfterFocusBlurFocus(t *testing.T) {
	m := New()
	c1 := m.Focus()
	m.Blur()
	c2 := m.Focus()
	if c1 == nil || c2 == nil {
		t.Fatal("Focus must schedule a tick")
	}
	m1 := tui.RunCmd(context.Background(), c1) // tick from the first, now stale, loop
	m2 := tui.RunCmd(context.Background(), c2)
	got, cmd := m.Update(m1)
	if cmd != nil || !got.cursorVisible {
		t.Fatalf("stale tick: cmd %v visible %v; want dropped", cmd != nil, got.cursorVisible)
	}
	got, cmd = m.Update(m2)
	if cmd == nil || got.cursorVisible {
		t.Fatal("live tick must toggle and reschedule")
	}
	if next, ok := tui.RunCmd(context.Background(), cmd).(blinkMsg); !ok || next != m2 {
		t.Error("rescheduled tick must carry the live id")
	}
}

// Criterion: a paste with escapes/C0 controls is stored without them.
func TestPasteSanitised(t *testing.T) {
	m := New()
	m.Focus()
	m, _ = m.Update(tui.PasteEvent{Text: "a\x1b]52;c;ZXZpbA==\x07b\x1b[31mc\x00\x08\td\n"})
	if got, want := m.Value(), "abc d"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if strings.Contains(m.View(), "\x1b]") {
		t.Errorf("View() renders OSC: %q", m.View())
	}
}
