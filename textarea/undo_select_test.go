package textarea

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
)

func send(m *Model, msgs ...tui.Msg) {
	for _, msg := range msgs {
		*m, _ = m.Update(msg)
	}
}

func focusedWith(s string) Model {
	m := New()
	m.Focus()
	m.SetValue(s)
	return m
}

var (
	undoKey = ck('z')
	redoKey = tui.Key{Type: tui.KeyRunes, Text: "z", Mod: input.ModCtrl | input.ModShift}
	shiftR  = tui.Key{Type: tui.KeyRight, Mod: input.ModShift}
	shiftD  = tui.Key{Type: tui.KeyDown, Mod: input.ModShift}
	bs      = tui.Key{Type: tui.KeyBackspace}
)

// #70
func TestUndoRedoTypeDeletePaste(t *testing.T) {
	m := focusedWith("hello\nworld")
	m.SetCursor(2)
	send(&m, rk('X'), rk('Y'))
	send(&m, undoKey)
	if m.Value() != "hello\nworld" || m.Cursor() != 2 {
		t.Fatalf("undo typing: %q %d", m.Value(), m.Cursor())
	}
	send(&m, redoKey)
	if m.Value() != "heXYllo\nworld" || m.Cursor() != 4 {
		t.Fatalf("redo typing: %q %d", m.Value(), m.Cursor())
	}
	send(&m, undoKey, tui.Key{Type: tui.KeyEnter})
	if m.Value() != "he\nllo\nworld" || m.Cursor() != 3 {
		t.Fatalf("newline: %q %d", m.Value(), m.Cursor())
	}
	send(&m, undoKey)
	if m.Value() != "hello\nworld" || m.Cursor() != 2 {
		t.Fatalf("undo newline: %q %d", m.Value(), m.Cursor())
	}
	m.SetCursor(6) // start of line 2: backspace joins lines
	send(&m, bs)
	if m.Value() != "helloworld" {
		t.Fatal(m.Value())
	}
	send(&m, undoKey)
	if m.Value() != "hello\nworld" || m.Cursor() != 6 {
		t.Fatalf("undo join: %q %d", m.Value(), m.Cursor())
	}
	send(&m, tui.Key{Type: tui.KeyDelete}, undoKey)
	if m.Value() != "hello\nworld" || m.Cursor() != 6 {
		t.Fatalf("undo delete: %q %d", m.Value(), m.Cursor())
	}
	send(&m, ck('k'), undoKey) // delete to line end
	if m.Value() != "hello\nworld" || m.Cursor() != 6 {
		t.Fatalf("undo ctrl+k: %q %d", m.Value(), m.Cursor())
	}
	m.SetCursor(11)
	send(&m, ck('w'), undoKey)
	if m.Value() != "hello\nworld" || m.Cursor() != 11 {
		t.Fatalf("undo ctrl+w: %q %d", m.Value(), m.Cursor())
	}
	m.SetCursor(5)
	send(&m, tui.PasteEvent{Text: "AB CD\nEF"})
	if m.Value() != "helloAB CD\nEF\nworld" {
		t.Fatal(m.Value())
	}
	send(&m, undoKey) // one paste is one step
	if m.Value() != "hello\nworld" || m.Cursor() != 5 {
		t.Fatalf("undo paste: %q %d", m.Value(), m.Cursor())
	}
	send(&m, tui.Key{Type: tui.KeyRunes, Text: "y", Mod: input.ModCtrl})
	if m.Value() != "helloAB CD\nEF\nworld" || m.Cursor() != 13 {
		t.Fatalf("ctrl+y: %q %d", m.Value(), m.Cursor())
	}
}

func TestTypingRunUndoesAsWords(t *testing.T) {
	m := focusedWith("")
	for _, r := range "ab cd" {
		send(&m, rk(r))
	}
	send(&m, undoKey)
	if m.Value() != "ab" {
		t.Fatalf("%q", m.Value())
	}
	send(&m, undoKey)
	if m.Value() != "" {
		t.Fatalf("%q", m.Value())
	}
}

func TestUndoRebindable(t *testing.T) {
	m := focusedWith("")
	m.KeyMap = DefaultKeyMap()
	m.KeyMap.Undo.Keys = []string{"alt+u"}
	m.KeyMap.Redo.Keys = []string{"alt+r"}
	send(&m, rk('a'), undoKey)
	if m.Value() != "a" {
		t.Fatalf("old key still undoes: %q", m.Value())
	}
	send(&m, tui.Key{Type: tui.KeyRunes, Text: "u", Mod: input.ModAlt})
	if m.Value() != "" {
		t.Fatalf("rebound undo: %q", m.Value())
	}
	send(&m, tui.Key{Type: tui.KeyRunes, Text: "r", Mod: input.ModAlt})
	if m.Value() != "a" {
		t.Fatalf("rebound redo: %q", m.Value())
	}
}

