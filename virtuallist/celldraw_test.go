package virtuallist

import (
	"fmt"
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

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
				t.Fatalf("cell (%d,%d): DrawCells %q w%d %+v, View %q w%d %+v", x, y,
					g.Cluster, g.Width, got.Style(g.Style), e.Cluster, e.Width, want.Style(e.Style))
			}
		}
	}
}

func TestDrawCellsMatchesView(t *testing.T) {
	plain := func(i int) string { return fmt.Sprint("item ", i) }
	styled := func(i int) string {
		if i%2 == 0 {
			return fmt.Sprintf("\x1b[1;31mitem\x1b[0m %d \x1b]8;;https://x.test\x1b\\link\x1b]8;;\x1b\\", i)
		}
		return fmt.Sprint("日本語 😀 ", i)
	}
	cases := map[string]Model{
		"top":      New(20, 5, plain),
		"empty":    New(0, 5, plain),
		"no rows":  New(5, 0, plain),
		"short":    New(3, 5, plain),
		"styled":   New(20, 6, styled),
		"nil item": New(5, 5, nil),
		"multiline": New(4, 6, func(i int) string {
			return fmt.Sprint("a", i, "\nb", i)
		}),
	}
	scrolled := New(100, 5, styled)
	scrolled.SetCursor(42)
	cases["scrolled"] = scrolled
	bottom := New(100, 5, plain)
	bottom.SetCursor(99)
	cases["bottom"] = bottom
	for name, m := range cases {
		t.Run(name, func(t *testing.T) { sameScreen(t, m, 40, 8) })
	}
}

func TestDrawCellsCallsRenderItemForVisibleOnly(t *testing.T) {
	var seen []int
	m := New(1000, 4, func(i int) string { seen = append(seen, i); return "x" })
	m.SetCursor(500)
	buf := cellbuf.New(10, 10)
	m.DrawCells(buf, buf.Bounds())
	if len(seen) != 4 {
		t.Errorf("RenderItem called for %v, want 4 items", seen)
	}
}

func TestDrawCellsClipsToRegion(t *testing.T) {
	m := New(10, 6, func(i int) string { return fmt.Sprint("row", i, "-wide") })
	buf := cellbuf.New(20, 10)
	m.DrawCells(buf, cellbuf.Rect{X: 2, Y: 1, W: 5, H: 3})
	if c := buf.At(1, 1); c.Cluster != " " {
		t.Errorf("left of region touched: %q", c.Cluster)
	}
	if c := buf.At(7, 1); c.Cluster != " " {
		t.Errorf("right of region touched: %q", c.Cluster)
	}
	if c := buf.At(2, 4); c.Cluster != " " {
		t.Errorf("below region touched: %q", c.Cluster)
	}
	if c := buf.At(2, 1); c.Cluster != "r" {
		t.Errorf("region origin = %q, want r", c.Cluster)
	}
}
