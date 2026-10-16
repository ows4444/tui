package chart_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets/chart"
)

func ExampleSparkline() {
	fmt.Println(ansi.StripANSI(chart.Sparkline([]float64{1, 3, 2, 8, 5, 9, 4})))
	// Output: ▁▃▂▇▅█▄
}

// A breakdown bar shows each part's share of a whole in one row, with a legend.
func ExampleBreakdownBar() {
	items := []chart.BarItem{{Label: "Go", Value: 50}, {Label: "Rust", Value: 30}, {Label: "C", Value: 20}}
	fmt.Println(ansi.StripANSI(chart.BreakdownBar(items, 20, theme.DarkTheme())))
	// Output:
	// ██████████▓▓▓▓▓▓▒▒▒▒
	// █ Go 50 (50%)
	// ▓ Rust 30 (30%)
	// ▒ C 20 (20%)
}

// A histogram counts values into equal-width bins and draws one bar per bin.
func ExampleHistogram() {
	fmt.Println(ansi.StripANSI(chart.Histogram([]float64{1, 2, 2, 3, 3, 3, 4, 4, 4, 4}, 2, 30, theme.DarkTheme().ASCII())))
	// Output:
	// [1, 2.5) #########------------
	// [2.5, 4] #####################
}
