package chart

import (
	"math"
	"strconv"

	"github.com/ows4444/tui/theme"
)

// histogramBins counts the finite values into bins equal-width bins from their
// minimum to their maximum (the last bin includes the maximum). It returns the
// bin items, labelled with their ranges, and false when there is nothing to
// count.
func histogramBins(values []float64, bins int) ([]BarItem, bool) {
	n, lo, hi, _ := stats(values)
	if n == 0 {
		return nil, false
	}
	if bins < 1 {
		bins = 10
	}
	if lo == hi {
		bins = 1
	}
	step := (hi - lo) / float64(bins)
	counts := make([]int, bins)
	for _, v := range values {
		if !finite(v) {
			continue
		}
		i := bins - 1
		if step > 0 {
			i = min(int(math.Floor((v-lo)/step)), bins-1)
		}
		counts[i]++
	}
	label := func(x float64) string { return strconv.FormatFloat(x, 'g', 4, 64) }
	items := make([]BarItem, bins)
	for i := range items {
		a, b := lo+float64(i)*step, lo+float64(i+1)*step
		if i == bins-1 {
			b = hi
		}
		closing := ")"
		if i == bins-1 {
			closing = "]"
		}
		items[i] = BarItem{Label: "[" + label(a) + ", " + label(b) + closing, Value: float64(counts[i])}
	}
	return items, true
}

// Histogram counts values into bins equal-width bins from their smallest to
// their largest and draws one bar per bin (see BarChart), each labelled with the
// range it covers. NaN and infinite values are ignored; a bins below 1 means 10;
// all values equal make one bin. It renders as "" when there is nothing to
// count.
func Histogram(values []float64, bins, width int, t theme.Theme) string {
	items, ok := histogramBins(values, bins)
	if !ok {
		return ""
	}
	return BarChart(items, width, t)
}

// LinearizeHistogram gives a Histogram a plain-text summary: how many values in
// how many bins, and the fullest bin.
func LinearizeHistogram(values []float64, bins int) string {
	items, ok := histogramBins(values, bins)
	if !ok {
		return "Histogram: no data"
	}
	n, _, _, _ := stats(values)
	best := 0
	for i, it := range items {
		if it.Value > items[best].Value {
			best = i
		}
	}
	return "Histogram: " + strconv.Itoa(n) + " values in " + strconv.Itoa(len(items)) + " bins; fullest " +
		items[best].Label + " with " + num(items[best].Value)
}
