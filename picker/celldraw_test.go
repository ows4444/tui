package picker

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
	cases := map[string]func() Model{
		"plain":      func() Model { return NewStrings("red", "green", "blue") },
		"cursor mid": func() Model { m := NewStrings("red", "green", "blue"); m.SetCursor(2); return m },
		"empty":      func() Model { return NewStrings() },
		"wide runes": func() Model { return NewStrings("日本語", "😀 smile", "café") },
		"filter on, no query": func() Model {
			m := NewStrings("red", "green")
			m.Filter = true
			return m
		},
		"filtered": func() Model {
			m := NewStrings("Red apple", "Green apple", "Banana", "grape")
			m.Filter = true
			return typeQuery(m, "ap")
		},
		"no matches": func() Model {
			m := NewStrings("a", "b")
			m.Filter = true
			return typeQuery(m, "zz")
		},
		"control chars falls back": func() Model { return NewStrings("a\x1b[31mb", "c") },
	}
	for name, build := range cases {
		t.Run(name, func(t *testing.T) {
			sameScreen(t, build(), 24, 8)
		})
	}
}

// A region smaller than the list clips: the rows that fit are drawn.
func TestDrawCellsClipsToTheRegion(t *testing.T) {
	m := NewStrings("one", "two", "three", "four")
	buf := cellbuf.New(10, 2)
	m.DrawCells(buf, buf.Bounds())
	if got := buf.At(2, 1).Cluster; got != "t" {
		t.Errorf("row 1 starts with %q, want t of two", got)
	}
	m.DrawCells(buf, cellbuf.Rect{X: 0, Y: 0, W: 0, H: 0}) // empty region: no panic
}

// A frame of a 24-row list: through View (build the string, parse it into
// cells) against DrawCells (write the cells directly).
func benchModel() Model {
	items := make([]string, 24)
	for i := range items {
		items[i] = "item number " + string(rune('a'+i%26)) + " with some text"
	}
	m := NewStrings(items...)
	m.SetCursor(7)
	return m
}

func BenchmarkViewThenParse(b *testing.B) {
	m := benchModel()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := cellbuf.Parse(m.View()); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDrawCells(b *testing.B) {
	m := benchModel()
	buf := cellbuf.New(40, 24)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.DrawCells(buf, buf.Bounds())
	}
}
