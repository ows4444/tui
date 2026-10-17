package main

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

// project is a small tree: a Go program, a package, docs and a dot file.
func project() fstest.MapFS {
	var long strings.Builder
	for i := 1; i <= 80; i++ {
		fmt.Fprintf(&long, "line %d of the long file with some text to search: needle%d\n", i, i%7)
	}
	return fstest.MapFS{
		"main.go":        {Data: []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")},
		"go.mod":         {Data: []byte("module demo\n\ngo 1.22\n")},
		"README.md":      {Data: []byte("# Demo\n\nA demo project.\n")},
		".env":           {Data: []byte("SECRET=1\n")},
		"pkg/util.go":    {Data: []byte("package pkg\n\n// Add adds.\nfunc Add(a, b int) int { return a + b }\n")},
		"pkg/deep/x.txt": {Data: []byte("deep file\n")},
		"docs/long.txt":  {Data: []byte(long.String())},
		"docs/wide.txt":  {Data: []byte(strings.Repeat("0123456789", 30) + "\n")},
		"bin/app":        {Data: []byte("\x7fELF\x00\x01")},
	}
}

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(newModel(project(), "demo"), tui.ResizeMsg{Width: 100, Height: 24}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

// open selects the row of path p and opens it.
func open(t testing.TB, m model, p string) model {
	t.Helper()
	for i, r := range m.rows() {
		if r.e.path == p {
			m.tCur = i
			m.pane = inTree
			return m.activate(r)
		}
	}
	t.Fatalf("no tree row for %s", p)
	return m
}

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func TestOpensWithTheFirstFileShown(t *testing.T) {
	m := ready()
	out := text(m)
	for _, want := range []string{"codebrowse", "demo/go.mod", "text", "bin", "docs", "pkg", "go.mod", "module demo", "Ln 1/3"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, ".env") {
		t.Error("a dot file shows by default")
	}
}

func TestFoldersOpenAndClose(t *testing.T) {
	m := open(t, ready(), "pkg")
	if !m.open["pkg"] || !strings.Contains(text(m), "util.go") {
		t.Fatalf("pkg did not open:\n%s", text(m))
	}
	m = open(t, m, "pkg/deep")
	if !strings.Contains(text(m), "x.txt") {
		t.Fatalf("nested folder did not open:\n%s", text(m))
	}
	m = key(m, tui.KeyLeft) // closes pkg/deep
	if m.open["pkg/deep"] || strings.Contains(text(m), "x.txt") {
		t.Fatal("Left did not close the folder")
	}
	m = key(m, tui.KeyDown) // util.go
	m = key(m, tui.KeyLeft) // steps out to pkg
	rs := m.rows()
	if rs[m.tCur].e.path != "pkg" {
		t.Fatalf("Left on a file went to %s", rs[m.tCur].e.path)
	}
	if m = key(m, tui.KeyLeft); m.open["pkg"] {
		t.Fatal("Left on an open folder did not close it")
	}
}

func TestOpeningAFileShowsItHighlightedWithLineNumbers(t *testing.T) {
	m := open(t, ready(), "main.go")
	out := text(m)
	for _, want := range []string{"go · Ln 1/7", "package main", "func main()", " 5 "} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
	if m.f.spans == nil {
		t.Fatal("a Go file was not coloured")
	}
	if strings.Contains(out, "\t") {
		t.Fatal("a tab reached the screen")
	}
}

func TestBinaryAndEmptyFilesAreExplained(t *testing.T) {
	m := open(t, open(t, ready(), "bin"), "bin/app")
	if !strings.Contains(text(m), "Binary file, 6 B.") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestMovingInTheFile(t *testing.T) {
	m := open(t, open(t, ready(), "docs"), "docs/long.txt")
	m = key(m, tui.KeyTab)
	if m.pane != inFile {
		t.Fatal("Tab did not move to the file")
	}
	m = key(key(m, tui.KeyDown), tui.KeyDown)
	if m.cur != 2 {
		t.Fatalf("cur = %d", m.cur)
	}
	m = key(m, tui.KeyPgDown)
	if m.cur != 2+m.fileRows()-1 || m.top == 0 && m.cur >= m.fileRows() {
		t.Fatalf("PgDn: cur=%d top=%d", m.cur, m.top)
	}
	m = runes(m, "G")
	if m.cur != 79 || m.top != 80-m.fileRows() {
		t.Fatalf("G: cur=%d top=%d", m.cur, m.top)
	}
	m = runes(m, "g")
	if m.cur != 0 || m.top != 0 {
		t.Fatalf("g: cur=%d top=%d", m.cur, m.top)
	}
	for i := 0; i < 200; i++ {
		m = key(m, tui.KeyUp)
	}
	if m.cur != 0 {
		t.Fatal("moved above the first line")
	}
}

func TestJumpToLine(t *testing.T) {
	m := key(open(t, open(t, ready(), "docs"), "docs/long.txt"), tui.KeyTab)
	m = runes(m, ":")
	if m.prompt != promptLine {
		t.Fatal(": did not ask for a line")
	}
	m = runes(m, "42")
	m = key(m, tui.KeyEnter)
	if m.cur != 41 || m.cur < m.top || m.cur >= m.top+m.fileRows() {
		t.Fatalf("cur=%d top=%d after jumping to 42", m.cur, m.top)
	}
	if !strings.Contains(text(m), "Ln 42/80") {
		t.Fatalf("\n%s", text(m))
	}
	for in, want := range map[string]int{"9999": 79, "end": 79, "$": 79, "1": 0} {
		m = runes(m, ":")
		m.in.SetValue(in)
		m = key(m, tui.KeyEnter)
		if m.cur != want {
			t.Errorf("jump to %q: cur = %d, want %d", in, m.cur, want)
		}
	}
	m = runes(m, ":")
	m.in.SetValue("abc")
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), "Type a line number, or end") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(m, ":")
	if m = key(m, tui.KeyEsc); m.prompt != noPrompt {
		t.Fatal("Esc did not cancel the prompt")
	}
}

func TestSearchFindsAndSteps(t *testing.T) {
	m := key(open(t, open(t, ready(), "docs"), "docs/long.txt"), tui.KeyTab)
	m = runes(m, "/")
	m.in.SetValue("needle3")
	m = key(m, tui.KeyEnter)
	want := 0
	for i := 1; i <= 80; i++ {
		if i%7 == 3 {
			want++
		}
	}
	if len(m.hits) != want || m.cur != 2 || !strings.Contains(text(m), "1 of ") {
		t.Fatalf("hits=%d (want %d) cur=%d\n%s", len(m.hits), want, m.cur, text(m))
	}
	m = runes(m, "n")
	if m.cur != 9 {
		t.Fatalf("n went to %d, want line index 9", m.cur)
	}
	m = runes(runes(m, "N"), "N")
	if m.cur != m.hits[len(m.hits)-1] {
		t.Fatalf("N did not wrap: %d", m.cur)
	}
	m.in.SetValue("zzzz")
	m = runes(m, "/")
	m.in.SetValue("zzzz")
	m = key(m, tui.KeyEnter)
	if len(m.hits) != 0 || !strings.Contains(text(m), "No match") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestSearchMarksMatchingLines(t *testing.T) {
	m := key(open(t, open(t, ready(), "docs"), "docs/long.txt"), tui.KeyTab)
	m = runes(m, "/")
	m.in.SetValue("needle0")
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), "•") && !strings.Contains(text(m), "›") {
		t.Fatalf("no marks:\n%s", text(m))
	}
	var marked int
	for _, line := range strings.Split(text(m), "\n") {
		if strings.Contains(line, "•") {
			marked++
		}
	}
	if marked == 0 {
		t.Fatalf("no • marks:\n%s", text(m))
	}
}

