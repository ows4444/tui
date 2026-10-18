package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
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

func ready() model { return send(newModel(), tui.ResizeMsg{Width: 120, Height: 36}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
}

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return press(m, r.X+min(1, r.W-1), r.Y)
}

// boxRect is a stage box's rectangle inside the stage.
func boxRect(t testing.TB, m model, name string) layout.Rect {
	t.Helper()
	for _, p := range m.boxes(theme.DarkTheme(), m.frame(m.vals["width"], m.vals["height"])) {
		if p.Name == name {
			return p.Rect
		}
	}
	t.Fatalf("no box %q", name)
	return layout.Rect{}
}

func setting(t testing.TB, m model, key string) int {
	t.Helper()
	for i, p := range m.params() {
		if p.key == key {
			return i
		}
	}
	t.Fatalf("no setting %q", key)
	return 0
}

func TestFirstScreenShowsFlexSettingsStageAndRectangles(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"layoutlab", "▸1 Flex", "2 Grid", "3 Wrap", "4 Anchor", "direction", "justify", "basis", "┌ 60x12", "1 10x12", "2 36x12", "3 12x12", "Rectangles", "c1     x=11"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestEverySettingOfEveryModeMovesWithinItsRange(t *testing.T) {
	m := ready()
	for md := mFlex; md < numModes; md++ {
		m = m.setMode(md)
		for i := range m.params() {
			p := m.params()[i]
			lo := m.put(i, p.lo-100)
			hi := m.put(i, p.hi+100)
			if lo.vals[p.key] != p.lo || hi.vals[p.key] != p.hi {
				t.Errorf("%s %s: %d..%d gave %d and %d", modeNames[md], p.key, p.lo, p.hi, lo.vals[p.key], hi.vals[p.key])
			}
			if out := text(hi); !strings.Contains(out, "layoutlab") {
				t.Errorf("%s %s at its top: no screen", modeNames[md], p.key)
			}
			if out := text(lo); strings.TrimSpace(out) == "" {
				t.Errorf("%s %s at its bottom: blank", modeNames[md], p.key)
			}
		}
	}
}

func TestFlexGrowSharesTheLeftoverWidth(t *testing.T) {
	m := ready()
	if w := boxRect(t, m, "c1").W; w != 36 {
		t.Fatalf("box 2 is %d wide", w)
	}
	m = m.put(setting(t, m, "width"), 40)
	if w := boxRect(t, m, "c1").W; w != 16 {
		t.Fatalf("box 2 is %d wide at 40", w)
	}
	m = m.put(setting(t, m, "gap"), 0)
	if w := boxRect(t, m, "c1").W; w != 18 {
		t.Fatalf("box 2 is %d wide with no gap", w)
	}
}

func TestFlexDirectionColumnAndJustifyMoveTheBoxes(t *testing.T) {
	m := ready()
	m = m.put(setting(t, m, "dir"), 1)
	if r := boxRect(t, m, "c0"); r.W != 60 || r.Y != 0 {
		t.Fatalf("column: %+v", r)
	}
	m = ready()
	for c := 0; c < 3; c++ {
		m.vals["grow."+string(rune('0'+c))] = 0
	}
	m = m.put(setting(t, m, "justify"), 1) // End
	if r := boxRect(t, m, "c2"); r.X+r.W != 60 {
		t.Fatalf("End: last box ends at %d", r.X+r.W)
	}
	m = m.put(setting(t, m, "justify"), 2) // Center
	if r := boxRect(t, m, "c0"); r.X <= 0 {
		t.Fatalf("Center: first box at %d", r.X)
	}
}

func TestPerChildSettingsFollowTheSelectedChild(t *testing.T) {
	m := ready()
	m = m.put(setting(t, m, "child"), 2)
	m = m.put(setting(t, m, "basis.2"), 20)
	if w := boxRect(t, m, "c2").W; w != 20 {
		t.Fatalf("box 3 is %d wide", w)
	}
	if boxRect(t, m, "c0").W != 10 {
		t.Fatal("box 1 changed")
	}
	if !strings.Contains(text(m), "╔") || !strings.Contains(text(m), "3 20x12") {
		t.Fatalf("selected box not marked:\n%s", text(m))
	}
	m = m.put(setting(t, m, "count"), 2)
	if m.vals["child"] != 1 {
		t.Fatalf("child %d after removing the last", m.vals["child"])
	}
}

func TestMinMaxAndCrossAlign(t *testing.T) {
	m := ready()
	m = m.put(setting(t, m, "max."+"0"), 6)
	m = m.put(setting(t, m, "basis.0"), 30)
	if w := boxRect(t, m, "c0").W; w != 6 {
		t.Fatalf("max 6: %d", w)
	}
	m = ready()
	m = m.put(setting(t, m, "cross.0"), 2) // Center
	if r := boxRect(t, m, "c0"); r.H != 3 || r.Y != 4 {
		t.Fatalf("centred: %+v", r)
	}
}

func TestGridTracksSpansAndGaps(t *testing.T) {
	m := ready().setMode(mGrid)
	if r := boxRect(t, m, "c0"); r.W != 39 || r.H != 3 {
		t.Fatalf("spanning cell %+v", r)
	}
	m = m.put(setting(t, m, "spanc"), 1)
	if r := boxRect(t, m, "c0"); r.W != 19 {
		t.Fatalf("one column: %+v", r)
	}
	m = m.put(setting(t, m, "size.0"), 12)
	if r := boxRect(t, m, "c0"); r.W != 12 {
		t.Fatalf("fixed first column: %+v", r)
	}
	m = m.put(setting(t, m, "rowgap"), 2)
	if r := boxRect(t, m, "c3"); r.Y != 5 {
		t.Fatalf("row gap: %+v", r)
	}
	m = m.put(setting(t, m, "cols"), 2)
	m = m.put(setting(t, m, "cells"), 4)
	if len(m.boxes(theme.DarkTheme(), m.frame(60, 14))) != 4 {
		t.Fatal("cell count")
	}
	if m.vals["col"] != 0 {
		t.Fatalf("col %d", m.vals["col"])
	}
}

func TestWrapStartsANewRowAndHidesWhatDoesNotFit(t *testing.T) {
	m := ready().setMode(mWrap)
	r0, r3 := boxRect(t, m, "c0"), boxRect(t, m, "c3")
	if r0.Y != 0 || r3.Y != 4 || r3.X != 0 {
		t.Fatalf("c0 %+v c3 %+v", r0, r3)
	}
	m = m.put(setting(t, m, "width"), 120)
	if r := boxRect(t, m, "c9"); r.Y != 0 {
		t.Fatalf("one row at 120 wide: %+v", r)
	}
	m = m.put(setting(t, m, "width"), 40)
	if r := boxRect(t, m, "c9"); r.H != 0 || !strings.Contains(text(m), "hidden") {
		t.Fatalf("the last chip should be pushed out of view: %+v", r)
	}
}

func TestAnchorPinsTheBadgeToTheChosenCorner(t *testing.T) {
	m := ready().setMode(mAnchor)
	if r := boxRect(t, m, "badge"); r.X+r.W != 39 || r.Y != 0 {
		t.Fatalf("top right with dx 1: %+v", r)
	}
	m = m.put(setting(t, m, "corner"), 2)
	if r := boxRect(t, m, "badge"); r.X != 1 || r.Y+r.H != 10 {
		t.Fatalf("bottom left: %+v", r)
	}
	m = m.put(setting(t, m, "corner"), 4)
	if r := boxRect(t, m, "badge"); r.X != 16 || r.Y != 3 {
		t.Fatalf("centre: %+v", r)
	}
	m = m.put(setting(t, m, "dy"), 2)
	if r := boxRect(t, m, "badge"); r.Y != 5 {
		t.Fatalf("dy moves the centred badge down: %+v", r)
	}
}

func TestModesKeepTheirOwnSettings(t *testing.T) {
	m := ready()
	m = m.put(setting(t, m, "gap"), 3)
	m = runes(m, "2")
	m = m.put(setting(t, m, "colgap"), 2)
	m = key(m, tui.KeyTab)
	m = runes(m, "1")
	if m.vals["gap"] != 3 || m.mode != mFlex {
		t.Fatalf("flex gap %d", m.vals["gap"])
	}
	m = runes(runes(m, "2"), "r")
	if m.vals["colgap"] != 1 || !strings.Contains(m.msg, "put back") {
		t.Fatalf("reset: colgap %d", m.vals["colgap"])
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: 1})
	if m.mode != mFlex {
		t.Fatalf("Shift+Tab: mode %v", m.mode)
	}
}

