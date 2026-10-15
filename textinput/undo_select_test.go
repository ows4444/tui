package textinput

import (
	"io"
	"strings"
	"testing"
	"time"

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

var (
	undoKey = ck('z')
	redoKey = tui.Key{Type: tui.KeyRunes, Text: "z", Mod: input.ModCtrl | input.ModShift}
	shiftR  = tui.Key{Type: tui.KeyRight, Mod: input.ModShift}
	shiftL  = tui.Key{Type: tui.KeyLeft, Mod: input.ModShift}
)

// #70
func TestUndoRedoTypeDeletePaste(t *testing.T) {
	m := focused()
	send(&m, tui.PasteEvent{Text: "hello"})
	m.SetValue("hello") // history starts here
	m.SetCursor(2)

	send(&m, rk('X'), rk('Y'))
	if m.Value() != "heXYllo" {
		t.Fatal(m.Value())
	}
	send(&m, undoKey)
	if m.Value() != "hello" || m.Cursor() != 2 {
		t.Fatalf("undo typing: %q cursor %d", m.Value(), m.Cursor())
	}
	send(&m, redoKey)
	if m.Value() != "heXYllo" || m.Cursor() != 4 {
		t.Fatalf("redo typing: %q cursor %d", m.Value(), m.Cursor())
	}
	send(&m, undoKey, tui.Key{Type: tui.KeyBackspace})
	if m.Value() != "hllo" || m.Cursor() != 1 {
		t.Fatalf("backspace: %q %d", m.Value(), m.Cursor())
	}
	send(&m, undoKey)
	if m.Value() != "hello" || m.Cursor() != 2 {
		t.Fatalf("undo backspace: %q %d", m.Value(), m.Cursor())
	}
	send(&m, tui.Key{Type: tui.KeyDelete}, undoKey)
	if m.Value() != "hello" || m.Cursor() != 2 {
		t.Fatalf("undo delete: %q %d", m.Value(), m.Cursor())
	}
	send(&m, tui.PasteEvent{Text: "ABC"})
	if m.Value() != "heABCllo" {
		t.Fatal(m.Value())
	}
	send(&m, undoKey)
	if m.Value() != "hello" || m.Cursor() != 2 {
		t.Fatalf("undo paste: %q %d", m.Value(), m.Cursor())
	}
	send(&m, tui.Key{Type: tui.KeyRunes, Text: "y", Mod: input.ModCtrl}) // ctrl+y redo
	if m.Value() != "heABCllo" {
		t.Fatalf("ctrl+y redo: %q", m.Value())
	}
}

func TestUndoRebindable(t *testing.T) {
	m := focused()
	m.KeyMap = DefaultKeyMap()
	m.KeyMap.Undo.Keys = []string{"alt+u"}
	m.KeyMap.Redo.Keys = []string{"alt+r"}
	send(&m, rk('a'))
	send(&m, undoKey)
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
	var found bool
	for _, b := range m.Bindings() {
		found = found || (b.Desc == "undo" && b.Keys[0] == "alt+u")
	}
	if !found {
		t.Fatal("Bindings does not list the rebound undo")
	}
}

// #71
func TestUndoLimit(t *testing.T) {
	m := focused()
	m.UndoLimit = 3
	m.SetValue(strings.Repeat("x", 20))
	for i := 0; i < 10; i++ {
		send(&m, tui.Key{Type: tui.KeyBackspace})
	}
	for i := 0; i < 10; i++ {
		send(&m, undoKey)
	}
	if len(m.Value()) != 13 {
		t.Fatalf("undid %d steps, want 3", len(m.Value())-10)
	}
	m = focused()
	m.UndoLimit = -1
	send(&m, rk('a'), undoKey)
	if m.Value() != "a" {
		t.Fatal("undo ran with history off")
	}
}

// runCopy runs cmd, the Cmd a Copy or Cut returned, in a Program and returns
// what the Program wrote: the OSC 52 sequence for the copied text.
func runCopy(t *testing.T, cmd tui.Cmd) string {
	t.Helper()
	var out strings.Builder
	// A reader that never reaches EOF, so the Program quits on the Tick, not
	// on the end of its input.
	in, w := io.Pipe()
	defer w.Close()
	p := tui.NewProgram(runAndQuit{cmd}, tui.WithInput(in),
		tui.WithOutput(&out), tui.WithAltScreen(false))
	if _, err := p.Run(); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

// runAndQuit is a Model that runs one Cmd and then quits.
type runAndQuit struct{ cmd tui.Cmd }

func (r runAndQuit) Init() tui.Cmd {
	// The Cmd's mode change is applied by the loop, so quit a moment later
	// rather than in sequence with it.
	return tui.Batch(r.cmd, tui.Tick(100*time.Millisecond, func(time.Time) tui.Msg { return tui.Quit()() }))
}
func (r runAndQuit) Update(tui.Msg) (tui.Model, tui.Cmd) { return r, nil }
func (r runAndQuit) View() string                        { return "" }

func TestShiftArrowsSelectAndHighlight(t *testing.T) {
	m := focused()
	m.SetValue("hello")
	m.SetCursor(1)
	send(&m, shiftR, shiftR)
	lo, hi, ok := m.ed.Selection()
	if !ok || lo != 1 || hi != 3 {
		t.Fatalf("selection %d %d %v", lo, hi, ok)
	}
	v := m.View()
	sel := m.SelectionStyle
	if !strings.Contains(v, sel.Render("e")) || !strings.Contains(v, sel.Render("l")) {
		t.Fatalf("selection not highlighted: %q", v)
	}
	if strings.Contains(v, sel.Render("h")) || strings.Contains(v, sel.Render("o")) {
		t.Fatalf("too much highlighted: %q", v)
	}
	send(&m, tui.Key{Type: tui.KeyRight}) // plain move clears
	if _, _, ok := m.ed.Selection(); ok || strings.Contains(m.View(), sel.Render("e")) {
		t.Fatal("selection survived a plain move")
	}
}

func TestSelectionFallbackStyle(t *testing.T) {
	m := focused()
	m.SelectionStyle = ansi.Style{}
	m.SetValue("ab")
	m.SetCursor(0)
	send(&m, shiftR)
	if !strings.Contains(m.View(), ansi.NewStyle().Reverse().Render("a")) {
		t.Fatalf("no fallback highlight: %q", m.View())
	}
}

// #72
func TestCopySendsOSC52(t *testing.T) {
	m := focused()
	m.SetValue("hello world")
	m.SetCursor(0)
	for i := 0; i < 5; i++ {
		send(&m, shiftR)
	}
	next, cmd := m.Update(ck('c'))
	if cmd == nil {
		t.Fatal("copy returned no Cmd")
	}
	if out := runCopy(t, cmd); !strings.Contains(out, ansi.OSC52Copy("hello")) {
		t.Fatalf("wrote %q", out)
	}
	if next.Value() != "hello world" {
		t.Fatal("copy changed value")
	}
	next.ed.ClearSelection()
	if _, cmd := next.Update(ck('c')); cmd != nil {
		t.Fatal("copy without selection returned a Cmd")
	}
}

func TestCutAndPasteKey(t *testing.T) {
	m := focused()
	m.SetValue("hello world")
	m.SetCursor(6)
	send(&m, shiftR, shiftR, shiftR, shiftR, shiftR)
	next, cmd := m.Update(ck('x'))
	m = next
	if cmd == nil || m.Value() != "hello " {
		t.Fatalf("cut: cmd nil = %v, value %q", cmd == nil, m.Value())
	}
	if out := runCopy(t, cmd); !strings.Contains(out, ansi.OSC52Copy("world")) {
		t.Fatalf("cut wrote %q", out)
	}
	// The Paste key asks the Program; its answer is a PasteEvent.
	send(&m, ck('v'), tui.PasteEvent{Text: "world"})
	if m.Value() != "hello world" {
		t.Fatalf("paste: %q", m.Value())
	}
	send(&m, undoKey)
	if m.Value() != "hello " {
		t.Fatalf("undo paste: %q", m.Value())
	}
	send(&m, undoKey)
	if m.Value() != "hello world" {
		t.Fatalf("undo cut: %q", m.Value())
	}
}

func TestDisableCopy(t *testing.T) {
	m := focused()
	m.DisableCopy = true
	m.SetValue("secret")
	m.SetCursor(0)
	send(&m, shiftR, shiftR)
	_, copyCmd := m.Update(ck('c'))
	next, cutCmd := m.Update(ck('x'))
	if copyCmd != nil || cutCmd != nil || next.Value() != "secret" {
		t.Fatalf("copy leaked: copy nil = %v, cut nil = %v, value %q", copyCmd == nil, cutCmd == nil, next.Value())
	}
}

// #73
func TestTypingReplacesSelection(t *testing.T) {
	m := focused()
	m.SetValue("hello")
	m.SetCursor(1)
	send(&m, shiftR, shiftR, shiftR, rk('A'))
	if m.Value() != "hAo" || m.Cursor() != 2 {
		t.Fatalf("typed over: %q %d", m.Value(), m.Cursor())
	}
	send(&m, undoKey)
	if m.Value() != "hello" {
		t.Fatalf("undo: %q", m.Value())
	}
	m.SetCursor(4)
	send(&m, shiftL, shiftL, tui.PasteEvent{Text: "ZZ"})
	if m.Value() != "heZZo" {
		t.Fatalf("paste over: %q", m.Value())
	}
	m.SetCursor(0)
	send(&m, shiftR, shiftR, tui.Key{Type: tui.KeyBackspace})
	if m.Value() != "ZZo" {
		t.Fatalf("backspace over: %q", m.Value())
	}
}

func TestSelectionRespectsCharLimit(t *testing.T) {
	m := focused()
	m.CharLimit = 5
	m.SetValue("hello")
	m.SetCursor(0)
	send(&m, shiftR, shiftR, tui.PasteEvent{Text: "123456"})
	if m.Value() != "12llo" {
		t.Fatalf("%q", m.Value())
	}
}

// #74
func mouse(act tui.MouseAction, x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: act}
}

func TestClickMovesCursorToCell(t *testing.T) {
	m := focused()
	m.Mouse = true
	m.Prompt = "> "
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 30, H: 1}
	m.SetValue("hello")
	cases := []struct{ x, want int }{
		{5, 0}, {6, 0}, {7, 0}, {8, 1}, {11, 4}, {12, 5}, {30, 5},
	}
	for _, c := range cases {
		send(&m, mouse(tui.MouseActionPress, c.x, 2), mouse(tui.MouseActionRelease, c.x, 2))
		if m.Cursor() != c.want {
			t.Errorf("click x=%d: cursor %d, want %d", c.x, m.Cursor(), c.want)
		}
	}
	m.SetCursor(2)
	send(&m, mouse(tui.MouseActionPress, 0, 0)) // outside Bounds
	if m.Cursor() != 2 {
		t.Fatal("click outside moved the cursor")
	}
	m.Mouse = false
	send(&m, mouse(tui.MouseActionPress, 8, 2))
	if m.Cursor() != 2 {
		t.Fatal("Mouse off still handled the click")
	}
}

