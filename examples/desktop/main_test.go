package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

// feed runs a command's message back through Update, as the Program would.
func feed(m model, cmd tui.Cmd) model {
	if cmd != nil {
		m, _ = update(m, cmd())
	}
	return m
}

func key(m model, t tui.KeyType) (model, tui.Cmd) { return update(m, tui.Key{Type: t}) }

func typed(m model, s string) model {
	for _, r := range s {
		m, _ = update(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model {
	m := newModel()
	m.manual = true
	m, _ = update(m, tui.ResizeMsg{Width: 100, Height: 30})
	return m
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func mouse(m model, x, y int, b tui.MouseButton, a tui.MouseAction) (model, tui.Cmd) {
	return update(m, tui.MouseEvent{X: x, Y: y, Button: b, Action: a, Clicks: 1})
}

func click(m model, x, y int) model {
	m, cmd := mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionPress)
	m = feed(m, cmd)
	m, cmd = mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionRelease)
	return feed(m, cmd)
}

func clickTask(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return click(m, r.X+2, r.Y)
}

func winRect(t testing.TB, m model, id string) hittest.Rect {
	t.Helper()
	w, ok := m.desk.Window(id)
	if !ok {
		t.Fatalf("no window %q", id)
	}
	return w.Rect
}

// screenOf is where cell (x, y) of window id is on the screen.
func screenOf(t testing.TB, m model, id string, x, y int) (int, int) {
	r := winRect(t, m, id)
	return m.desk.Bounds.X + r.X + x, m.desk.Bounds.Y + r.Y + y
}

func TestFirstScreenHasFourWindowsAndATaskBar(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"desktop  4 windows", "┌ Clock", "┌ Counter", "┌ Notes", "┌ Event log", "09:00:00", "[ - ]", "▣ Notes", "▢ Clock", "▢ Event log", "started", "[f6] move"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if m.desk.Focused() != "notes" {
		t.Fatalf("focus %q", m.desk.Focused())
	}
}

func TestDraggingATitleBarMovesTheWindowAndLogsIt(t *testing.T) {
	m := ready()
	before := winRect(t, m, "clock")
	x, y := screenOf(t, m, "clock", 5, 0)
	m, cmd := mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionPress)
	m = feed(m, cmd)
	m, _ = mouse(m, x+10, y+3, tui.MouseButtonLeft, tui.MouseActionMotion)
	m, cmd = mouse(m, x+10, y+3, tui.MouseButtonLeft, tui.MouseActionRelease)
	m = feed(m, cmd)
	after := winRect(t, m, "clock")
	if after.X != before.X+10 || after.Y != before.Y+3 || m.desk.Focused() != "clock" {
		t.Fatalf("%+v -> %+v focus %q", before, after, m.desk.Focused())
	}
	if !strings.Contains(strings.Join(m.logs, "\n"), "Clock moved to") || !strings.Contains(strings.Join(m.logs, "\n"), "Clock has focus") {
		t.Fatalf("log %v", m.logs)
	}
}

func TestDraggingAnEdgeResizes(t *testing.T) {
	m := ready()
	r := winRect(t, m, "counter")
	x, y := screenOf(t, m, "counter", r.W-1, 3) // the right edge
	m, cmd := mouse(m, x, y, tui.MouseButtonLeft, tui.MouseActionPress)
	m = feed(m, cmd)
	m, _ = mouse(m, x+6, y, tui.MouseButtonLeft, tui.MouseActionMotion)
	m, cmd = mouse(m, x+6, y, tui.MouseButtonLeft, tui.MouseActionRelease)
	m = feed(m, cmd)
	if got := winRect(t, m, "counter"); got.W != r.W+6 || got.X != r.X {
		t.Fatalf("%+v -> %+v", r, got)
	}
	if !strings.Contains(strings.Join(m.logs, "\n"), "Counter moved to") {
		t.Fatalf("log %v", m.logs)
	}
}

func TestAPressRaisesTheWindowUnderIt(t *testing.T) {
	m := ready()
	x, y := screenOf(t, m, "log", 3, 0)
	m = click(m, x, y)
	if m.desk.Focused() != "log" {
		t.Fatalf("focus %q", m.desk.Focused())
	}
	if !strings.Contains(text(m), "▣ Event log") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestTabAndShiftTabCycleThroughTheWindows(t *testing.T) {
	m := ready()
	seen := map[string]bool{m.desk.Focused(): true}
	for range 3 {
		m, _ = key(m, tui.KeyTab)
		seen[m.desk.Focused()] = true
	}
	if len(seen) != 4 {
		t.Fatalf("Tab reached %v", seen)
	}
	first := m.desk.Focused()
	m, _ = update(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	m, _ = key(m, tui.KeyTab)
	if m.desk.Focused() != first {
		t.Fatalf("Shift+Tab then Tab should come back: %q vs %q", m.desk.Focused(), first)
	}
}

func TestMoveModeMovesAndResizesWithTheArrows(t *testing.T) {
	m := ready()
	before := winRect(t, m, "notes")
	m, _ = key(m, tui.KeyF6)
	if !m.moving || !strings.Contains(text(m), "Move mode") {
		t.Fatal("not in move mode")
	}
	m, _ = key(m, tui.KeyRight)
	m, _ = key(m, tui.KeyDown)
	m, _ = update(m, tui.Key{Type: tui.KeyRight, Mod: input.ModShift})
	got := winRect(t, m, "notes")
	if got.X != before.X+2 || got.Y != before.Y+2 || got.W != before.W+2 {
		t.Fatalf("%+v -> %+v", before, got)
	}
	m, _ = key(m, tui.KeyEsc)
	if m.moving {
		t.Fatal("Esc did not leave move mode")
	}
	m = typed(m, "a")
	if got := winRect(t, m, "notes"); got.X != before.X+2 {
		t.Fatal("typing moved the window")
	}
}

func TestMaximizeWithF2AndBack(t *testing.T) {
	m := ready()
	before := winRect(t, m, "notes")
	m, _ = key(m, tui.KeyF2)
	if got := winRect(t, m, "notes"); got.W != 100 || got.H != 26 {
		t.Fatalf("maximized %+v", got)
	}
	m, _ = key(m, tui.KeyF2)
	if got := winRect(t, m, "notes"); got != before {
		t.Fatalf("restored %+v want %+v", got, before)
	}
}

func TestNotesTakesTextWhenFocused(t *testing.T) {
	m := ready()
	m = typed(m, "hi")
	m, _ = key(m, tui.KeyEnter)
	m = typed(m, "yo")
	if !strings.Contains(string(m.notes), "hi\nyo") || !m.dirty || !strings.Contains(text(m), "Notes *") {
		t.Fatalf("%q dirty %v", string(m.notes), m.dirty)
	}
	m, _ = key(m, tui.KeyBackspace)
	if !strings.HasSuffix(string(m.notes), "hi\ny") {
		t.Fatalf("%q", string(m.notes))
	}
	m, _ = key(m, tui.KeyTab)
	before := string(m.notes)
	m = typed(m, "zzz")
	if string(m.notes) != before {
		t.Fatal("text went to the notepad while another window had focus")
	}
}

func TestCounterByMouseAndKeys(t *testing.T) {
	m := ready()
	cr, _ := m.desk.ContentRect("counter")
	m = click(m, cr.X+plusX+1, cr.Y+counterRow)
	m = click(m, cr.X+plusX+1, cr.Y+counterRow)
	if m.counter != 2 || m.desk.Focused() != "counter" || !strings.Contains(text(m), "   2") {
		t.Fatalf("counter %d focus %q", m.counter, m.desk.Focused())
	}
	m = click(m, cr.X+minusX+1, cr.Y+counterRow)
	if m.counter != 1 {
		t.Fatalf("counter %d", m.counter)
	}
	m = typed(m, "+")
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyDown)
	m, _ = key(m, tui.KeyDown)
	if m.counter != -1 {
		t.Fatalf("counter %d", m.counter)
	}
	m = click(m, cr.X+resetX+2, cr.Y+counterRow+2)
	if m.counter != 0 {
		t.Fatalf("reset: %d", m.counter)
	}
}

func TestCloseAskingFirstWhenTheNotepadHasUnsavedText(t *testing.T) {
	m := typed(ready(), "x")
	m, _ = key(m, tui.KeyF4)
	if m.modal == nil || !strings.Contains(text(m), "Close Notes?") || !strings.Contains(text(m), "y Yes") {
		t.Fatalf("no dialog:\n%s", text(m))
	}
	if _, open := m.desk.Window("notes"); !open {
		t.Fatal("the window closed before the answer")
	}
	// a click on the desktop behind the dialog does nothing
	x, y := screenOf(t, m, "clock", 3, 0)
	m = click(m, x, y)
	if m.desk.Focused() != "notes" || m.modal == nil {
		t.Fatal("the dialog did not block the mouse")
	}
	m = typed(m, "z")
	if strings.Contains(string(m.notes), "z") {
		t.Fatal("the dialog did not block the keys")
	}
	m, _ = key(m, tui.KeyEsc)
	if m.modal != nil {
		t.Fatal("Esc did not answer No")
	}
	if _, open := m.desk.Window("notes"); !open {
		t.Fatal("No closed the window")
	}
	m, _ = key(m, tui.KeyF4)
	m = clickTask(t, m, "modal-yes")
	if _, open := m.desk.Window("notes"); open || len(m.closed) != 1 || m.dirty {
		t.Fatalf("closed %d dirty %v", len(m.closed), m.dirty)
	}
}

func TestCloseBoxOnACleanWindowClosesAtOnceAndF3BringsItBack(t *testing.T) {
	m := ready()
	w := winRect(t, m, "clock")
	x, y := screenOf(t, m, "clock", w.W-3, 0)
	m = click(m, x, y)
	if _, open := m.desk.Window("clock"); open {
		t.Fatal("the close box did nothing")
	}
	if len(m.closed) != 1 || !strings.Contains(strings.Join(m.logs, "\n"), "Clock closed") || !strings.Contains(text(m), "+ 1 closed") {
		t.Fatalf("closed %d log %v", len(m.closed), m.logs)
	}
	m, _ = key(m, tui.KeyF3)
	if _, open := m.desk.Window("clock"); !open || m.desk.Focused() != "clock" || len(m.closed) != 0 {
		t.Fatal("F3 did not bring it back")
	}
	if m, _ = key(m, tui.KeyF3); !strings.Contains(m.msg, "No closed window") {
		t.Fatalf("msg %q", m.msg)
	}
}

func TestCloseBoxOnTheDirtyNotepadAsksToo(t *testing.T) {
	m := typed(ready(), "x")
	w := winRect(t, m, "notes")
	x, y := screenOf(t, m, "notes", w.W-3, 0)
	m = click(m, x, y)
	if m.modal == nil {
		t.Fatal("no question")
	}
	if _, open := m.desk.Window("notes"); !open {
		t.Fatal("closed without asking")
	}
}

func TestTaskBarRaisesHidesAndShows(t *testing.T) {
	m := ready()
	m = clickTask(t, m, "tb-clock")
	if m.desk.Focused() != "clock" {
		t.Fatalf("focus %q", m.desk.Focused())
	}
	m = clickTask(t, m, "tb-clock") // the focused one: hide
	if _, open := m.desk.Window("clock"); open || len(m.hidden) != 1 || !strings.Contains(text(m), "Clock (hidden)") {
		t.Fatalf("not hidden:\n%s", text(m))
	}
	m = clickTask(t, m, "tb-clock")
	if _, open := m.desk.Window("clock"); !open || m.desk.Focused() != "clock" || len(m.hidden) != 0 {
		t.Fatal("not shown again")
	}
	before := winRect(t, m, "clock")
	m, _ = key(m, tui.KeyF5)
	m = clickTask(t, m, "tb-clock")
	if winRect(t, m, "clock") != before {
		t.Fatal("a window came back somewhere else")
	}
}

func TestPlusButtonBringsBackAClosedWindow(t *testing.T) {
	m := ready()
	m, _ = key(m, tui.KeyTab)
	m, _ = key(m, tui.KeyF4)
	if len(m.closed) != 1 {
		t.Fatalf("closed %d", len(m.closed))
	}
	m = clickTask(t, m, "tb-add")
	if len(m.closed) != 0 || len(m.desk.Windows) != 4 {
		t.Fatalf("closed %d windows %d", len(m.closed), len(m.desk.Windows))
	}
}

func TestEventLogScrolls(t *testing.T) {
	m := ready()
	for i := 0; i < 40; i++ {
		m = m.log("event " + itoa(i))
	}
	x, y := screenOf(t, m, "log", 3, 3)
	m, _ = mouse(m, x, y, tui.MouseButtonWheelUp, tui.MouseActionPress)
	if m.logOff != 3 {
		t.Fatalf("offset %d", m.logOff)
	}
	m, _ = mouse(m, x, y, tui.MouseButtonWheelDown, tui.MouseActionPress)
	m, _ = mouse(m, x, y, tui.MouseButtonWheelDown, tui.MouseActionPress)
	if m.logOff != 0 {
		t.Fatalf("offset %d", m.logOff)
	}
	m, _ = mouse(m, 0, 0, tui.MouseButtonWheelUp, tui.MouseActionPress)
	if m.logOff != 0 {
		t.Fatal("wheel outside the log scrolled it")
	}
}

func TestClockTicks(t *testing.T) {
	m := ready()
	m, _ = update(m, tickMsg{})
	m, _ = update(m, tickMsg{})
	if m.secs != 2 || !strings.Contains(text(m), "09:00:02") {
		t.Fatalf("secs %d", m.secs)
	}
}

func TestResizeKeepsWindowsOnTheDesktop(t *testing.T) {
	m := ready()
	m, _ = update(m, tui.ResizeMsg{Width: 50, Height: 14})
	for _, w := range m.desk.Windows {
		if w.Rect.X+w.Rect.W > 50 || w.Rect.Y+w.Rect.H > 10 {
			t.Errorf("%s at %+v is off a 50x10 desktop", w.ID, w.Rect)
		}
	}
	if lines := strings.Split(text(m), "\n"); len(lines) != 14 || !strings.Contains(lines[13], "[tab]") && !strings.Contains(lines[13], "tab") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestEverythingClosedLeavesAUsableDesktop(t *testing.T) {
	m := ready()
	for range 4 {
		m, _ = key(m, tui.KeyF5)
	}
	if len(m.desk.Windows) != 0 || !strings.Contains(text(m), "No windows") || !strings.Contains(text(m), "(hidden)") {
		t.Fatalf("\n%s", text(m))
	}
	m, _ = key(m, tui.KeyF5) // nothing to hide, nothing breaks
	m, _ = key(m, tui.KeyTab)
	if len(m.desk.Windows) != 0 {
		t.Fatal("a window appeared from nowhere")
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := ready()
	m, _ = key(m, tui.KeyF1)
	if !strings.Contains(text(m), "move mode: arrows move") && !strings.Contains(text(m), "move mode") {
		t.Fatalf("no help:\n%s", text(m))
	}
	m, _ = key(m, tui.KeyEsc)
	if m.help {
		t.Fatal("a key should close help")
	}
	m, _ = update(m, tui.Key{Type: tui.KeyCtrl, Code: 'q'})
	if m.modal == nil || m.modal.yes != "quit" {
		t.Fatal("Ctrl+Q should ask")
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"}); cmd == nil {
		t.Fatal("y should quit")
	}
	m2, _ := update(ready(), tui.Key{Type: tui.KeyCtrlC})
	if m2.modal == nil {
		t.Fatal("Ctrl+C should ask")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyRunes, Text: "q"}); cmd != nil {
		t.Fatal("q must not quit")
	}
}
