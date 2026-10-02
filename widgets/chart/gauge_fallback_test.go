package chart_test

import (
	"testing"

	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

// Below the arc's minimum width Gauge draws a one-line bar. chart keeps its own
// copy of that bar (widgets imports chart, so chart cannot import widgets);
// this holds the copy equal to widgets.ProgressBar.
func TestGaugeFallbackEqualsProgressBar(t *testing.T) {
	th := theme.DarkTheme()
	for _, w := range []int{1, 3, 7} {
		for _, p := range []float64{-1, 0, 0.3, 0.5, 0.99, 1, 2} {
			if got, want := chart.Gauge(p, w, th), widgets.ProgressBar(p, w, th); got != want {
				t.Errorf("Gauge(%v, %d) = %q, ProgressBar = %q", p, w, got, want)
			}
		}
	}
}
