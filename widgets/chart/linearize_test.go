package chart

import (
	"math"
	"strings"
	"testing"
)

func TestLinearizeSparkline(t *testing.T) {
	got := LinearizeSparkline([]float64{1, 4, 2, 9})
	want := "Sparkline: 4 values, min 1, max 9, average 4, last 9, rising"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if got := LinearizeSparkline(nil); got != "Sparkline: no data" {
		t.Errorf("empty = %q", got)
	}
	if got := LinearizeSparkline([]float64{3}); got != "Sparkline: 1 value, min 3, max 3, average 3, last 3" {
		t.Errorf("single = %q", got)
	}
	if got := LinearizeSparkline([]float64{5, 1}); !strings.HasSuffix(got, "falling") {
		t.Errorf("falling = %q", got)
	}
	if got := LinearizeSparkline([]float64{2, 7, 2}); !strings.HasSuffix(got, "flat") {
		t.Errorf("flat = %q", got)
	}
}

func TestLinearizeIgnoresNonFinite(t *testing.T) {
	got := LinearizeLineChart([]float64{1, math.NaN(), 3, math.Inf(1)})
	if !strings.HasPrefix(got, "Line chart: 4 values, min 1, max 3, average 2") {
		t.Errorf("got %q", got)
	}
	if got := LinearizeLineChart([]float64{math.NaN()}); got != "Line chart: 1 value, none finite" {
		t.Errorf("all NaN = %q", got)
	}
}

func TestLinearizeBarChart(t *testing.T) {
	got := LinearizeBarChart([]BarItem{{"a", 1}, {"b", 5}, {"c", 3}})
	want := "Bar chart: 3 bars: a 1, b 5, c 3; largest b"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if got := LinearizeBarChart(nil); got != "Bar chart: no data" {
		t.Errorf("empty = %q", got)
	}
	if got := LinearizeBarChart([]BarItem{{"x", 2}}); got != "Bar chart: 1 bar: x 2; largest x" {
		t.Errorf("one = %q", got)
	}
}

func TestLinearizeGauge(t *testing.T) {
	for in, want := range map[float64]string{0.424: "Gauge: 42 percent", -1: "Gauge: 0 percent", 7: "Gauge: 100 percent", math.NaN(): "Gauge: 0 percent"} {
		if got := LinearizeGauge(in); got != want {
			t.Errorf("%v: got %q want %q", in, got, want)
		}
	}
}

func TestLinearizeHeatMap(t *testing.T) {
	got := LinearizeHeatMap([][]float64{{1, 2}, {3}, {0, 9, 4}})
	if want := "Heat map: 3 rows by 3 columns, min 0, max 9"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
	if got := LinearizeHeatMap([][]float64{{}, {}}); got != "Heat map: no data" {
		t.Errorf("empty = %q", got)
	}
}
