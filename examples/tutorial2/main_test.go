package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

func screen(s *tuitest.Session) string {
	return ansi.StripANSI(strings.Join(s.Screen(), "\n"))
}

// Three tasks are added, the list is searched, one is marked done, and the
// count under the list follows: the steps of docs/tutorial-2.md.
func TestSearchAndFinishATask(t *testing.T) {
	s := tuitest.New(newModel(), 64, 14)
	defer s.Close()
	for _, title := range []string{"write docs", "fix bug", "write tests"} {
		s.Keys(title, "enter")
	}
	if got := screen(s); !strings.Contains(got, "3 of 3 left") {
		t.Fatalf("after adding:\n%s", got)
	}
	s.Keys("tab", "w", "r", "i") // to the list; narrow to the "write" tasks
	got := screen(s)
	if !strings.Contains(got, "write docs") || !strings.Contains(got, "write tests") || strings.Contains(got, "fix bug") {
		t.Fatalf("the list was not narrowed:\n%s", got)
	}
	s.Keys("down", "enter") // second match: "write tests"
	got = screen(s)
	if !strings.Contains(got, "[x] write tests") || !strings.Contains(got, "2 of 3 left") {
		t.Fatalf("after marking:\n%s", got)
	}
	s.Keys("ctrl+c")
}

func TestNothingYetShowsTheHint(t *testing.T) {
	s := tuitest.New(newModel(), 64, 10)
	defer s.Close()
	if got := screen(s); !strings.Contains(got, "Nothing yet") {
		t.Errorf("screen:\n%s", got)
	}
	s.Keys("ctrl+c")
}
