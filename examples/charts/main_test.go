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

func ready() model { return send(newModel(), tui.ResizeMsg{Width: 100, Height: 30}) }

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

func mouse(m model, x, y int, b tui.MouseButton, a tui.MouseAction) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: b, Action: a})
}

func hover(m model, x int) model {
	return mouse(m, x, m.plotRect().Y+2, tui.MouseButtonNone, tui.MouseActionMotion)
}

func TestShowsTheFirstDatasetAsALineChart(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"charts", "Temperature (°C)", "Latency", "Waves", "Growth", "Line", "Scatter", "Bars", "Histogram", "1 Berlin", "2 Cairo", "3 Oslo", "Jan", "Dec", "all 12 points", "┤", "└"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if !strings.ContainsAny(out, "⠁⠂⠄⡀⢀⠈⠐⠠⣀⠉⠒⠤") {
		t.Fatalf("no braille dots drawn:\n%s", out)
	}
}

func TestEveryDatasetAndKindDraws(t *testing.T) {
	m := ready()
	for ds := range m.sets {
		for k := kLine; k < numKinds; k++ {
			m.ds, m.kind = ds, k
			out := text(m)
			if k == kHist {
				if !strings.Contains(out, "bins") || !strings.Contains(out, "█") && !strings.Contains(out, "▏") && !strings.Contains(out, "━") && !strings.ContainsAny(out, "▌▉▊▋▍▎") {
					t.Errorf("%s histogram:\n%s", m.sets[ds].name, out)
				}
				continue
			}
			if !strings.ContainsAny(out, "⠁⠂⠄⡀⢀⠈⠐⠠⣀⠉⠒⠤⣿⡇⢸⣤⣶⣷⣾⣄⣠⠿⠛⠇⠸") {
				t.Errorf("%s %s drew nothing:\n%s", m.sets[ds].name, kindNames[k], out)
			}
		}
	}
}

