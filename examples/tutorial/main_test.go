package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

func screen(s *tuitest.Session) string {
	return ansi.StripANSI(strings.Join(s.Screen(), "\n"))
}

// A task is typed, added with enter, moved to and marked done with tab, and
// the program quits on ctrl+c: the steps of docs/tutorial.md, through a real
// Program and a virtual screen.
func TestAddAndFinishATask(t *testing.T) {
	s := tuitest.New(newModel(), 60, 12)
	defer s.Close()
	s.Keys("write", "enter")
	if got := screen(s); !strings.Contains(got, "[ ] write") {
		t.Fatalf("task not listed:\n%s", got)
	}
	s.Keys("tab")
	if got := screen(s); !strings.Contains(got, "[x] ") {
		t.Fatalf("task not marked done:\n%s", got)
	}
	s.Keys("ctrl+c")
}

func TestEmptyTitleIsNotAdded(t *testing.T) {
	m := newModel()
	next, _ := m.Update(tui.Key{Type: tui.KeyEnter})
	if got := len(next.(model).tasks); got != 0 {
		t.Fatalf("%d tasks after enter on an empty field", got)
	}
}
