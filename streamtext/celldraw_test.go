package streamtext

import (
	"testing"

	"github.com/ows4444/tui/cellbuf"
)

// Criterion #47: DrawCells shows the screen View shows, cell by cell with styles.
func TestDrawCellsMatchesView(t *testing.T) {
	cases := []struct {
		name string
		set  func(*Model)
		text string
	}{
		{"empty", func(m *Model) {}, ""},
		{"plain", func(m *Model) {}, "hello world"},
		{"multiline", func(m *Model) {}, "one\ntwo\n\nfour"},
		{"trailing newline", func(m *Model) {}, "one\n"},
		{"wide", func(m *Model) {}, "世界 ab 世界"},
		{"wrapped", func(m *Model) { m.Width = 8 }, "the quick brown fox jumps over"},
		{"styled", func(m *Model) { m.Raw = true }, "\x1b[1;31mred bold\x1b[0m and \x1b[4mul\x1b[0m"},
		{"styled wrapped", func(m *Model) { m.Raw = true; m.Width = 10 }, "\x1b[32mgreen words that wrap\x1b[0m end"},
		{"no cursor", func(m *Model) { m.Cursor = "" }, "abc"},
		{"typewriter", func(m *Model) { *m = NewTypewriter() }, "ab. cd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := New()
			c.set(&m)
			m.SetText(c.text)
			total := m.total()
			for shown := 0; shown <= total+1; shown++ {
				m.shown = shown
				for _, off := range []bool{false, true} {
					m.blinkOff = off
					want, err := cellbuf.Parse(m.View())
					if err != nil {
						t.Fatal(err)
					}
					got := cellbuf.New(30, 8)
					m.DrawCells(got, got.Bounds())
					for y := 0; y < 8; y++ {
						for x := 0; x < 30; x++ {
							a, b := want.At(x, y), got.At(x, y)
							if a.Cluster != b.Cluster || a.Width != b.Width || want.Style(a.Style) != got.Style(b.Style) {
								t.Fatalf("shown %d off %v cell (%d,%d): View %q %+v, DrawCells %q %+v\nview %q",
									shown, off, x, y, a.Cluster, want.Style(a.Style), b.Cluster, got.Style(b.Style), m.View())
							}
						}
					}
				}
			}
		})
	}
}

// DrawCells clips to the region it is given.
func TestDrawCellsClips(t *testing.T) {
	m := New()
	m.SetText("abcdef\nghijkl")
	m.Skip()
	buf := cellbuf.New(10, 4)
	m.DrawCells(buf, cellbuf.Rect{X: 1, Y: 1, W: 3, H: 1})
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			in := x >= 1 && x < 4 && y == 1
			if c := buf.At(x, y); !in && c.Cluster != " " {
				t.Fatalf("(%d,%d) outside the region holds %q", x, y, c.Cluster)
			}
		}
	}
	if got := buf.At(1, 1).Cluster + buf.At(3, 1).Cluster; got != "ac" {
		t.Fatalf("region content %q", got)
	}
}