func TestKeyboardPicksAndChangesSettings(t *testing.T) {
	m := key(ready(), tui.KeyDown)
	m = key(m, tui.KeyRight)
	if m.sel != 1 || m.vals["gap"] != 2 {
		t.Fatalf("sel %d gap %d", m.sel, m.vals["gap"])
	}
	m = key(m, tui.KeyPgUp)
	if m.vals["gap"] != 4 {
		t.Fatalf("gap %d", m.vals["gap"])
	}
	m = key(m, tui.KeyHome)
	if m.vals["gap"] != 1 {
		t.Fatalf("Home: gap %d", m.vals["gap"])
	}
	m = runes(runes(m, "j"), "l")
	if m.sel != 2 || m.vals["justify"] != 1 || !strings.Contains(text(m), "End") {
		t.Fatalf("sel %d justify %d", m.sel, m.vals["justify"])
	}
	if m = runes(runes(runes(m, "k"), "k"), "k"); m.sel != 0 {
		t.Fatalf("sel %d at the top", m.sel)
	}
}

func TestMouseSlidersArrowsWheelAndBoxes(t *testing.T) {
	m := ready()
	i := setting(t, m, "gap")
	r := rect(t, m, "bar-1")
	m = press(m, r.X+r.W-1, r.Y)
	if m.vals["gap"] != 4 || m.sel != i {
		t.Fatalf("gap %d", m.vals["gap"])
	}
	m = click(t, m, "dec-1")
	if m.vals["gap"] != 3 {
		t.Fatalf("◂: gap %d", m.vals["gap"])
	}
	m = click(t, m, "inc-1")
	if m.vals["gap"] != 4 {
		t.Fatalf("▸: gap %d", m.vals["gap"])
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.vals["gap"] != 3 {
		t.Fatalf("wheel: gap %d", m.vals["gap"])
	}
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion})
	if m.vals["gap"] != 0 {
		t.Fatalf("drag to the left end: gap %d", m.vals["gap"])
	}
	m = click(t, m, "mode-2")
	if m.mode != mWrap {
		t.Fatalf("mode %v", m.mode)
	}
	m = click(t, m.setMode(mFlex), "c2")
	if m.vals["child"] != 2 {
		t.Fatalf("child %d after clicking box 3", m.vals["child"])
	}
	m = click(t, m.setMode(mGrid), "c4")
	if m.vals["col"] != 1 {
		t.Fatalf("column %d after clicking cell 5", m.vals["col"])
	}
}

