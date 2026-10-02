package tui

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// placerModel draws a fixed view and asks for the hardware cursor at x, y.
type placerModel struct {
	view string
	x, y int
	ok   bool
}

func (m placerModel) Init() Cmd               { return nil }
func (m placerModel) Update(Msg) (Model, Cmd) { return m, nil }
func (m placerModel) View() string            { return m.view }
func (m placerModel) CursorPos() (int, int, bool) {
	return m.x, m.y, m.ok
}

// frameTail returns what a render wrote after its content: the cursor moves
// between the last row and the end of the synchronized frame.
func frameTail(t *testing.T, read func() []byte, before int) string {
	t.Helper()
	added := string(read()[before:])
	i := strings.LastIndex(added, ansi.SyncOutputDisable)
	if i < 0 {
		t.Fatalf("no synchronized frame end in %q", added)
	}
	return added[:i]
}

// Criterion #98: where the model gives a cursor position, the cursor is moved
// to that cell and shown.
func TestCursorPlacerShowsCursorAtCell(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(placerModel{view: "one\ntwo\nthree", x: 2, y: 1, ok: true}, WithOutput(out))
	p.render()
	got := read()
	want := ansi.CursorUp(1) + "\r" + ansi.CursorForward(2) + ansi.CursorShow + ansi.SyncOutputDisable
	if !strings.HasSuffix(string(got), want) {
		t.Fatalf("frame ends %q, want the cursor moved to row 1 col 2 and shown", string(got))
	}
}

// Criterion #99: with no position the cursor stays hidden and the frame is
// exactly what it was before CursorPlacer existed.
func TestNoCursorPlacerWritesNothingExtra(t *testing.T) {
	plain, readPlain := captureOutput(t)
	pp := NewProgram(staticModel{view: "one\ntwo"}, WithOutput(plain))
	pp.render()

	off, readOff := captureOutput(t)
	po := NewProgram(placerModel{view: "one\ntwo"}, WithOutput(off)) // ok false
	po.render()

	if string(readPlain()) != string(readOff()) {
		t.Fatalf("ok=false frame %q differs from a model without CursorPlacer %q", readOff(), readPlain())
	}
	if strings.Contains(string(readOff()), ansi.CursorShow) {
		t.Fatal("cursor was shown without a position")
	}
}

// Criterion #100: a position outside the screen is clamped to it.
func TestCursorPlacerClampsToView(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(placerModel{view: "one\ntwo", x: 500, y: 99, ok: true}, WithOutput(out))
	p.width, p.height = 10, 5
	p.render()
	want := "\r" + ansi.CursorForward(9) + ansi.CursorShow + ansi.SyncOutputDisable
	if !strings.HasSuffix(string(read()), want) {
		t.Fatalf("frame ends %q, want the cursor clamped to the last row, col 9", string(read()))
	}

	out2, read2 := captureOutput(t)
	p2 := NewProgram(placerModel{view: "one\ntwo", x: -3, y: -3, ok: true}, WithOutput(out2), WithAltScreen(false))
	p2.render()
	want = ansi.CursorUp(1) + "\r" + ansi.CursorShow + ansi.SyncOutputDisable
	if !strings.HasSuffix(string(read2()), want) {
		t.Fatalf("frame ends %q, want the cursor clamped to row 0, col 0", string(read2()))
	}
}

// Criterion #101: the next frame, a print and leaving the terminal start from
// the last row again, so the cursor is hidden and moved back first.
func TestCursorPlacerIsUnparkedBeforeNextWrite(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(placerModel{view: "one\ntwo\nthree", x: 1, y: 0, ok: true}, WithOutput(out))
	p.render()
	before := len(read())

	p.model = placerModel{view: "one\ntwo\nTHREE"} // no cursor now
	p.render()
	added := string(read()[before:])
	want := ansi.SyncOutputEnable + ansi.CursorHide + ansi.CursorDown(2)
	if !strings.HasPrefix(added, want) {
		t.Fatalf("next frame starts %q, want it to hide the cursor and move back down first", added)
	}
	if strings.Contains(added, ansi.CursorShow) || p.curShown || p.curUp != 0 {
		t.Fatal("cursor stayed parked after a frame with no position")
	}

	p.model = placerModel{view: "one\ntwo\nTHREE", x: 0, y: 2, ok: true}
	p.render()
	before = len(read())
	p.printLines("hello", p.output)
	if got := string(read()[before:]); !strings.HasPrefix(got, ansi.CursorHide) {
		t.Fatalf("printLines starts %q, want the cursor unparked first", got)
	}

	p.model = placerModel{view: "x\ny", x: 0, y: 0, ok: true}
	p.render()
	before = len(read())
	p.leaveModes()
	if got := string(read()[before:]); !strings.HasPrefix(got, ansi.CursorHide+ansi.CursorDown(1)) {
		t.Fatalf("leaveModes starts %q, want the cursor unparked first", got)
	}
}

// Criterion #101: a frame skipped because the View is unchanged still follows
// a cursor that moved.
func TestCursorPlacerFollowsWhenViewUnchanged(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(placerModel{view: "abc\ndef", x: 0, y: 0, ok: true}, WithOutput(out))
	p.render()
	before := len(read())

	p.model = placerModel{view: "abc\ndef", x: 2, y: 1, ok: true}
	p.renderIfChanged()
	tail := frameTail(t, read, before)
	if !strings.Contains(tail, ansi.CursorForward(2)) || strings.Contains(tail, "abc") {
		t.Fatalf("moved cursor wrote %q, want only a cursor move", tail)
	}

	before = len(read())
	p.renderIfChanged() // nothing changed at all
	if len(read()) != before {
		t.Fatalf("unchanged view and cursor wrote %q", read()[before:])
	}
}

// Criterion #102: in inline mode the position is relative to the top of the
// View, moved to with relative movement only. Rows a tall view has already
// pushed into scrollback are not part of the live region.
func TestCursorPlacerInlineIsRelativeToView(t *testing.T) {
	out, read := captureOutput(t)
	view := "r0\nr1\nr2\nr3\nr4\nr5"
	p := NewProgram(placerModel{view: view, x: 1, y: 4, ok: true}, WithOutput(out), WithAltScreen(false))
	p.width, p.height = 20, 4 // 6 rows in 4: rows 0 and 1 are committed
	p.render()
	if p.committed != 2 || p.liveLines != 4 {
		t.Fatalf("committed %d, live %d; the fixture no longer overflows", p.committed, p.liveLines)
	}
	want := ansi.CursorUp(1) + "\r" + ansi.CursorForward(1) + ansi.CursorShow + ansi.SyncOutputDisable
	if got := string(read()); !strings.HasSuffix(got, want) {
		t.Fatalf("frame ends %q, want view row 4 mapped to live row 2", got)
	}
	if strings.Contains(string(read()), ansi.CursorPosition(1, 1)) {
		t.Fatal("absolute addressing used")
	}

	// A cursor on a committed row clamps to the top of the live region.
	out2, read2 := captureOutput(t)
	p2 := NewProgram(placerModel{view: view, x: 0, y: 0, ok: true}, WithOutput(out2), WithAltScreen(false))
	p2.width, p2.height = 20, 4
	p2.render()
	want = ansi.CursorUp(3) + "\r" + ansi.CursorShow + ansi.SyncOutputDisable
	if got := string(read2()); !strings.HasSuffix(got, want) {
		t.Fatalf("frame ends %q, want the cursor clamped to the first live row", got)
	}
}
