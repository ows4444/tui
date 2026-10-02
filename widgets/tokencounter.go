package widgets

import (
	"strconv"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TokenCounter renders a one-line token readout for an AI-chat UI:
// "1.2k / 8k tokens" with a limit, "1.2k tokens" without (limit <= 0).
// Counts are compact — exact below 1000, then k/M/B with one decimal
// (rounded half up, a trailing ".0" dropped) — and a negative used counts
// as 0. Like Badge it is stateless: pass the current numbers on every
// render.
//
// The colour warns as the limit nears: t.Muted below 80% of limit,
// t.Warning from 80% up to 100%, and bold t.Error from 100% on. With no
// limit there is nothing to warn about, so it stays t.Muted.
func TokenCounter(used, limit int, t theme.Theme) string {
	if used < 0 {
		used = 0
	}
	text := CompactCount(used)
	if limit > 0 {
		text += " / " + CompactCount(limit)
	}
	text += " tokens"

	return tokenCounterStyle(used, limit, t).Render(text)
}

// tokenCounterStyle returns the Muted/Warning/Error threshold style shared
// by TokenCounter and UsageMonitor: t.Muted below 80% of limit, t.Warning
// from 80% up to 100%, bold t.Error from 100% on, and always t.Muted when
// limit <= 0 (nothing to warn about).
func tokenCounterStyle(used, limit int, t theme.Theme) ansi.Style {
	switch {
	case limit <= 0:
		return ansi.NewStyle().Foreground(t.Muted)
	case used >= limit:
		return ansi.NewStyle().Bold().Foreground(t.Error)
	case int64(used)*5 >= int64(limit)*4:
		return ansi.NewStyle().Foreground(t.Warning)
	default:
		return ansi.NewStyle().Foreground(t.Muted)
	}
}

// CompactCount formats n (>= 0) as "999", "1.2k", "8k", "1.5M", ... A value
// that rounds up to 1000 of its unit moves to the next one (999950 -> "1M").
func CompactCount(n int) string {
	if n < 1000 {
		return strconv.Itoa(n)
	}
	units := []struct {
		div    int64
		suffix string
	}{{1e3, "k"}, {1e6, "M"}, {1e9, "B"}}
	for i, u := range units {
		tenths := (int64(n)*10 + u.div/2) / u.div
		if tenths >= 10000 && i < len(units)-1 {
			continue
		}
		if tenths%10 == 0 {
			return strconv.FormatInt(tenths/10, 10) + u.suffix
		}
		return strconv.FormatInt(tenths/10, 10) + "." + strconv.FormatInt(tenths%10, 10) + u.suffix
	}
	return strconv.Itoa(n) // unreachable
}
