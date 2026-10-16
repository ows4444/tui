package spinner

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
		"default":   New,
		"label":     func() Model { m := New(); m.Label = "Loading"; return m },
		"advanced":  func() Model { m := New(); m.Label = "Working"; m.frame = 3; return m },
		"no frames": func() Model { m := New(); m.Frames = nil; m.Label = "plain"; return m },
		"custom":    func() Model { m := New(); m.Frames = []string{"|", "/", "-", "\\"}; m.Label = "x"; return m },
		"wide":      func() Model { m := New(); m.Frames = []string{"日", "本"}; m.Label = "語"; return m },
		"ascii":     func() Model { m := New().SetTheme(New().Theme.ASCII()); m.Label = "a"; return m },
		"escape falls back": func() Model {
			m := New()
			m.Label = "a\x1b[31mb"
			return m
		},
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			sameScreen(t, build(), 20, 1)
		})
	}
}
