package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
)

// drive passes each message to Update and, as the Program would, feeds the
// message any returned Cmd produces back in. It reports whether a Cmd quit.
func drive(t *testing.T, m model, msgs ...tui.Msg) (model, bool) {
	t.Helper()
	quit := false
	for len(msgs) > 0 {
		msg := msgs[0]
		msgs = msgs[1:]
		next, cmd := m.Update(msg)
		m = next.(model)
		if cmd == nil {
			continue
		}
		switch out := cmd().(type) {
		case nil:
		case tui.QuitMsg:
			quit = true
		default:
			msgs = append([]tui.Msg{out}, msgs...)
		}
	}
	return m, quit
}

func sized(t *testing.T) model {
	t.Helper()
	m, _ := drive(t, initialModel(), tui.ResizeMsg{Width: 80, Height: 24})
	return m
}

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func char(r rune) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r} }

func TestMenuBarOpensMovesAndSelects(t *testing.T) {
	m, _ := drive(t, sized(t), key(tui.KeyF10))
	if !m.bar.Open() || m.bar.Active() != 0 {
		t.Fatalf("F10: open=%v active=%d, want the first menu open", m.bar.Open(), m.bar.Active())
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "ctrl+n") {
		t.Errorf("the File dropdown is not drawn over the screen:\n%s", out)
	}
	m, quit := drive(t, m, key(tui.KeyRight), key(tui.KeyDown), key(tui.KeyEnter))
	if quit || m.bar.Open() {
		t.Fatalf("after selecting: quit=%v open=%v", quit, m.bar.Open())
	}
	if m.last != "menu bar: View > Wide" {
		t.Fatalf("last = %q", m.last)
	}
}

func TestAltLetterOpensAMenuAndQuitQuits(t *testing.T) {
	m, _ := drive(t, sized(t), tui.Key{Type: tui.KeyRunes, Text: "f", Code: 'f', Mod: input.ModAlt})
	if !m.bar.Open() || m.bar.Active() != 0 {
		t.Fatalf("alt+f: open=%v active=%d", m.bar.Open(), m.bar.Active())
	}
	// While the dropdown is open, "q" is the Quit item's accelerator, not the
	// program's own quit key.
	if _, quit := drive(t, m, char('q')); !quit {
		t.Fatal("File > Quit did not quit")
	}
}

func TestNestedMenuDrillsInAndBack(t *testing.T) {
	m, _ := drive(t, sized(t), key(tui.KeyEnter))
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "alpha") {
		t.Fatalf("Enter on Projects did not open its children:\n%s", out)
	}
	m, _ = drive(t, m, key(tui.KeyDown), key(tui.KeyEnter))
	if m.last != "navigation: project beta" {
		t.Fatalf("last = %q", m.last)
	}
	m, _ = drive(t, m, key(tui.KeyEsc))
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "Environments") {
		t.Fatalf("Esc did not go back to the top level:\n%s", out)
	}
}

func TestContextMenuTakesKeysWhileOpen(t *testing.T) {
	m, _ := drive(t, sized(t), char('a'))
	if !m.ctx.Open() {
		t.Fatal(`"a" did not open the actions menu`)
	}
	// Down moves in the context menu, not in the nested menu under it.
	m, _ = drive(t, m, key(tui.KeyDown), key(tui.KeyEnter))
	if m.ctx.Open() || m.last != "actions: Rename" {
		t.Fatalf("open=%v last=%q, want closed and Rename", m.ctx.Open(), m.last)
	}
	// Closed again, "q" is the program's quit key.
	if _, quit := drive(t, m, char('q')); !quit {
		t.Fatal("q did not quit with no menu open")
	}
}

func TestDisabledActionCannotBeChosen(t *testing.T) {
	m, _ := drive(t, sized(t), char('a'), key(tui.KeyEnd), key(tui.KeyEnter))
	if m.last == "actions: Delete" {
		t.Fatal("the disabled Delete action was chosen")
	}
}

func TestRightClickOpensTheContextMenuAtThePointer(t *testing.T) {
	m, _ := drive(t, sized(t), tui.MouseEvent{X: 30, Y: 8, Button: tui.MouseButtonRight, Action: tui.MouseActionPress})
	if !m.ctx.Open() || m.ctx.AnchorX != 30 || m.ctx.AnchorY != 8 {
		t.Fatalf("open=%v anchor=(%d,%d), want open at (30,8)", m.ctx.Open(), m.ctx.AnchorX, m.ctx.AnchorY)
	}
}

func TestClickOnABarTitleOpensIt(t *testing.T) {
	x := strings.Index(ansi.StripANSI(sized(t).bar.View()), "View")
	if x < 0 {
		t.Fatal("the bar does not show View")
	}
	m, _ := drive(t, sized(t), tui.MouseEvent{X: x, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if !m.bar.Open() || m.bar.Active() != 1 {
		t.Fatalf("open=%v active=%d, want the View menu open", m.bar.Open(), m.bar.Active())
	}
}

// An open overlay never makes the frame larger than the terminal.
func TestOverlaysStayInsideTheTerminal(t *testing.T) {
	m, _ := drive(t, initialModel(), tui.ResizeMsg{Width: 40, Height: 10}, key(tui.KeyF10))
	for _, opened := range []model{m, func() model { c, _ := drive(t, sized40(t), char('a')); return c }()} {
		lines := strings.Split(ansi.StripANSI(opened.View()), "\n")
		if len(lines) > 10 {
			t.Errorf("frame is %d rows in a 10-row terminal", len(lines))
		}
		for _, l := range lines {
			if w := ansi.Width(l); w > 40 {
				t.Errorf("a line is %d wide in a 40-column terminal: %q", w, l)
			}
		}
	}
}

func sized40(t *testing.T) model {
	t.Helper()
	m, _ := drive(t, initialModel(), tui.ResizeMsg{Width: 40, Height: 10})
	return m
}
