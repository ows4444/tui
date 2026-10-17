package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/tuitest"
)

func start(t *testing.T) *tuitest.Session {
	t.Helper()
	s := tuitest.New(newModel(sample(), "sample"), 100, 30, tui.WithMultiClick(400*time.Millisecond))
	t.Cleanup(s.Close)
	return s
}

func screen(s *tuitest.Session) string { return strings.Join(s.Screen(), "\n") }

func want(t *testing.T, s *tuitest.Session, text string) {
	t.Helper()
	if !s.WaitForText(text, 3*time.Second) {
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

func TestStartsOnTheTopFolder(t *testing.T) {
	s := start(t)
	for _, text := range []string{"Folders", "assets/", "README.md", "0 folders, 2 files", "Preview"} {
		want(t, s, text)
	}
	absent(t, s, ".gitignore")
}

func TestClickingAFileShowsItsPreview(t *testing.T) {
	s := start(t)
	s.ClickText("README.md")
	want(t, s, "A small project.")
	want(t, s, "README.md  27 B")
}

func TestBinaryAndEmptyFilesAreDescribed(t *testing.T) {
	s := start(t)
	s.ClickText("assets/")
	s.Keys("enter")
	want(t, s, "logo.bin")
	s.ClickText("logo.bin")
	want(t, s, "Binary file, 7 B.")
	s.ClickText("empty.txt")
	want(t, s, "Empty file.")
}

func TestKeyboardOpensAFolderAndBackspaceGoesUp(t *testing.T) {
	s := start(t)
	s.ClickText("docs/")
	s.Keys("enter")
	want(t, s, "guide.md")
	want(t, s, "sample ▸ docs")
	s.Keys("backspace")
	want(t, s, "Zebra.txt")
	absent(t, s, "guide.md  ")
}

func TestDoubleClickOpensAFolder(t *testing.T) {
	s := start(t)
	x, y, ok := s.Find("src/")
	if !ok {
		t.Fatalf("no src/ row:\n%s", screen(s))
	}
	for _, clicks := range []int{1, 2} {
		s.Send(tui.MouseEvent{X: x + 1, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
		s.Send(tui.MouseEvent{X: x + 1, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease, Clicks: clicks})
	}
	want(t, s, "sample ▸ src")
	want(t, s, "util/")
}

func TestClickingAPlaceInThePathGoesBack(t *testing.T) {
	s := start(t)
	s.ClickText("docs/")
	s.Keys("enter")
	s.ClickText("api/")
	s.Keys("enter")
	want(t, s, "sample ▸ docs ▸ api")
	want(t, s, "index.md")
	s.ClickText("docs") // the middle place of the path bar
	want(t, s, "guide.md")
	absent(t, s, "index.md")
}

func TestClickingAFolderInTheTreeShowsItsFiles(t *testing.T) {
	s := start(t)
	x, y, ok := s.Find("▸ src")
	if !ok {
		t.Fatalf("no src in the tree:\n%s", screen(s))
	}
	s.Click(x+3, y)
	want(t, s, "sample ▸ src")
	want(t, s, "util/")
}

func TestTreeKeysMoveThroughFolders(t *testing.T) {
	s := start(t)
	x, y, _ := s.Find("▸ docs")
	s.Click(x+3, y) // focuses the tree on docs
	want(t, s, "guide.md")
	s.Keys("down")
	want(t, s, "util/") // src, the next folder
	absent(t, s, "guide.md  ")
	s.Keys("up", "up", "up")
	want(t, s, "Zebra.txt") // src, docs, assets, then the top folder
}

func TestRightClickOpensMenuAndGoUpWorks(t *testing.T) {
	s := start(t)
	s.ClickText("docs/")
	s.Keys("enter")
	x, y, _ := s.Find("guide.md")
	s.Send(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonRight, Action: tui.MouseActionPress})
	want(t, s, "Copy path")
	want(t, s, "Go up")
	s.Keys("esc")
	absent(t, s, "Copy path")
	s.Send(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonRight, Action: tui.MouseActionPress})
	want(t, s, "Go up")
	s.ClickText("Go up")
	want(t, s, "Zebra.txt")
	absent(t, s, "guide.md")
}

func TestHiddenFilesToggle(t *testing.T) {
	s := start(t)
	s.Keys(".")
	want(t, s, ".gitignore")
	s.Keys(".")
	absent(t, s, ".gitignore")
}

func TestHelpOpensAndCloses(t *testing.T) {
	s := start(t)
	s.Keys("?")
	want(t, s, "double-click   open")
	s.Keys("esc")
	absent(t, s, "double-click   open")
	if s.Done() {
		t.Fatal("Esc closed the program instead of the help")
	}
}

func TestQuit(t *testing.T) {
	s := start(t)
	s.Keys("q")
	for i := 0; i < 100 && !s.Done(); i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if !s.Done() {
		t.Fatal("q did not quit")
	}
}

func TestDraggingTheDividerResizesThePanes(t *testing.T) {
	m := newModel(sample(), "sample")
	next, _ := m.Update(tui.ResizeMsg{Width: 100, Height: 30})
	m = next.(model)
	d := m.split1.DividerRect()
	before := m.split1.Pos()
	press := tui.MouseEvent{X: d.X, Y: d.Y + 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	next, _ = m.Update(press)
	next, _ = next.(model).Update(tui.MouseEvent{X: d.X + 10, Y: d.Y + 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionMotion})
	next, _ = next.(model).Update(tui.MouseEvent{X: d.X + 10, Y: d.Y + 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease})
	if got := next.(model).split1.Pos(); got != before+10 {
		t.Fatalf("divider at %d after a 10 column drag from %d", got, before)
	}
}

func TestSmallTerminalsDropPanes(t *testing.T) {
	m := newModel(sample(), "sample")
	next, _ := m.Update(tui.ResizeMsg{Width: 60, Height: 12})
	out := ansi.StripANSI(next.View())
	if strings.Contains(out, "Folders") || strings.Contains(out, "Preview") {
		t.Fatalf("a 60x12 terminal still shows the tree or preview:\n%s", out)
	}
	if !strings.Contains(out, "assets/") {
		t.Fatalf("the list vanished:\n%s", out)
	}
}

func TestWheelScrollsALongPreview(t *testing.T) {
	m := newModel(sample(), "sample")
	next, _ := m.Update(tui.ResizeMsg{Width: 100, Height: 30})
	m = next.(model)
	m.prev.SetContent(strings.Repeat("line\n", 100))
	r := m.geometry().preview
	next, _ = m.Update(tui.MouseEvent{X: r.X + 2, Y: r.Y + 2, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if next.(model).prev.YOffset() == 0 {
		t.Fatal("the wheel did not scroll the preview")
	}
}

func TestEscapesInNamesCannotReachTheScreen(t *testing.T) {
	if got := ansi.StripANSI(strings.Join(row(item{name: "a\x1b[31mb"}), "|")); strings.Contains(got, "\x1b") {
		t.Fatalf("escape in a name reached the row: %q", got)
	}
}
