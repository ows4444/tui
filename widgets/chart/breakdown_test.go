package chart

import (
	"math"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func plain(s string) string { return ansi.StripANSI(s) }

func TestBreakdownBarSegmentsFillTheBarExactly(t *testing.T) {
	items := []BarItem{{"Go", 50}, {"Rust", 30}, {"C", 20}}
	got := plain(BreakdownBar(items, 20, theme.DarkTheme()))
	lines := strings.Split(got, "\n")
	if ansi.Width(lines[0]) != 20 {
		t.Errorf("bar is %d wide: %q", ansi.Width(lines[0]), lines[0])
	}
	if lines[0] != strings.Repeat("█", 10)+strings.Repeat("▓", 6)+strings.Repeat("▒", 4) {
		t.Errorf("bar = %q", lines[0])
	}
	want := []string{"█ Go 50 (50%)", "▓ Rust 30 (30%)", "▒ C 20 (20%)"}
	for i, w := range want {
		if lines[i+1] != w {
			t.Errorf("legend %d = %q, want %q", i, lines[i+1], w)
		}
	}
}

func TestBreakdownBarRoundsToTheWidthAndKeepsZeroItems(t *testing.T) {
	// Thirds of 10 columns: 3+3+3 and the odd column goes to a largest remainder.
	for _, w := range []int{3, 7, 10, 13, 100} {
		got := plain(BreakdownBar([]BarItem{{"a", 1}, {"b", 1}, {"c", 1}}, w, theme.DarkTheme()))
		if bw := ansi.Width(strings.Split(got, "\n")[0]); bw != w {
			t.Errorf("width %d: bar is %d wide", w, bw)
		}
	}
	got := plain(BreakdownBar([]BarItem{{"on", 4}, {"off", 0}, {"neg", -2}}, 8, theme.DarkTheme()))
	lines := strings.Split(got, "\n")
	if lines[0] != strings.Repeat("█", 8) || len(lines) != 4 || !strings.Contains(lines[2], "off 0 (0%)") {
		t.Errorf("zero and negative items: %q", got)
	}
}

func TestBreakdownBarEmptyAndASCII(t *testing.T) {
	for _, c := range []struct {
		items []BarItem
		w     int
	}{{nil, 10}, {[]BarItem{{"x", 0}}, 10}, {[]BarItem{{"x", 5}}, 0}} {
		if got := BreakdownBar(c.items, c.w, theme.DarkTheme()); got != "" {
			t.Errorf("BreakdownBar(%v, %d) = %q", c.items, c.w, got)
		}
	}
	got := plain(BreakdownBar([]BarItem{{"a", 1}, {"b", 1}}, 4, theme.DarkTheme().ASCII()))
	if strings.Split(got, "\n")[0] != "##++" {
		t.Errorf("ASCII bar = %q", got)
	}
}

func TestLinearizeBreakdownBar(t *testing.T) {
	got := LinearizeBreakdownBar([]BarItem{{"b", 1}, {"a", 3}})
	if got != "Breakdown: a 3 (75%), b 1 (25%)" {
		t.Errorf("got %q", got)
	}
	if LinearizeBreakdownBar(nil) != "Breakdown: no data" {
		t.Error("empty")
	}
}

func TestHistogramCountsIntoEqualBins(t *testing.T) {
	values := []float64{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}
	items, ok := histogramBins(values, 3)
	if !ok || len(items) != 3 {
		t.Fatalf("bins = %v, %v", items, ok)
	}
	var total float64
	for _, it := range items {
		total += it.Value
	}
	if total != float64(len(values)) {
		t.Errorf("counts add up to %v, want %d", total, len(values))
	}
	if items[0].Label != "[1, 2)" || items[2].Label != "[3, 4]" {
		t.Errorf("labels = %q ... %q", items[0].Label, items[2].Label)
	}
	// Edges at 1, 2, 3, 4: [1,2)=1, [2,3)=2, and the maximum lands in the last bin [3,4]=7.
	if items[0].Value != 1 || items[1].Value != 2 || items[2].Value != 7 {
		t.Errorf("counts = %v %v %v, want 1 2 7", items[0].Value, items[1].Value, items[2].Value)
	}
}

func TestHistogramEdgeCases(t *testing.T) {
	if Histogram(nil, 5, 30, theme.DarkTheme()) != "" || LinearizeHistogram(nil, 5) != "Histogram: no data" {
		t.Error("no values should give nothing")
	}
	same, _ := histogramBins([]float64{7, 7, 7}, 5)
	if len(same) != 1 || same[0].Value != 3 {
		t.Errorf("equal values = %v", same)
	}
	nan, _ := histogramBins([]float64{1, 2, math.NaN(), math.Inf(1)}, 2)
	if len(nan) != 2 || nan[0].Value+nan[1].Value != 2 {
		t.Errorf("non-finite values counted: %v", nan)
	}
	out := plain(Histogram([]float64{1, 2, 3, 4}, 2, 30, theme.DarkTheme()))
	if strings.Count(out, "\n") != 1 || !strings.Contains(out, "[1, 2.5)") {
		t.Errorf("Histogram = %q", out)
	}
	if got := LinearizeHistogram([]float64{1, 2, 3, 4, 4}, 2); !strings.Contains(got, "5 values in 2 bins") {
		t.Errorf("Linearize = %q", got)
	}
}
