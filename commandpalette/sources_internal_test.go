package commandpalette

import (
	"testing"

	"github.com/ows4444/tui"
)

func TestTaskDoneAcceptance(t *testing.T) {
	base := New()
	base.Name = "a"
	base.Sources = []Source{nil}
	base.Input.SetValue("x")
	done := func(name, query string, canceled bool) tui.TaskDoneMsg {
		return tui.TaskDoneMsg{Key: "k", Canceled: canceled, Msg: sourceMsg{name: name, query: query, found: []Command{{Name: "hit"}}}}
	}
	for _, c := range []struct {
		name string
		msg  tui.TaskDoneMsg
		mine bool
		kept bool
	}{
		{"another palette's", done("b", "x", false), false, false},
		{"not a source result", tui.TaskDoneMsg{Msg: 3}, false, false},
		{"cancelled", done("a", "x", true), true, false},
		{"for an older query", done("a", "w", false), true, false},
		{"current", done("a", "x", false), true, true},
	} {
		m, mine := base.taskDone(c.msg)
		if mine != c.mine || (len(m.fromSources("x")) == 1) != c.kept {
			t.Errorf("%s: mine=%v kept=%v; want %v, %v", c.name, mine, len(m.fromSources("x")) == 1, c.mine, c.kept)
		}
	}
}

func TestNilSourceIsSkipped(t *testing.T) {
	m := New()
	m.Sources = []Source{nil}
	m.Input.SetValue("q")
	if _, cmd := m.restart(); cmd != nil {
		t.Fatal("a nil Source started a task")
	}
}
