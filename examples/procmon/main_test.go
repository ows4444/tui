package main

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func send(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func press(m model, s string) model {
	m, _ = send(m, tui.Key{Type: tui.KeyRunes, Text: s})
	return m
}

func at(t testing.TB, m model, id string) (x, y int) {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen", id)
	}
	return r.X + r.W/2, r.Y
}

func click(t testing.TB, m model, id string) model {
	t.Helper()
	x, y := at(t, m, id)
	m, _ = send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	m, _ = send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
	return m
}

func TestStartsSortedByCPUWithTheBusiestSelected(t *testing.T) {
	m := initialModel()
	if m.sortCol != colCPU || !m.desc {
		t.Fatal("not sorted by CPU, busiest first")
	}
	rows := m.shown()
	for i := 1; i < len(rows); i++ {
		if rows[i-1].cpu < rows[i].cpu {
			t.Fatalf("rows %d and %d out of order", i-1, i)
		}
	}
	if m.selected != rows[0].pid {
		t.Fatal("the busiest process is not selected")
	}
}

func TestClickingAHeaderSortsAndAgainReverses(t *testing.T) {
	m := click(t, initialModel(), "col-"+strconv.Itoa(colName))
	if m.sortCol != colName || m.desc || m.shown()[0].name != "bluetoothd" {
		t.Fatalf("name sort: col=%d desc=%v first=%s", m.sortCol, m.desc, m.shown()[0].name)
	}
	m = click(t, m, "col-"+strconv.Itoa(colName))
	if !m.desc || m.shown()[0].name != "zsh" {
		t.Fatalf("reversed: desc=%v first=%s", m.desc, m.shown()[0].name)
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "NAME▼") {
		t.Fatalf("no sort arrow in the header:\n%s", out)
	}
}

func TestClickingARowSelectsIt(t *testing.T) {
	m := click(t, initialModel(), "row-3")
	if m.selected != m.shown()[3].pid {
		t.Fatalf("selected %d, want %d", m.selected, m.shown()[3].pid)
	}
}

func TestSelectionSurvivesAResort(t *testing.T) {
	m := click(t, initialModel(), "row-3")
	pid := m.selected
	m = m.sortBy(colName)
	if m.selected != pid {
		t.Fatal("the selection moved to another process when the sort changed")
	}
}

func TestKeysMoveTheSelection(t *testing.T) {
	m := initialModel()
	first := m.selected
	m, _ = send(m, tui.Key{Type: tui.KeyDown})
	m = press(m, "j")
	if m.selected != m.shown()[2].pid {
		t.Fatal("two moves down did not reach row 2")
	}
	m, _ = send(m, tui.Key{Type: tui.KeyEnd})
	if m.selected != m.shown()[len(m.shown())-1].pid {
		t.Fatal("End did not select the last row")
	}
	m, _ = send(m, tui.Key{Type: tui.KeyHome})
	if m.selected != first {
		t.Fatal("Home did not return to the first row")
	}
}