// #71
func TestUndoLimit(t *testing.T) {
	m := focusedWith(strings.Repeat("x", 20))
	m.UndoLimit = 3
	for i := 0; i < 10; i++ {
		send(&m, bs)
	}
	for i := 0; i < 10; i++ {
		send(&m, undoKey)
	}
	if len(m.Value()) != 13 {
		t.Fatalf("undid %d steps, want 3", len(m.Value())-10)
	}
	m = focusedWith("")
	m.UndoLimit = -1
	send(&m, rk('a'), undoKey)
	if m.Value() != "a" {
		t.Fatal("undo ran with history off")
	}
}

func TestSetValueForgetsHistory(t *testing.T) {
	m := focusedWith("")
	send(&m, rk('a'))
	m.SetValue("zz")
	send(&m, undoKey)
	if m.Value() != "zz" {
		t.Fatal(m.Value())
	}
}

func TestUndoCopyIsolation(t *testing.T) {
	m := focusedWith("")
	send(&m, rk('a'), ck('u'))
	snap := m
	send(&m, undoKey, rk('q'), rk('r'))
	if snap.Value() != "" {
		t.Fatalf("copy changed: %q", snap.Value())
	}
	send(&snap, undoKey)
	if snap.Value() != "a" {
		t.Fatalf("copy undo: %q", snap.Value())
	}
}

func clip() (func(string) (int, error), *string) {
	var out string
	return func(s string) (int, error) { out += s; return len(s), nil }, &out
}

func TestShiftArrowsSelectAndHighlight(t *testing.T) {
	m := focusedWith("hello\nworld")
	m.SetCursor(1)
	send(&m, shiftR, shiftR)
	if got := m.SelectedText(); got != "el" {
		t.Fatalf("selected %q", got)
	}
	v, sel := m.View(), m.SelectionStyle
	if !strings.Contains(v, sel.Render("e")+sel.Render("l")) || strings.Contains(v, sel.Render("h")) {
		t.Fatalf("not highlighted: %q", v)
	}
	send(&m, shiftD)
	if got := m.SelectedText(); got != "ello\nwor" {
		t.Fatalf("shift+down selected %q", got)
	}
	send(&m, tui.Key{Type: tui.KeyLeft})
	if m.SelectedText() != "" || strings.Contains(m.View(), sel.Render("e")) {
		t.Fatal("selection survived a plain move")
	}
}

// Selection highlighting is the same through DrawCells as through View.
func TestDrawCellsSelectionMatchesView(t *testing.T) {
	cases := []struct {
		name string
		set  func(*Model)
		text string
		a, b int
	}{
		{"plain", func(m *Model) {}, "one\ntwo two\n" + family + "x\nend", 2, 14},
		{"cursor first", func(m *Model) {}, "one\ntwo two", 8, 1},
		{"wide", func(m *Model) {}, "世界ab\n世界", 1, 7},
		{"scrolled", func(m *Model) { m.Width = 6 }, "abcdefghijklmnop\nshort", 3, 18},
		{"wrapped", func(m *Model) { m.Width = 5; m.SoftWrap = true }, "abcdefghijkl\nxy\nzz", 2, 15},
		{"windowed", func(m *Model) { m.Height = 2 }, "a\nbb\nccc\ndddd\neeeee", 2, 12},
		{"wrapped window", func(m *Model) { m.Width = 4; m.SoftWrap = true; m.Height = 3 }, "abcdefghijklmnop\nqrs", 3, 17},
		{"custom style", func(m *Model) { m.SelectionStyle = ansi.NewStyle().Underline() }, "abc def", 1, 5},
		{"fallback style", func(m *Model) { m.SelectionStyle = ansi.Style{} }, "abc def", 1, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := New()
			c.set(&m)
			m.SetValue(c.text)
			m.SetCursor(c.a)
			m.sel = c.a + 1
			m.cursor = c.b
			m.scrollToCursor()
			sameScreen(t, m, 30, 12)
			m.Focus()
			sameScreen(t, m, 30, 12)
			m.cursorVisible = false
			sameScreen(t, m, 30, 12)
		})
	}
}

// #72
func TestCopySendsOSC52(t *testing.T) {
	m := focusedWith("hello\nworld")
	w, out := clip()
	m.ClipboardWrite = w
	m.SetCursor(3)
	send(&m, shiftR, shiftR, shiftR, shiftR)
	send(&m, ck('c'))
	if *out != ansi.OSC52Copy("lo\nw") {
		t.Fatalf("wrote %q", *out)
	}
	if m.Value() != "hello\nworld" {
		t.Fatal("copy changed the value")
	}
	*out = ""
	m.SetCursor(0)
	send(&m, ck('c'))
	if *out != "" {
		t.Fatal("copy without a selection wrote")
	}
}

