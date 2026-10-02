package logview

import (
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

func TestDrawCellsMatchesView(t *testing.T) {
	mk := func(n int, ind bool, scroll bool) Model {
		m := New(12, 3)
		m.NewLinesIndicator = ind
		for i := 0; i < n; i++ {
			m.Append("\x1b[32mline\x1b[0m 日本 " + string(rune('a'+i%26)))
		}
		if scroll {
			m.Viewport.GotoTop()
			m.Append("late")
			m.Append("later")
		}
		return m
	}
	cases := map[string]Model{
		"empty":         New(12, 3),
		"filled":        mk(6, false, false),
		"scrolled":      mk(6, false, true),
		"indicator":     mk(6, true, true),
		"indicator-off": mk(1, true, false),
	}
	one := mk(6, true, false)
	one.Viewport.GotoTop()
	one.Append("x")
	cases["indicator-one"] = one
	for name, m := range cases {
		t.Run(name, func(t *testing.T) {
			buf := cellbuf.New(14, 5)
			m.DrawCells(buf, buf.Bounds())
			want, _ := cellbuf.Parse(m.View())
			for y := 0; y < 3; y++ {
				for x := 0; x < 12; x++ {
					wc, ws := cellbuf.Cell{Cluster: " ", Width: 1}, cellbuf.Style{}
					if x < want.Width() && y < want.Height() {
						wc = want.At(x, y)
						ws = want.Style(wc.Style)
					}
					gc := buf.At(x, y)
					if wc.Cluster != gc.Cluster || wc.Width != gc.Width || ws != buf.Style(gc.Style) {
						t.Fatalf("cell %d,%d differs: view %q, draw %q\n%q", x, y, wc.Cluster, gc.Cluster, m.View())
					}
				}
			}
		})
	}
}
