package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
)

var _ tui.CellDrawer = screen{}

func TestDrawMixesStringAndCellChildren(t *testing.T) {
	buf := cellbuf.New(20, 8)
	s := newScreen()
	s.DrawCells(buf, buf.Bounds())
	lines := strings.Split(buf.String(), "\n")
	if !strings.Contains(lines[0], "Mixed string") {
		t.Fatalf("header row = %q", lines[0])
	}
	if !strings.Contains(lines[6], "█") {
		t.Fatalf("chart's bottom row = %q, want bars", lines[6])
	}
	if !strings.Contains(lines[7], "q quits") {
		t.Fatalf("footer row = %q", lines[7])
	}
}
