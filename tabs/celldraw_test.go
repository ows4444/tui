package tabs

import (
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

// sameScreen fails t unless DrawCells draws what View shows, cell by cell,
// cluster, width and style, over a w x h grid.
func sameScreen(t *testing.T, m Model, w, h int) {
	t.Helper()
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatalf("Parse(View): %v", err)
	}
	got := cellbuf.New(w, h)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			g, e := got.At(x, y), want.At(x, y)
			if g.Cluster != e.Cluster || g.Width != e.Width || got.Style(g.Style) != want.Style(e.Style) {
				t.Fatalf("cell (%d,%d): DrawCells %q w%d %+v, View %q w%d %+v\nView:\n%s", x, y,
					g.Cluster, g.Width, got.Style(g.Style), e.Cluster, e.Width, want.Style(e.Style), m.View())
			}
		}
	}
}

func TestDrawCellsMatchesView(t *testing.T) {
	cases := map[string]func() Model{
		"three":                   func() Model { return New("Files", "Edit", "Help") },
		"second":                  func() Model { m := New("Files", "Edit", "Help"); m.active = 1; return m },
		"one":                     func() Model { return New("Only") },
		"none":                    func() Model { return New() },
		"wide runes":              func() Model { return New("日本語", "😀", "café") },
		"control chars fall back": func() Model { return New("a\x1b[31mb", "c") },
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			sameScreen(t, build(), 40, 1)
		})
	}
}

func TestDrawCellsEmptyRegion(t *testing.T) {
	m := New("a", "b")
	m.DrawCells(cellbuf.New(10, 1), cellbuf.Rect{})
}
