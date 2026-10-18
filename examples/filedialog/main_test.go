package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func key(m model, t tui.KeyType) (model, tui.Cmd) { return update(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m, _ = update(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func open(t testing.TB, save bool) (model, string) {
	t.Helper()
	root := fixture(t)
	m := newModel(root, save)
	m.root, m.loc = root, time.UTC
	m, _ = update(m, tui.ResizeMsg{Width: 100, Height: 30})
	return m, root
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

func click(t testing.TB, m model, id string, clicks int) (model, tui.Cmd) {
	r := rect(t, m, id)
	return update(m, tui.MouseEvent{X: r.X + min(2, r.W-1), Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
}

func names(m model) []string {
	var out []string
	for _, i := range m.visible() {
		out = append(out, m.fp.Entries()[i].Name)
	}
	return out
}

func selected(m model) string { n, _, _ := m.current(); return n }

func isQuit(c tui.Cmd) bool {
	if c == nil {
		return false
	}
	_, ok := c().(tui.QuitMsg)
	return ok
}

func TestOpenDialogListsFoldersFirstAndHidesDotfiles(t *testing.T) {
	m, _ := open(t, false)
	want := []string{"docs", "empty", "src", "README.md", "data.bin", "logo.png", "main.go", "notes.txt", "util.go"}
	if got := strings.Join(names(m), " "); got != strings.Join(want, " ") {
		t.Fatalf("%s", got)
	}
	out := text(m)
	for _, w := range []string{"Open file", "project", "▸All files", "Go source", "Show hidden (.)", "docs/", "main.go", "9 entries", "Enter Open", "Esc Cancel"} {
		if !strings.Contains(out, w) {
			t.Errorf("screen lacks %q:\n%s", w, out)
		}
	}
	if strings.Contains(out, ".env") {
		t.Fatal("a dotfile is showing")
	}
	if selected(m) != "docs" {
		t.Fatalf("selected %q", selected(m))
	}
}

func TestDotShowsHiddenFilesAndTheCursorSkipsThemWhenHidden(t *testing.T) {
	m, _ := open(t, false)
	m = runes(m, ".")
	if !m.showHidden || !strings.Contains(text(m), ".env") || !strings.Contains(text(m), "[x] Show hidden") {
		t.Fatal("dotfiles not shown")
	}
	for selected(m) != ".env" {
		prev := m.fp.Cursor()
		m, _ = key(m, tui.KeyDown)
		if m.fp.Cursor() == prev {
			t.Fatal("never reached .env")
		}
	}
	m = runes(m, ".")
	if strings.Contains(text(m), ".env") || strings.HasPrefix(selected(m), ".") {
		t.Fatalf("the cursor is left on a hidden file: %q", selected(m))
	}
}

func TestArrowsNeverLandOnAHiddenEntry(t *testing.T) {
	m, _ := open(t, false)
	seen := map[string]bool{}
	for range 30 {
		seen[selected(m)] = true
		m, _ = key(m, tui.KeyDown)
	}
	for n := range seen {
		if strings.HasPrefix(n, ".") {
			t.Fatalf("landed on %q", n)
		}
	}
	for range 30 {
		m, _ = key(m, tui.KeyUp)
	}
	if selected(m) != "docs" {
		t.Fatalf("selected %q after going to the top", selected(m))
	}
}

func TestFiltersNarrowTheFiles(t *testing.T) {
	m, _ := open(t, false)
	m, _ = key(m, tui.KeyF3)
	if got := strings.Join(names(m), " "); got != "docs empty src main.go util.go" {
		t.Fatalf("Go filter: %s", got)
	}
	m, _ = key(m, tui.KeyF3)
	if got := strings.Join(names(m), " "); got != "docs empty src README.md notes.txt" {
		t.Fatalf("Text filter: %s", got)
	}
	m, _ = click(t, m, "filter-3", 1)
	if got := strings.Join(names(m), " "); got != "docs empty src logo.png" || !strings.Contains(text(m), "▸Images") {
		t.Fatalf("Images filter: %s", got)
	}
	m, _ = key(m, tui.KeyF3)
	if m.filter != 0 {
		t.Fatalf("F3 wraps: %d", m.filter)
	}
}

func TestEnterOpensAFolderAndLeftGoesUp(t *testing.T) {
	m, root := open(t, false)
	m, _ = key(m, tui.KeyEnter)
	if m.fp.Dir != filepath.Join(root, "docs") || !strings.Contains(text(m), "guide.md") {
		t.Fatalf("dir %s", m.fp.Dir)
	}
	if got := names(m); len(got) != 1 {
		t.Fatalf("the hidden .draft should not show: %v", got)
	}
	if !strings.Contains(strings.Join(strings.Fields(text(m)), " "), "project › docs") {
		t.Fatalf("path bar:\n%s", text(m))
	}
	m, _ = key(m, tui.KeyLeft)
	if m.fp.Dir != root || selected(m) != "docs" {
		t.Fatalf("dir %s selected %q", m.fp.Dir, selected(m))
	}
	m, _ = key(m, tui.KeyRight)
	m, _ = key(m, tui.KeyBackspace)
	if m.fp.Dir != root {
		t.Fatal("Backspace did not go up")
	}
}

func TestTypeAheadJumpsToAName(t *testing.T) {
	m, _ := open(t, false)
	m = runes(m, "no")
	if selected(m) != "notes.txt" {
		t.Fatalf("selected %q", selected(m))
	}
	m = runes(m, "x") // the word restarts at an x nobody has: the cursor stays
	if selected(m) != "notes.txt" {
		t.Fatalf("selected %q", selected(m))
	}
}

func TestChoosingAFileEndsWithItsPath(t *testing.T) {
	m, root := open(t, false)
	m = runes(m, "main")
	m, cmd := key(m, tui.KeyEnter)
	if m.result != filepath.Join(root, "main.go") || !isQuit(cmd) {
		t.Fatalf("result %q", m.result)
	}
}

func TestOpenButtonNeedsAFile(t *testing.T) {
	m, _ := open(t, false)
	m, cmd := click(t, m, "btn-ok", 1)
	if m.result != "" || isQuit(cmd) || !strings.Contains(m.msg, "Select a file") {
		t.Fatalf("msg %q", m.msg)
	}
	m = runes(m, "util")
	m, cmd = click(t, m, "btn-ok", 1)
	if !strings.HasSuffix(m.result, "util.go") || !isQuit(cmd) {
		t.Fatalf("result %q", m.result)
	}
}

func TestEscCancelsWithNoResult(t *testing.T) {
	m, _ := open(t, false)
	m, cmd := key(m, tui.KeyEsc)
	if m.result != "" || !isQuit(cmd) {
		t.Fatal("Esc should end the dialog with nothing")
	}
	m, _ = open(t, false)
	if _, cmd := click(t, m, "btn-cancel", 1); !isQuit(cmd) {
		t.Fatal("Cancel did not quit")
	}
	if _, cmd := key(m, tui.KeyCtrlC); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit")
	}
}

func TestClickSelectsThenClickingAgainOrDoubleClickOpens(t *testing.T) {
	m, root := open(t, false)
	m, _ = click(t, m, "row-"+itoa(indexOf(m, "src")), 1)
	if selected(m) != "src" || m.fp.Dir != root {
		t.Fatalf("selected %q dir %s", selected(m), m.fp.Dir)
	}
	m, _ = click(t, m, "row-"+itoa(indexOf(m, "src")), 1)
	if m.fp.Dir != filepath.Join(root, "src") {
		t.Fatalf("a second click should open: %s", m.fp.Dir)
	}
	m, _ = key(m, tui.KeyLeft)
	m, _ = click(t, m, "row-"+itoa(indexOf(m, "docs")), 2)
	if m.fp.Dir != filepath.Join(root, "docs") {
		t.Fatalf("double click: %s", m.fp.Dir)
	}
	m, _ = click(t, m, "row-"+itoa(indexOf(m, "guide.md")), 2)
	if !strings.HasSuffix(m.result, "guide.md") {
		t.Fatalf("double click on a file: %q", m.result)
	}
}

func indexOf(m model, name string) int {
	for i, e := range m.fp.Entries() {
		if e.Name == name {
			return i
		}
	}
	return -1
}

func itoa(n int) string { return strconv.Itoa(n) }

func TestPathBarJumpsToAFolder(t *testing.T) {
	m, root := open(t, false)
	m, _ = key(m, tui.KeyEnter)
	m, _ = click(t, m, "crumb-0", 1)
	if m.fp.Dir != root {
		t.Fatalf("dir %s", m.fp.Dir)
	}
	names, paths := m.crumbs()
	if len(names) != 1 || names[0] != "project" || paths[0] != root {
		t.Fatalf("%v %v", names, paths)
	}
}

func TestRealPathBarStartsAtTheRoot(t *testing.T) {
	m := newModel(fixture(t), false)
	names, paths := m.crumbs()
	if names[0] != string(filepath.Separator) || len(names) < 3 || paths[len(paths)-1] != filepath.Clean(m.fp.Dir) {
		t.Fatalf("%v %v", names, paths)
	}
}

func TestSaveAsTakesANameAndAsksBeforeReplacing(t *testing.T) {
	m, root := open(t, true)
	if !m.onName || !strings.Contains(text(m), "Save file as") || !strings.Contains(text(m), "Name ") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(m, "report.txt")
	m, cmd := key(m, tui.KeyEnter)
	if m.result != filepath.Join(root, "report.txt") || !isQuit(cmd) {
		t.Fatalf("result %q", m.result)
	}
	if _, err := os.Stat(m.result); err == nil {
		t.Fatal("the dialog created a file; it must only choose a name")
	}
	m, _ = open(t, true)
	m = runes(m, "main.go")
	m, cmd = key(m, tui.KeyEnter)
	if m.confirm == "" || m.result != "" || isQuit(cmd) || !strings.Contains(text(m), "Replace") {
		t.Fatalf("no overwrite question:\n%s", text(m))
	}
	m = runes(m, "n")
	if m.confirm != "" || m.result != "" {
		t.Fatal("n should go back")
	}
	m, _ = key(m, tui.KeyEnter)
	m, cmd = key(m, tui.KeyEnter)
	if !strings.HasSuffix(m.result, "main.go") || !isQuit(cmd) {
		t.Fatalf("result %q", m.result)
	}
}

func TestSaveNameIsChecked(t *testing.T) {
	m, _ := open(t, true)
	m, _ = key(m, tui.KeyEnter)
	if !strings.Contains(m.msg, "Type a file name") {
		t.Fatalf("msg %q", m.msg)
	}
	m = runes(m, "a/b")
	m, _ = key(m, tui.KeyEnter)
	if !strings.Contains(m.msg, "without slashes") || m.result != "" {
		t.Fatalf("msg %q result %q", m.msg, m.result)
	}
	m, _ = key(m, tui.KeyBackspace)
	m, _ = key(m, tui.KeyBackspace)
	m, _ = key(m, tui.KeyBackspace)
	m = runes(m, "ok")
	if string(m.name) != "ok" {
		t.Fatalf("%q", string(m.name))
	}
}

func TestTabMovesBetweenTheListAndTheNameField(t *testing.T) {
	m, _ := open(t, true)
	m, _ = key(m, tui.KeyTab)
	if m.onName {
		t.Fatal("Tab should leave the field")
	}
	m = runes(m, "ut") // type-ahead, not text
	if selected(m) != "util.go" || len(m.name) != 0 {
		t.Fatalf("selected %q name %q", selected(m), string(m.name))
	}
	m, _ = key(m, tui.KeyTab)
	if !m.onName {
		t.Fatal("Tab should enter the field")
	}
	m, _ = click(t, m, "name", 1)
	if !m.onName {
		t.Fatal("clicking the field")
	}
}

func TestChoosingAFileInSaveModeFillsTheName(t *testing.T) {
	m, root := open(t, true)
	m, _ = key(m, tui.KeyTab)
	m = runes(m, "readme")
	m, _ = key(m, tui.KeyEnter)
	if m.confirm != filepath.Join(root, "README.md") || string(m.name) != "README.md" {
		t.Fatalf("confirm %q name %q", m.confirm, string(m.name))
	}
}

func TestF2SwitchesModes(t *testing.T) {
	m, _ := open(t, false)
	m, _ = key(m, tui.KeyF2)
	if !m.save || !m.onName || !strings.Contains(text(m), "Enter Save") {
		t.Fatal("not Save As")
	}
	m, _ = key(m, tui.KeyF2)
	if m.save || m.onName {
		t.Fatal("not Open")
	}
}

func TestInfoPanelDescribesTheSelection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file modes are Unix permissions; Windows reports its own")
	}
	m, _ := open(t, false)
	m = runes(m, "data")
	out := text(m)
	for _, w := range []string{"data.bin", "File", "Size      10 B", "Modified  2026-10-01 09:30:00", "Mode      -rw-r--r--"} {
		if !strings.Contains(out, w) {
			t.Errorf("lacks %q:\n%s", w, out)
		}
	}
	m, _ = update(m, tui.ResizeMsg{Width: 60, Height: 20})
	if strings.Contains(text(m), "Modified") {
		t.Fatal("the side panel should go in a narrow window")
	}
}

func TestEmptyFolderAndUnreadableFolder(t *testing.T) {
	m, root := open(t, false)
	m = m.goTo(filepath.Join(root, "empty"))
	if !strings.Contains(text(m), "Nothing here") || selected(m) != "" {
		t.Fatalf("\n%s", text(m))
	}
	m, cmd := key(m, tui.KeyEnter)
	if isQuit(cmd) || m.result != "" {
		t.Fatal("Enter on nothing did something")
	}
	m = m.goTo(filepath.Join(root, "no-such-folder"))
	if !strings.Contains(text(m), "Cannot read this folder") {
		t.Fatalf("\n%s", text(m))
	}
	m = m.setFilter(1).goTo(root)
	m = m.setFilter(3)
	m = m.goTo(filepath.Join(root, "empty"))
	if !strings.Contains(text(m), "with the Images filter") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestNamesWithEscapesAreSanitized(t *testing.T) {
	root := filepath.Join(t.TempDir(), "p")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "evil\x1b[2Jname"), []byte("x"), 0o644); err != nil {
		t.Skip("this file system will not hold that name")
	}
	m, _ := update(newModel(root, false), tui.ResizeMsg{Width: 100, Height: 30})
	if strings.ContainsRune(m.View(), 0x1b) && strings.Contains(m.View(), "\x1b[2J") {
		t.Fatal("an escape from a file name reached the screen")
	}
}

func TestWheelMovesTheSelection(t *testing.T) {
	m, _ := open(t, false)
	m, _ = update(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if selected(m) != "README.md" {
		t.Fatalf("selected %q", selected(m))
	}
	m, _ = update(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if selected(m) != "docs" {
		t.Fatalf("selected %q", selected(m))
	}
}

func TestHiddenBoxByMouseAndHelp(t *testing.T) {
	m, _ := open(t, false)
	m, _ = click(t, m, "hidden", 1)
	if !m.showHidden {
		t.Fatal("the box did not toggle")
	}
	m = runes(m, "?")
	if !strings.Contains(text(m), "show or hide dotfiles") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m, _ = key(m, tui.KeyEsc); m.help || m.result != "" {
		t.Fatal("Esc should only close help")
	}
}

func TestPagingKeys(t *testing.T) {
	m, _ := open(t, false)
	m, _ = key(m, tui.KeyPgDown)
	if selected(m) != "util.go" {
		t.Fatalf("selected %q", selected(m))
	}
	m, _ = key(m, tui.KeyPgUp)
	if selected(m) != "docs" {
		t.Fatalf("selected %q", selected(m))
	}
}