func TestSidewaysScroll(t *testing.T) {
	m := key(open(t, open(t, ready(), "docs"), "docs/wide.txt"), tui.KeyTab)
	if m.longest() == 0 {
		t.Fatal("a 300-column line should scroll")
	}
	m = key(m, tui.KeyRight)
	if m.left != 8 || !strings.Contains(text(m), "89012345") {
		t.Fatalf("left=%d\n%s", m.left, text(m))
	}
	for i := 0; i < 100; i++ {
		m = key(m, tui.KeyRight)
	}
	if m.left != m.longest() {
		t.Fatalf("left = %d, want the end %d", m.left, m.longest())
	}
	m = key(m, tui.KeyHome)
	if m.left != 0 {
		t.Fatalf("Home: left = %d", m.left)
	}
}

func TestHiddenFilesToggle(t *testing.T) {
	m := runes(ready(), ".")
	if !strings.Contains(text(m), ".env") || !strings.Contains(text(m), "Dot files shown") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(m, ".")
	if strings.Contains(text(m), ".env") {
		t.Fatal(". did not hide them again")
	}
}

func TestClickTreeAndFile(t *testing.T) {
	m := ready()
	tr := rect(t, m, "tree")
	var docs int
	for i, r := range m.rows() {
		if r.e.path == "docs" {
			docs = i
		}
	}
	m = press(m, tr.X+3, tr.Y+docs)
	if !m.open["docs"] {
		t.Fatal("clicking a folder did not open it")
	}
	var long int
	for i, r := range m.rows() {
		if r.e.path == "docs/long.txt" {
			long = i
		}
	}
	m = press(m, tr.X+5, tr.Y+long)
	if m.f.path != "docs/long.txt" || m.tCur != long {
		t.Fatalf("file=%s tCur=%d", m.f.path, m.tCur)
	}
	fr := rect(t, m, "file")
	m = press(m, fr.X+8, fr.Y+4)
	if m.cur != 4 || m.pane != inFile {
		t.Fatalf("cur=%d pane=%v", m.cur, m.pane)
	}
}

