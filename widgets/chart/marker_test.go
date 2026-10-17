package chart

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

var allMarkers = []Marker{Braille, Sextant, Quadrant, HalfBlock, Block}

func TestMarkerDotsAndNames(t *testing.T) {
	want := map[Marker][3]any{
		Braille: {2, 4, "braille"}, Sextant: {2, 3, "sextant"}, Quadrant: {2, 2, "quadrant"},
		HalfBlock: {1, 2, "half-block"}, Block: {1, 1, "block"},
	}
	for m, w := range want {
		if c, r := m.Dots(); c != w[0] || r != w[1] || m.String() != w[2] {
			t.Errorf("%v: Dots %dx%d String %q, want %v", m, c, r, m.String(), w)
		}
	}
	if c, r := Marker(99).Dots(); c != 2 || r != 4 || Marker(99).String() != "braille" {
		t.Error("an unknown marker is not braille")
	}
}

func TestSextantGlyphsAreTheUnicodeBlock(t *testing.T) {
	seen := map[string]int{}
	for n := 0; n < 64; n++ {
		g := sextant(n)
		if prev, dup := seen[g]; dup {
			t.Errorf("patterns %d and %d share the glyph %q", prev, n, g)
		}
		seen[g] = n
	}
	for n, want := range map[int]string{
		0: " ", 21: "▌", 42: "▐", 63: "█",
		1: "\U0001FB00", 20: "\U0001FB13", 22: "\U0001FB14", 41: "\U0001FB27", 43: "\U0001FB28", 62: "\U0001FB3B",
	} {
		if got := sextant(n); got != want {
			t.Errorf("sextant(%d) = %q (U+%X), want %q", n, got, []rune(got), want)
		}
	}
}

func TestEveryPatternOfEveryMarkerIsOneCellWide(t *testing.T) {
	for _, m := range allMarkers {
		c, r := m.Dots()
		for p := 0; p < 1<<(c*r); p++ {
			g := m.cell(rune(p), theme.UnicodeGlyphSet())
			if ansi.Width(g) != 1 {
				t.Errorf("%v pattern %d = %q, width %d", m, p, g, ansi.Width(g))
			}
		}
	}
}

func rowOf(c *Canvas) string { return ansi.StripANSI(c.String(theme.DarkTheme())) }

func TestCanvasMarkersDrawTheirGlyphs(t *testing.T) {
	cases := []struct {
		m    Marker
		dots [][2]int
		want string
	}{
		{Quadrant, [][2]int{{0, 0}}, "▘"}, {Quadrant, [][2]int{{1, 0}}, "▝"},
		{Quadrant, [][2]int{{0, 1}}, "▖"}, {Quadrant, [][2]int{{1, 1}}, "▗"},
		{Quadrant, [][2]int{{0, 0}, {1, 0}}, "▀"}, {Quadrant, [][2]int{{0, 0}, {1, 1}}, "▚"},
		{Quadrant, [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}}, "█"},
		{HalfBlock, [][2]int{{0, 0}}, "▀"}, {HalfBlock, [][2]int{{0, 1}}, "▄"}, {HalfBlock, [][2]int{{0, 0}, {0, 1}}, "█"},
		{Block, [][2]int{{0, 0}}, "█"},
		{Sextant, [][2]int{{0, 0}}, "\U0001FB00"},
		{Sextant, [][2]int{{0, 0}, {0, 1}, {0, 2}}, "▌"},
		{Sextant, [][2]int{{1, 0}, {1, 1}, {1, 2}}, "▐"},
		{Sextant, [][2]int{{1, 2}}, "\U0001FB1E"}, // bit 5 alone is pattern 32
		{Braille, [][2]int{{0, 0}}, "⠁"},
	}
	for _, tc := range cases {
		c := NewCanvasMarker(1, 1, tc.m)
		for _, d := range tc.dots {
			c.Set(d[0], d[1], nil)
		}
		if got := rowOf(c); got != tc.want {
			t.Errorf("%v %v = %q, want %q", tc.m, tc.dots, got, tc.want)
		}
	}
}

