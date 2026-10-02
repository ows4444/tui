package chart_test

import (
	"fmt"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/widgets/chart"
)

func ExampleSparkline() {
	fmt.Println(ansi.StripANSI(chart.Sparkline([]float64{1, 3, 2, 8, 5, 9, 4})))
	// Output: ▁▃▂▇▅█▄
}
