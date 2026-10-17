package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
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

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func at(t testing.TB, m model, id string) (x, y int) {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q on screen", id)
	}
	return r.X + r.W/2, r.Y
}

func click(m model, id string) model {
	r, _ := layout.RectOf(m.screen(), m.size(), id)
	return send(m, tui.MouseEvent{X: r.X + r.W/2, Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func TestClickRowTicksIt(t *testing.T) {
	m := click(initialModel(), "row-1")
	if !m.tasks[1].done {
		t.Fatal("clicking row 1 did not tick it")
	}
	m = click(m, "row-1")
	if m.tasks[1].done {
		t.Fatal("a second click did not untick it")
	}
}

func TestClickCrossDeletes(t *testing.T) {
	m := initialModel()
	n := len(m.tasks)
	want := m.tasks[1].text
	m = click(m, "del-0")
	if len(m.tasks) != n-1 || m.tasks[0].text != want {
		t.Fatalf("tasks after deleting row 0: %v", m.tasks)
	}
}

func TestTabsFilter(t *testing.T) {
	m := click(initialModel(), "tab-2")
	if v := m.visible(); len(v) != 1 || !m.tasks[v[0]].done {
		t.Fatalf("Done tab shows %v, want the one finished task", v)
	}
	m = click(m, "tab-1")
	if len(m.visible()) != 5 {
		t.Fatalf("Active tab shows %d, want 5", len(m.visible()))
	}
}

func TestClearDone(t *testing.T) {
	m := click(initialModel(), "clear")
	for _, task := range m.tasks {
		if task.done {
			t.Fatalf("a done task survived Clear done: %v", m.tasks)
		}
	}
	if len(m.tasks) != 5 {
		t.Fatalf("%d tasks left, want 5", len(m.tasks))
	}
}

func TestWheelScrolls(t *testing.T) {
	m := initialModel()
	for i := 0; i < 20; i++ {
		m.tasks = append(m.tasks, task{text: "extra"})
	}
	m = send(m, tui.ResizeMsg{Width: 80, Height: 24})
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.top != 1 {
		t.Fatalf("top = %d after one wheel step, want 1", m.top)
	}
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.top != 0 {
		t.Fatalf("top = %d after wheel up, want 0", m.top)
	}
}

func TestHover(t *testing.T) {
	m := initialModel()
	x, y := at(t, m, "del-2")
	m = send(m, tui.MouseEvent{X: x, Y: y, Action: tui.MouseActionMotion})
	if m.hover != "del-2" {
		t.Fatalf("hover = %q, want del-2", m.hover)
	}
}

func TestAddWithKeyboard(t *testing.T) {
	m := runes(initialModel(), "a")
	if !m.typing {
		t.Fatal("a did not start typing")
	}
	m = runes(m, "Buy tea")
	m = send(m, tui.Key{Type: tui.KeyEnter})
	if got := m.tasks[len(m.tasks)-1]; got.text != "Buy tea" || got.done {
		t.Fatalf("last task = %+v", got)
	}
	if m.input.Value() != "" {
		t.Fatal("the input kept its text after Enter")
	}
	m = send(m, tui.Key{Type: tui.KeyEsc})
	if m.typing {
		t.Fatal("Esc did not stop typing")
	}
}

func TestClickInputStartsTyping(t *testing.T) {
	m := click(initialModel(), "input")
	if !m.typing {
		t.Fatal("clicking the input did not start typing")
	}
	m = runes(m, "x")
	m = send(m, tui.Key{Type: tui.KeyEnter})
	if m.tasks[len(m.tasks)-1].text != "x" {
		t.Fatal("the typed task was not added")
	}
}

func TestBlankTaskIsNotAdded(t *testing.T) {
	m := runes(initialModel(), "a")
	m = runes(m, "   ")
	n := len(m.tasks)
	m = send(m, tui.Key{Type: tui.KeyEnter})
	if len(m.tasks) != n {
		t.Fatal("a blank task was added")
	}
}

func TestKeys(t *testing.T) {
	m := runes(initialModel(), "j")
	m = runes(m, " ")
	if !m.tasks[1].done {
		t.Fatal("j then space did not tick the second task")
	}
	m = runes(m, "d")
	if len(m.tasks) != 5 {
		t.Fatalf("d left %d tasks, want 5", len(m.tasks))
	}
	m = send(m, tui.Key{Type: tui.KeyTab})
	if m.filter != active {
		t.Fatal("Tab did not move to the Active filter")
	}
}

func TestEmptyListIsExplained(t *testing.T) {
	m := initialModel()
	m.tasks = nil
	if out := ansi.StripANSI(m.View()); !strings.Contains(out, "Nothing here") {
		t.Fatalf("empty list shows no message:\n%s", out)
	}
}

func TestQuit(t *testing.T) {
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyEsc}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(initialModel(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
	m := runes(initialModel(), "a")
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Error("q quit while typing a task")
	}
}

func TestSavesOnlyWhenTheListChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tasks.txt")
	m, err := fromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyDown}); cmd != nil {
		t.Fatal("moving the cursor returned a save Cmd")
	}
	next, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: " "})
	if cmd == nil {
		t.Fatal("ticking a task returned no save Cmd")
	}
	save(path, next.saved)()
	b, err := os.ReadFile(path)
	if err != nil || string(b) != encode(next.tasks) {
		t.Fatalf("file = %q, %v; want %q", b, err, encode(next.tasks))
	}
}

func TestFileRoundTrip(t *testing.T) {
	ts := []task{{"one", true}, {"two words", false}}
	got := decode(encode(ts))
	if len(got) != 2 || got[0] != ts[0] || got[1] != ts[1] {
		t.Fatalf("round trip gave %v", got)
	}
	path := filepath.Join(t.TempDir(), "t.txt")
	if err := os.WriteFile(path, []byte(encode(ts)), 0o600); err != nil {
		t.Fatal(err)
	}
	m, err := fromFile(path)
	if err != nil || len(m.tasks) != 2 || !m.tasks[0].done {
		t.Fatalf("fromFile = %v, %v", m.tasks, err)
	}
}