func TestTabsAndKeysChangeDatasetAndKind(t *testing.T) {
	m := key(ready(), tui.KeyTab)
	if m.ds != 1 || !strings.Contains(text(m), "Latency (ms)") {
		t.Fatalf("ds = %d", m.ds)
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.ds != 3 {
		t.Fatalf("Shift+Tab wrapped to %d", m.ds)
	}
	m = runes(m, "t")
	if m.kind != kScatter {
		t.Fatalf("t: kind = %v", m.kind)
	}
	r := rect(t, m, "kind-2")
	m = mouse(m, r.X+1, r.Y, tui.MouseButtonLeft, tui.MouseActionPress)
	if m.kind != kBars {
		t.Fatalf("clicking Bars: kind = %v", m.kind)
	}
	r = rect(t, m, "ds-2")
	m = mouse(m, r.X+1, r.Y, tui.MouseButtonLeft, tui.MouseActionPress)
	if m.ds != 2 {
		t.Fatalf("clicking Waves: ds = %d", m.ds)
	}
}

func TestToggleSeriesByKeyAndLegendClick(t *testing.T) {
	m := runes(ready(), "2")
	if m.vis[0][1] || !strings.Contains(text(m), "2 Cairo (off)") {
		t.Fatalf("\n%s", text(m))
	}
	lo1, hi1, _ := bounds(m.cur().values(), m.vis[0], m.win())
	if hi1 >= 29 || lo1 != -4 {
		t.Fatalf("the y range still counts the hidden series: %v..%v", lo1, hi1)
	}
	r := rect(t, m, "leg-1")
	m = mouse(m, r.X+1, r.Y, tui.MouseButtonLeft, tui.MouseActionPress)
	if !m.vis[0][1] {
		t.Fatal("clicking the legend did not bring the series back")
	}
	m = runes(runes(runes(m, "1"), "2"), "3")
	if !m.vis[0][2] && !m.vis[0][0] && !m.vis[0][1] || !strings.Contains(m.msg, "at least one") {
		t.Fatalf("every series can be hidden: %v msg %q", m.vis[0], m.msg)
	}
	m = runes(m, "9")
	if len(m.vis[0]) != 3 {
		t.Fatal("a digit past the series changed something")
	}
}

func TestHiddenSeriesAreRememberedPerDataset(t *testing.T) {
	m := runes(ready(), "1")
	m = key(m, tui.KeyTab)
	if !m.vis[1][0] {
		t.Fatal("hiding in one data set hid in another")
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.vis[0][0] {
		t.Fatal("the choice was forgotten on coming back")
	}
}

func TestHoverShowsTheValuesUnderThePointer(t *testing.T) {
	m := ready()
	r := m.plotRect()
	m = hover(m, r.X)
	if m.cursor != 0 || !strings.Contains(text(m), "Jan") || !strings.Contains(text(m), "Berlin 0") {
		t.Fatalf("cursor = %d\n%s", m.cursor, text(m))
	}
	m = hover(m, r.X+r.W-1)
	if m.cursor != 11 || !strings.Contains(text(m), "Dec") || !strings.Contains(text(m), "Oslo -2") {
		t.Fatalf("cursor = %d\n%s", m.cursor, text(m))
	}
	m = mouse(m, 0, 0, tui.MouseButtonNone, tui.MouseActionMotion)
	if m.cursor != -1 {
		t.Fatal("leaving the plot kept the cursor")
	}
}

func TestTooltipSkipsHiddenSeries(t *testing.T) {
	m := runes(ready(), "2")
	m = hover(m, m.plotRect().X)
	if strings.Contains(text(m), "Cairo 14") {
		t.Fatalf("a hidden series is in the tooltip:\n%s", text(m))
	}
}

func TestKeyboardCursor(t *testing.T) {
	m := key(ready(), tui.KeyRight)
	if m.cursor < 0 {
		t.Fatal("Right did not start the cursor")
	}
	start := m.cursor
	m = key(m, tui.KeyRight)
	m = key(m, tui.KeyLeft)
	if m.cursor != start {
		t.Fatalf("cursor = %d, want %d", m.cursor, start)
	}
	for i := 0; i < 40; i++ {
		m = key(m, tui.KeyRight)
	}
	if m.cursor != 11 {
		t.Fatalf("cursor = %d past the end", m.cursor)
	}
	if m = key(m, tui.KeyEsc); m.cursor != -1 {
		t.Fatal("Esc did not clear the cursor")
	}
}

func TestWheelZoomsAroundThePointerAndDoubleClickResets(t *testing.T) {
	m := key(ready(), tui.KeyTab) // 300 points
	r := m.plotRect()
	m = mouse(m, r.X+r.W/2, r.Y+1, tui.MouseButtonWheelUp, tui.MouseActionPress)
	z := m.win()
	if z.span() >= full(300).span() || z.isFull() {
		t.Fatalf("window = %+v", z)
	}
	mid := (z.lo + z.hi) / 2
	if mid < 130 || mid > 170 {
		t.Fatalf("zooming at the middle moved the centre to %v", mid)
	}
	m = runes(m, "0")
	for i := 0; i < 30; i++ {
		m = mouse(m, r.X+2, r.Y+1, tui.MouseButtonWheelUp, tui.MouseActionPress)
	}
	if m.win().lo > 25 {
		t.Fatalf("zooming at the left edge drifted to lo=%v", m.win().lo)
	}
	if !strings.Contains(text(m), "points ") {
		t.Fatalf("no zoom range in the status:\n%s", text(m))
	}
	m = send(m, tui.MouseEvent{X: r.X + 3, Y: r.Y + 1, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 2})
	if !m.win().isFull() {
		t.Fatal("a double click did not reset the zoom")
	}
}

func TestZoomIsRememberedPerDataset(t *testing.T) {
	m := key(ready(), tui.KeyTab)
	m = runes(runes(m, "+"), "+")
	zoomed := m.win()
	m = key(m, tui.KeyTab)
	if !m.win().isFull() {
		t.Fatal("zoom carried over to another data set")
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.win() != zoomed {
		t.Fatal("zoom was forgotten")
	}
	if m = runes(m, "0"); !m.win().isFull() {
		t.Fatal("0 did not reset")
	}
}

func TestDragPans(t *testing.T) {
	m := key(ready(), tui.KeyTab)
	m = runes(runes(runes(m, "+"), "+"), "+")
	before := m.win()
	r := m.plotRect()
	m = mouse(m, r.X+40, r.Y+1, tui.MouseButtonLeft, tui.MouseActionPress)
	if m.dragX != r.X+40 {
		t.Fatal("no drag started")
	}
	m = mouse(m, r.X+30, r.Y+1, tui.MouseButtonLeft, tui.MouseActionMotion)
	if m.win().lo <= before.lo {
		t.Fatalf("dragging left should move the window right: %v -> %v", before.lo, m.win().lo)
	}
	if d := m.win().span() - before.span(); d > 1e-9 || d < -1e-9 {
		t.Fatal("panning changed the zoom")
	}
	m = mouse(m, r.X+30, r.Y+1, tui.MouseButtonLeft, tui.MouseActionRelease)
	moved := m.win()
	m = mouse(m, r.X+10, r.Y+1, tui.MouseButtonNone, tui.MouseActionMotion)
	if m.win() != moved {
		t.Fatal("moving after the release still dragged")
	}
	m = key(m, tui.KeyPgUp)
	if m.win().lo >= moved.lo {
		t.Fatal("PgUp did not pan left")
	}
}

func TestHistogramBins(t *testing.T) {
	m := key(ready(), tui.KeyTab)
	m.kind = kHist
	if !strings.Contains(text(m), "p50, 300 values, 10 bins") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(m, "]")
	m = runes(m, "]")
	if m.bins != 12 {
		t.Fatalf("bins = %d", m.bins)
	}
	for i := 0; i < 30; i++ {
		m = runes(m, "[")
	}
	if m.bins != 3 {
		t.Fatalf("bins = %d at the floor", m.bins)
	}
	m = runes(m, "+")
	if !strings.Contains(text(m), "values,") {
		t.Fatal("the histogram vanished on zoom")
	}
}

func TestBarsAndZeroBaseline(t *testing.T) {
	m := ready()
	m.kind = kBars
	lo, _, _, _ := m.scale()
	if lo > 0 {
		t.Fatalf("a bar chart's axis starts at %v, not at or below zero", lo)
	}
	m.kind = kLine
	m = runes(runes(m, "1"), "3") // only Cairo, 14 to 29 degrees
	lo, _, _, _ = m.scale()
	if lo <= 0 {
		t.Fatalf("Cairo's axis starts at %v; the test wants it above zero", lo)
	}
	m = runes(m, "y")
	if lo, _, _, _ = m.scale(); lo > 0 {
		t.Fatalf("y did not start the axis at zero: %v", lo)
	}
}

func TestXAxisLabelsDoNotCollide(t *testing.T) {
	m := ready()
	_, labels := m.xAxis(themeDark())
	plain := ansi.StripANSI(labels)
	if strings.Contains(plain, "JanFeb") || strings.Contains(plain, "MarApr") || !strings.Contains(plain, "Jan") || !strings.Contains(plain, "Dec") {
		t.Fatalf("labels = %q", plain)
	}
	for ds := range m.sets {
		m.ds = ds
		for _, w := range []int{20, 40, 100, 300} {
			m = send(m, tui.ResizeMsg{Width: w, Height: 30})
			_, l := m.xAxis(themeDark())
			if ansi.Width(l) > w {
				t.Fatalf("labels %d wide in %d columns", ansi.Width(l), w)
			}
		}
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "zoom around the pointer") {
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
	m := hover(key(ready(), tui.KeyTab), 20)
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {3, 3}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for k := kLine; k < numKinds; k++ {
			m.kind = k
			for i, line := range strings.Split(m.View(), "\n") {
				if w := ansi.Width(line); w > sz[0] {
					t.Fatalf("%dx%d %s: line %d is %d wide: %q", sz[0], sz[1], kindNames[k], i, w, line)
				}
			}
		}
	}
}
