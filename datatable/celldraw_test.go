package datatable

import (
	"fmt"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/cellbuf"
)

// sameScreen fails t unless the DrawCells buffer shows what View does, cell by
// cell including styles, over a w x h area.
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

func sampleRows(n int) [][]string {
	rows := make([][]string, n)
	for i := range rows {
		rows[i] = []string{fmt.Sprint("name", i), fmt.Sprint(i * 7), "ok"}
	}
	return rows
}

func TestDrawCellsMatchesView(t *testing.T) {
	hdr := []string{"Name", "Count", "State"}
	cases := map[string]func() Model{
		"cursor top": func() Model { return New(hdr, sampleRows(5)) },
		"cursor mid": func() Model { m := New(hdr, sampleRows(5)); m.SetCursor(3); return m },
		"scrolled": func() Model {
			m := New(hdr, sampleRows(30))
			m.Height = 4
			m.SetCursor(17)
			return m
		},
		"scrolled bottom": func() Model {
			m := New(hdr, sampleRows(30))
			m.Height = 4
			m.SetCursor(29)
			return m
		},
		"empty rows":    func() Model { return New(hdr, nil) },
		"no headers":    func() Model { return New(nil, nil) },
		"short rows":    func() Model { return New(hdr, [][]string{{"a"}, {"b", "2"}, {"c", "3", "x", "extra"}}) },
		"wide runes":    func() Model { return New([]string{"名前", "Emoji"}, [][]string{{"日本語", "😀"}, {"a", "b́"}}) },
		"dirty cells":   func() Model { return New(hdr, [][]string{{"a\x1b[31mred", "x\ty", "z\x07"}, {"ok", "ok", "ok"}}) },
		"raw":           func() Model { m := New(hdr, sampleRows(3)); m.Raw = true; return m },
		"literal model": func() Model { return Model{Headers: hdr, Rows: sampleRows(4)} },
		"sorted asc": func() Model {
			m := New(hdr, sampleRows(6))
			m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "s"})
			return m
		},
		"sorted desc, window": func() Model {
			m := New(hdr, sampleRows(12))
			m.Height = 3
			m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "s"})
			m, _ = m.Update(tui.Key{Type: tui.KeyRunes, Text: "s"})
			return m
		},
		"narrow width, column scrolled": func() Model {
			m := New(hdr, sampleRows(8))
			m.Width = 16
			m, _ = m.Update(tui.Key{Type: tui.KeyRight})
			m, _ = m.Update(tui.Key{Type: tui.KeyRight})
			return m
		},
	}
	for name, mk := range cases {
		t.Run(name, func(t *testing.T) { sameScreen(t, mk(), 50, 16) })
	}
}

func TestDrawCellsClipsLikeDrawView(t *testing.T) {
	m := New([]string{"名前", "Count"}, [][]string{{"日本語", "1"}, {"abc", "22"}})
	m.SetCursor(1)
	for w := 0; w <= 16; w++ {
		for _, h := range []int{0, 1, 2, 3, 4} {
			got, want := cellbuf.New(w, h), cellbuf.New(w, h)
			m.DrawCells(got, got.Bounds())
			tui.DrawView(want, want.Bounds(), m.View())
			for y := 0; y < h; y++ {
				for x := 0; x < w; x++ {
					g, e := got.At(x, y), want.At(x, y)
					if g.Cluster != e.Cluster || g.Width != e.Width || got.Style(g.Style) != want.Style(e.Style) {
						t.Fatalf("%dx%d cell (%d,%d): %+v vs %+v", w, h, x, y, g, e)
					}
				}
			}
		}
	}
}

func TestDrawCellsUsesWidthCache(t *testing.T) {
	m := New([]string{"A", "B"}, sampleRows(40))
	m.Headers = []string{"A", "B"}
	m.Rows = sampleRows(40)
	for i := range m.Rows {
		m.Rows[i] = m.Rows[i][:2]
	}
	m.SetRows(m.Rows)
	scans := 0
	scanHook = func(int) { scans++ }
	defer func() { scanHook = nil }()
	buf := cellbuf.New(40, 50)
	for range 5 {
		m.DrawCells(buf, buf.Bounds())
	}
	if scans != 1 {
		t.Errorf("scans = %d over 5 draws, want 1", scans)
	}
}

func BenchmarkDrawCells(b *testing.B) {
	m := New([]string{"Name", "Count", "State"}, sampleRows(1000))
	m.Height = 30
	m.SetCursor(500)
	buf := cellbuf.New(80, 40)
	b.ReportAllocs()
	for b.Loop() {
		buf.Clear()
		m.DrawCells(buf, buf.Bounds())
	}
}
