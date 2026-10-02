package widgets_test

import (
	"testing"

	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

// The deprecated forwarders in widgets return exactly what widgets/chart does,
// so callers can migrate one import at a time.
func TestDeprecatedChartForwardersMatchChart(t *testing.T) {
	th := theme.DarkTheme()
	vals := []float64{1, 4, 2, 8, 5}
	items := []widgets.BarItem{{Label: "a", Value: 3}, {Label: "bb", Value: 9}}
	var citems []chart.BarItem = items // BarItem is an alias, so the types are one
	cases := map[string][2]string{
		"Sparkline":     {widgets.Sparkline(vals), chart.Sparkline(vals)},
		"SparklineWith": {widgets.SparklineWith(vals, th), chart.SparklineWith(vals, th)},
		"BarChart":      {widgets.BarChart(items, 30, th), chart.BarChart(citems, 30, th)},
		"LineChart":     {widgets.LineChart(vals, 20, 5, th), chart.LineChart(vals, 20, 5, th)},
		"HeatMap":       {widgets.HeatMap([][]float64{{0, 1}, {2, 3}}, th), chart.HeatMap([][]float64{{0, 1}, {2, 3}}, th)},
		"Gauge":         {widgets.Gauge(0.4, 20, th), chart.Gauge(0.4, 20, th)},
	}
	for name, c := range cases {
		if c[0] == "" || c[0] != c[1] {
			t.Errorf("%s: forwarder %q, chart %q", name, c[0], c[1])
		}
	}
}
