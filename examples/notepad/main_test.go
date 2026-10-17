package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/tuitest"
)

func start(t *testing.T, st memStore, path string) *tuitest.Session {
	t.Helper()
	s := tuitest.New(newModel(st, path), 80, 24)
	t.Cleanup(s.Close)
	return s
}

func screen(s *tuitest.Session) string { return strings.Join(s.Screen(), "\n") }

func want(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	if !s.WaitForText(text, time.Second) {
		t.Fatalf("screen never showed %q:\n%s", text, screen(s))
	}
}

func absent(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	time.Sleep(60 * time.Millisecond)
	if strings.Contains(screen(s), text) {
		t.Fatalf("screen still shows %q:\n%s", text, screen(s))
	}
}

func done(s *tuitest.Session) bool {
	for i := 0; i < 100 && !s.Done(); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	return s.Done()
}

func TestOpensTheFileGivenOnTheCommandLine(t *testing.T) {
	s := start(t, sampleStore(), "main.go")
	want(t, s, "func greet(names []string)")
	want(t, s, "Ln 1, Col 1")
	want(t, s, "15 lines")
	if strings.Contains(screen(s), "●") {
		t.Fatal("a freshly opened file shows as modified")
	}
}

func TestTypingMarksTheDocumentModified(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	s.Keys("X")
	want(t, s, "notes.txt ●")
	want(t, s, "Ln 1, Col 2")
}

func TestSaveWritesTheFile(t *testing.T) {
	st := sampleStore()
	s := start(t, st, "notes.txt")
	s.Keys("X", "ctrl+s")
	want(t, s, "Saved notes.txt")
	if !strings.HasPrefix(st["notes.txt"], "Xbuy milk") {
		t.Fatalf("file = %q", st["notes.txt"])
	}
	absent(t, s, "notes.txt ●")
}

func TestSaveWithoutAPathAsksForOne(t *testing.T) {
	st := memStore{}
	s := start(t, st, "")
	s.Keys("hello", "ctrl+s")
	want(t, s, "Save as:")
	s.Keys("out.txt", "enter")
	want(t, s, "Saved out.txt")
	if st["out.txt"] != "hello" {
		t.Fatalf("file = %q", st["out.txt"])
	}
	want(t, s, "out.txt")
}

func TestSaveErrorIsReported(t *testing.T) {
	s := start(t, sampleStore(), "readonly.txt")
	s.Keys("x", "ctrl+s")
	want(t, s, "Cannot save: readonly.txt: permission denied")
	want(t, s, "readonly.txt ●") // still unsaved
}

func TestOpenAFile(t *testing.T) {
	s := start(t, sampleStore(), "")
	s.Keys("ctrl+o")
	want(t, s, "Open:")
	s.Keys("notes.txt", "enter")
	want(t, s, "write tests")
	want(t, s, "Opened notes.txt")
}

func TestOpenMissingFileSaysSo(t *testing.T) {
	s := start(t, sampleStore(), "")
	s.Keys("ctrl+o", "nope.txt", "enter")
	want(t, s, "nope.txt: no such file")
}

func TestUnsavedChangesAreNotDiscardedSilently(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	s.Keys("X", "ctrl+n")
	want(t, s, "Discard unsaved changes?")
	s.Keys("n")
	absent(t, s, "Discard unsaved changes?")
	want(t, s, "notes.txt ●")
	s.Keys("ctrl+n", "y")
	want(t, s, "New document")
	want(t, s, "untitled")
}

func TestQuitAsksWhenModifiedAndQuitsWhenClean(t *testing.T) {
	clean := start(t, sampleStore(), "notes.txt")
	clean.Keys("ctrl+q")
	if !done(clean) {
		t.Fatal("ctrl+q did not quit a clean document")
	}
	dirty := start(t, sampleStore(), "notes.txt")
	dirty.Keys("X", "ctrl+q")
	want(t, dirty, "Discard unsaved changes?")
	if dirty.Done() {
		t.Fatal("quit with unsaved changes did not ask")
	}
	dirty.Keys("y")
	if !done(dirty) {
		t.Fatal("answering yes did not quit")
	}
}

func TestFindSelectsMatchesAndEnterGoesOn(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	s.Keys("ctrl+f", "buy", "enter")
	want(t, s, "2 matches")
	want(t, s, "Ln 1, Col 4") // the first match, its end
	s.Keys("enter")
	want(t, s, "Ln 2, Col 4")
	s.Keys("enter")
	want(t, s, "Ln 1, Col 4") // wraps
	s.Keys("esc")
	absent(t, s, "Find:")
}

func TestFindNotFound(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	s.Keys("ctrl+f", "zzz", "enter")
	want(t, s, "Not found: zzz")
}

func TestReplaceAll(t *testing.T) {
	st := sampleStore()
	s := start(t, st, "notes.txt")
	s.Keys("ctrl+r", "buy", "enter")
	want(t, s, "With:")
	s.Keys("get", "enter")
	want(t, s, "Replaced 2")
	want(t, s, "get milk")
	s.Keys("ctrl+s")
	want(t, s, "Saved notes.txt")
	if !strings.Contains(st["notes.txt"], "get tea") {
		t.Fatalf("file = %q", st["notes.txt"])
	}
}

func TestUndo(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	s.Keys("X")
	want(t, s, "Xbuy milk")
	s.Keys("ctrl+z")
	absent(t, s, "Xbuy milk")
}

func TestMenuBarWithTheMouse(t *testing.T) {
	st := sampleStore()
	s := start(t, st, "notes.txt")
	s.ClickText("File")
	want(t, s, "Save as…")
	want(t, s, "ctrl+q")
	s.ClickText("Save as…")
	want(t, s, "Save as:")
}

func TestViewMenuTogglesLineNumbers(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	want(t, s, "1 buy milk")
	s.ClickText("View")
	want(t, s, "Line numbers")
	s.ClickText("Line numbers")
	absent(t, s, "1 buy milk")
	if !strings.Contains(screen(s), "buy milk") {
		t.Fatal("the text vanished with the numbers")
	}
}

func TestAltFOpensTheFileMenu(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	s.Keys("alt+f")
	want(t, s, "Open…")
	s.Keys("esc")
	absent(t, s, "Open…")
}

func TestClickPlacesTheCursor(t *testing.T) {
	s := start(t, sampleStore(), "notes.txt")
	x, y, ok := s.Find("write tests")
	if !ok {
		t.Fatalf("no third line:\n%s", screen(s))
	}
	s.Click(x+6, y)
	want(t, s, "Ln 3, Col 7")
}

func TestTabsInAFileBecomeSpaces(t *testing.T) {
	m := newModel(memStore{"t.go": "\tx\n"}, "t.go").load("t.go")
	if !strings.Contains(m.msg, "tabs became 4 spaces") || strings.Contains(m.ed.Value(), "\t") {
		t.Fatalf("msg=%q value=%q", m.msg, m.ed.Value())
	}
}

func TestResizeKeepsTheEditorBetweenTheBars(t *testing.T) {
	m := newModel(sampleStore(), "main.go")
	next, _ := m.Update(tui.ResizeMsg{Width: 50, Height: 12})
	got := next.(model)
	if got.ed.Height != 9 || got.ed.Bounds.Y != 1 {
		t.Fatalf("editor height %d at y %d, want 9 at 1", got.ed.Height, got.ed.Bounds.Y)
	}
}
