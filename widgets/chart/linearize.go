package chart

import (
	"math"
	"strconv"
	"strings"
)

// The Linearize functions give each chart a plain-text summary for accessible
// output (see tui.Linearizer): no glyphs, colour or scaling marks, just the
// series' count, range and latest value. Like the charts themselves they are
// stateless; callers pass the same data they pass to the chart, and typically
// return the summary from their own Model's Linearize method. NaN and
// infinite values are ignored when finding minimum, maximum and average.

func num(v float64) string { return strconv.FormatFloat(v, 'g', 6, 64) }

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// stats returns the count of finite values, their min, max and mean.
func stats(values []float64) (n int, min, max, mean float64) {
	sum := 0.0
	for _, v := range values {
		if !finite(v) {
			continue
		}
		if n == 0 || v < min {
			min = v
		}
		if n == 0 || v > max {
			max = v
		}
		sum += v
		n++
	}
	if n > 0 {
		mean = sum / float64(n)
	}
	return
}

func series(kind string, values []float64) string {
	if len(values) == 0 {
		return kind + ": no data"
	}
	n, min, max, mean := stats(values)
	s := kind + ": " + strconv.Itoa(len(values)) + " values"
	if len(values) == 1 {
		s = kind + ": 1 value"
	}
	if n == 0 {
		return s + ", none finite"
	}
	trend := ""
	if len(values) > 1 {
		first, last := values[0], values[len(values)-1]
		switch {
		case !finite(first) || !finite(last):
		case last > first:
			trend = ", rising"
		case last < first:
			trend = ", falling"
		default:
			trend = ", flat"
		}
	}
	return s + ", min " + num(min) + ", max " + num(max) + ", average " + num(mean) +
		", last " + num(values[len(values)-1]) + trend
}

// LinearizeSparkline summarises a Sparkline's values as text: the count, min,
// max, average and last value, and whether the series rose, fell or stayed
// flat from first to last. Empty input returns "Sparkline: no data".
func LinearizeSparkline(values []float64) string { return series("Sparkline", values) }

// LinearizeLineChart summarises a LineChart's values as text, in the same
// form as LinearizeSparkline with the prefix "Line chart". Empty input
// returns "Line chart: no data".
func LinearizeLineChart(values []float64) string { return series("Line chart", values) }

// LinearizeBarChart summarises a BarChart's items as text: the item count and
// one "label value" entry per item in order, then the largest. Empty input
// returns "Bar chart: no data".
func LinearizeBarChart(items []BarItem) string {
	if len(items) == 0 {
		return "Bar chart: no data"
	}
	parts := make([]string, len(items))
	top := 0
	for i, it := range items {
		parts[i] = it.Label + " " + num(it.Value)
		if it.Value > items[top].Value || (math.IsNaN(items[top].Value) && !math.IsNaN(it.Value)) {
			top = i
		}
	}
	n := strconv.Itoa(len(items)) + " bars"
	if len(items) == 1 {
		n = "1 bar"
	}
	return "Bar chart: " + n + ": " + strings.Join(parts, ", ") + "; largest " + items[top].Label
}

// LinearizeGauge summarises a Gauge's percent (a fraction, clamped to [0,1],
// NaN counting as 0) as text, for example "Gauge: 42 percent".
func LinearizeGauge(percent float64) string {
	if math.IsNaN(percent) || percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}
	return "Gauge: " + strconv.Itoa(int(math.Round(percent*100))) + " percent"
}

// LinearizeHeatMap summarises a HeatMap's grid as text: its row and column
// counts (columns being the longest row) and the min and max over all cells.
// An empty grid returns "Heat map: no data".
func LinearizeHeatMap(values [][]float64) string {
	rows, cols := len(values), 0
	var all []float64
	for _, r := range values {
		if len(r) > cols {
			cols = len(r)
		}
		all = append(all, r...)
	}
	if rows == 0 || cols == 0 {
		return "Heat map: no data"
	}
	s := "Heat map: " + strconv.Itoa(rows) + " rows by " + strconv.Itoa(cols) + " columns"
	if n, min, max, _ := stats(all); n > 0 {
		s += ", min " + num(min) + ", max " + num(max)
	}
	return s
}
