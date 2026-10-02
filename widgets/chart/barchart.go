package chart

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// BarItem is a single labeled value rendered as one row by BarChart.
type BarItem struct {
	Label string
	Value float64
}

// BarChart renders items as one horizontal bar per row, width columns
// wide: the Label (left-aligned, padded to the widest label across all
// items so every bar starts at the same column, the same alignment
// convention as widgets.KeyValue) followed by a bar whose filled length is
// proportional to Value scaled against the dataset's max Value, the same
// min/max-scaling approach as Sparkline. If the max Value is <= 0
// (all zero or negative), bars render zero-length rather than dividing by
// zero. An empty items slice renders as "".
func BarChart(items []BarItem, width int, t theme.Theme) string {
	if len(items) == 0 {
		return ""
	}
	g := t.GlyphSet()

	labelWidth := 0
	max := items[0].Value
	for _, it := range items {
		if w := ansi.Width(it.Label); w > labelWidth {
			labelWidth = w
		}
		if it.Value > max {
			max = it.Value
		}
	}

	barWidth := width - labelWidth - 1
	if barWidth < 0 {
		barWidth = 0
	}

	fillStyle := ansi.NewStyle().Foreground(t.Primary)
	trackStyle := ansi.NewStyle().Foreground(t.Muted)

	lines := make([]string, len(items))
	for i, it := range items {
		pad := strings.Repeat(" ", labelWidth-ansi.Width(it.Label)+1)

		filled := 0
		if max > 0 {
			frac := it.Value / max
			switch {
			case frac < 0:
				frac = 0
			case frac > 1:
				frac = 1
			}
			filled = int(frac*float64(barWidth) + 0.5)
		}
		if filled > barWidth {
			filled = barWidth
		}

		bar := fillStyle.Render(strings.Repeat(g.BarFull, filled)) +
			trackStyle.Render(strings.Repeat(g.BarEmpty, barWidth-filled))

		line := it.Label + pad + bar
		if ansi.Width(line) > width {
			line = ansi.Truncate(line, width)
		}
		lines[i] = line
	}

	return strings.Join(lines, "\n")
}
