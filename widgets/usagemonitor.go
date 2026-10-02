package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// UsageStat is a single tracked quantity rendered as one UsageMonitor row:
// a label, its current Value, and an optional Limit (Limit <= 0 means "no
// limit").
type UsageStat struct {
	Label string
	Value int
	Limit int
}

// UsageMonitor renders a titled panel of usage stat rows, one per line,
// each formatted "label: value" or "label: value / limit" when Limit > 0.
// Values are formatted with TokenCounter's compact exact/k/M/B rule
// (CompactCount) and colored with the same Muted/Warning/Error threshold
// (tokenCounterStyle): Muted below 80% of limit, Warning from 80% up to
// 100%, bold Error at/over limit, and always Muted when Limit <= 0.
//
// Rows are aligned to a common label column following KeyValue's padding
// convention (padding measured via ansi.Width to the widest label). An
// empty stats slice renders just the title, or "" if title is also empty.
func UsageMonitor(title string, stats []UsageStat, t theme.Theme) string {
	if len(stats) == 0 {
		return title
	}

	width := 0
	for _, s := range stats {
		if w := ansi.Width(s.Label); w > width {
			width = w
		}
	}

	labelStyle := ansi.NewStyle().Foreground(t.Muted).Bold()

	lines := make([]string, 0, len(stats)+1)
	if title != "" {
		lines = append(lines, title)
	}

	for _, s := range stats {
		used := s.Value
		if used < 0 {
			used = 0
		}
		text := CompactCount(used)
		if s.Limit > 0 {
			text += " / " + CompactCount(s.Limit)
		}
		value := tokenCounterStyle(used, s.Limit, t).Render(text)

		pad := strings.Repeat(" ", width-ansi.Width(s.Label)+1)
		lines = append(lines, labelStyle.Render(s.Label+":")+pad+value)
	}

	return strings.Join(lines, "\n")
}
