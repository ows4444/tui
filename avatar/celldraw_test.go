package avatar

import (
	"fmt"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
	"github.com/ows4444/tui/theme"
)

// DrawCells puts into the grid the cells View parses to, for every kind of
// picture: names, sizes, plates, poses, an ASCII glyph set.
func TestDrawCellsMatchesView(t *testing.T) {
	for _, name := range []string{"ada", "grace", "linus", "ken", "user-21"} {
		for _, size := range [][2]int{{1, 1}, {8, 4}, {16, 8}, {33, 11}} {
			for _, bg := range []Background{BackgroundNone, BackgroundCircle, BackgroundSquare} {
				for _, e := range []Expression{ExpressionNone, ExpressionMad, ExpressionLove} {
					for _, th := range []theme.Theme{theme.DarkTheme(), theme.LightTheme(), theme.DarkTheme().ASCII()} {
						m := New(name)
						m.Width, m.Height, m.Background, m.Expression, m.Theme = size[0], size[1], bg, e, th
						want := cellbuf.New(size[0], size[1])
						tui.DrawView(want, want.Bounds(), m.View())
						got := cellbuf.New(size[0], size[1])
						m.DrawCells(got, got.Bounds())
						for y := 0; y < size[1]; y++ {
							for x := 0; x < size[0]; x++ {
								a, b := got.At(x, y), want.At(x, y)
								if a.Cluster != b.Cluster || got.Style(a.Style) != want.Style(b.Style) {
									t.Fatalf("%s %v bg %v %v: cell %d,%d is %q %+v, View gives %q %+v",
										name, size, bg, e, x, y, a.Cluster, got.Style(a.Style), b.Cluster, want.Style(b.Style))
								}
							}
						}
					}
				}
			}
		}
	}
}

func TestDrawCellsClipsAndSkipsEmpty(t *testing.T) {
	m := New("clip")
	m.DrawCells(cellbuf.New(4, 2), cellbuf.Rect{})
	buf := cellbuf.New(4, 2) // smaller than the 8x4 avatar
	m.DrawCells(buf, buf.Bounds())
	m.Width = 0
	m.DrawCells(buf, buf.Bounds())
}

func BenchmarkDrawCellsVersusView(b *testing.B) {
	m := New("bench")
	m.Width, m.Height = 16, 8
	buf := cellbuf.New(16, 8)
	b.Run("DrawCells", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			m.DrawCells(buf, buf.Bounds())
		}
	})
	b.Run("View+DrawView", func(b *testing.B) {
		b.ReportAllocs()
		uncached := m
		uncached.cache = nil
		for b.Loop() {
			tui.DrawView(buf, buf.Bounds(), uncached.View())
		}
	})
	_ = fmt.Sprint
}
