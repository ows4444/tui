package tuitest_test

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

// clicker draws two buttons and counts the clicks on each, so a test can
// find them by their labels.
type clicker struct{ ok, cancel int }

func (clicker) Init() tui.Cmd { return nil }

func (c clicker) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if m, ok := msg.(tui.MouseEvent); ok && m.Action == tui.MouseActionRelease && m.Y == 2 {
		switch {
		case m.X >= 2 && m.X < 6:
			c.ok++
		case m.X >= 10 && m.X < 18:
			c.cancel++
		}
	}
	return c, nil
}

func (c clicker) View() string {
	return "title\n\n  [OK]    [Cancel]\n" + strings.Repeat("x", c.ok) + "|" + strings.Repeat("y", c.cancel)
}

func TestFind(t *testing.T) {
	s := tuitest.New(clicker{}, 30, 5)
	defer s.Close()
	if x, y, ok := s.Find("[Cancel]"); !ok || x != 10 || y != 2 {
		t.Fatalf("Find([Cancel]) = %d, %d, %v; want 10, 2, true", x, y, ok)
	}
	if _, _, ok := s.Find("missing"); ok {
		t.Fatal("Find found text that is not drawn")
	}
	if _, _, ok := s.Find(""); ok {
		t.Fatal("Find found an empty string")
	}
}

func TestClickText(t *testing.T) {
	s := tuitest.New(clicker{}, 30, 5)
	defer s.Close()
	if !s.ClickText("[Cancel]") || !s.ClickText("[OK]") || !s.ClickText("[OK]") {
		t.Fatal("ClickText did not find a button")
	}
	if s.ClickText("[Nope]") {
		t.Fatal("ClickText reported a click on text that is not drawn")
	}
	if !s.WaitForText("xx|y", 0) {
		t.Fatalf("clicks not counted; screen:\n%s", strings.Join(s.Screen(), "\n"))
	}
}

type caret struct{ x, y int }

func (caret) Init() tui.Cmd                         { return nil }
func (c caret) Update(tui.Msg) (tui.Model, tui.Cmd) { return c, nil }
func (c caret) View() string                        { return "name: ada\nage: 36" }
func (c caret) CursorPos() (int, int, bool)         { return c.x, c.y, true }

func TestCursorOn(t *testing.T) {
	s := tuitest.New(caret{x: 7, y: 0}, 20, 3)
	defer s.Close()
	for text, want := range map[string]bool{"ada": true, "name": false, "d": true, "a": false, "age": false, "": false} {
		if got := s.CursorOn(text); got != want {
			t.Errorf("CursorOn(%q) = %v; want %v", text, got, want)
		}
	}
}

func TestCursorOnHidden(t *testing.T) {
	s := tuitest.New(typer{}, 20, 3)
	defer s.Close()
	if s.CursorOn(">") {
		t.Fatal("CursorOn reported a hidden cursor")
	}
}

func TestFindWideRow(t *testing.T) {
	s := tuitest.New(wide{}, 20, 2)
	defer s.Close()
	if x, _, ok := s.Find("end"); !ok || x != 5 {
		t.Fatalf("Find(end) = %d, %v; want column 5", x, ok)
	}
}

type wide struct{}

func (wide) Init() tui.Cmd                         { return nil }
func (w wide) Update(tui.Msg) (tui.Model, tui.Cmd) { return w, nil }
func (wide) View() string                          { return "é→ ü end" }
