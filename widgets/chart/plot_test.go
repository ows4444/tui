package chart

import (
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func plotText(s []Series, o PlotOptions) string {
	return ansi.StripANSI(Plot(s, o, theme.DarkTheme()))
}

func TestPlotEmptyOrZeroSize(t *testing.T) {
	if got := Plot([]Series{{Values: []float64{1, 2}}}, PlotOptions{Width: 0, Height: 4}, theme.DarkTheme()); got != "" {
		t.Errorf("zero width = %q", got)
	}
	if got := Plot([]Series{{Values: []float64{1, 2}}}, PlotOptions{Width: 10, Height: -1}, theme.DarkTheme()); got != "" {
		t.Errorf("negative height = %q", got)
	}
	// No series still draws the empty axes.
	got := plotText(nil, PlotOptions{Width: 6, Height: 4})
	if !strings.Contains(got, "──") {
		t.Errorf("empty plot:\n%s", got)
	}
}

func TestPlotAxesLabelsAndShape(t *testing.T) {
	s := []Series{{Name: "cpu", Values: []float64{0, 25, 50, 75, 100}}}
	got := plotText(s, PlotOptions{
		Width: 20, Height: 8, YTicks: 3,
		XLabels: []string{"mon", "wed", "fri"}, Legend: true,
	})
	lines := strings.Split(got, "\n")
	// 8 plot rows, the x axis, the x labels and the legend.
	if len(lines) != 8+1+1+1 {
		t.Fatalf("%d lines:\n%s", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "100─") {
		t.Errorf("top tick: %q", lines[0])
	}
	if !strings.HasPrefix(lines[7], "  0─") {
		t.Errorf("bottom tick: %q", lines[7])
	}
	if !strings.Contains(lines[3], "50─") && !strings.Contains(lines[4], "50─") {
		t.Errorf("middle tick missing:\n%s", got)
	}
	if !strings.HasPrefix(lines[8], "   ─") || !strings.Contains(lines[8], "────") {
		t.Errorf("x axis: %q", lines[8])
	}
	for _, l := range []string{"mon", "wed", "fri"} {
		if !strings.Contains(lines[9], l) {
			t.Errorf("x labels %q lack %q", lines[9], l)
		}
	}
	if !strings.Contains(lines[10], "cpu") {
		t.Errorf("legend %q", lines[10])
	}
	// A rising line: the top row has dots on the right, the bottom on the left.
	top, bottom := strings.TrimPrefix(lines[0], "100─"), strings.TrimPrefix(lines[7], "  0─")
	if strings.TrimSpace(top) == "" || strings.TrimSpace(bottom) == "" {
		t.Errorf("line not drawn:\n%s", got)
	}
	if strings.Index(top, strings.TrimSpace(top)[:1]) < strings.Index(bottom, strings.TrimSpace(bottom)[:1]) {
		t.Errorf("series does not rise left to right:\n%s", got)
	}
}

func TestPlotWidthIsBounded(t *testing.T) {
	s := []Series{{Name: "a", Values: []float64{1, 5, 2, 8}}, {Name: "b", Values: []float64{3, 3, 9}}}
	for _, w := range []int{1, 2, 7, 30} {
		got := plotText(s, PlotOptions{Width: w, Height: 5, XLabels: []string{"start", "end"}, Legend: true})
		for i, l := range strings.Split(got, "\n") {
			if i < 6 && ansi.Width(l) > w+6 { // label column and axis
				t.Errorf("width %d line %d is %d wide: %q", w, i, ansi.Width(l), l)
			}
		}
	}
}

func TestPlotTwoSeriesHaveTheirOwnColour(t *testing.T) {
	th := theme.DarkTheme()
	s := []Series{
		{Name: "a", Values: []float64{0, 10}},
		{Name: "b", Values: []float64{10, 0}},
	}
	got := Plot(s, PlotOptions{Width: 10, Height: 4}, th)
	a := ansi.NewStyle().Foreground(th.Primary).Render("x")
	b := ansi.NewStyle().Foreground(th.Secondary).Render("x")
	open := func(rendered string) string { return rendered[:strings.Index(rendered, "x")] }
	if !strings.Contains(got, open(a)) || !strings.Contains(got, open(b)) {
		t.Errorf("both palette colours expected:\n%q", got)
	}
}

func TestPlotGapsAndFixedRange(t *testing.T) {
	s := []Series{{Values: []float64{1, math.NaN(), 3, math.Inf(1), 5}}}
	got := plotText(s, PlotOptions{Width: 12, Height: 4, Min: 0, Max: 10})
	if !strings.HasPrefix(strings.TrimSpace(got), "10─") {
		t.Errorf("fixed range:\n%s", got)
	}
	flat := plotText([]Series{{Values: []float64{7, 7, 7}}}, PlotOptions{Width: 8, Height: 4})
	if !strings.Contains(flat, "8─") || !strings.Contains(flat, "6─") {
		t.Errorf("a flat series pads the range by one:\n%s", flat)
	}
}

func TestPlotYFormatAndTicksClamp(t *testing.T) {
	s := []Series{{Values: []float64{0, 1}}}
	got := plotText(s, PlotOptions{Width: 6, Height: 3, YTicks: 99, YFormat: func(v float64) string { return fmt.Sprintf("%.1f%%", v*100) }})
	if !strings.Contains(got, "100.0%─") || strings.Count(got, "%─") != 3 {
		t.Errorf("ticks:\n%s", got)
	}
}

func TestPlotASCIIFallback(t *testing.T) {
	th := theme.DarkTheme().ASCII()
	got := ansi.StripANSI(Plot([]Series{{Name: "a", Values: []float64{0, 5, 10}}}, PlotOptions{Width: 10, Height: 4, Legend: true}, th))
	if !strings.Contains(got, "---") {
		t.Errorf("ASCII axes:\n%s", got)
	}
	for _, r := range got {
		if r >= 0x80 {
			t.Fatalf("non-ASCII rune %q in:\n%s", r, got)
		}
	}
}

func TestXLabelsThatCollideAreDropped(t *testing.T) {
	got := plotText([]Series{{Values: []float64{1, 2, 3}}},
		PlotOptions{Width: 10, Height: 3, XLabels: []string{"January", "February", "March"}})
	last := got[strings.LastIndex(got, "\n")+1:]
	if strings.Contains(last, "February") && strings.Contains(last, "January") && len(strings.Fields(last)) == 3 && ansi.Width(last) > 11+3 {
		t.Errorf("labels overflow: %q", last)
	}
	if ansi.Width(last) > 3+10+1 {
		t.Errorf("x label line is %d wide: %q", ansi.Width(last), last)
	}
}

func TestLinearizePlot(t *testing.T) {
	got := LinearizePlot([]Series{
		{Name: "cpu", Values: []float64{1, 2, 3}},
		{Values: []float64{9, 4}},
	}, PlotOptions{})
	for _, want := range []string{"Plot: 2 series, y from 1 to 9", "cpu: 3 values", "Series 2: 2 values, min 4, max 9", "falling"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if LinearizePlot(nil, PlotOptions{}) != "Plot: no data" {
		t.Error("no series")
	}
}

// Two lines on one y axis with ticks, x labels and a legend.
func ExamplePlot() {
	s := []Series{
		{Name: "requests", Values: []float64{10, 40, 30, 80, 60, 100}},
	}
	out := Plot(s, PlotOptions{
		Width: 16, Height: 5, YTicks: 3,
		XLabels: []string{"00:00", "12:00"}, Legend: true,
	}, theme.DarkTheme())
	for _, l := range strings.Split(ansi.StripANSI(out), "\n") {
		fmt.Println(strings.TrimRight(l, " "))
	}
	// Output:
	// 100─              ⡠⠊
	//    │        ⡰⠑⠢⣀⢀⠔⠁
	//  55─       ⡰⠁   ⠁
	//    │  ⡠⠒⠤⢄⡰⠁
	//  10─⡠⠊
	//    ─────────────────
	//     00:00      12:00
	//     ● requests
}
