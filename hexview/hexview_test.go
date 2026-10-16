package hexview

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
)

func plain(m Model) string { return ansi.StripANSI(m.View()) }

func key(s string) tui.Key {
	switch s {
	case "left":
		return tui.Key{Type: tui.KeyLeft}
	case "right":
		return tui.Key{Type: tui.KeyRight}
	case "up":
		return tui.Key{Type: tui.KeyUp}
	case "down":
		return tui.Key{Type: tui.KeyDown}
	case "home":
		return tui.Key{Type: tui.KeyHome}
	case "end":
		return tui.Key{Type: tui.KeyEnd}
	case "pgdown":
		return tui.Key{Type: tui.KeyPgDown}
	}
	return tui.Key{Type: tui.KeyRunes, Text: s, Code: rune(s[0])}
}

func press(m Model, keys ...string) Model {
	for _, k := range keys {
		m, _ = m.Update(key(k))
	}
	return m
}

func bytes40() []byte {
	b := make([]byte, 40)
	for i := range b {
		b[i] = byte('A' + i%26)
	}
	b[3], b[20] = 0, 0x0a
	return b
}

func TestViewLaysOutOffsetHexAndASCII(t *testing.T) {
	got := plain(New([]byte("hello, hex view!\nabc")))
	want := "0000  68 65 6c 6c 6f 2c 20 68  65 78 20 76 69 65 77 21  |hello, hex view!|\n" +
		"0010  0a 61 62 63                                       |.abc|"
	if got != want {
		t.Errorf("View =\n%s\nwant\n%s", got, want)
	}
}

func TestViewOptions(t *testing.T) {
	m := New([]byte("abcdef"))
	m.BytesPerRow, m.GroupSize, m.HideOffset, m.HideASCII = 4, 2, true, true
	if got := plain(m); got != "61 62  63 64\n65 66       " {
		t.Errorf("View = %q", got)
	}
	m = New([]byte("ab"))
	m.BytesPerRow = 4
	m.GroupSize = 99 // no groups
	if got := plain(m); got != "0000  61 62        |ab|" {
		t.Errorf("no groups: %q", got)
	}
	if New(nil).View() == "" || !strings.HasPrefix(plain(New(nil)), "0000") {
		t.Errorf("an empty dump should still draw one row: %q", plain(New(nil)))
	}
	big := New(make([]byte, 0x10001))
	big.Height = 1
	if !strings.HasPrefix(plain(big), "00000000") {
		t.Errorf("a buffer over 64 KiB should use 8-digit offsets: %q", plain(big))
	}
}

func TestCursorMovesByByteAndRow(t *testing.T) {
	m := New(bytes40())
	m = press(m, "right", "right", "down")
	if m.Cursor() != 18 {
		t.Errorf("cursor = %d, want 18", m.Cursor())
	}
	m = press(m, "home")
	if m.Cursor() != 16 {
		t.Errorf("home = %d", m.Cursor())
	}
	m = press(m, "end")
	if m.Cursor() != 31 {
		t.Errorf("end = %d", m.Cursor())
	}
	m = press(m, "G")
	if m.Cursor() != 39 {
		t.Errorf("last = %d", m.Cursor())
	}
	m = press(m, "right", "down") // clamped
	if m.Cursor() != 39 {
		t.Errorf("past the end = %d", m.Cursor())
	}
	m = press(m, "g", "left", "up")
	if m.Cursor() != 0 {
		t.Errorf("start = %d", m.Cursor())
	}
	if New(nil).Cursor() != 0 {
		t.Error("empty cursor")
	}
}

func TestCursorByteIsHighlightedInBothColumns(t *testing.T) {
	m := press(New([]byte("abc")), "right")
	view := m.View()
	if strings.Count(view, "\x1b[") < 4 {
		t.Errorf("the cursor byte is not highlighted twice: %q", view)
	}
	if plain(m) != plain(New([]byte("abc"))) {
		t.Error("the cursor changed the text")
	}
}

func TestSelectAndCopyAsHex(t *testing.T) {
	m := New([]byte("hello world"))
	m = press(m, "v", "right", "right", "right")
	from, to, ok := m.Selection()
	if !ok || from != 0 || to != 3 {
		t.Fatalf("Selection = %d..%d %v", from, to, ok)
	}
	if string(m.SelectedBytes()) != "hell" {
		t.Errorf("SelectedBytes = %q", m.SelectedBytes())
	}
	_, cmd := m.Update(key("y"))
	if cmd == nil {
		t.Error("Copy returned no Cmd")
	}
	if got := hexText(m.SelectedBytes()); got != "68 65 6c 6c" {
		t.Errorf("hexText = %q", got)
	}
	m = press(m, "v") // ends the selection
	if _, _, ok := m.Selection(); ok || string(m.SelectedBytes()) != "l" {
		t.Errorf("after ending: ok=%v bytes=%q", ok, m.SelectedBytes())
	}
	if _, cmd := New(nil).Update(key("y")); cmd != nil {
		t.Error("Copy with no data returned a Cmd")
	}
}

