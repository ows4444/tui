package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

var _ tui.CellDrawer = canvas{}

func TestDrawCellsMatchesView(t *testing.T) {
	m, _ := canvas{}.Update(tui.ResizeMsg{Width: 20, Height: 6})
	m, _ = m.Update(tui.Key{Type: tui.KeyRight})
	m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	c := m.(canvas)
	if c.x != 1 || c.y != 1 {
		t.Fatalf("marker at %d,%d, want 1,1", c.x, c.y)
	}
	buf := cellbuf.New(20, 6)
	c.DrawCells(buf, buf.Bounds())
	if got := buf.At(1, 1).Cluster; got != "@" {
		t.Fatalf("cell 1,1 = %q, want the marker", got)
	}
	if v := ansi.StripANSI(c.View()); !strings.Contains(v, "@") || !strings.Contains(v, "q quits") {
		t.Fatalf("View lacks the marker or help: %q", v)
	}
}

func TestMarkerStaysOnCanvas(t *testing.T) {
	var m tui.Model = canvas{}
	m, _ = m.Update(tui.ResizeMsg{Width: 5, Height: 4})
	for range 10 {
		m, _ = m.Update(tui.Key{Type: tui.KeyRight})
		m, _ = m.Update(tui.Key{Type: tui.KeyDown})
	}
	if c := m.(canvas); c.x != 4 || c.y != 2 {
		t.Fatalf("marker at %d,%d, want 4,2", c.x, c.y)
	}
}
