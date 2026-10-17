package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
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

func shift(m model, t tui.KeyType) model {
	return send(m, tui.Key{Type: t, Mod: input.ModShift})
}

func ctrl(m model, c rune) model { return send(m, tui.Key{Type: tui.KeyCtrl, Code: c}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func open(t testing.TB, data []byte) (model, memStore) {
	t.Helper()
	st := memStore{"f.bin": data}
	m, err := newModel(st, "f.bin")
	if err != nil {
		t.Fatal(err)
	}
	return send(m, tui.ResizeMsg{Width: 120, Height: 24}), st
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func seq(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i)
	}
	return b
}

func TestShowsOffsetsHexAndText(t *testing.T) {
	m, _ := open(t, []byte("Hello, hex world!\x00\x01"))
	out := text(m)
	for _, want := range []string{"hexedit", "00000000  48 65 6c 6c 6f 2c 20 68  65 78 20 77 6f 72 6c 64", "Hello, hex world", "00000010", "f.bin", "19 bytes", "00 01 02"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "●") {
		t.Error("a fresh file shows as modified")
	}
}

func TestSampleOpensWithoutAFile(t *testing.T) {
	m, err := newModel(memStore{}, "")
	if err != nil || !strings.HasPrefix(string(m.buf.data), "GIF89a") || len(m.buf.data) < 300 {
		t.Fatalf("err=%v len=%d", err, len(m.buf.data))
	}
}

func TestMissingFileIsAnError(t *testing.T) {
	if _, err := newModel(memStore{}, "none.bin"); err == nil {
		t.Fatal("a missing file opened")
	}
}

func TestMovingStaysInsideAndScrolls(t *testing.T) {
	m, _ := open(t, seq(1000))
	m = send(m, tui.ResizeMsg{Width: 120, Height: 12})
	m = key(m, tui.KeyDown)
	if m.cur != 16 {
		t.Fatalf("Down: cur = %d", m.cur)
	}
	m = key(key(m, tui.KeyRight), tui.KeyLeft)
	m = key(m, tui.KeyEnd)
	if m.cur != 31 {
		t.Fatalf("End: cur = %d", m.cur)
	}
	m = key(m, tui.KeyHome)
	if m.cur != 16 {
		t.Fatalf("Home: cur = %d", m.cur)
	}
	for i := 0; i < 20; i++ {
		m = key(m, tui.KeyDown)
	}
	if m.top == 0 || m.cur/16 < m.top || m.cur/16 >= m.top+m.bodyH() {
		t.Fatalf("cursor row %d not in view (top %d)", m.cur/16, m.top)
	}
	for i := 0; i < 200; i++ {
		m = key(m, tui.KeyDown)
	}
	if m.cur != 1000 && m.cur/16 != 62 {
		t.Fatalf("cur = %d past the end", m.cur)
	}
	m = key(m, tui.KeyPgUp)
	if m.cur >= 1000-16*5 {
		t.Fatalf("PgUp: cur = %d", m.cur)
	}
	for i := 0; i < 200; i++ {
		m = key(m, tui.KeyUp)
	}
	if m.cur < 0 || m.cur > 15 {
		t.Fatalf("cur = %d at the top", m.cur)
	}
}

func TestTypingHexOverwritesNibbleByNibble(t *testing.T) {
	m, _ := open(t, []byte{0x00, 0x11, 0x22})
	m = runes(m, "a")
	if m.buf.data[0] != 0xa0 || m.nib != 1 || m.cur != 0 {
		t.Fatalf("after one digit: %x nib=%d cur=%d", m.buf.data, m.nib, m.cur)
	}
	m = runes(m, "B")
	if m.buf.data[0] != 0xab || m.nib != 0 || m.cur != 1 {
		t.Fatalf("after two: %x nib=%d cur=%d", m.buf.data, m.nib, m.cur)
	}
	m = runes(m, "f")
	if m.buf.data[1] != 0xf1 {
		t.Fatalf("the low nibble was lost: %x", m.buf.data)
	}
	if !m.dirty() || !strings.Contains(text(m), "●") {
		t.Fatal("no modified mark")
	}
}

func TestOneUndoRevertsAWholeTypedByte(t *testing.T) {
	m, _ := open(t, []byte{0x00, 0x11})
	m = runes(m, "ab")
	m = ctrl(m, 'z')
	if m.buf.data[0] != 0x00 || m.dirty() || m.cur != 0 {
		t.Fatalf("data=%x dirty=%v cur=%d", m.buf.data, m.dirty(), m.cur)
	}
	m = ctrl(m, 'y')
	if m.buf.data[0] != 0xab {
		t.Fatalf("redo: %x", m.buf.data)
	}
	if m = ctrl(ctrl(m, 'y'), 'y'); !strings.Contains(text(m), "Nothing to redo") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestAppendPastTheEnd(t *testing.T) {
	m, _ := open(t, []byte{0x01})
	m = key(m, tui.KeyRight)
	if m.cur != 1 {
		t.Fatalf("cur = %d, want the place after the last byte", m.cur)
	}
	m = runes(m, "7f")
	if !bytes.Equal(m.buf.data, []byte{0x01, 0x7f}) || m.cur != 2 {
		t.Fatalf("data=%x cur=%d", m.buf.data, m.cur)
	}
}

func TestInsertMode(t *testing.T) {
	m, _ := open(t, []byte{0xaa, 0xbb})
	m = key(m, tui.KeyInsert)
	if !m.insert || !strings.Contains(text(m), "INS") {
		t.Fatal("Insert did not switch mode")
	}
	m = runes(m, "12")
	if !bytes.Equal(m.buf.data, []byte{0x12, 0xaa, 0xbb}) || m.cur != 1 {
		t.Fatalf("data=%x cur=%d", m.buf.data, m.cur)
	}
	m = ctrl(m, 'z')
	if !bytes.Equal(m.buf.data, []byte{0xaa, 0xbb}) {
		t.Fatalf("undo of an insert: %x", m.buf.data)
	}
	if m = key(m, tui.KeyInsert); m.insert {
		t.Fatal("Insert did not switch back")
	}
}

func TestTextPaneOverwritesAndInsertsCharacters(t *testing.T) {
	m, _ := open(t, []byte("abc"))
	m = key(m, tui.KeyTab)
	if m.pane != inText {
		t.Fatal("Tab did not switch pane")
	}
	m = runes(m, "XY")
	if string(m.buf.data) != "XYc" || m.cur != 2 {
		t.Fatalf("data=%q cur=%d", m.buf.data, m.cur)
	}
	m = runes(m, "/:q") // data in this pane, not commands
	if string(m.buf.data) != "XY/:q" || m.mode != modeNormal {
		t.Fatalf("data=%q mode=%v", m.buf.data, m.mode)
	}
	m = key(m, tui.KeyInsert)
	m = send(key(m, tui.KeyHome), tui.Key{Type: tui.KeyRunes, Text: "-"})
	if string(m.buf.data) != "-XY/:q" {
		t.Fatalf("insert in text: %q", m.buf.data)
	}
}

func TestDeleteAndBackspace(t *testing.T) {
	m, _ := open(t, []byte{1, 2, 3, 4})
	m = key(m, tui.KeyRight)
	m = key(m, tui.KeyDelete)
	if !bytes.Equal(m.buf.data, []byte{1, 3, 4}) || m.cur != 1 {
		t.Fatalf("Delete: %x cur=%d", m.buf.data, m.cur)
	}
	m = key(m, tui.KeyBackspace)
	if !bytes.Equal(m.buf.data, []byte{3, 4}) || m.cur != 0 {
		t.Fatalf("Backspace: %x cur=%d", m.buf.data, m.cur)
	}
	if m = key(m, tui.KeyBackspace); !bytes.Equal(m.buf.data, []byte{3, 4}) {
		t.Fatal("Backspace at the start removed something")
	}
	m = key(key(m, tui.KeyDelete), tui.KeyDelete)
	if len(m.buf.data) != 0 || m.cur != 0 {
		t.Fatalf("data = %x cur=%d", m.buf.data, m.cur)
	}
	if m = key(m, tui.KeyDelete); len(m.buf.data) != 0 {
		t.Fatal("Delete on an empty file did something")
	}
}

func TestSelectionWithShiftArrowsAndDelete(t *testing.T) {
	m, _ := open(t, seq(10))
	m = shift(shift(m, tui.KeyRight), tui.KeyRight)
	lo, hi, ok := m.selection()
	if !ok || lo != 0 || hi != 2 {
		t.Fatalf("selection = %d..%d %v", lo, hi, ok)
	}
	if !strings.Contains(text(m), "sel 3") {
		t.Fatalf("\n%s", text(m))
	}
	m = key(m, tui.KeyDelete)
	if !bytes.Equal(m.buf.data, seq(10)[3:]) || m.cur != 0 {
		t.Fatalf("data=%x cur=%d", m.buf.data, m.cur)
	}
	if _, _, ok := m.selection(); ok {
		t.Fatal("selection survived a delete")
	}
	m = ctrl(m, 'a')
	if lo, hi, _ := m.selection(); lo != 0 || hi != 6 {
		t.Fatalf("Ctrl+A = %d..%d", lo, hi)
	}
	m = key(m, tui.KeyRight)
	if _, _, ok := m.selection(); ok {
		t.Fatal("moving without Shift kept the selection")
	}
}

func TestFindHexAndText(t *testing.T) {
	m, _ := open(t, []byte("xxGIF89axxGIF87a"))
	m = runes(m, "/")
	if m.mode != modeSearch {
		t.Fatal("/ did not open the search")
	}
	m = runes(m, `"GIF8"`)
	m = key(m, tui.KeyEnter)
	if !strings.Contains(text(m), "Found at 0x2") {
		t.Fatalf("\n%s", text(m))
	}
	lo, hi, ok := m.selection()
	if !ok || lo != 2 || hi != 5 {
		t.Fatalf("the match is not selected: %d..%d %v", lo, hi, ok)
	}
	m = runes(m, "n")
	if lo, _, _ := m.selection(); lo != 10 {
		t.Fatalf("n went to %d, want 10", lo)
	}
	m = runes(m, "N")
	if lo, _, _ := m.selection(); lo != 2 {
		t.Fatalf("N went to %d, want 2", lo)
	}
	m = ctrl(m, 'f')
	m = runes(m, "47 49 46")
	m = key(m, tui.KeyEnter)
	if lo, _, _ := m.selection(); lo != 2 && lo != 10 {
		t.Fatalf("hex search found %d", lo)
	}
}

func TestFindErrors(t *testing.T) {
	m, _ := open(t, []byte("abc"))
	if m = runes(m, "n"); m.msg != "Search first with /" {
		t.Fatalf("msg = %q", m.msg)
	}
	m = runes(runes(m, "/"), "zz")
	m = key(m, tui.KeyEnter)
	if !strings.Contains(m.msg, "not a hex byte") {
		t.Fatalf("msg = %q", m.msg)
	}
	m = runes(runes(m, "/"), "ff")
	m = key(m, tui.KeyEnter)
	if !strings.Contains(m.msg, "Not found") {
		t.Fatalf("msg = %q", m.msg)
	}
	m = runes(m, "/")
	if m = key(m, tui.KeyEsc); m.mode != modeNormal {
		t.Fatal("Esc did not close the prompt")
	}
}

func TestGoto(t *testing.T) {
	m, _ := open(t, seq(1000))
	// In order: the relative jumps are from wherever the one before landed.
	for _, c := range []struct {
		in   string
		want int
	}{{"10", 16}, {"0x1f0", 0x1f0}, {"#100", 100}, {"+4", 104}, {"-8", 96}, {"end", 999}, {"ffff", 999}} {
		m = ctrl(m, 'g')
		m.in.SetValue(c.in)
		m = key(m, tui.KeyEnter)
		if m.cur != c.want {
			t.Errorf("goto %q: cur = %d, want %d", c.in, m.cur, c.want)
		}
	}
	m = runes(m, ":")
	m.in.SetValue("nope")
	m = key(m, tui.KeyEnter)
	if !strings.Contains(m.msg, "Type an offset") {
		t.Fatalf("msg = %q", m.msg)
	}
	if m.cur/16 < m.top || m.cur/16 >= m.top+m.bodyH() {
		t.Fatal("goto left the cursor off screen")
	}
}

func TestSaveWritesAndClearsTheMark(t *testing.T) {
	m, st := open(t, []byte{0, 1})
	m = runes(m, "ff")
	m = ctrl(m, 's')
	if m.dirty() || !bytes.Equal(st["f.bin"], []byte{0xff, 1}) || !strings.Contains(text(m), "Saved f.bin (2 bytes)") {
		t.Fatalf("file=%x\n%s", st["f.bin"], text(m))
	}
	m = ctrl(m, 'z')
	if !m.dirty() {
		t.Fatal("undoing after a save should mark the file modified again")
	}
}

func TestEditBackToTheOriginalIsNotDirty(t *testing.T) {
	m, _ := open(t, []byte{0x12})
	m = runes(m, "ff")
	m = key(m, tui.KeyLeft)
	m = runes(m, "12")
	if m.dirty() {
		t.Fatal("restoring the original bytes still shows modified")
	}
}

func TestSaveAsAndError(t *testing.T) {
	st := memStore{}
	m, _ := newModel(st, "")
	m = send(m, tui.ResizeMsg{Width: 120, Height: 24})
	m = ctrl(m, 's')
	if m.mode != modeSaveAs {
		t.Fatal("no Save as prompt")
	}
	m.in.SetValue("out.bin")
	m = key(m, tui.KeyEnter)
	if len(st["out.bin"]) != len(sampleBytes()) || m.path != "out.bin" {
		t.Fatalf("saved %d bytes, path %q", len(st["out.bin"]), m.path)
	}
	m2, _ := open(t, []byte{1})
	m2.st = memStore{}
	m2.path = "readonly.bin"
	m2 = runes(m2, "ff")
	if m2 = ctrl(m2, 's'); !strings.Contains(text(m2), "Cannot save:") || !m2.dirty() {
		t.Fatalf("\n%s", text(m2))
	}
}

func TestQuitAsksOnlyWhenChangesWouldBeLost(t *testing.T) {
	m, _ := open(t, []byte{1})
	if _, cmd := update(m, tui.Key{Type: tui.KeyCtrl, Code: 'q'}); !isQuit(cmd) {
		t.Fatal("Ctrl+Q on a clean file did not quit")
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "q"}); !isQuit(cmd) {
		t.Fatal("q on a clean file did not quit")
	}
	m = runes(m, "ff")
	m, cmd := update(m, tui.Key{Type: tui.KeyCtrl, Code: 'q'})
	if isQuit(cmd) || m.mode != modeDiscard {
		t.Fatalf("mode = %v", m.mode)
	}
	if m, _ = update(m, tui.Key{Type: tui.KeyRunes, Text: "n"}); m.mode != modeNormal {
		t.Fatal("n did not cancel")
	}
	m = ctrl(m, 'q')
	if _, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"}); !isQuit(cmd) {
		t.Fatal("y did not quit")
	}
	if _, cmd := update(m, tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit")
	}
}

