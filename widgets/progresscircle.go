package widgets

import (
	"fmt"
	"math"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/braille"
	"github.com/ows4444/tui/theme"
)

// progressCircleMinWidth is the narrowest width that still fits a full
// ring and its label; below it ProgressCircle falls back to a single-line
// ProgressBar, matching Gauge's fallback convention. A full circle needs
// dot-rows on both sides of the centre (roughly double Gauge's row count
// for the same width, since Gauge only ever fills the upper half-plane),
// so it needs more headroom than Gauge for the ring to still read clearly
// at the same width; progressCircleMinWidth is twice gaugeMinWidth to keep
// the same dot density per unit of arc.
const progressCircleMinWidth = 2 * braille.ArcMinWidth

// ProgressCircle renders a full 360-degree ring width columns wide, filled
// clockwise from the top in proportion to percent (clamped to [0,1]; NaN
// counts as 0), with the rounded percentage centred in the middle of the
// ring. Like Gauge and ProgressBar it is stateless: the caller passes the
// current value in on every render.
//
// It reuses Gauge's braille dot-arc-filling technique (brailleArc): the
// only differences are that the centre sits in the middle of the grid
// instead of on the bottom edge, so both the top and bottom halves of the
// ring are drawn, and the angle test accepts the full 0..2*Pi range instead
// of only the upper half-plane. The filled part uses t.Primary, the
// remainder t.Muted, and the label t.Text. Every line is exactly width
// columns wide and the line count depends only on width. Below
// progressCircleMinWidth columns there's no room for the ring, and
// ProgressCircle returns a one-line ProgressBar instead.
func ProgressCircle(percent float64, width int, t theme.Theme) string {
	if width <= 0 {
		return ""
	}
	if math.IsNaN(percent) || percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}
	if width < progressCircleMinWidth {
		return ProgressBar(percent, width, t)
	}

	outer := float64(width) // outer radius in dot units: 2*width dots wide
	thick := math.Max(2, math.Round(outer/8))
	inner := outer - thick
	rows := int(math.Ceil(outer / 2)) // full diameter tall: ~2x Gauge's rows
	cx, cy := outer, float64(rows*4)/2

	fill := ansi.NewStyle().Foreground(t.Primary)
	track := ansi.NewStyle().Foreground(t.Muted)

	// Progress around the ring: 0 at the top, sweeping clockwise through a
	// full turn back to the top at 1. Unlike Gauge, every angle is tested
	// since the centre now sits in the middle of the grid.
	cells := braille.Arc(width, rows, cx, cy, outer, inner, func(px, py, d float64) bool {
		theta := math.Atan2(px, py)
		if theta < 0 {
			theta += 2 * math.Pi
		}
		return theta/(2*math.Pi) <= percent
	}, fill, track, t.GlyphSet())

	// The label overwrites the middle row, inside the ring: the inner
	// radius comfortably clears the label at progressCircleMinWidth by
	// construction (see TestProgressCircleLabelClearOfRing).
	label := fmt.Sprintf("%d%%", int(math.Round(percent*100)))
	labelRow := rows / 2
	start := (width - len(label)) / 2
	text := ansi.NewStyle().Foreground(t.Text)
	for i := range label {
		cells[labelRow][start+i] = text.Render(label[i : i+1])
	}

	lines := make([]string, rows)
	for r := range cells {
		lines[r] = strings.Join(cells[r], "")
	}
	return strings.Join(lines, "\n")
}
