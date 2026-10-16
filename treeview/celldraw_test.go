package treeview

import (
	"testing"

	"github.com/ows4444/tui"
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
	typeQuery := func(m Model, q string) Model {
		for _, r := range q {
			m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: string(r), Code: r})
		}
		return m
	}
	tree := func() Model {
		return New(
			Node{Label: "src", Children: []Node{
				{Label: "main.go"},
				{Label: "widget", Children: []Node{{Label: "button.go"}, {Label: "slider.go"}}},
			}},
			Node{Label: "docs", Children: []Node{{Label: "readme.md"}}},
			Node{Label: "Makefile"},
		)
	}
	open := func(m Model, paths ...string) Model {
		m.expanded = map[string]bool{}
		for _, p := range paths {
			m.expanded[p] = true
		}
		return m
	}
	cases := map[string]func() Model{
		"collapsed":  tree,
		"expanded":   func() Model { return open(tree(), "0", "0.1") },
		"cursor mid": func() Model { m := open(tree(), "0"); m.SetCursor(2); return m },
		"empty":      func() Model { return New() },
		"wide runes": func() Model { return New(Node{Label: "日本語", Children: []Node{{Label: "😀"}}}) },
		"filtered": func() Model {
			m := tree()
			m.Filter = true
			return typeQuery(m, "slid")
		},
		"no matches": func() Model {
			m := tree()
			m.Filter = true
			return typeQuery(m, "zzz")
		},
		"control chars fall back": func() Model { m := tree(); m.Roots[2].Label = "a\x1b[31mb"; m.Raw = true; return m },
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			sameScreen(t, build(), 30, 10)
		})
	}
}

func TestDrawCellsClipsToTheRegion(t *testing.T) {
	m := New(Node{Label: "a"}, Node{Label: "b"}, Node{Label: "c"})
	buf := cellbuf.New(10, 2)
	m.DrawCells(buf, buf.Bounds())
	if got := buf.At(4, 1).Cluster; got != "b" {
		t.Errorf("row 1 label = %q, want b", got)
	}
	m.DrawCells(buf, cellbuf.Rect{}) // empty region: no panic
}
