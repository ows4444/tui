package tabs

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
)

func altKey(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s, Mod: input.ModAlt} }

func TestAltAndALettersActivatesThatTab(t *testing.T) {
	m := New("&Files", "&Edit", "&Help")
	m, cmd := m.Update(altKey("h"))
	if m.Active() != 2 {
		t.Fatalf("active = %d, want 2", m.Active())
	}
	if c, ok := cmd().(ChangedMsg); !ok || c.Index != 2 {
		t.Errorf("message = %+v", cmd())
	}
	if _, cmd = m.Update(altKey("h")); cmd != nil {
		t.Error("choosing the active tab again sent a ChangedMsg")
	}
	m, cmd = m.Update(altKey("F"))
	if m.Active() != 0 || cmd == nil {
		t.Errorf("Alt+F: active %d, cmd %v", m.Active(), cmd != nil)
	}
	for _, k := range []tui.Key{altKey("x"), {Type: tui.KeyRunes, Text: "e"}, {Type: tui.KeyRunes, Text: "e", Mod: input.ModCtrl}} {
		if m2, cmd := m.Update(k); cmd != nil || m2.Active() != m.Active() {
			t.Errorf("%v changed the tab", k)
		}
	}
}

func TestMarkersAreNotDrawnAndUnderlineTheLetter(t *testing.T) {
	m := New("&Files", "E&dit")
	if got := ansi.StripANSI(m.View()); got != "[Files]  Edit " {
		t.Errorf("View = %q", got)
	}
	if !strings.Contains(m.View(), ansi.NewStyle().Foreground(m.themed().Muted).Underline().Render("d")) {
		t.Errorf("d is not underlined in %q", m.View())
	}
	if got, want := ansi.Width(m.View()), ansi.Width(New("Files", "Edit").View()); got != want {
		t.Errorf("width %d, want %d: markers must take no room", got, want)
	}
	if got := ansi.StripANSI(New("R&&D").View()); got != "[R&D]" {
		t.Errorf("&& = %q", got)
	}
}

func TestLabelsWithoutMarkersDrawAsBefore(t *testing.T) {
	m := New("Files", "Edit")
	active := m.themed().ResolvedStates().Selected.Bold()
	want := active.Render("[Files]") + " " + ansi.NewStyle().Foreground(m.themed().Muted).Render(" Edit ")
	if m.View() != want {
		t.Errorf("View = %q, want %q", m.View(), want)
	}
}

func TestMouseWidthsIgnoreTheMarker(t *testing.T) {
	m := New("&Files", "&Edit")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 40, H: 1}
	// "[Files]" is 7 wide, a gap at x=7, then " Edit " from x=8: x=8 is the second tab.
	m, cmd := m.Update(tui.MouseEvent{X: 8, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.Active() != 1 || cmd == nil {
		t.Errorf("click on the second tab: active %d", m.Active())
	}
}

func TestLinearizeSpeaksTheShortcut(t *testing.T) {
	got := New("&Files", "Plain").Linearize()
	want := "Files, tab 1 of 2, selected, shortcut Alt+F\nPlain, tab 2 of 2"
	if got != want {
		t.Errorf("Linearize = %q, want %q", got, want)
	}
}

func TestDrawCellsMatchesViewWithHotKeys(t *testing.T) {
	for name, m := range map[string]Model{
		"marked":    New("&Files", "E&dit", "Help"),
		"active":    func() Model { m := New("&Files", "E&dit"); m.active = 1; return m }(),
		"escaped":   New("R&&D", "&Team"),
		"unmarked":  New("Files", "Edit"),
		"wide":      New("日&本", "x"),
		"only&amps": New("a & b", "c"),
	} {
		t.Run(name, func(t *testing.T) { sameScreen(t, m, 50, 1) })
	}
}
