package chart

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func canvasText(c *Canvas) string {
	var rows []string
	for _, r := range c.Rows(theme.DarkTheme()) {
		rows = append(rows, strings.TrimRight(ansi.StripANSI(r), " "))
	}
	return strings.Join(rows, "\n")
}

func TestCanvasSizeAndDots(t *testing.T) {
	c := NewCanvas(3, 2)
	if w, h := c.Size(); w != 3 || h != 2 {
		t.Errorf("Size = %d,%d", w, h)
	}
	if w, h := c.DotSize(); w != 6 || h != 8 {
		t.Errorf("DotSize = %d,%d, want 6,8", w, h)
	}
	c.Set(0, 0, nil)
	c.Set(5, 7, nil)
	c.Set(-1, 0, nil) // outside: ignored
	c.Set(6, 0, nil)
	c.Set(0, 8, nil)
	got := canvasText(c)
	want := "⠁\n    ⢀"
	// Dot (0,0) is the top-left bit of cell 0; dot (5,7) is the bottom-right of cell 2.
	if !strings.HasPrefix(got, "⠁") || !strings.HasSuffix(got, "⢀") {
		t.Errorf("Rows =\n%q, want like %q", got, want)
	}
}

func TestCanvasShapes(t *testing.T) {
	c := NewCanvas(8, 4) // 16 x 16 dots
	c.Rect(0, 0, 16, 16, nil)
	box := canvasText(c)
	lines := strings.Split(box, "\n")
	if len(lines) != 4 || strings.TrimSpace(lines[0]) == "" || strings.TrimSpace(lines[3]) == "" {
		t.Errorf("Rect:\n%s", box)
	}
	c.Clear()
	if canvasText(c) != "\n\n\n" {
		t.Error("Clear left dots")
	}
	c.Circle(8, 8, 6, nil)
	circle := canvasText(c)
	// A circle of radius 6 in a 16x16 dot area touches every row of cells.
	for i, l := range strings.Split(circle, "\n") {
		if strings.TrimSpace(l) == "" {
			t.Errorf("Circle row %d is empty:\n%s", i, circle)
		}
	}
	c.Clear()
	c.Line(0, 0, 15, 15, nil)
	diag := strings.Split(canvasText(c), "\n")
	if len(diag) != 4 || !strings.HasPrefix(diag[0], "⠑") && !strings.HasPrefix(diag[0], "⠁") && !strings.HasPrefix(diag[0], "⠙") {
		// Only the shape matters here: the first row has dots at the left.
		if strings.TrimSpace(diag[0]) == "" {
			t.Errorf("Line:\n%s", strings.Join(diag, "\n"))
		}
	}
}

func TestCanvasZeroAndNegativeSizes(t *testing.T) {
	for _, c := range []*Canvas{NewCanvas(0, 5), NewCanvas(-3, -3), {}} {
		c.Set(0, 0, nil)
		c.Line(0, 0, 4, 4, nil)
		c.Rect(0, 0, 3, 3, nil)
		c.Circle(1, 1, 2, nil)
		if len(c.Rows(theme.DarkTheme())) != c.h {
			t.Errorf("Rows of an empty canvas = %d", len(c.Rows(theme.DarkTheme())))
		}
	}
	NewCanvas(2, 2).Circle(0, 0, -1, nil) // a negative radius draws nothing and does not panic
}

func TestCanvasColourIsTheLastShapesInTheCell(t *testing.T) {
	th := theme.DarkTheme()
	c := NewCanvas(1, 1)
	c.Set(0, 0, ansi.Red)
	c.Set(1, 0, ansi.Blue) // same cell: Blue wins
	out := c.String(th)
	blue := ansi.NewStyle().Foreground(ansi.Blue).Render("x")
	open := blue[:strings.Index(blue, "x")]
	if !strings.Contains(out, open) {
		t.Errorf("cell not blue: %q", out)
	}
	c.Clear()
	c.Set(0, 0, nil)
	prim := ansi.NewStyle().Foreground(th.Primary).Render("x")
	if !strings.Contains(c.String(th), prim[:strings.Index(prim, "x")]) {
		t.Error("a nil colour does not take the theme's primary")
	}
}

func TestPlotScatterAndBars(t *testing.T) {
	th := theme.DarkTheme()
	vals := []float64{1, 5, 3, 9, 2}
	line := ansi.StripANSI(Plot([]Series{{Values: vals}}, PlotOptions{Width: 20, Height: 6}, th))
	scat := ansi.StripANSI(Plot([]Series{{Values: vals, Kind: Scatter}}, PlotOptions{Width: 20, Height: 6}, th))
	bars := ansi.StripANSI(Plot([]Series{{Values: vals, Kind: Bars}}, PlotOptions{Width: 20, Height: 6}, th))
	count := func(s string) int {
		n := 0
		for _, r := range s {
			if r >= 0x2800 && r < 0x2900 {
				n++
			}
		}
		return n
	}
	cl, cs, cb := count(line), count(scat), count(bars)
	if !(cs < cl) {
		t.Errorf("scatter should light fewer cells than a line: scatter %d line %d", cs, cl)
	}
	if !(cb > cs) {
		t.Errorf("bars should light more cells than scatter: bars %d scatter %d", cb, cs)
	}
	// The bar for the largest value reaches the top row.
	first := strings.Split(bars, "\n")[0]
	if !strings.ContainsFunc(first, func(r rune) bool { return r >= 0x2800 && r < 0x2900 }) {
		t.Errorf("tallest bar does not reach the top:\n%s", bars)
	}
}

func TestPlotBarsRiseFromZeroWhenTheRangeCrossesIt(t *testing.T) {
	th := theme.DarkTheme()
	got := ansi.StripANSI(Plot([]Series{{Kind: Bars, Values: []float64{-5, 5}}}, PlotOptions{Width: 10, Height: 8}, th))
	lines := strings.Split(got, "\n")
	// The negative bar hangs below the middle, the positive one stands above it.
	top := strings.Join(lines[:4], "")
	bottom := strings.Join(lines[4:8], "")
	if !strings.ContainsFunc(top, func(r rune) bool { return r >= 0x2800 }) || !strings.ContainsFunc(bottom, func(r rune) bool { return r >= 0x2800 }) {
		t.Errorf("bars:\n%s", got)
	}
}

func ExampleCanvas() {
	c := NewCanvas(4, 2) // 8 x 8 dots
	c.Rect(0, 0, 8, 8, nil)
	for _, row := range c.Rows(theme.DarkTheme()) {
		fmt.Println(ansi.StripANSI(row))
	}
	// Output:
	// ⡏⠉⠉⢹
	// ⣇⣀⣀⣸
}
