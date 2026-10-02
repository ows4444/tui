package widgets

import (
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets/chart"
)

// BarItem is a single labeled value rendered as one row by BarChart.
//
// Deprecated: use chart.BarItem. The chart widgets moved to widgets/chart.
type BarItem = chart.BarItem

// BarChart renders items as one horizontal bar per row.
//
// Deprecated: use chart.BarChart.
func BarChart(items []BarItem, width int, t theme.Theme) string {
	return chart.BarChart(items, width, t)
}

// Gauge renders a semicircular meter.
//
// Deprecated: use chart.Gauge.
func Gauge(percent float64, width int, t theme.Theme) string { return chart.Gauge(percent, width, t) }

// HeatMap renders a 2D grid of values as shaded cells.
//
// Deprecated: use chart.HeatMap.
func HeatMap(values [][]float64, t theme.Theme) string { return chart.HeatMap(values, t) }

// LineChart renders values as a braille line plot.
//
// Deprecated: use chart.LineChart.
func LineChart(values []float64, width, height int, t theme.Theme) string {
	return chart.LineChart(values, width, height, t)
}

// Sparkline renders values as a single line of block characters.
//
// Deprecated: use chart.Sparkline.
func Sparkline(values []float64) string { return chart.Sparkline(values) }

// SparklineWith is Sparkline drawn with t's glyphs.
//
// Deprecated: use chart.SparklineWith.
func SparklineWith(values []float64, t theme.Theme) string { return chart.SparklineWith(values, t) }
