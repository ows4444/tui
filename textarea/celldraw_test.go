package textarea

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// sameScreen compares cellbuf.Parse(View()) with the DrawCells buffer, cell by
// cell, cluster, width and style, over a w x h grid.
func sameScreen(t *testing.T, m Model, w, h int) {
	t.Helper()
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatalf("parse View: %v", err)
	}
	got := cellbuf.New(w, h)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a, b := want.At(x, y), got.At(x, y)
			if a.Cluster != b.Cluster || a.Width != b.Width || want.Style(a.Style) != got.Style(b.Style) {
				t.Fatalf("cell (%d,%d): View %q w%d %+v, DrawCells %q w%d %+v\nview: %q",
					x, y, a.Cluster, a.Width, want.Style(a.Style), b.Cluster, b.Width, got.Style(b.Style), m.View())
			}
		}
	}
}

// Criterion #47: DrawCells shows the screen View shows.
func TestDrawCellsMatchesView(t *testing.T) {
	styled := func(m *Model) {
		m.TextStyle = ansi.NewStyle().Foreground(ansi.Green).Bold()
		m.CursorStyle = ansi.NewStyle().Background(ansi.RGB{R: 10, G: 20, B: 30})
	}
	cases := []struct {
		name  string
		set   func(*Model)
		text  string
		moves []int
	}{
		{"empty", func(m *Model) {}, "", []int{0}},
		{"placeholder", func(m *Model) { m.Placeholder = "type here" }, "", []int{0}},
		{"placeholder multiline", func(m *Model) { m.Placeholder = "one\n\ntwo" }, "", []int{0}},
		{"plain", func(m *Model) {}, "one\ntwo two\n" + family + "x\nend", []int{0, 2, 3, 5, 12, 14, 16}},
		{"styled", styled, "one\ntwo two\nend", []int{0, 4, 14}},
		{"wide", func(m *Model) {}, "世界ab\n世界", []int{0, 1, 2, 4, 6, 8}},
		{"scrolled", func(m *Model) { m.Width = 6 }, "abcdefghijklmnop\nshort\n" + family + "ab" + family + "cdefgh", []int{0, 3, 10, 17, 22, 30}},
		{"scrolled wide", func(m *Model) { m.Width = 5; styled(m) }, "世界世界世界ab\nx", []int{0, 3, 6, 8, 10}},
		{"wrapped", func(m *Model) { m.Width = 5; m.SoftWrap = true }, "abcdefghijkl\nxy\n" + family + family + "zz", []int{0, 4, 5, 9, 12, 13, 20}},
		{"wrapped styled", func(m *Model) { m.Width = 4; m.SoftWrap = true; styled(m) }, "世界世界ab\nabcd", []int{0, 2, 4, 6, 8, 11}},
		{"windowed", func(m *Model) { m.Height = 2 }, "a\nbb\nccc\ndddd\neeeee", []int{0, 2, 5, 9, 15}},
		{"wrapped window", func(m *Model) { m.Width = 4; m.SoftWrap = true; m.Height = 3 }, "abcdefghijklmnop\nqrs", []int{0, 5, 9, 15, 17, 20}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := New()
			c.set(&m)
			m.SetValue(c.text)
			sameScreen(t, m, 30, 12) // unfocused: no cursor
			m.Focus()
			for _, pos := range c.moves {
				m.SetCursor(pos)
				m.scrollToCursor()
				sameScreen(t, m, 30, 12)
			}
			m.cursorVisible = false // the dark half of a blink
			sameScreen(t, m, 30, 12)
		})
	}
}

// DrawCells clips to the region it is given.
func TestDrawCellsClips(t *testing.T) {
	m := New()
	m.SetValue("abcdef\nghijkl\nmnop")
	m.Focus()
	buf := cellbuf.New(10, 5)
	m.DrawCells(buf, cellbuf.Rect{X: 2, Y: 1, W: 3, H: 2})
	for y := 0; y < 5; y++ {
		for x := 0; x < 10; x++ {
			in := x >= 2 && x < 5 && y >= 1 && y < 3
			if c := buf.At(x, y); !in && c.Cluster != " " {
				t.Fatalf("(%d,%d) outside the region holds %q", x, y, c.Cluster)
			}
		}
	}
	if got := buf.At(2, 1).Cluster + buf.At(4, 2).Cluster; got != "ai" {
		t.Fatalf("region content %q, want \"ai\"", got)
	}
}