func TestClickWideRunes(t *testing.T) {
	m := focused()
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 1}
	m.SetValue("a世界b") // columns: a0 世1-2 界3-4 b5
	for x, want := range map[int]int{0: 0, 1: 1, 2: 1, 3: 2, 4: 2, 5: 3, 6: 4} {
		send(&m, mouse(tui.MouseActionPress, x, 0))
		if m.Cursor() != want {
			t.Errorf("x=%d: cursor %d, want %d", x, m.Cursor(), want)
		}
	}
}

func TestClickAccountsForScroll(t *testing.T) {
	m := focused()
	m.Mouse = true
	m.Width = 6
	m.Bounds = hittest.Rect{W: 6, H: 1}
	m.SetValue("0123456789") // cursor at end: window scrolled
	start, _ := m.window()
	if start == 0 {
		t.Fatal("expected a scrolled window")
	}
	send(&m, mouse(tui.MouseActionPress, 1, 0))
	if m.Cursor() != start+1 {
		t.Fatalf("cursor %d, want %d", m.Cursor(), start+1)
	}
}

func TestMouseDragSelects(t *testing.T) {
	m := focused()
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 1}
	m.SetValue("hello world")
	send(&m, mouse(tui.MouseActionPress, 2, 0), mouse(tui.MouseActionMotion, 4, 0), mouse(tui.MouseActionMotion, 7, 0))
	if got := m.ed.SelectedText(); got != "llo w" {
		t.Fatalf("drag selected %q", got)
	}
	send(&m, mouse(tui.MouseActionRelease, 7, 0), mouse(tui.MouseActionMotion, 1, 0))
	if got := m.ed.SelectedText(); got != "llo w" {
		t.Fatalf("motion after release changed selection: %q", got)
	}
	_, cmd := m.Update(ck('c'))
	if cmd == nil {
		t.Fatal("copy after drag returned no Cmd")
	}
	if out := runCopy(t, cmd); !strings.Contains(out, ansi.OSC52Copy("llo w")) {
		t.Fatalf("copy after drag wrote %q", out)
	}
	if !strings.Contains(m.View(), m.SelectionStyle.Render("l")) {
		t.Fatal("drag selection not highlighted")
	}
}

// The Model does no I/O and holds no copied text: Copy and Paste return Cmds
// for the Program, and the paste arrives as a PasteEvent.
func TestCopyAndPasteGoThroughTheProgram(t *testing.T) {
	m := focused()
	m.SetValue("hello")
	m.SetCursor(0)
	send(&m, shiftR, shiftR)
	next, cmd := m.Update(ck('c'))
	if cmd == nil {
		t.Fatal("Copy returned no Cmd")
	}
	if next.Value() != "hello" {
		t.Fatalf("copy changed the value: %q", next.Value())
	}
	next, cmd = next.Update(ck('v'))
	if cmd == nil {
		t.Fatal("Paste returned no Cmd")
	}
	if next.Value() != "hello" {
		t.Fatalf("the paste key inserted %q before the Program answered", next.Value())
	}
	next.ed.ClearSelection()
	next.SetCursor(5)
	next, _ = next.Update(tui.PasteEvent{Text: "he"})
	if next.Value() != "hellohe" {
		t.Fatalf("after the PasteEvent: %q", next.Value())
	}
	// Nothing selected: nothing to copy, no Cmd.
	if _, cmd := next.Update(ck('c')); cmd != nil {
		t.Error("Copy with no selection returned a Cmd")
	}
}