func TestCutAndPasteKey(t *testing.T) {
	m := focusedWith("hello\nworld")
	w, out := clip()
	m.ClipboardWrite = w
	m.SetCursor(3)
	send(&m, shiftR, shiftR, shiftR, shiftR, ck('x'))
	if m.Value() != "helorld" || *out != ansi.OSC52Copy("lo\nw") || m.Cursor() != 3 {
		t.Fatalf("cut: %q %q %d", m.Value(), *out, m.Cursor())
	}
	send(&m, ck('v'))
	if m.Value() != "hello\nworld" {
		t.Fatalf("paste: %q", m.Value())
	}
	send(&m, undoKey, undoKey)
	if m.Value() != "hello\nworld" || m.Cursor() != 7 {
		t.Fatalf("undo cut: %q %d", m.Value(), m.Cursor())
	}
}

// #73
func TestTypingReplacesSelection(t *testing.T) {
	m := focusedWith("hello\nworld")
	m.SetCursor(3)
	send(&m, shiftR, shiftR, shiftR, shiftR, rk('X'))
	if m.Value() != "helXorld" || m.Cursor() != 4 {
		t.Fatalf("typed over: %q %d", m.Value(), m.Cursor())
	}
	send(&m, undoKey)
	if m.Value() != "hello\nworld" || m.Cursor() != 7 {
		t.Fatalf("undo: %q %d", m.Value(), m.Cursor())
	}
	m.SetCursor(0)
	send(&m, shiftR, shiftR, tui.PasteEvent{Text: "Z\nQ"})
	if m.Value() != "Z\nQllo\nworld" {
		t.Fatalf("paste over: %q", m.Value())
	}
	send(&m, undoKey)
	if m.Value() != "hello\nworld" {
		t.Fatalf("undo paste over: %q", m.Value())
	}
	m.SetCursor(0)
	send(&m, shiftR, shiftR, bs)
	if m.Value() != "llo\nworld" || m.Cursor() != 0 {
		t.Fatalf("backspace over: %q %d", m.Value(), m.Cursor())
	}
	m.SetCursor(0)
	send(&m, shiftR, tui.Key{Type: tui.KeyEnter})
	if m.Value() != "\nlo\nworld" {
		t.Fatalf("enter over: %q", m.Value())
	}
}

func TestSelectAll(t *testing.T) {
	m := focusedWith("a\nb")
	m.SelectAll()
	if m.SelectedText() != "a\nb" {
		t.Fatal(m.SelectedText())
	}
	if lo, hi, ok := m.Selection(); !ok || lo != 0 || hi != 3 {
		t.Fatal(lo, hi, ok)
	}
	send(&m, rk('z'))
	if m.Value() != "z" {
		t.Fatal(m.Value())
	}
}

// #74
func mouse(act tui.MouseAction, x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: act}
}

func click(m *Model, x, y int) {
	send(m, mouse(tui.MouseActionPress, x, y), mouse(tui.MouseActionRelease, x, y))
}

func TestClickMovesCursorToCell(t *testing.T) {
	m := focusedWith("hello\n\nab世界cd")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 4, Y: 3, W: 30, H: 10}
	cases := []struct{ x, y, want int }{
		{4, 3, 0}, {6, 3, 2}, {9, 3, 5}, {20, 3, 5}, // past line end
		{4, 4, 6},                                                            // empty line
		{4, 5, 7}, {6, 5, 9}, {7, 5, 9}, {8, 5, 10}, {9, 5, 10}, {10, 5, 11}, // 世 at cols 2-3, 界 at 4-5
		{20, 5, 13},
		{6, 12, 9}, // below the text: last line
	}
	for _, c := range cases {
		click(&m, c.x, c.y)
		if m.Cursor() != c.want {
			t.Errorf("click (%d,%d): cursor %d, want %d", c.x, c.y, m.Cursor(), c.want)
		}
	}
	m.SetCursor(2)
	click(&m, 0, 0)
	if m.Cursor() != 2 {
		t.Fatal("click outside Bounds moved the cursor")
	}
	m.Mouse = false
	click(&m, 6, 3)
	if m.Cursor() != 2 {
		t.Fatal("Mouse off still handled the click")
	}
}

func TestClickAccountsForHeightWindow(t *testing.T) {
	m := focusedWith("a0\na1\na2\na3\na4\na5")
	m.Mouse = true
	m.Height = 3
	m.Bounds = hittest.Rect{W: 10, H: 3}
	m.SetCursor(m.value.len()) // window shows lines 3..5
	if m.firstRow() != 3 {
		t.Fatalf("window top %d", m.firstRow())
	}
	click(&m, 1, 0)
	if m.Cursor() != 10 { // line 3 starts at offset 9
		t.Fatalf("cursor %d, want 10", m.Cursor())
	}
	click(&m, 0, 1)
	if m.Cursor() != 12 {
		t.Fatalf("cursor %d, want 12", m.Cursor())
	}
}

