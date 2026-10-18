package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

func session(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(newModel(), 100, 28)
	t.Cleanup(s.Close)
	return s
}

func screen(s *tuitest.Session) string { return strings.Join(s.Screen(), "\n") }

func want(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	if !s.WaitForText(text, 3*time.Second) {
		t.Fatalf("never saw %q:\n%s", text, screen(s))
	}
}

func gone(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !strings.Contains(screen(s), text) {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("%q never went away:\n%s", text, screen(s))
}

func TestPagesChangeByKeyAndByClick(t *testing.T) {
	s := session(t)
	want(t, s, "Overlays")
	s.Keys("f2")
	want(t, s, "Loading bar")
	s.Keys("ctrl+n")
	want(t, s, "Email")
	s.Keys("ctrl+p")
	want(t, s, "Loading bar")
	s.Click(tabAt(1)+1, 1) // the Menus tab
	_ = tabAt
	want(t, s, "Overlays")
	if tabAt(1) != 0 || tabAt(99) != -1 {
		t.Error("tabAt")
	}
}

func TestMenuChoosesAndDrillsIn(t *testing.T) {
	s := session(t)
	s.Keys("enter")
	want(t, s, "chose Open")
	s.Keys("down", "enter") // Recent has children
	want(t, s, "report.md")
	s.Keys("enter")
	want(t, s, "chose report.md")
}

func TestEachOverlayOpensAndCloses(t *testing.T) {
	for key, text := range map[string]string{"d": "This cannot be undone.", "w": "[x] open", "p": "Anchored near a point"} {
		s := session(t)
		s.Keys(key)
		want(t, s, text)
		s.Keys("esc")
		gone(t, s, text)
	}
}

func TestToastShowsAndDismissesItself(t *testing.T) {
	s := session(t)
	s.Keys("t")
	want(t, s, "Saved")
	gone(t, s, "Saved")
}

func TestLoadingAnimatesAndTheListScrolls(t *testing.T) {
	s := session(t)
	s.Keys("f2")
	want(t, s, "row 0000")
	s.Keys("pgdown")
	want(t, s, "row 0005")
	before := screen(s)
	time.Sleep(300 * time.Millisecond)
	if screen(s) == before {
		t.Error("the loading bar and skeleton did not move")
	}
}

func TestFieldsTakeTextAndTabMoves(t *testing.T) {
	s := session(t)
	s.Keys("f3")
	s.Keys("a", "d", "a", "@", "x", ".", "i", "o")
	want(t, s, "(valid)")
	s.Keys("tab", "s", "e", "c")
	want(t, s, "•••")
	s.Keys("tab", "a", "l")
	want(t, s, "alpha")
}

func TestWizardNeedsAValidEmailThenFinishes(t *testing.T) {
	s := session(t)
	s.Keys("f4")
	s.Keys("n")
	want(t, s, "give a valid email")
	s.Keys("f3", "a", "@", "b", ".", "c", "o", "f4", "n", "n")
	gone(t, s, "give a valid email")
	s.Keys("b")
}

func TestHelpOpensAndAnyKeyClosesIt(t *testing.T) {
	s := session(t)
	s.Keys("?")
	want(t, s, "change the page")
	s.Keys("x")
	gone(t, s, "change the page")
}

func TestQuitKeys(t *testing.T) {
	m := newModel()
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := m.Update(k); cmd == nil {
			t.Errorf("%+v did not quit", k)
		}
	}
	m.page = pageLoading
	if _, cmd := m.Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); cmd == nil {
		t.Error("q on Loading did not quit")
	}
}
