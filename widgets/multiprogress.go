package widgets

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// ProgressStatus is the lifecycle state of a MultiProgress row, driving both
// its StatusIndicator text and color (via Variant).
type ProgressStatus int

const (
	// ProgressPending is a row not started yet ("Pending", muted). It is
	// the zero value.
	ProgressPending ProgressStatus = iota
	// ProgressRunning is a row in progress ("Running", Info colour).
	ProgressRunning
	// ProgressDone is a finished row ("Done", Success colour).
	ProgressDone
	// ProgressError is a failed row ("Error", Error colour).
	ProgressError
)

// String is the label StatusIndicator renders for s.
func (s ProgressStatus) String() string {
	switch s {
	case ProgressRunning:
		return "Running"
	case ProgressDone:
		return "Done"
	case ProgressError:
		return "Error"
	default:
		return "Pending"
	}
}

// Variant maps s to the Variant (and so the Theme color) StatusIndicator
// renders it with.
func (s ProgressStatus) Variant() Variant {
	switch s {
	case ProgressRunning:
		return VariantInfo
	case ProgressDone:
		return VariantSuccess
	case ProgressError:
		return VariantError
	default:
		return VariantNeutral
	}
}

// ProgressItem is one row of a MultiProgress: a labeled, independently
// tracked progress meter with its own lifecycle status.
type ProgressItem struct {
	Label   string
	Percent float64
	Status  ProgressStatus
}

// MultiProgress renders one row per item — label, status indicator, and a
// ProgressBar — stacked vertically as a layout.Column. Labels are
// padded to a common column across all rows following KeyValue's alignment
// convention (measured with ansi.Width), and status text is likewise padded
// to a common column so every row's bar starts at the same offset. The bar
// is sized to fill whatever of width remains after the label and status
// columns; there is no aggregate or combined-total row. An empty items
// renders as "".
func MultiProgress(items []ProgressItem, width int, t theme.Theme) string {
	if len(items) == 0 {
		return ""
	}

	labelWidth, statusWidth := 0, 0
	for _, it := range items {
		if w := ansi.Width(it.Label); w > labelWidth {
			labelWidth = w
		}
		if w := ansi.Width(it.Status.String()); w > statusWidth {
			statusWidth = w
		}
	}

	const gap = 1
	// Fixed overhead before the bar: label + 1 pad + status dot(1) + space(1)
	// + status text (padded) + trailing gap.
	fixed := labelWidth + 1 + 2 + statusWidth + gap
	barWidth := width - fixed
	if barWidth < 0 {
		barWidth = 0
	}

	rows := make([]string, len(items))
	for i, it := range items {
		labelPad := strings.Repeat(" ", labelWidth-ansi.Width(it.Label)+1)
		statusText := it.Status.String()
		statusPad := strings.Repeat(" ", statusWidth-ansi.Width(statusText))
		indicator := StatusIndicator(statusText, it.Status.Variant(), t) + statusPad
		bar := ProgressBar(it.Percent, barWidth, t)

		row := it.Label + labelPad + indicator + strings.Repeat(" ", gap) + bar
		if width > 0 && ansi.Width(row) > width {
			row = ansi.Truncate(row, width)
		}
		rows[i] = row
	}

	return stackBlocks(0, rows...)
}
