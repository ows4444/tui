package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// send passes each message to Update and returns the model that results. It
// reports whether any returned Cmd quit.
func send(t *testing.T, m model, msgs ...tui.Msg) (model, bool) {
	t.Helper()
	quit := false
	for _, msg := range msgs {
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd != nil {
			if _, ok := cmd().(tui.QuitMsg); ok {
				quit = true
			}
		}
	}
	return m, quit
}

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func char(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r} }

func screen(m model) string { return ansi.StripANSI(m.View()) }

func TestOnlyTheVisibleRowsAreRendered(t *testing.T) {
	m := initialModel()
	calls := 0
	m.list.RenderItem = func(i int) string { calls++; return row(i) }
	_ = m.list.View()
	if calls == 0 || calls > m.list.Height+2*m.list.Overscan {
		t.Fatalf("drawing %d rows of %d entries called RenderItem %d times", m.list.Height, entryCount, calls)
	}
}

func TestMovingSelectsAnEntryAndTheDetailFollows(t *testing.T) {
	m, _ := send(t, initialModel(), key(tui.KeyDown), key(tui.KeyDown))
	if m.list.Cursor() != 2 {
		t.Fatalf("cursor = %d, want 2", m.list.Cursor())
	}
	out := screen(m)
	if !strings.Contains(out, "Entry 3 of 10000") || !strings.Contains(out, "> "+row(2)[:14]) {
		t.Fatalf("the detail pane or the row marker does not follow the cursor:\n%s", out)
	}
}

func TestScrollbarFollowsTheList(t *testing.T) {
	m := initialModel()
	if m.bar.Offset != 0 || m.bar.Total != entryCount || m.bar.Visible != m.list.Height {
		t.Fatalf("at the top: bar = %+v", m.bar)
	}
	m, _ = send(t, m, key(tui.KeyEnd))
	if m.bar.Offset != m.list.Offset() || m.bar.Offset != m.bar.MaxOffset() {
		t.Fatalf("at the end: bar offset %d, list offset %d, max %d", m.bar.Offset, m.list.Offset(), m.bar.MaxOffset())
	}
}

func TestBracketsMoveTheDividerWithinItsLimits(t *testing.T) {
	m := initialModel()
	before, _ := m.split.Sizes()
	m, _ = send(t, m, char(']'), char(']'))
	after, _ := m.split.Sizes()
	if after != before+2 {
		t.Fatalf("list pane went from %d to %d, want %d", before, after, before+2)
	}
	for range 200 {
		m, _ = send(t, m, char('['))
	}
	first, second := m.split.Sizes()
	if first != m.split.Min1 || second < m.split.Min2 {
		t.Fatalf("after shrinking all the way: panes %d and %d, want the first at its minimum %d", first, second, m.split.Min1)
	}
}

func TestPopoverOpensBesideTheSelectedRowAndTakesTheKeys(t *testing.T) {
	m, _ := send(t, initialModel(), key(tui.KeyDown), char('i'))
	if !m.info.Open() {
		t.Fatal(`"i" did not open the popover`)
	}
	if out := screen(m); !strings.Contains(out, "INFO from worker") {
		t.Fatalf("the popover does not describe the selected entry:\n%s", out)
	}
	// While it is open the list does not move, and "q" does not quit.
	m, quit := send(t, m, key(tui.KeyDown), char('q'))
	if quit || m.list.Cursor() != 1 {
		t.Fatalf("with the popover open: quit=%v cursor=%d", quit, m.list.Cursor())
	}
	m, _ = send(t, m, key(tui.KeyEsc))
	if m.info.Open() {
		t.Fatal("esc did not close the popover")
	}
}

func TestDrawerOpensOnTheRightEdgeAndCloses(t *testing.T) {
	m, _ := send(t, initialModel(), char('l'))
	if !m.key.Open() {
		t.Fatal(`"l" did not open the drawer`)
	}
	for _, line := range strings.Split(screen(m), "\n") {
		if strings.Contains(line, "routine work") && !strings.HasSuffix(strings.TrimRight(line, " "), "│") {
			t.Fatalf("the drawer is not on the right edge: %q", line)
		}
	}
	m, _ = send(t, m, key(tui.KeyEnter))
	if m.key.Open() {
		t.Fatal("enter did not close the drawer")
	}
}

func TestQuit(t *testing.T) {
	if _, quit := send(t, initialModel(), char('q')); !quit {
		t.Fatal("q did not quit")
	}
	if _, quit := send(t, initialModel(), key(tui.KeyCtrlC)); !quit {
		t.Fatal("ctrl+c did not quit")
	}
}

// With either overlay open, the frame still fits a 40x10 terminal.
func TestOverlaysFitASmallTerminal(t *testing.T) {
	for _, open := range []rune{'i', 'l'} {
		m, _ := send(t, initialModel(), tui.ResizeMsg{Width: 40, Height: 10}, char(open))
		lines := strings.Split(screen(m), "\n")
		if len(lines) > 10 {
			t.Errorf("%q open: %d rows in a 10-row terminal", open, len(lines))
		}
		for _, l := range lines {
			if w := ansi.Width(l); w > 40 {
				t.Errorf("%q open: a line is %d wide in a 40-column terminal: %q", open, w, l)
			}
		}
	}
}