func TestClickAccountsForHorizontalScroll(t *testing.T) {
	m := focusedWith("0123456789abcdef\nxyz")
	m.Mouse = true
	m.Width = 6
	m.Bounds = hittest.Rect{W: 6, H: 2}
	m.SetCursor(16) // end of line 0: window scrolled right
	x0, _, _ := m.CursorCell()
	if x0 == 0 {
		t.Fatal("expected a scrolled window")
	}
	click(&m, 0, 0)
	start := 16 - x0
	if m.Cursor() != start {
		t.Fatalf("cursor %d, want %d", m.Cursor(), start)
	}
	m.SetCursor(16) // the window follows the cursor, so go back
	click(&m, 2, 0)
	if m.Cursor() != start+2 {
		t.Fatalf("cursor %d, want %d", m.Cursor(), start+2)
	}
	click(&m, 2, 1) // other line shows from its start
	if m.Cursor() != 17+2 {
		t.Fatalf("cursor %d, want 19", m.Cursor())
	}
}

func TestClickWithSoftWrap(t *testing.T) {
	m := focusedWith("abcdefghij\nxy")
	m.Mouse = true
	m.Width, m.SoftWrap = 4, true
	m.Bounds = hittest.Rect{X: 1, Y: 1, W: 4, H: 6}
	// rows: abcd | efgh | ij | xy
	for _, c := range []struct{ x, y, want int }{
		{1, 1, 0}, {3, 1, 2}, {4, 1, 3}, {1, 2, 4}, {3, 2, 6}, {2, 3, 9}, {3, 4, 13}, {1, 4, 11},
	} {
		click(&m, c.x, c.y)
		if m.Cursor() != c.want {
			t.Errorf("click (%d,%d): cursor %d, want %d", c.x, c.y, m.Cursor(), c.want)
		}
	}
	m.Height = 2
	m.SetCursor(m.value.len())
	top := m.firstRow()
	click(&m, 1, 1)
	if rows, _ := m.allRows(); m.Cursor() != rows[top].s {
		t.Fatalf("cursor %d, want start of row %d", m.Cursor(), top)
	}
}

func TestMouseDragSelects(t *testing.T) {
	m := focusedWith("hello\nworld")
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 5}
	send(&m, mouse(tui.MouseActionPress, 2, 0), mouse(tui.MouseActionMotion, 4, 0), mouse(tui.MouseActionMotion, 3, 1))
	if got := m.SelectedText(); got != "llo\nwor" {
		t.Fatalf("drag selected %q", got)
	}
	send(&m, mouse(tui.MouseActionRelease, 3, 1), mouse(tui.MouseActionMotion, 0, 0))
	if got := m.SelectedText(); got != "llo\nwor" {
		t.Fatalf("motion after release changed selection: %q", got)
	}
	if !strings.Contains(m.View(), m.SelectionStyle.Render("l")) {
		t.Fatal("drag selection not highlighted")
	}
	w, out := clip()
	m.ClipboardWrite = w
	send(&m, ck('c'))
	if *out != ansi.OSC52Copy("llo\nwor") {
		t.Fatalf("wrote %q", *out)
	}
	click(&m, 1, 0) // a plain click drops the selection
	if m.SelectedText() != "" {
		t.Fatal("click kept the selection")
	}
}

func TestWheelScrollsWindow(t *testing.T) {
	m := focusedWith(strings.Repeat("line\n", 19) + "line")
	m.Mouse = true
	m.Height = 4
	m.Bounds = hittest.Rect{W: 10, H: 4}
	m.SetCursor(0)
	wheel := func(b tui.MouseButton) tui.MouseEvent {
		return tui.MouseEvent{X: 1, Y: 1, Button: b, Action: tui.MouseActionPress}
	}
	send(&m, wheel(tui.MouseButtonWheelDown))
	if l, _ := m.lineCol(); l != 3 {
		t.Fatalf("line %d after one notch", l)
	}
	send(&m, wheel(tui.MouseButtonWheelDown), wheel(tui.MouseButtonWheelDown))
	if m.firstRow() != 6 {
		t.Fatalf("window top %d, want 6", m.firstRow())
	}
	send(&m, wheel(tui.MouseButtonWheelUp))
	if l, _ := m.lineCol(); l != 6 {
		t.Fatalf("line %d after wheel up", l)
	}
	send(&m, tui.MouseEvent{X: 50, Y: 50, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if l, _ := m.lineCol(); l != 6 {
		t.Fatal("wheel outside Bounds scrolled")
	}
}
