package textinput

import (
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/cellbuf"
)

// sameScreen compares cellbuf.Parse(View()) with the DrawCells buffer, cell by
// cell, cluster, width and style, over a w x h grid.
func sameScreen(t *testing.T, m Model, w, h int) {
	t.Helper()
	want, err := cellbuf.Parse(m.View())
	if err != nil {
		t.Fatalf("parse View: %v", err)
	}
	got := cellbuf.New(w, h)
	m.DrawCells(got, got.Bounds())
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			a, b := want.At(x, y), got.At(x, y)
			if a.Cluster != b.Cluster || a.Width != b.Width || want.Style(a.Style) != got.Style(b.Style) {
				t.Fatalf("cell (%d,%d): View %q w%d %+v, DrawCells %q w%d %+v\nview: %q",
					x, y, a.Cluster, a.Width, want.Style(a.Style), b.Cluster, b.Width, got.Style(b.Style), m.View())
			}
		}
	}
}

const family = "👨‍👩‍👧"

// DrawCells shows the screen View shows, for every shape of field.
func TestDrawCellsMatchesView(t *testing.T) {
	styled := func(m *Model) {
		m.TextStyle = ansi.NewStyle().Foreground(ansi.Green).Bold()
		m.CursorStyle = ansi.NewStyle().Background(ansi.RGB{R: 10, G: 20, B: 30})
		m.PlaceholderStyle = ansi.NewStyle().Faint()
	}
	cases := []struct {
		name  string
		set   func(*Model)
		text  string
		moves []int
	}{
		{"empty", func(m *Model) {}, "", []int{0}},
		{"placeholder", func(m *Model) { m.Placeholder = "type here" }, "", []int{0}},
		{"prompt", func(m *Model) { m.Prompt = "name: " }, "ada", []int{0, 1, 3}},
		{"styled", styled, "hello", []int{0, 2, 5}},
		{"wide", func(m *Model) {}, "世界ab", []int{0, 1, 2, 4}},
		{"cluster", func(m *Model) {}, "a" + family + "b", []int{0, 1, 2, 3}},
		{"windowed", func(m *Model) { m.Width = 6 }, "abcdefghijklmnopqrstuvwxyz", []int{0, 5, 13, 26}},
		{"windowed wide", func(m *Model) { m.Width = 5; styled(m) }, "世界世界世界ab", []int{0, 3, 6, 8}},
		{"styled prompt", func(m *Model) { m.Prompt = ansi.NewStyle().Bold().Render("> ") }, "abc", []int{0, 3}},
		{"control in value", func(m *Model) {}, "a\tb", []int{0, 2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := New()
			c.set(&m)
			m.SetValue(c.text)
			sameScreen(t, m, 30, 2) // unfocused: no cursor
			m.Focus()
			for _, pos := range c.moves {
				m.SetCursor(pos)
				sameScreen(t, m, 30, 2)
			}
			m.cursorVisible = false // the dark half of a blink
			sameScreen(t, m, 30, 2)
		})
	}
}

func TestDrawCellsShowsASelection(t *testing.T) {
	m := New()
	m.SetValue("select me")
	m.Focus()
	m.ed.SelectAll()
	sameScreen(t, m, 20, 1)
}

func TestDrawCellsIsClippedToItsRegion(t *testing.T) {
	m := New()
	m.SetValue("abcdefgh")
	buf := cellbuf.New(10, 3)
	m.DrawCells(buf, cellbuf.Rect{X: 2, Y: 1, W: 4, H: 1})
	if got := buf.At(2, 1).Cluster; got != "a" {
		t.Errorf("cell (2,1) = %q, want a", got)
	}
	if buf.At(6, 1).Cluster == "g" || buf.At(1, 1).Cluster != "" && buf.At(1, 1).Cluster != " " {
		t.Errorf("drew outside the region: %q", buf.Lines())
	}
	m.DrawCells(buf, cellbuf.Rect{}) // an empty region draws nothing and does not panic
}

// A field of 60 columns, drawn into a cell grid: through View and parse, and
// straight with DrawCells.
func benchModel() Model {
	m := New()
	m.Prompt = "> "
	m.SetValue("the quick brown fox jumps over the lazy dog, again and again")
	m.Focus()
	return m
}

func BenchmarkFrameViaView(b *testing.B) {
	m := benchModel()
	buf := cellbuf.New(70, 1)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		p, err := cellbuf.Parse(m.View())
		if err != nil {
			b.Fatal(err)
		}
		_ = p
		_ = buf
	}
}

func BenchmarkFrameDrawCells(b *testing.B) {
	m := benchModel()
	buf := cellbuf.New(70, 1)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m.DrawCells(buf, buf.Bounds())
	}
}
