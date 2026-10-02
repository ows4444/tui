package tui

import (
	"bytes"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// When a view line contains a tab, the system shall expand it to spaces to
// the next multiple of 8 columns before Program.fit measures or truncates it.
func TestFitExpandsTabsBeforeTruncating(t *testing.T) {
	p := &Program{width: 10}
	got := p.fit([]string{"ab\tcdefgh", "\x1b[31ma\tb\x1b[0m"})
	if want := "ab      cd"; got[0] != want {
		t.Errorf("fit[0] = %q, want %q", got[0], want)
	}
	if want := "\x1b[31ma       b\x1b[0m"; got[1] != want {
		t.Errorf("fit[1] = %q, want %q", got[1], want)
	}
	for i, l := range got {
		if w := ansi.Width(l); w > 10 {
			t.Errorf("line %d width %d exceeds 10", i, w)
		}
	}
}

// When the cell renderer draws a line containing a tab, the system shall
// produce the same cells as the line renderer, without falling back.
func TestCellRendererTabMatchesLineRenderer(t *testing.T) {
	view := "a\tb\n\x1b[1mxx\ty\x1b[0m\n12345678\tz"
	var lb, cb bytes.Buffer
	lp := equivProgram(&lb, 40, false)
	cp := equivProgram(&cb, 40, true)
	lp.model = staticModel{view: view}
	cp.model = staticModel{view: view}
	lp.render()
	cp.render()
	if cp.cells.Reason() != "" || !cp.cells.Valid() {
		t.Fatalf("cell renderer fell back (%q)", cp.cells.Reason())
	}
	if l, c := ansiStrip(lb.String()), ansiStrip(cb.String()); l != c {
		t.Errorf("line %q vs cell %q", l, c)
	}
}

// When a view containing tabs is rendered, the system shall not shift the
// columns of text to the right of the tab.
func TestTabDoesNotShiftFollowingColumns(t *testing.T) {
	for _, cell := range []bool{false, true} {
		var buf bytes.Buffer
		p := equivProgram(&buf, 40, cell)
		p.model = staticModel{view: "ab\tX"}
		p.render()
		if got, want := ansiStrip(buf.String()), "ab      X"; !bytes.Contains([]byte(got), []byte(want)) {
			t.Errorf("cell=%v: %q lacks %q", cell, got, want)
		}
	}
}
