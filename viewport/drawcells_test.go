package viewport

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

// sameScreen fails unless the w x h cells of got equal those of View's text.
func sameScreen(t *testing.T, view string, got *cellbuf.Buffer, w, h int) {
	t.Helper()
	want, err := cellbuf.Parse(view)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			wc, gc := cellbuf.Cell{Cluster: " ", Width: 1}, got.At(x, y)
			ws := cellbuf.Style{}
			if x < want.Width() && y < want.Height() {
				wc = want.At(x, y)
				ws = want.Style(wc.Style)
			}
			if wc.Cluster != gc.Cluster || wc.Width != gc.Width || ws != got.Style(gc.Style) {
				t.Fatalf("cell %d,%d: View %q/%d/%+v, DrawCells %q/%d/%+v", x, y,
					wc.Cluster, wc.Width, ws, gc.Cluster, gc.Width, got.Style(gc.Style))
			}
		}
	}
}

func TestDrawCellsMatchesView(t *testing.T) {
	long := strings.Repeat("0123456789", 6)
	cases := map[string]struct {
		content string
		w, h    int
		y, x    int
	}{
		"empty":    {"", 10, 4, 0, 0},
		"short":    {"a\nb", 10, 4, 0, 0},
		"scrolled": {"1\n2\n3\n4\n5\n6\n7", 10, 3, 2, 0},
		"clipped":  {long + "\n" + long, 12, 2, 0, 0},
		"xscroll":  {long + "\nshort", 12, 2, 0, 7},
		"wide":     {"日本語テキスト\nab日本c", 9, 2, 0, 0},
		"widecut":  {"日本語テキスト\nab日本c", 5, 2, 0, 3},
		"styled":   {"\x1b[1;31mred bold\x1b[0m plain \x1b[38;2;1;2;3mrgb\x1b[0m\n\x1b[4mul", 14, 3, 0, 2},
		"link":     {"\x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\ rest", 10, 2, 0, 0},
		"tabs":     {"a\tb", 12, 1, 0, 0},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			m := New(c.w, c.h)
			m.SetContent(c.content)
			m.setYOffset(c.y)
			m.setXOffset(c.x)
			buf := cellbuf.New(c.w+3, c.h+2) // larger than the viewport
			m.DrawCells(buf, buf.Bounds())
			sameScreen(t, m.View(), buf, c.w, c.h)
			// nothing outside Width x Height is drawn
			for y := 0; y < buf.Height(); y++ {
				for x := 0; x < buf.Width(); x++ {
					if (x >= c.w || y >= c.h) && buf.At(x, y).Cluster != " " {
						t.Fatalf("ink outside viewport at %d,%d", x, y)
					}
				}
			}
		})
	}
}

func TestDrawCellsSubRegion(t *testing.T) {
	m := New(6, 2)
	m.SetContent("abcdefgh\nijklmnop")
	buf := cellbuf.New(10, 5)
	m.DrawCells(buf, cellbuf.Rect{X: 2, Y: 1, W: 4, H: 1})
	if got := buf.Lines()[1]; got != "  abcd    " {
		t.Fatalf("row 1 = %q", got)
	}
	if buf.Lines()[2] != strings.Repeat(" ", 10) {
		t.Fatal("drew outside r")
	}
}

func TestDrawCellsZero(t *testing.T) {
	m := New(0, 0)
	buf := cellbuf.New(3, 3)
	m.DrawCells(buf, buf.Bounds())
}
