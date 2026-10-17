package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func ctrl(m model, c rune) model { return send(m, tui.Key{Type: tui.KeyCtrl, Code: c}) }

func typed(m model, s string) model {
	for _, r := range s {
		if r == '\n' {
			m = key(m, tui.KeyEnter)
			continue
		}
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

const doc = "# Alpha\n\nfirst paragraph\n\n## Beta\n\nsecond paragraph\n\n## Gamma\n\nthird\n"

func open(t testing.TB, st memStore, path string, w, h int) model {
	t.Helper()
	m, err := newModel(st, path)
	if err != nil {
		t.Fatal(err)
	}
	return send(m, tui.ResizeMsg{Width: w, Height: h})
}

func ready(t testing.TB) (model, memStore) {
	st := memStore{"d.md": doc}
	return open(t, st, "d.md", 120, 30), st
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func press(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func clickTool(t testing.TB, m model, id string) model {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no target %q:\n%s", id, text(m))
	}
	return press(m, r.X+1, r.Y)
}

func TestOpensWithThreePanes(t *testing.T) {
	m, _ := ready(t)
	out := text(m)
	for _, want := range []string{"mdstudio", "d.md", "Alpha", "Beta", "Gamma", "first paragraph", "[Save]", "editor", "3 headings", "Ln 1, Col 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	ps := m.panes()
	if len(ps) != 3 || ps[inOutline].W != outlineW || ps[inEditor].X <= ps[inOutline].X {
		t.Fatalf("panes = %+v", ps)
	}
	if m.dirty() || strings.Contains(out, "●") {
		t.Fatal("a freshly opened file shows as modified")
	}
}

func TestMissingFileIsAnError(t *testing.T) {
	if _, err := newModel(memStore{}, "none.md"); err == nil {
		t.Fatal("a missing file opened")
	}
}

func TestSampleOpensWithoutAFile(t *testing.T) {
	m := open(t, memStore{}, "", 120, 30)
	if !strings.Contains(text(m), "Release notes") || !strings.Contains(text(m), "untitled.md") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestPreviewRendersMarkdown(t *testing.T) {
	m := open(t, memStore{"d.md": "# Title\n\nSome **bold** text\n\n- one\n- two\n"}, "d.md", 120, 30)
	out := text(m)
	if !strings.Contains(out, "Some bold text") {
		t.Fatalf("preview:\n%s", out)
	}
}

func TestTypingUpdatesPreviewOutlineAndMarksDirty(t *testing.T) {
	m, _ := ready(t)
	m = typed(m, "## New\n\n") // at the top of the file
	if len(m.hs) != 4 || !m.dirty() {
		t.Fatalf("headings=%d dirty=%v", len(m.hs), m.dirty())
	}
	if !strings.Contains(text(m), "d.md ●") || !strings.Contains(text(m), "4 headings") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestPreviewFollowsTheCursorSection(t *testing.T) {
	long := "# One\n" + strings.Repeat("filler line\n\n", 40) + "## Two\n" + "target text here\n" + strings.Repeat("more\n\n", 40)
	m := open(t, memStore{"l.md": long}, "l.md", 120, 20)
	if strings.Contains(text(m), "target text here") {
		t.Fatal("the target is visible before the cursor goes there")
	}
	m.ed.SetCursorPosition(len(strings.Split(long, "\n"))-30, 0)
	m = m.sync(false)
	if m.syncIdx != 1 || !strings.Contains(text(m), "target text here") {
		t.Fatalf("syncIdx=%d, the preview did not follow:\n%s", m.syncIdx, text(m))
	}
}

func TestOutlineClickJumpsTheEditor(t *testing.T) {
	m, _ := ready(t)
	r := m.panes()[inOutline]
	m = press(m, r.X+3, r.Y+2) // Gamma
	if m.ed.Line() != 8 || m.focus != inEditor || !strings.Contains(text(m), "Ln 9, Col 1") {
		t.Fatalf("line=%d focus=%v\n%s", m.ed.Line(), m.focus, text(m))
	}
}

func TestOutlineKeys(t *testing.T) {
	m, _ := ready(t)
	m = ctrl(ctrl(m, 'w'), 'w') // outline
	if m.focus != inOutline {
		t.Fatalf("focus = %v", m.focus)
	}
	m = key(key(m, tui.KeyDown), tui.KeyDown)
	for i := 0; i < 5; i++ {
		m = key(m, tui.KeyDown)
	}
	if m.outSel != 2 {
		t.Fatalf("outSel = %d past the end", m.outSel)
	}
	m = key(m, tui.KeyEnter)
	if m.ed.Line() != 8 || m.focus != inEditor {
		t.Fatalf("line=%d focus=%v after Enter", m.ed.Line(), m.focus)
	}
	m = ctrl(ctrl(m, 'w'), 'w')
	if m = key(m, tui.KeyEsc); m.focus != inEditor {
		t.Fatal("Esc did not return to the editor")
	}
}

func TestFocusCycleAndPreviewKeys(t *testing.T) {
	m, _ := ready(t)
	m = ctrl(m, 'w')
	if m.focus != inPreview {
		t.Fatalf("focus = %v", m.focus)
	}
	m = send(m, tui.Key{Type: tui.KeyRunes, Text: "]"}) // next heading, handled by the reader
	m = key(m, tui.KeyEsc)
	if m.focus != inEditor || !m.ed.Focused() {
		t.Fatal("Esc in the preview did not give the editor focus back")
	}
	m = ctrl(ctrl(ctrl(m, 'w'), 'w'), 'w')
	if m.focus != inEditor {
		t.Fatalf("three Ctrl+W did not come round: %v", m.focus)
	}
}

func TestBoldWrapsTheSelectionOrMakesAPair(t *testing.T) {
	m := open(t, memStore{"b.md": "hello world\n"}, "b.md", 120, 30)
	m.ed.SetCursor(0)
	m.ed.SetCursorPosition(0, 6)
	m = ctrl(m, 'b')
	if got := m.ed.Value(); got != "hello ****world\n" {
		t.Fatalf("empty selection: %q", got)
	}
	if m.ed.Column() != 8 {
		t.Fatalf("cursor at %d, want between the pairs (8)", m.ed.Column())
	}
	m = open(t, memStore{"b.md": "hello world\n"}, "b.md", 120, 30)
	m.ed.SetCursor(6)
	m.ed.SelectAll()
	m = ctrl(m, 'b')
	if got := m.ed.Value(); got != "**hello world\n**" {
		t.Fatalf("selection: %q", got)
	}
}

func TestToolbarButtons(t *testing.T) {
	m := open(t, memStore{"b.md": "word\n"}, "b.md", 120, 30)
	m = clickTool(t, m, "tb-h")
	if m.ed.Value() != "## word\n" || m.ed.Column() != 3 {
		t.Fatalf("H: %q col %d", m.ed.Value(), m.ed.Column())
	}
	if len(m.hs) != 1 || m.hs[0].level != 2 || m.hs[0].text != "word" {
		t.Fatalf("outline after H: %+v", m.hs)
	}
	m = clickTool(t, m, "tb-list")
	if !strings.HasPrefix(m.ed.Value(), "- ## word") {
		t.Fatalf("List: %q", m.ed.Value())
	}
	m = clickTool(t, m, "tb-link")
	if !strings.Contains(m.ed.Value(), "[text](https://)") {
		t.Fatalf("Link: %q", m.ed.Value())
	}
	m = clickTool(t, m, "tb-code")
	if !strings.Contains(m.ed.Value(), "``") {
		t.Fatalf("Code: %q", m.ed.Value())
	}
	m = clickTool(t, m, "tb-i")
	if !strings.Contains(m.ed.Value(), "**") {
		t.Fatalf("Italic: %q", m.ed.Value())
	}
}

func TestSaveWritesAndClearsTheMark(t *testing.T) {
	m, st := ready(t)
	m = typed(m, "X")
	m = ctrl(m, 's')
	if m.dirty() || !strings.Contains(text(m), "Saved d.md") || !strings.HasPrefix(st["d.md"], "X# Alpha") {
		t.Fatalf("dirty=%v file=%q", m.dirty(), st["d.md"])
	}
}

func TestSaveAsAsksForANameAndEscCancels(t *testing.T) {
	st := memStore{}
	m := open(t, st, "", 120, 30)
	m = ctrl(m, 's')
	if m.mode != modeSaveAs || !strings.Contains(text(m), "Save as ▸") {
		t.Fatalf("\n%s", text(m))
	}
	m = key(m, tui.KeyEsc)
	if m.mode != modeEdit || len(st) != 0 {
		t.Fatal("Esc did not cancel the save")
	}
	m = typed(ctrl(m, 's'), "notes.md")
	m = key(m, tui.KeyEnter)
	if !strings.HasPrefix(st["notes.md"], "# Release notes") || m.path != "notes.md" {
		t.Fatalf("file=%q path=%q", st["notes.md"], m.path)
	}
}

func TestSaveErrorIsShown(t *testing.T) {
	m := open(t, memStore{"readonly.txt": "x"}, "readonly.txt", 120, 30)
	m = ctrl(typed(m, "y"), 's')
	if !strings.Contains(text(m), "Cannot save:") || !m.dirty() {
		t.Fatalf("\n%s", text(m))
	}
}

func TestQuitAsksOnlyWhenChangesWouldBeLost(t *testing.T) {
	m, _ := ready(t)
	if _, cmd := update(m, tui.Key{Type: tui.KeyCtrl, Code: 'q'}); !isQuit(cmd) {
		t.Fatal("Ctrl+Q on a clean file did not quit")
	}
	m = typed(m, "x")
	m, cmd := update(m, tui.Key{Type: tui.KeyCtrl, Code: 'q'})
	if isQuit(cmd) || m.mode != modeDiscard || !strings.Contains(text(m), "Discard unsaved changes?") {
		t.Fatalf("\n%s", text(m))
	}
	if m, _ = update(m, tui.Key{Type: tui.KeyRunes, Text: "n"}); m.mode != modeEdit {
		t.Fatal("n did not cancel")
	}
	m = ctrl(m, 'q')
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"}); !isQuit(cmd) {
		t.Fatal("y did not quit")
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit at once")
	}
}

func TestTwoPaneAndOnePaneLayouts(t *testing.T) {
	m, _ := ready(t)
	m = send(m, tui.ResizeMsg{Width: 80, Height: 24})
	ps := m.panes()
	if len(ps) != 2 {
		t.Fatalf("panes at 80 columns = %+v", ps)
	}
	if _, ok := ps[inOutline]; ok {
		t.Fatal("the outline shows at 80 columns")
	}
	m = send(m, tui.ResizeMsg{Width: 40, Height: 12})
	if len(m.panes()) != 1 {
		t.Fatalf("panes at 40 columns = %+v", m.panes())
	}
	m = clickTool(t, m, "tb-pane")
	if m.focus != inPreview || len(m.panes()) != 1 {
		t.Fatalf("focus=%v panes=%v", m.focus, m.panes())
	}
	if !strings.Contains(text(m), "first paragraph") {
		t.Fatalf("the preview pane is not shown:\n%s", text(m))
	}
	m = ctrl(m, 'w')
	if !strings.Contains(text(m), "Alpha") || m.focus != inOutline {
		t.Fatalf("outline pane:\n%s", text(m))
	}
}

func TestClickInEditorMovesTheCursorAndFocuses(t *testing.T) {
	m, _ := ready(t)
	m = ctrl(m, 'w')
	r := m.panes()[inEditor]
	m = press(m, r.X+m.gutter()+3, r.Y+2) // third line, "first paragraph"
	if m.focus != inEditor || m.ed.Line() != 2 {
		t.Fatalf("focus=%v line=%d", m.focus, m.ed.Line())
	}
}

func TestWheelScrollsThePreview(t *testing.T) {
	long := "# T\n" + strings.Repeat("line\n\n", 80)
	m := open(t, memStore{"l.md": long}, "l.md", 120, 14)
	r := m.panes()[inPreview]
	before := m.pv.YOffset()
	m = send(m, tui.MouseEvent{X: r.X + 2, Y: r.Y + 2, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.pv.YOffset() <= before {
		t.Fatalf("offset %d -> %d after the wheel", before, m.pv.YOffset())
	}
}

func TestHeadingAnchorsMatchEvenWithSetextHeadings(t *testing.T) {
	md := "Setext\n======\n\n# Atx One\n\ntext\n\n## Atx Two\n"
	m := open(t, memStore{"s.md": md}, "s.md", 120, 30)
	a := m.anchors()
	if len(m.hs) != 2 || a == nil || a[0] == "" || a[1] == "" || a[0] == a[1] {
		t.Fatalf("hs=%d anchors=%v", len(m.hs), a)
	}
}

func TestHelp(t *testing.T) {
	m, _ := ready(t)
	m = key(m, tui.KeyF1)
	if !strings.Contains(text(m), "editor, preview, outline in turn") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m, _ := ready(t)
	m = typed(m, "# x\n")
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {99, 24}, {59, 24}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
