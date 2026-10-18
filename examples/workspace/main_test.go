package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func send(m model, msg tui.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(newModel(), tui.ResizeMsg{Width: 100, Height: 30}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func mouse(m model, x, y int, b tui.MouseButton, a tui.MouseAction, clicks int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: b, Action: a, Clicks: clicks})
}

func press(m model, x, y int) model {
	return mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionPress, 1)
}

func TestFirstScreenHasThreePanesAndDividers(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"workspace", "▸Files", "Outline", "▸main.go", "notes.md", "▸Output", "Problems", "package main", "build step 1", "│", "─", "Editor focused"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	g := m.geo()
	if g.pane[0].W != defaultL || g.pane[2].H != defaultB || g.pane[1].X != defaultL+1 {
		t.Fatalf("geometry %+v", g)
	}
	sum := g.pane[1].H + 1 + g.pane[2].H
	if sum != 27 {
		t.Fatalf("right column is %d rows, want 27", sum)
	}
}

func TestTabMovesFocusAndSkipsHiddenPanes(t *testing.T) {
	m := key(ready(), tui.KeyTab)
	if m.focus != 2 {
		t.Fatalf("focus %d", m.focus)
	}
	m = key(m, tui.KeyTab)
	if m.focus != 0 {
		t.Fatalf("focus %d", m.focus)
	}
	if m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift}); m.focus != 2 {
		t.Fatalf("Shift+Tab: focus %d", m.focus)
	}
	m = runes(runes(m, "c"), "")
	if m.focus != 1 || !m.collapsedB {
		t.Fatalf("collapsing the focused panel: focus %d collapsed %v", m.focus, m.collapsedB)
	}
	if m = key(m, tui.KeyTab); m.focus != 0 {
		t.Fatalf("Tab went to a hidden pane: %d", m.focus)
	}
}

func TestTabKeysAndClicks(t *testing.T) {
	m := runes(ready(), ".")
	if m.tab[1] != 1 || !strings.Contains(text(m), "▸notes.md") || !strings.Contains(text(m), "# Notes") {
		t.Fatalf("tab %d", m.tab[1])
	}
	m = runes(runes(runes(m, "."), "."), ",")
	if m.tab[1] != 2 {
		t.Fatalf("tab %d after wrapping", m.tab[1])
	}
	r := rect(t, m, "tab-2-1")
	m = press(m, r.X+1, r.Y)
	if m.focus != 2 || m.tab[2] != 1 || !strings.Contains(text(m), "3 problems") {
		t.Fatalf("focus %d tab %d", m.focus, m.tab[2])
	}
}

func TestClickFocusesAPane(t *testing.T) {
	m := ready()
	r := rect(t, m, "pane-0")
	if m = press(m, r.X+2, r.Y+4); m.focus != 0 {
		t.Fatalf("focus %d", m.focus)
	}
}

func TestDraggingTheVerticalDividerResizesTheExplorer(t *testing.T) {
	m := ready()
	d := rect(t, m, "vdiv")
	m = press(m, d.X, d.Y+3)
	m = mouse(m, 40, 5, tui.MouseButtonLeft, tui.MouseActionMotion, 0)
	if !strings.Contains(text(m), "Explorer 40 columns") || !strings.Contains(text(m), "┃") {
		t.Fatalf("no drag feedback:\n%s", text(m))
	}
	m = mouse(m, 40, 5, tui.MouseButtonLeft, tui.MouseActionRelease, 0)
	if m.geo().pane[0].W != 40 || m.drag != noDrag {
		t.Fatalf("width %d", m.geo().pane[0].W)
	}
	m = mouse(m, 3, 5, tui.MouseButtonLeft, tui.MouseActionMotion, 0)
	if m.geo().pane[0].W != 40 {
		t.Fatal("moving without a button resized")
	}
}

func TestDividerStopsAtTheMinimums(t *testing.T) {
	m := ready()
	d := rect(t, m, "vdiv")
	m = press(m, d.X, d.Y)
	m = mouse(m, 1, 5, tui.MouseButtonLeft, tui.MouseActionMotion, 0)
	if m.geo().pane[0].W != minLeft {
		t.Fatalf("left %d", m.geo().pane[0].W)
	}
	m = mouse(m, 99, 5, tui.MouseButtonLeft, tui.MouseActionMotion, 0)
	if g := m.geo(); g.pane[1].W < minRight {
		t.Fatalf("editor %d wide", g.pane[1].W)
	}
}

func TestDraggingTheHorizontalDividerResizesThePanel(t *testing.T) {
	m := ready()
	d := rect(t, m, "hdiv")
	m = press(m, d.X+5, d.Y)
	m = mouse(m, d.X+5, 15, tui.MouseButtonLeft, tui.MouseActionMotion, 0)
	m = mouse(m, d.X+5, 15, tui.MouseButtonLeft, tui.MouseActionRelease, 0)
	if g := m.geo(); g.pane[2].H != 12 || g.pane[1].H != 14 {
		t.Fatalf("panel %d, editor %d", g.pane[2].H, g.pane[1].H)
	}
}

