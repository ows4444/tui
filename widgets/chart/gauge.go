package chart

import (
	"fmt"
	"math"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/braille"
	"github.com/ows4444/tui/theme"
)

// gaugeMinWidth is the narrowest width that still fits the semicircle and
// its label; below it Gauge falls back to a single-line bar.
const gaugeMinWidth = braille.ArcMinWidth

// Gauge renders a semicircular meter width columns wide, filled from the
// left end of the arc clockwise to the right in proportion to percent
// (clamped to [0,1]; NaN counts as 0), with the rounded percentage centred
// under the arc. Like ProgressBar it is stateless: the caller passes the
// current value in on every render.
//
// The arc is drawn with braille dots (2x4 per cell), which are close to
// square on a terminal grid, so the half-ring keeps its proportions. The
// filled part uses t.Primary, the remainder t.Muted, and the label t.Text.
// Every line is exactly width columns wide and the line count depends only
// on width, so the layout never shifts as the value changes. Below
// gaugeMinWidth columns there's no room for the arc, and Gauge returns a
// one-line ProgressBar instead.
func Gauge(percent float64, width int, t theme.Theme) string {
	if width <= 0 {
		return ""
	}
	if math.IsNaN(percent) || percent < 0 {
		percent = 0
	} else if percent > 1 {
		percent = 1
	}
	if width < gaugeMinWidth {
		return bar(percent, width, t)
	}

	outer := float64(width) // outer radius in dot units: 2*width dots wide
	thick := math.Max(2, math.Round(outer/8))
	inner := outer - thick
	rows := int(math.Ceil(outer / 4))
	cx, cy := outer, float64(rows*4) // arc centre: middle of the bottom edge

	fill := ansi.NewStyle().Foreground(t.Primary)
	track := ansi.NewStyle().Foreground(t.Muted)

	// Progress along the arc: 0 at the left end, 1 at the right, sweeping
	// through the top (only the upper half-plane is ever tested, since the
	// centre sits on the bottom edge of the grid).
	cells := braille.Arc(width, rows, cx, cy, outer, inner, func(px, py, d float64) bool {
		return (math.Pi-math.Atan2(py, px))/math.Pi <= percent
	}, fill, track, t.GlyphSet())

	// The label overwrites the middle of the bottom row, which is inside
	// the ring: at gaugeMinWidth the ring clears the 4-column "100%" by
	// construction (see TestGaugeArcIsSymmetricRing).
	label := fmt.Sprintf("%d%%", int(math.Round(percent*100)))
	start := (width - len(label)) / 2
	text := ansi.NewStyle().Foreground(t.Text)
	for i := range label {
		cells[rows-1][start+i] = text.Render(label[i : i+1])
	}

	lines := make([]string, rows)
	for r := range cells {
		lines[r] = strings.Join(cells[r], "")
	}
	return strings.Join(lines, "\n")
}

// bar is the one-line meter Gauge falls back to below gaugeMinWidth. It draws
// exactly what widgets.ProgressBar does (a test holds them equal); it is local
// because widgets depends on this package for its forwarders, not the reverse.
func bar(percent float64, width int, t theme.Theme) string {
	switch {
	case percent < 0:
		percent = 0
	case percent > 1:
		percent = 1
	}
	filled := min(int(float64(width)*percent+0.5), width)
	fill := ansi.NewStyle().Foreground(t.Primary).Render(strings.Repeat(t.GlyphSet().BarFull, filled))
	track := ansi.NewStyle().Foreground(t.Muted).Render(strings.Repeat(t.GlyphSet().BarEmpty, width-filled))
	return fill + track
}