func TestHeightScrollsToKeepTheCursorInSight(t *testing.T) {
	m := New(bytes40()) // 3 rows
	m.Height = 2
	if rows := strings.Count(plain(m), "\n") + 1; rows != 2 {
		t.Fatalf("%d rows shown, want 2", rows)
	}
	m = press(m, "pgdown") // two rows down: the last row
	if !strings.HasPrefix(strings.Split(plain(m), "\n")[1], "0020") {
		t.Errorf("the cursor row is not the last shown:\n%s", plain(m))
	}
	m = press(m, "g")
	if !strings.HasPrefix(plain(m), "0000") {
		t.Errorf("did not scroll back:\n%s", plain(m))
	}
}

func TestSetDataKeepsTheCursorInside(t *testing.T) {
	m := New(bytes40())
	m = press(m, "G")
	m.SetData([]byte("abc"))
	if m.Cursor() != 2 {
		t.Errorf("cursor = %d, want 2", m.Cursor())
	}
	m.SetData(nil)
	if m.Cursor() != 0 || m.SelectedBytes() != nil {
		t.Error("empty data")
	}
}

func mouseEv(a tui.MouseAction, x, y int) tui.MouseEvent {
	return tui.MouseEvent{Button: tui.MouseButtonLeft, Action: a, X: x, Y: y}
}

func TestMouseClickDragAndWheel(t *testing.T) {
	m := New(bytes40())
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 80, H: 5}
	// A click on the 3rd byte of the first row: hex area starts at column 6, a byte every 3.
	m, _ = m.Update(mouseEv(tui.MouseActionPress, 6+2*3, 0))
	m, _ = m.Update(mouseEv(tui.MouseActionRelease, 6+2*3, 0))
	if m.Cursor() != 2 {
		t.Errorf("hex click: cursor %d, want 2", m.Cursor())
	}
	if _, _, ok := m.Selection(); ok {
		t.Error("a click left a selection")
	}
	// A click in the ASCII column: "  |" follows the hex area, so its first character is at asciiStart.
	m, _ = m.Update(mouseEv(tui.MouseActionPress, m.asciiStart()+5, 1))
	if m.Cursor() != 21 {
		t.Errorf("ASCII click: cursor %d, want 21", m.Cursor())
	}
	// Dragging from the first byte to the 20th selects 20 bytes.
	m, _ = m.Update(mouseEv(tui.MouseActionPress, 6, 0))
	m, _ = m.Update(mouseEv(tui.MouseActionMotion, 6+3, 1))
	m, _ = m.Update(mouseEv(tui.MouseActionRelease, 6+3, 1))
	if from, to, ok := m.Selection(); !ok || from != 0 || to != 17 {
		t.Errorf("drag: %d..%d %v", from, to, ok)
	}
	// Off by default.
	off := New(bytes40())
	off.Bounds = m.Bounds
	off, _ = off.Update(mouseEv(tui.MouseActionPress, 6+2*3, 0))
	if off.Cursor() != 0 {
		t.Error("the mouse acted with Mouse off")
	}
	// The wheel scrolls a Height window.
	w := New(make([]byte, 16*10))
	w.Mouse, w.Height, w.Bounds = true, 3, hittest.Rect{W: 80, H: 3}
	w, _ = w.Update(tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress, X: 5, Y: 1})
	if !strings.HasPrefix(plain(w), "0030") {
		t.Errorf("wheel did not scroll three rows: %q", plain(w))
	}
}

func TestLinearize(t *testing.T) {
	m := New([]byte("hi"))
	if got := m.Linearize(); got != `Hex view, 2 bytes, cursor at offset 0 (0x0), value 0x68 "h"` {
		t.Errorf("got %q", got)
	}
	m = press(m, "v", "right")
	if got := m.Linearize(); !strings.HasSuffix(got, ", selected 2 bytes from offset 0") {
		t.Errorf("got %q", got)
	}
	if New(nil).Linearize() != "Hex view: no data" {
		t.Error("empty")
	}
	if got := New([]byte{0}).Linearize(); strings.Contains(got, `"`) {
		t.Errorf("a non-printable byte was quoted: %q", got)
	}
}

func TestLayoutNodeAndBindings(t *testing.T) {
	m := New([]byte("abc"))
	if m.LayoutNode() == nil || len(m.Bindings()) != 12 {
		t.Error("LayoutNode or Bindings")
	}
	var zero Model
	zero.Data = []byte("x")
	if len(zero.Bindings()) != 12 { // a zero KeyMap reads as the default
		t.Error("zero KeyMap")
	}
}