func TestWheelScrollsEachPane(t *testing.T) {
	m := open(t, open(t, ready(), "docs"), "docs/long.txt")
	fr := rect(t, m, "file")
	m = send(m, tui.MouseEvent{X: fr.X + 2, Y: fr.Y + 2, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.top != 3 || m.cur < m.top {
		t.Fatalf("top=%d cur=%d", m.top, m.cur)
	}
	m = send(m, tui.MouseEvent{X: fr.X + 2, Y: fr.Y + 2, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.top != 0 {
		t.Fatalf("top = %d", m.top)
	}
	tr := rect(t, m, "tree")
	m = send(m, tui.MouseEvent{X: tr.X, Y: tr.Y, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.tTop < 0 {
		t.Fatal("tree top negative")
	}
}

func TestNarrowShowsOnePane(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 50, Height: 16})
	if strings.Contains(text(m), "module demo") {
		t.Fatalf("the file shows beside the tree at 50 columns:\n%s", text(m))
	}
	m = open(t, m, "main.go")
	if m.pane != inFile || !strings.Contains(text(m), "package main") || strings.Contains(text(m), "▸ docs") {
		t.Fatalf("opening a file did not switch to it:\n%s", text(m))
	}
	m = key(m, tui.KeyEsc)
	if m.pane != inTree {
		t.Fatal("Esc did not go back to the tree")
	}
}

func TestUnreadableFolderIsExplained(t *testing.T) {
	m := ready()
	m.errs = map[string]string{"pkg": "permission denied"}
	m.open = map[string]bool{".": true, "pkg": true}
	if !strings.Contains(text(m), "✗ permission denied") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "go to a line (a number or end)") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	for _, k := range []tui.Key{{Type: tui.KeyCtrlC}, {Type: tui.KeyRunes, Text: "q"}} {
		if _, cmd := update(ready(), k); !isQuit(cmd) {
			t.Errorf("%v did not quit", k)
		}
	}
	m = key(open(t, ready(), "main.go"), tui.KeyTab)
	m = runes(m, "/")
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); isQuit(cmd) {
		t.Fatal("q quit while typing a search")
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := open(t, open(t, ready(), "docs"), "docs/wide.txt")
	m = key(m, tui.KeyRight)
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {69, 24}, {70, 24}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