func TestWheelScrollsTheList(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 80, Height: 12})
	m, _ = send(m, tui.MouseEvent{X: 5, Y: 8, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.top != 3 {
		t.Fatalf("top = %d after a wheel step, want 3", m.top)
	}
	for i := 0; i < 20; i++ {
		m, _ = send(m, tui.MouseEvent{X: 5, Y: 8, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	}
	if m.top != len(m.shown())-m.rows() {
		t.Fatalf("top = %d, want the last page %d", m.top, len(m.shown())-m.rows())
	}
}

func TestFilter(t *testing.T) {
	m := press(initialModel(), "/")
	for _, r := range "post" {
		m = press(m, string(r))
	}
	if got := len(m.shown()); got != 2 {
		t.Fatalf("%d processes match post, want 2", got)
	}
	m, _ = send(m, tui.Key{Type: tui.KeyEnter})
	if m.typing || m.query != "post" {
		t.Fatal("Enter must keep the filter and stop typing")
	}
	m, _ = send(m, tui.Key{Type: tui.KeyEsc})
	if m.query != "" || len(m.shown()) != len(sample()) {
		t.Fatal("Esc did not clear the filter")
	}
}

func TestFilterWithNoMatchIsExplained(t *testing.T) {
	m := press(initialModel(), "/")
	for _, r := range "zzz" {
		m = press(m, string(r))
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "No process matches.") {
		t.Fatalf("empty result not explained:\n%s", out)
	}
}

func TestKillAsksFirstAndNoKeeps(t *testing.T) {
	m := initialModel()
	pid := m.selected
	m = press(m, "k")
	if m.askKill != pid {
		t.Fatal("k did not open the dialog for the selected process")
	}
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "Kill compile?") {
		t.Fatalf("dialog missing:\n%s", out)
	}
	m = press(m, "n")
	if m.askKill != 0 || m.index(pid) < 0 {
		t.Fatal("answering no must keep the process")
	}
}

func TestKillByClickingYes(t *testing.T) {
	m := click(t, initialModel(), "kill")
	if m.askKill == 0 {
		t.Fatal("the Kill button did not open the dialog")
	}
	pid := m.askKill
	m = click(t, m, "kill-yes")
	if m.index(pid) >= 0 || len(m.procs) != len(sample())-1 {
		t.Fatal("the process is still there after Yes")
	}
	if !strings.Contains(m.status, "Killed compile") {
		t.Fatalf("status = %q", m.status)
	}
	if m.selected == 0 || m.index(m.selected) < 0 {
		t.Fatal("no valid selection after a kill")
	}
}

func TestDialogBlocksTheListBehindIt(t *testing.T) {
	m := click(t, initialModel(), "kill")
	before := m.selected
	m = press(m, "j")
	m, _ = send(m, tui.Key{Type: tui.KeyDown})
	if m.selected != before {
		t.Fatal("keys moved the selection while the dialog was open")
	}
	m, _ = send(m, tui.Key{Type: tui.KeyEsc})
	if m.askKill != 0 {
		t.Fatal("Esc did not close the dialog")
	}
}

func TestPauseStopsTheNumbers(t *testing.T) {
	m := click(t, initialModel(), "pause")
	if !m.paused {
		t.Fatal("the Pause button did nothing")
	}
	before := m.procs[0]
	m, cmd := send(m, tickMsg(time.Now()))
	if m.procs[0] != before || cmd == nil {
		t.Fatal("a paused tick must change nothing and keep ticking")
	}
	m = press(m, " ")
	m, _ = send(m, tickMsg(time.Now()))
	if m.paused {
		t.Fatal("space did not resume")
	}
}

func TestTickMovesTheNumbersAndKeepsHistoryLength(t *testing.T) {
	m := initialModel()
	cpu := m.procs[0].cpu
	m, cmd := send(m, tickMsg(time.Now()))
	if cmd == nil || len(m.cpuHist) != history || len(m.memHist) != history {
		t.Fatal("tick did not re-arm or history grew")
	}
	if m.procs[0].cpu == cpu && m.procs[1].cpu == cpu {
		t.Fatal("the numbers did not change")
	}
}

func TestHoverOnAHeader(t *testing.T) {
	m := initialModel()
	x, y := at(t, m, "col-"+strconv.Itoa(colMem))
	m, _ = send(m, tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion})
	if m.hover != "col-"+strconv.Itoa(colMem) {
		t.Fatalf("hover = %q", m.hover)
	}
}

func TestNarrowTerminalDropsUser(t *testing.T) {
	m, _ := send(initialModel(), tui.ResizeMsg{Width: 50, Height: 20})
	if out := ansi.StripANSI(m.View()); strings.Contains(out, "USER") {
		t.Fatalf("USER still shown at 50 columns:\n%s", out)
	}
}

func TestQuit(t *testing.T) {
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}, {Type: tui.KeyEsc}} {
		if _, cmd := send(initialModel(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
}