func TestClickAndDragSelect(t *testing.T) {
	m, _ := open(t, seq(100))
	press := func(x, y int) model {
		return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	}
	m = press(m.hexX(3), 2+1) // row 1, byte 3
	if m.cur != 19 || m.pane != inHex || !m.drag {
		t.Fatalf("cur=%d pane=%v drag=%v", m.cur, m.pane, m.drag)
	}
	m = send(m, tui.MouseEvent{X: m.hexX(7), Y: 2 + 2, Action: tui.MouseActionMotion})
	lo, hi, ok := m.selection()
	if !ok || lo != 19 || hi != 39 {
		t.Fatalf("drag selection = %d..%d %v", lo, hi, ok)
	}
	m = send(m, tui.MouseEvent{X: 0, Y: 0, Action: tui.MouseActionRelease})
	if m.drag {
		t.Fatal("release did not end the drag")
	}
	m = press(m.textX()+5, 2)
	if m.cur != 5 || m.pane != inText {
		t.Fatalf("click in the text pane: cur=%d pane=%v", m.cur, m.pane)
	}
	if _, _, ok := m.selection(); ok {
		t.Fatal("a plain click left a selection")
	}
}

func TestClickEveryByteOfARow(t *testing.T) {
	m, _ := open(t, seq(64))
	for i := 0; i < 16; i++ {
		got, p := m.byteAt(m.hexX(i)+1, 2)
		if got != i || p != inHex {
			t.Errorf("hex byte %d: got %d in %v", i, got, p)
		}
		got, p = m.byteAt(m.textX()+i, 2)
		if got != i || p != inText {
			t.Errorf("text byte %d: got %d in %v", i, got, p)
		}
	}
	if got, _ := m.byteAt(2, 2); got != -1 {
		t.Error("a click on the offset column hit a byte")
	}
	if got, _ := m.byteAt(m.hexX(0), 1); got != -1 {
		t.Error("a click on the header hit a byte")
	}
}

