package main

import (
	"flag"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// Declared here because tuitest registers no flag: `go test -update` rewrites
// the goldens.
var _ = flag.Bool("update", false, "rewrite golden files")

// scripted is the model without its Init timers, so the test alone decides
// when each line arrives and no wall-clock time is involved.
type scripted struct{ model }

func (scripted) Init() tui.Cmd { return nil }

func (s scripted) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	m, cmd := s.model.Update(msg)
	return scripted{m.(model)}, cmd
}

// Driving Update with each lineMsg (no timers) is deterministic: a full run
// leaves every line in scrollback, the program exits, and the final screen
// matches the golden file.
func TestScriptedBuildGolden(t *testing.T) {
	s := tuitest.New(scripted{model{}}, 40, 20, tui.WithAltScreen(false))
	t.Cleanup(s.Close)
	for i := range lines {
		s.Send(lineMsg{i})
	}
	if !s.Done() {
		t.Fatal("program did not exit after the last line")
	}
	got := strings.Join(s.Screen(), "\n")
	for _, l := range lines {
		if !strings.Contains(got, "ok "+l) {
			t.Fatalf("scrollback lacks %q:\n%s", l, got)
		}
	}
	s.Golden(t, "scripted")
}

func TestTailIsBounded(t *testing.T) {
	var m tui.Model = model{}
	for i := range lines {
		var cmd tui.Cmd
		m, cmd = m.Update(lineMsg{i})
		if cmd == nil {
			t.Fatalf("line %d committed nothing", i)
		}
	}
	mm := m.(model)
	if len(mm.tail) != tailRows {
		t.Fatalf("tail = %d rows, want %d", len(mm.tail), tailRows)
	}
	if got := strings.Count(mm.View(), "\n") + 1; got != tailRows+1 {
		t.Fatalf("view rows = %d", got)
	}
	if !mm.done || !strings.Contains(mm.View(), "build finished") {
		t.Fatal("not finished")
	}
}

func TestInitSchedulesAll(t *testing.T) {
	if (model{}).Init() == nil {
		t.Fatal("Init returned nil")
	}
}
