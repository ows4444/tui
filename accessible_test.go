package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

type linModel struct{ staticModel }

func (linModel) Linearize() string { return "linear one\nlinear two" }

func TestAccessibleImpliesReducedMotionAndOverridesAltScreen(t *testing.T) {
	p := NewProgram(staticModel{}, WithReducedMotion(false), WithAccessible(true), WithAltScreen(true))
	if !p.Accessible() || !p.ReducedMotion() {
		t.Errorf("Accessible=%v ReducedMotion=%v, want both true", p.Accessible(), p.ReducedMotion())
	}
	if p.altScreen {
		t.Error("accessible mode must force the alt screen off, whatever the option order")
	}
	q := NewProgram(staticModel{}, WithAltScreen(true), WithReducedMotion(false))
	if q.Accessible() || !q.altScreen || q.ReducedMotion() {
		t.Error("unset WithAccessible must not change defaults")
	}
}

func TestAccessibleOffRenderUnchanged(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "\x1b[1mhi\x1b[0m"}, WithOutput(out))
	p.render()
	got := string(read())
	if !strings.Contains(got, ansi.SyncOutputEnable) || !strings.Contains(got, "\x1b[1mhi") {
		t.Errorf("non-accessible render changed: %q", got)
	}
}

func TestAccessibleEmitsNoControlSequences(t *testing.T) {
	out, read := captureOutput(t)
	view := "\x1b[1;31mred\x1b[0m " + ansi.Hyperlink("link", "https://x.test") + "\nplain"
	p := NewProgram(staticModel{view: view}, WithOutput(out), WithAccessible(true))
	p.render()
	p.model = staticModel{view: view + "\nmore"}
	p.render()
	got := string(read())
	if got != "red link\r\nplain\r\nmore\r\n" {
		t.Errorf("output = %q, want %q", got, "red link\r\nplain\r\nmore\r\n")
	}
	if strings.Contains(got, "\x1b") {
		t.Errorf("accessible output contains an escape sequence: %q", got)
	}
}

func TestAccessibleAppendsOnlyChangedLinesInOrder(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb\nc"}, WithOutput(out), WithAccessible(true))
	p.render()
	before := len(read())
	p.model = staticModel{view: "a\nB\nc\nd"}
	p.render()
	added := string(read()[before:])
	if added != "B\r\nd\r\n" {
		t.Errorf("added = %q, want only changed lines %q", added, "B\r\nd\r\n")
	}
	before = len(read())
	p.render() // identical frame
	if len(read()) != before {
		t.Error("unchanged frame wrote output")
	}
}

func TestAccessiblePrefersLinearizer(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(linModel{staticModel{view: "VIEW"}}, WithOutput(out), WithAccessible(true))
	p.render()
	if got := string(read()); got != "linear one\r\nlinear two\r\n" {
		t.Errorf("output = %q, want Linearize() text", got)
	}
}

func TestAccessibleResizeAndPrintlnEmitNoCursorMoves(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb"}, WithOutput(out), WithAccessible(true))
	p.render()
	p.repaintFresh()
	p.printLines("note", out)
	p.render() // must not reprint a, b
	got := string(read())
	if got != "a\r\nb\r\nnote\r\n" {
		t.Errorf("output = %q, want %q", got, "a\r\nb\r\nnote\r\n")
	}
}

