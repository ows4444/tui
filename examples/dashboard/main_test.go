package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(newModel(), tui.ResizeMsg{Width: 120, Height: 40}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func TestShowsEveryPanelInAGrid(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"dashboard", "CPU", "Memory", "Disks", "Network", "Processes", "Total", "Used", "/home", "▼", "PID", "● live", "every 1s"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if !m.grid() || len(m.rects()) != numPanels {
		t.Fatalf("%d panels in the grid", len(m.rects()))
	}
}

func TestGridRectsTileTheBody(t *testing.T) {
	m := ready()
	covered := map[[2]int]int{}
	for _, r := range m.rects() {
		if r.X < 0 || r.Y < 1 || r.X+r.W > 120 || r.Y+r.H > 1+m.bodyH() {
			t.Fatalf("rect %+v outside the body", r)
		}
		for y := r.Y; y < r.Y+r.H; y++ {
			for x := r.X; x < r.X+r.W; x++ {
				covered[[2]int{x, y}]++
			}
		}
	}
	for y := 1; y < 1+m.bodyH(); y++ {
		for x := 0; x < 120; x++ {
			if covered[[2]int{x, y}] != 1 {
				t.Fatalf("cell %d,%d covered %d times", x, y, covered[[2]int{x, y}])
			}
		}
	}
}

func TestSmallTerminalsShowOnePanelAtATime(t *testing.T) {
	for _, sz := range [][2]int{{83, 40}, {120, 18}, {40, 10}} {
		m := send(ready(), tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		if m.grid() {
			t.Errorf("%dx%d shows a grid", sz[0], sz[1])
		}
	}
	m := send(ready(), tui.ResizeMsg{Width: 60, Height: 20})
	out := text(m)
	if !strings.Contains(out, "CPU") || strings.Contains(out, "Memory") {
		t.Fatalf("one panel expected:\n%s", out)
	}
	m = key(m, tui.KeyTab)
	if !strings.Contains(text(m), "Memory") {
		t.Fatalf("Tab did not show the next panel:\n%s", text(m))
	}
}

func TestFocusKeys(t *testing.T) {
	m := ready()
	m = key(m, tui.KeyRight)
	m = key(m, tui.KeyDown)
	if m.focus != 3 {
		t.Fatalf("focus = %d after Right, Down", m.focus)
	}
	m = key(m, tui.KeyUp)
	m = key(m, tui.KeyLeft)
	if m.focus != 0 {
		t.Fatalf("focus = %d", m.focus)
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.focus != numPanels-1 {
		t.Fatalf("Shift+Tab wrapped to %d", m.focus)
	}
	m = key(m, tui.KeyTab)
	if m.focus != 0 {
		t.Fatalf("Tab wrapped to %d", m.focus)
	}
}

func TestZoom(t *testing.T) {
	m := key(key(ready(), tui.KeyTab), tui.KeyEnter)
	if !m.zoom || len(m.rects()) != 1 {
		t.Fatal("Enter did not zoom")
	}
	out := text(m)
	if !strings.Contains(out, "Memory (zoomed)") || strings.Contains(out, "Processes") {
		t.Fatalf("\n%s", out)
	}
	m = key(m, tui.KeyEsc)
	if m.zoom || !m.grid() {
		t.Fatal("Esc did not leave the zoom")
	}
	m = runes(runes(m, "z"), "z")
	if m.zoom {
		t.Fatal("z twice should end unzoomed")
	}
}

func TestZoomedPanelsShowMoreDetail(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 100, Height: 28}) // rows of 8: too short to list the cores
	grid := text(m)
	m.focus = pCPU
	m.zoom = true
	if !strings.Contains(text(m), "core 7") || strings.Contains(grid, "core 7") {
		t.Fatalf("zoomed CPU should list every core; grid:\n%s", grid)
	}
	m.focus = pProcs
	if n := strings.Count(text(m), "\n"); n < 10 {
		t.Fatal("zoomed processes should list more rows")
	}
	m.focus = pNet
	if !strings.Contains(text(m), "since start") {
		t.Fatal("zoomed network lacks the totals")
	}
}

func TestMovePanelsWithKeys(t *testing.T) {
	m := ready()
	m = runes(m, ">")
	if m.order[0] != pMem || m.order[1] != pCPU || m.focus != 1 {
		t.Fatalf("order = %v focus = %d", m.order, m.focus)
	}
	m = runes(m, "<")
	if m.order[0] != pCPU || m.focus != 0 {
		t.Fatalf("order = %v focus = %d", m.order, m.focus)
	}
	if m = runes(m, "<"); !strings.Contains(m.msg, "Already at the start") || m.order[0] != pCPU {
		t.Fatalf("msg = %q order = %v", m.msg, m.order)
	}
	m.focus = numPanels - 1
	if m = runes(m, ">"); !strings.Contains(m.msg, "Already at the end") {
		t.Fatalf("msg = %q", m.msg)
	}
}

func TestDragATitleOntoAnotherPanelSwaps(t *testing.T) {
	m := ready()
	rs := m.rects()
	from, to := rs[0], rs[3]
	m = send(m, tui.MouseEvent{X: from.X + 5, Y: from.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.dragFrom != 0 {
		t.Fatalf("dragFrom = %d", m.dragFrom)
	}
	m = send(m, tui.MouseEvent{X: to.X + 5, Y: to.Y + 2, Action: tui.MouseActionMotion})
	if m.dragOver != 3 || !strings.Contains(text(m), "Network ⇄") {
		t.Fatalf("dragOver = %d\n%s", m.dragOver, text(m))
	}
	m = send(m, tui.MouseEvent{X: to.X + 5, Y: to.Y + 2, Action: tui.MouseActionRelease})
	if m.order[0] != pNet || m.order[3] != pCPU || m.focus != 3 || m.dragFrom != -1 {
		t.Fatalf("order = %v focus = %d dragFrom = %d", m.order, m.focus, m.dragFrom)
	}
}

func TestDroppingOnItselfOrNowhereChangesNothing(t *testing.T) {
	m := ready()
	r := m.rects()[1]
	m = send(m, tui.MouseEvent{X: r.X + 3, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	m = send(m, tui.MouseEvent{X: r.X + 4, Y: r.Y + 1, Action: tui.MouseActionMotion})
	m = send(m, tui.MouseEvent{X: r.X + 4, Y: r.Y + 1, Action: tui.MouseActionRelease})
	if m.order[1] != pMem || m.focus != 1 {
		t.Fatalf("order = %v focus = %d", m.order, m.focus)
	}
	m = send(m, tui.MouseEvent{X: r.X + 3, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	m = send(m, tui.MouseEvent{X: 500, Y: 500, Action: tui.MouseActionMotion})
	m = send(m, tui.MouseEvent{X: 500, Y: 500, Action: tui.MouseActionRelease})
	if m.order[1] != pMem {
		t.Fatalf("a drop outside every panel moved something: %v", m.order)
	}
}

func TestClickBodyFocusesAndDoubleClickTitleZooms(t *testing.T) {
	m := ready()
	r := m.rects()[2]
	m = send(m, tui.MouseEvent{X: r.X + 4, Y: r.Y + 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.focus != 2 || m.dragFrom != -1 {
		t.Fatalf("focus = %d dragFrom = %d", m.focus, m.dragFrom)
	}
	m = send(m, tui.MouseEvent{X: r.X + 4, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 2})
	if !m.zoom {
		t.Fatal("a double click on the title did not zoom")
	}
	m = send(m, tui.MouseEvent{X: 4, Y: 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 2})
	if m.zoom {
		t.Fatal("a double click on the zoomed title did not unzoom")
	}
}

func TestTicksUpdateTheDataAndPauseStopsThem(t *testing.T) {
	m := ready()
	before := m.w.cur
	m, cmd := update(m, tickMsg{m.seq})
	if m.w.cur == before || m.ticks != 1 || cmd == nil {
		t.Fatal("a tick did not advance the machine and ask for the next")
	}
	m, _ = update(runes(m, " "), nil)
	if !m.paused || !strings.Contains(text(m), "‖ paused") {
		t.Fatal("space did not pause")
	}
	stale := tickMsg{m.seq - 1}
	paused := m.w.cur
	m, cmd = update(m, stale)
	if m.w.cur != paused || cmd != nil {
		t.Fatal("a tick from before the pause moved the machine")
	}
	m, cmd = update(m, tickMsg{m.seq})
	if m.w.cur != paused || cmd != nil {
		t.Fatal("a tick while paused moved the machine")
	}
	m, cmd = update(m, tui.Key{Type: tui.KeyRunes, Text: " "})
	if m.paused || cmd == nil {
		t.Fatal("space did not resume with a new timer")
	}
}

func TestClickPauseAndSpeed(t *testing.T) {
	m := ready()
	r := rect(t, m, "pause")
	m = send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if !m.paused {
		t.Fatal("clicking the state did not pause")
	}
	r = rect(t, m, "speed")
	m = send(m, tui.MouseEvent{X: r.X + 1, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.interval != 2 || !strings.Contains(text(m), "every 2s") {
		t.Fatalf("interval = %d", m.interval)
	}
}

func TestIntervalKeysStopAtTheEnds(t *testing.T) {
	m := ready()
	for i := 0; i < 5; i++ {
		m = runes(m, "-")
	}
	if m.interval != len(intervals)-1 || !strings.Contains(m.msg, "as slow as it goes") {
		t.Fatalf("interval = %d msg = %q", m.interval, m.msg)
	}
	for i := 0; i < 6; i++ {
		m = runes(m, "+")
	}
	if m.interval != 0 || !strings.Contains(m.msg, "as fast as it goes") {
		t.Fatalf("interval = %d msg = %q", m.interval, m.msg)
	}
	old := m.seq
	m, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "-"})
	if m.seq == old || cmd == nil {
		t.Fatal("changing the interval did not restart the timer")
	}
	if _, cmd = update(m, tickMsg{old}); cmd != nil {
		t.Fatal("the old timer still ran")
	}
}

func TestPanelContentAdaptsToItsSize(t *testing.T) {
	m := ready()
	th := themeForTest()
	for id := 0; id < numPanels; id++ {
		for _, sz := range [][2]int{{60, 20}, {30, 8}, {12, 3}, {5, 1}, {1, 1}, {100, 2}} {
			lines := m.lines(id, sz[0], sz[1], th)
			if len(lines) != sz[1] {
				t.Errorf("panel %d at %dx%d: %d lines", id, sz[0], sz[1], len(lines))
			}
			for _, l := range lines {
				if ansi.Width(l) > sz[0] {
					t.Errorf("panel %d at %dx%d: a line %d wide", id, sz[0], sz[1], ansi.Width(l))
				}
			}
		}
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "drag a title onto another panel swaps them") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(ready(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := runes(ready(), ">")
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {84, 21}, {83, 21}, {3, 3}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
