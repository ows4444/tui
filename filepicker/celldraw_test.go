package filepicker

import (
	"os"
	"path/filepath"
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
	dir := t.TempDir()
	for _, d := range []string{"alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"a.txt", "日本語.md", "z.go"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cases := map[string]func() Model{
		"top": func() Model { return New(dir) },
		"cursor mid": func() Model {
			m := New(dir)
			m.cursor = 2
			return m
		},
		"scrolled": func() Model {
			m := New(dir)
			m.Height = 2
			m.cursor = 4
			return m
		},
		"unreadable": func() Model { return New(filepath.Join(dir, "missing")) },
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			sameScreen(t, build(), 30, 8)
		})
	}
}