func TestStripOSC(t *testing.T) {
	cases := map[string]string{
		"a\x1b]8;;u\x07b\x1b]8;;\x1b\\c": "abc",
		"no osc":                         "no osc",
		"x\x1b]52;c;abc":                 "x",
	}
	for in, want := range cases {
		if got := stripOSC(in); got != want {
			t.Errorf("stripOSC(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAccessibleNonLastLineSuffix(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "The \nstatus"}, WithOutput(out), WithAccessible(true))
	p.render()
	before := len(read())
	p.model = staticModel{view: "The quick\nstatus"}
	p.render()
	if got := string(read()[before:]); got != "quick\r\n" {
		t.Errorf("added = %q, want only appended text", got)
	}
}

func TestAccessibleNoEscapeBytes(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a\nb"}, WithOutput(out), WithAccessible(true), WithAltScreen(true), WithMouse(MouseAllMotion))
	p.enterModes()
	p.render()
	_ = p.suspend(func() error { return nil })
	p.render()
	p.leaveModes()
	if bytes.IndexByte(read(), 0x1b) >= 0 {
		t.Errorf("accessible output contains ESC: %q", read())
	}
}

// When a list scrolls by one item, only the newly visible item is written.
func TestAccessibleScrollAnnouncesOnlyNewItem(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "item 1\nitem 2\nitem 3"}, WithOutput(out), WithAccessible(true))
	p.render()
	before := len(read())
	p.model = staticModel{view: "item 2\nitem 3\nitem 4"}
	p.render()
	if added := string(read()[before:]); added != "item 4\r\n" {
		t.Errorf("scroll down announced %q, want only %q", added, "item 4\r\n")
	}
	before = len(read())
	p.model = staticModel{view: "item 1\nitem 2\nitem 3"}
	p.render()
	if added := string(read()[before:]); added != "item 1\r\n" {
		t.Errorf("scroll up announced %q, want only %q", added, "item 1\r\n")
	}
}

func TestAccessibleDuplicateItemsCountedOnce(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "x\nb"}, WithOutput(out), WithAccessible(true))
	p.render()
	before := len(read())
	p.model = staticModel{view: "b\nx\nx"}
	p.render()
	if added := string(read()[before:]); added != "x\r\n" {
		t.Errorf("added = %q, want the one new duplicate", added)
	}
}

func TestAnnouncePoliteDedup(t *testing.T) {
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: "a"}, WithOutput(out), WithAccessible(true))
	p.handleAnnounce(AnnounceWith("saved", Polite)())
	p.handleAnnounce(AnnounceWith("saved", Polite)())
	if got := strings.Count(string(read()), "saved"); got != 1 {
		t.Errorf("polite duplicate written %d times, want 1", got)
	}
	p.handleAnnounce(AnnounceWith("saved", Assertive)())
	if got := strings.Count(string(read()), "saved"); got != 2 {
		t.Errorf("assertive written count %d, want 2", got)
	}
}

// accessibleDelta renders prev then next and returns what the second frame wrote.
func accessibleDelta(t *testing.T, prev, next string) string {
	t.Helper()
	out, read := captureOutput(t)
	p := NewProgram(staticModel{view: prev}, WithOutput(out), WithAccessible(true))
	p.render()
	before := len(read())
	p.model = staticModel{view: next}
	p.render()
	return string(read()[before:])
}

func TestAccessibleNewRowRepeatingOldTextIsAnnounced(t *testing.T) {
	got := accessibleDelta(t, "Status: OK\nName: \nError: x", "Status: Error\nName: \nStatus: OK")
	if want := "Status: Error\r\nStatus: OK\r\n"; got != want {
		t.Errorf("announced %q, want %q", got, want)
	}
}

func TestAccessibleToggledRowsAnnounced(t *testing.T) {
	got := accessibleDelta(t, "[x] a\n[ ] a\n[ ] b", "[ ] a\n[x] a\n[ ] b")
	if want := "[ ] a\r\n[x] a\r\n"; got != want {
		t.Errorf("announced %q, want %q", got, want)
	}
}

func TestAccessibleRemovedThenReappearingRowAnnounced(t *testing.T) {
	got := accessibleDelta(t, "Loading\nbody\nfoot", "Done\nbody\nLoading")
	if want := "Done\r\nLoading\r\n"; got != want {
		t.Errorf("announced %q, want %q", got, want)
	}
}

func TestAccessibleDuplicateLinesNotShiftedAnnounced(t *testing.T) {
	got := accessibleDelta(t, "---\na\n---\nb", "---\nc\n---\n---")
	if want := "c\r\n---\r\n"; got != want {
		t.Errorf("announced %q, want %q", got, want)
	}
}
