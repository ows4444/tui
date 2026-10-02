package layout

import (
	"strings"

	"github.com/ows4444/tui/ansi"
)

// Justify is a main-axis (horizontal) space distribution for
// JoinHorizontalJustify — CSS flexbox's justify-content, as distinct from
// Align's cross-axis alignment.
type Justify int

const (
	// JustifyStart packs blocks flush to the left with no gap between
	// them, matching joinHorizontal's own layout. It is the zero value.
	JustifyStart Justify = iota
	// JustifyEnd packs blocks flush to the right: all leftover width
	// becomes a single leading gap.
	JustifyEnd
	// JustifyCenter centers the blocks as a group, splitting leftover
	// width before and after them; an odd remainder goes to the leading
	// gap, matching Align's own placement-of-the-extra-unit convention.
	JustifyCenter
	// JustifySpaceBetween distributes leftover width as equal gaps
	// strictly between consecutive blocks, with no leading or trailing
	// gap. With fewer than two blocks there is no "between" to distribute
	// into, so it behaves like JustifyStart.
	JustifySpaceBetween
	// JustifySpaceAround gives each block an equal half-gap on both
	// sides, so the gap between two blocks (adjoining half-gaps) is
	// twice the width of the leading and trailing gaps.
	JustifySpaceAround
	// JustifySpaceEvenly distributes leftover width as equal gaps
	// before the first block, between every pair, and after the last.
	JustifySpaceEvenly
)

// JoinHorizontalJustify lays out blocks side by side within width, top
// cross-axis-aligned exactly as joinHorizontal is (see joinHorizontalAlign
// for that logic), distributing any leftover width along the main axis
// per justify. If the blocks' combined rendered width is already >= width,
// no gap is added and the result may exceed width — width is a target to
// fill, not a hard truncation limit. It is additive to joinHorizontal/
// joinHorizontalAlign, which take an explicit literal gap instead of
// computing one from a target width.
func JoinHorizontalJustify(width int, justify Justify, blocks ...string) string {
	if len(blocks) == 0 {
		return ""
	}

	split := make([][]string, len(blocks))
	widths := make([]int, len(blocks))
	maxLines := 0
	total := 0
	for i, blk := range blocks {
		split[i] = strings.Split(blk, "\n")
		if len(split[i]) > maxLines {
			maxLines = len(split[i])
		}
		for _, l := range split[i] {
			if w := ansi.Width(l); w > widths[i] {
				widths[i] = w
			}
		}
		total += widths[i]
	}

	leading, between, trailing := justifyGaps(justify, len(blocks), width-total)

	leadSpacer := strings.Repeat(" ", leading)
	betweenSpacer := strings.Repeat(" ", between)
	trailSpacer := strings.Repeat(" ", trailing)

	rows := make([]string, maxLines)
	for row := 0; row < maxLines; row++ {
		var line strings.Builder
		line.WriteString(leadSpacer)
		for i, lines := range split {
			var cell string
			if row < len(lines) { // top cross-axis alignment, matching joinHorizontal/AlignStart
				cell = lines[row]
			}
			pad := widths[i] - ansi.Width(cell)
			if pad < 0 {
				pad = 0
			}
			line.WriteString(cell + strings.Repeat(" ", pad))
			if i < len(split)-1 {
				line.WriteString(betweenSpacer)
			}
		}
		line.WriteString(trailSpacer)
		rows[row] = line.String()
	}
	return strings.Join(rows, "\n")
}

// justifyGaps computes the leading, between-block and trailing gap widths
// for justify given n blocks and leftover (width minus the blocks'
// combined rendered width, already possibly negative). A negative
// leftover clamps to zero gaps throughout, letting the result exceed
// width rather than go negative.
func justifyGaps(justify Justify, n, leftover int) (leading, between, trailing int) {
	if leftover < 0 {
		leftover = 0
	}

	switch justify {
	case JustifyEnd:
		leading = leftover
	case JustifyCenter:
		leading = (leftover + 1) / 2
		trailing = leftover - leading
	case JustifySpaceBetween:
		if n > 1 {
			between = leftover / (n - 1)
		}
	case JustifySpaceAround:
		half := leftover / (2 * n)
		leading, trailing = half, half
		between = half * 2
	case JustifySpaceEvenly:
		unit := leftover / (n + 1)
		leading, trailing = unit, unit
		between = unit
	default: // JustifyStart
	}
	return leading, between, trailing
}