func TestCanvasMarkerSizesAndBounds(t *testing.T) {
	for _, m := range allMarkers {
		c := NewCanvasMarker(3, 2, m)
		cols, rows := m.Dots()
		if w, h := c.DotSize(); w != 3*cols || h != 2*rows || c.Marker() != m {
			t.Errorf("%v: DotSize %dx%d Marker %v", m, w, h, c.Marker())
		}
		w, h := c.DotSize()
		for _, p := range [][2]int{{-1, 0}, {0, -1}, {w, 0}, {0, h}} {
			c.Set(p[0], p[1], nil) // outside: ignored, no panic
		}
		if strings.TrimSpace(rowOf(c)) != "" {
			t.Errorf("%v: a dot outside the canvas was drawn", m)
		}
		c.Line(0, 0, w-1, h-1, nil)
		if strings.TrimSpace(rowOf(c)) == "" {
			t.Errorf("%v: a line drew nothing", m)
		}
	}
	if NewCanvas(2, 2).Marker() != Braille || NewCanvasMarker(2, 2, Marker(-1)).Marker() != Braille {
		t.Error("NewCanvas and an unknown marker are not braille")
	}
}

func TestCanvasMarkerOnAnAsciiTheme(t *testing.T) {
	asc := theme.DarkTheme().ASCII()
	c := NewCanvasMarker(2, 1, Quadrant)
	c.Set(0, 0, nil) // one of four dots in the first cell
	c.Set(2, 0, nil) // and all four in the second
	c.Set(3, 0, nil)
	c.Set(2, 1, nil)
	c.Set(3, 1, nil)
	got := ansi.StripANSI(c.String(asc))
	if len(got) != 2 || got[0] == ' ' || got[1] == ' ' || got[0] == got[1] || strings.ContainsFunc(got, func(r rune) bool { return r > 0x7e }) {
		t.Errorf("ascii rows = %q, want two different ASCII shades", got)
	}
	if got := ansi.StripANSI(NewCanvasMarker(1, 1, Block).String(asc)); got != " " {
		t.Errorf("an empty cell = %q", got)
	}
}

func TestPlotDrawsWithEveryMarker(t *testing.T) {
	series := []Series{{Name: "a", Values: []float64{1, 5, 2, 8, 3}}}
	braille := Plot(series, PlotOptions{Width: 20, Height: 5}, theme.DarkTheme())
	for _, m := range allMarkers {
		out := Plot(series, PlotOptions{Width: 20, Height: 5, Marker: m}, theme.DarkTheme())
		rows := strings.Split(ansi.StripANSI(out), "\n")
		if len(rows) < 5 || strings.TrimSpace(out) == "" {
			t.Fatalf("%v: plot is empty:\n%s", m, out)
		}
		if m == Braille && out != braille {
			t.Error("the zero Marker changed the plot")
		}
		if m != Braille && out == braille {
			t.Errorf("%v drew the same as braille", m)
		}
		for _, r := range rows[:5] {
			if ansi.Width(r) != ansi.Width(rows[0]) {
				t.Errorf("%v: rows differ in width: %q", m, r)
			}
		}
	}
	// A bar series fills columns with a block marker too.
	bars := Plot([]Series{{Kind: Bars, Values: []float64{3, 1, 4}}}, PlotOptions{Width: 12, Height: 4, Marker: HalfBlock}, theme.DarkTheme())
	if !strings.ContainsAny(ansi.StripANSI(bars), "▀▄█") {
		t.Errorf("bars with HalfBlock:\n%s", bars)
	}
}

func TestADotLandsInTheCellItIsIn(t *testing.T) {
	for _, tc := range []struct {
		m    Marker
		x, y int
		want string
	}{
		{Block, 2, 0, "  █"},
		{HalfBlock, 2, 1, "  ▄"},
		{Quadrant, 5, 0, "  ▝"},
		{Sextant, 4, 0, "  \U0001FB00"},
		{Braille, 4, 0, "  ⠁"},
	} {
		c := NewCanvasMarker(3, 1, tc.m)
		c.Set(tc.x, tc.y, nil)
		if got := rowOf(c); got != tc.want {
			t.Errorf("%v dot (%d,%d) = %q, want %q", tc.m, tc.x, tc.y, got, tc.want)
		}
	}
	c := NewCanvasMarker(1, 3, HalfBlock) // down too
	c.Set(0, 5, nil)
	if got := rowOf(c); got != " \n \n▄" {
		t.Errorf("dot in the third row = %q", got)
	}
}