func TestCodeViewMatchesTheSettings(t *testing.T) {
	m := runes(ready(), "c")
	out := text(m)
	for _, want := range []string{"Code", "layout.RowJustify(1, layout.JustifyStart,", "layout.FlexChild{Node: tile1, Basis: 10, Shrink: 1},", "Grow: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("code lacks %q:\n%s", want, out)
		}
	}
	m = click(t, m, "view")
	if m.code || !strings.Contains(text(m), "Rectangles") {
		t.Fatal("clicking the header did not switch back")
	}
	g := code(mGrid, defaults(mGrid))
	if !strings.Contains(g, "layout.GridNodeGaps([]layout.Track{") || !strings.Contains(g, "layout.Span(cell1, 2, 1)") {
		t.Fatalf("\n%s", g)
	}
	if !strings.Contains(code(mAnchor, defaults(mAnchor)), "layout.Anchor(layout.TopRight, 1, 0, badge)") {
		t.Fatal("anchor code")
	}
	if !strings.Contains(code(mWrap, defaults(mWrap)), "layout.Wrap(1, 1, chip1 ... chip10)") {
		t.Fatal("wrap code")
	}
}

func TestStageIsCutToTheWindow(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 100, Height: 30})
	m = m.put(setting(t, m, "width"), 120)
	if !strings.Contains(text(m), "cut from 120x12") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestNarrowWindowsShowJustTheSettings(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 60, Height: 20})
	out := text(m)
	if strings.Contains(out, "Rectangles") || !strings.Contains(out, "direction") {
		t.Fatalf("\n%s", out)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "rectangles or code") {
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