func TestDoubleClickCollapsesAndAClickBringsItBack(t *testing.T) {
	m := ready()
	d := rect(t, m, "vdiv")
	m = mouse(m, d.X, d.Y+2, tui.MouseButtonLeft, tui.MouseActionPress, 2)
	if !m.collapsedL || m.geo().shown(0) || m.geo().pane[1].X != 1 || !strings.Contains(text(m), "▸") {
		t.Fatalf("not collapsed:\n%s", text(m))
	}
	d = rect(t, m, "vdiv")
	m = press(m, d.X, d.Y)
	m = mouse(m, 30, 5, tui.MouseButtonLeft, tui.MouseActionMotion, 0)
	m = mouse(m, 30, 5, tui.MouseButtonLeft, tui.MouseActionRelease, 0)
	if m.collapsedL || m.geo().pane[0].W != 30 {
		t.Fatalf("dragging the strip should reopen it: collapsed %v", m.collapsedL)
	}
	h := rect(t, m, "hdiv")
	m = mouse(m, h.X+3, h.Y, tui.MouseButtonLeft, tui.MouseActionPress, 2)
	if !m.collapsedB || !strings.Contains(text(m), "▴ Panel") {
		t.Fatalf("panel not collapsed:\n%s", text(m))
	}
	m = mouse(m, h.X+3, rect(t, m, "hdiv").Y, tui.MouseButtonLeft, tui.MouseActionPress, 2)
	if m.collapsedB {
		t.Fatal("double-click did not restore")
	}
}

func TestEditorCannotCollapse(t *testing.T) {
	m := runes(ready(), "c")
	if m.collapsedL || m.collapsedB || !strings.Contains(m.msg, "editor stays") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestKeyboardResizesAndResets(t *testing.T) {
	m := runes(runes(ready(), "]"), "]")
	if m.geo().pane[0].W != defaultL+4 {
		t.Fatalf("width %d", m.geo().pane[0].W)
	}
	m = runes(m, "[")
	if m.geo().pane[0].W != defaultL+2 {
		t.Fatalf("width %d", m.geo().pane[0].W)
	}
	m = runes(runes(m, "="), "=")
	if m.geo().pane[2].H != defaultB+2 {
		t.Fatalf("panel %d", m.geo().pane[2].H)
	}
	m = runes(m, "-")
	if m.geo().pane[2].H != defaultB+1 {
		t.Fatalf("panel %d", m.geo().pane[2].H)
	}
	if m = runes(m, "0"); m.geo().pane[0].W != defaultL || m.geo().pane[2].H != defaultB || m.msg != "Layout reset" {
		t.Fatal("not reset")
	}
}

func TestZoomShowsOnePane(t *testing.T) {
	m := runes(ready(), "z")
	g := m.geo()
	if !g.shown(1) || g.shown(0) || g.shown(2) || g.pane[1].W != 100 {
		t.Fatalf("%+v", g)
	}
	if out := text(m); !strings.Contains(out, "zoomed") || strings.Contains(out, "Problems") {
		t.Fatalf("\n%s", out)
	}
	if m = key(m, tui.KeyEsc); m.zoom {
		t.Fatal("Esc did not leave zoom")
	}
}

func TestScrollingByKeyWheelAndPageKeys(t *testing.T) {
	m := runes(ready(), "j")
	if m.off[1][0] != 1 {
		t.Fatalf("offset %d", m.off[1][0])
	}
	m = key(m, tui.KeyEnd)
	if m.off[1][0] != m.maxOff(1) || m.maxOff(1) == 0 {
		t.Fatalf("End should reach the last line: offset %d max %d", m.off[1][0], m.maxOff(1))
	}
	m = key(runes(ready(), "."), tui.KeyUp) // notes are short too
	if m.off[1][1] != 0 {
		t.Fatal("scrolled above the top")
	}
	m = m.setFocus(2)
	m = key(m, tui.KeyPgDown)
	if m.off[2][0] != m.visible(2)-1 {
		t.Fatalf("PgDn moved %d", m.off[2][0])
	}
	r := rect(t, m, "pane-2")
	m = mouse(m, r.X+2, r.Y+3, tui.MouseButtonWheelDown, tui.MouseActionPress, 0)
	if m.off[2][0] != m.visible(2)-1+3 {
		t.Fatalf("wheel: offset %d", m.off[2][0])
	}
	m = key(m, tui.KeyEnd)
	if !strings.Contains(text(m), "done in 2.4s") {
		t.Fatalf("end of the output not reached:\n%s", text(m))
	}
	if m = key(m, tui.KeyHome); m.off[2][0] != 0 {
		t.Fatal("Home did not go to the top")
	}
}

func TestScrollOffsetsAreKeptPerTab(t *testing.T) {
	m := ready().setFocus(2)
	m = key(m, tui.KeyPgDown)
	off := m.off[2][0]
	m = runes(m, ".")
	if m.off[2][1] != 0 {
		t.Fatal("the new tab inherited the scroll")
	}
	if m = runes(m, ","); m.off[2][0] != off {
		t.Fatal("the first tab lost its place")
	}
}

func TestSmallWindowsHidePanesAndSaySo(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 40, Height: 10})
	g := m.geo()
	if g.shown(0) || g.shown(2) || !g.shown(1) {
		t.Fatalf("%+v", g)
	}
	if !strings.Contains(text(m), "some panes hidden") {
		t.Fatalf("\n%s", text(m))
	}
	m = send(ready().setFocus(2), tui.ResizeMsg{Width: 40, Height: 10})
	if m.focus != 1 {
		t.Fatalf("focus stayed on a hidden pane: %d", m.focus)
	}
	m = send(ready(), tui.ResizeMsg{Width: 300, Height: 80})
	if g := m.geo(); g.pane[1].W != 300-defaultL-1 || g.pane[2].H != defaultB {
		t.Fatalf("%+v", g)
	}
}

func TestResizeClampsScrollOffsets(t *testing.T) {
	m := ready().setFocus(2)
	m = key(m, tui.KeyEnd)
	m = send(m, tui.ResizeMsg{Width: 100, Height: 60})
	if m.off[2][0] > m.maxOff(2) {
		t.Fatalf("offset %d past max %d", m.off[2][0], m.maxOff(2))
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "double-click collapses") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyCtrlC}); c == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); c == nil {
		t.Fatal("q did not quit")
	}
}