func TestWheelScrolls(t *testing.T) {
	m, _ := open(t, seq(2000))
	m = send(m, tui.ResizeMsg{Width: 120, Height: 12})
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.top != 3 {
		t.Fatalf("top = %d", m.top)
	}
	m = send(m, tui.MouseEvent{Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.top != 0 {
		t.Fatalf("top = %d", m.top)
	}
}

func TestCopyReturnsAClipboardCommand(t *testing.T) {
	m, _ := open(t, []byte("abcd"))
	m = shift(shift(m, tui.KeyRight), tui.KeyRight)
	m, cmd := update(m, tui.Key{Type: tui.KeyRunes, Text: "y"})
	if cmd == nil || !strings.Contains(text(m), "Copied 3 bytes") {
		t.Fatalf("\n%s", text(m))
	}
	if _, cmd = update(m, tui.Key{Type: tui.KeyRunes, Text: "Y"}); cmd == nil {
		t.Fatal("Y gave no command")
	}
	e, _ := open(t, nil)
	if e, cmd = update(e, tui.Key{Type: tui.KeyRunes, Text: "y"}); cmd != nil || !strings.Contains(text(e), "Nothing to copy") {
		t.Fatalf("copying nothing:\n%s", text(e))
	}
}

func TestEmptyFileCanBeTypedInto(t *testing.T) {
	m, _ := open(t, nil)
	if !strings.Contains(text(m), "0 bytes") {
		t.Fatalf("\n%s", text(m))
	}
	m = runes(m, "4142")
	if string(m.buf.data) != "AB" {
		t.Fatalf("data = %q", m.buf.data)
	}
}

func TestLayoutFollowsWidth(t *testing.T) {
	m, _ := open(t, seq(100))
	for w, want := range map[int]int{120: 16, 77: 16, 76: 8, 44: 8, 43: 4, 20: 4} {
		m = send(m, tui.ResizeMsg{Width: w, Height: 20})
		if m.perRow() != want {
			t.Errorf("width %d: %d bytes a row, want %d", w, m.perRow(), want)
		}
	}
	m = send(m, tui.ResizeMsg{Width: 120, Height: 20})
	if !strings.Contains(text(m), "u16 le") {
		t.Fatalf("no inspector at 120 columns:\n%s", text(m))
	}
	m = send(m, tui.ResizeMsg{Width: 90, Height: 20})
	if strings.Contains(text(m), "u16 le") {
		t.Fatal("inspector shown at 90 columns")
	}
}

func TestHelp(t *testing.T) {
	m, _ := open(t, []byte{1})
	m = key(m, tui.KeyF1)
	if !strings.Contains(text(m), "overwrite a hex digit") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m, _ := open(t, seq(300))
	m = shift(shift(m, tui.KeyRight), tui.KeyDown)
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}, {300, 80}, {104, 24}, {77, 24}, {3, 3}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
