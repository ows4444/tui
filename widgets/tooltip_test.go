package widgets

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestTooltipSizesToContent proves criterion #351: Tooltip with width 0
// renders a small bordered box sized to the text, styled via the given
// Theme's border color.
func TestTooltipSizesToContent(t *testing.T) {
	dt := theme.DarkTheme()
	got := Tooltip("hi", dt, 0)

	if !strings.Contains(got, "hi") {
		t.Fatalf("Tooltip output does not contain text %q: %q", "hi", got)
	}

	lines := strings.Split(got, "\n")
	if len(lines) < 3 {
		t.Fatalf("Tooltip should render a bordered box (>=3 lines), got %d lines: %q", len(lines), got)
	}

	topPlain := ansi.StripANSI(lines[0])
	wantTop := ansi.NewStyle().Foreground(dt.Info).Render(topPlain)
	if lines[0] != wantTop {
		t.Errorf("top border line = %q, want %q (colored via t.Info)", lines[0], wantTop)
	}

	// Sized to content: the content width should be small, not padded
	// out to some arbitrary fixed width.
	if w := ansi.Width(lines[0]); w > 10 {
		t.Errorf("Tooltip width = %d for 2-char text, want small (sized to content)", w)
	}
}

// TestTooltipWrapsToWidth proves criterion #352: a positive width wraps
// text longer than it, keeping every rendered line's ansi.Width within
// that width.
func TestTooltipWrapsToWidth(t *testing.T) {
	dt := theme.DarkTheme()
	width := 16
	text := "this is a fairly long tooltip message that must wrap across lines"

	got := Tooltip(text, dt, width)

	for i, line := range strings.Split(got, "\n") {
		if w := ansi.Width(line); w > width {
			t.Errorf("line %d width = %d, want <= %d; line=%q", i, w, width, line)
		}
	}
	if !strings.Contains(got, "tooltip") {
		t.Errorf("Tooltip output missing wrapped text content: %q", got)
	}
}

// grid returns a w x h rectangle of '.' characters, one row per line, a
// fixed-size base to test TooltipOverlay placement against.
func grid(w, h int) string {
	row := strings.Repeat(".", w)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// TestTooltipOverlayDefaultPlacement proves criterion #353: by default
// the tooltip is composited below and right-aligned to the anchor (its
// left edge starts at anchorX, its top edge at anchorY+1), reusing
// layout.Overlay.
func TestTooltipOverlayDefaultPlacement(t *testing.T) {
	dt := theme.DarkTheme()
	base := grid(40, 20)
	anchorX, anchorY := 5, 3

	got := TooltipOverlay(base, "hi", anchorX, anchorY, dt)
	tooltip := Tooltip("hi", dt, 0)
	tw, th := tooltipSize(tooltip)

	lines := strings.Split(got, "\n")
	tipLines := strings.Split(tooltip, "\n")

	for i := 0; i < th; i++ {
		row := anchorY + 1 + i
		if row >= len(lines) {
			t.Fatalf("expected tooltip row %d within base, base has %d lines", row, len(lines))
		}
		gotSegment := ansi.Truncate(ansi.TrimLeftWidth(lines[row], anchorX), tw)
		if gotSegment != tipLines[i] {
			t.Errorf("row %d at x=%d: got %q, want tooltip line %q", row, anchorX, gotSegment, tipLines[i])
		}
	}
}

// TestTooltipOverlayShiftsLeftWhenClipped proves criterion #354: when
// the default placement would push the tooltip's right edge past base's
// width, the tooltip is shifted left to stay within bounds instead of
// being clipped.
func TestTooltipOverlayShiftsLeftWhenClipped(t *testing.T) {
	dt := theme.DarkTheme()
	tooltip := Tooltip("a fairly long tooltip text", dt, 0)
	tw, _ := tooltipSize(tooltip)

	// base is just wide enough to hold the tooltip when it's flush
	// against the right edge, but not at the anchor's default position.
	baseWidth := tw + 5
	base := grid(baseWidth, 20)

	// Anchor near the right edge so the default (left edge at anchorX)
	// placement would overflow base's width.
	anchorX, anchorY := baseWidth-2, 3

	got := TooltipOverlay(base, "a fairly long tooltip text", anchorX, anchorY, dt)
	lines := strings.Split(got, "\n")

	for _, line := range lines {
		if w := ansi.Width(line); w > baseWidth {
			t.Fatalf("overlaid line width = %d, want <= base width %d: %q", w, baseWidth, line)
		}
	}

	if tw >= baseWidth {
		t.Fatalf("test setup invalid: tooltip width %d >= base width %d", tw, baseWidth)
	}
}

// TestTooltipOverlayFlipsAboveWhenNoRoomBelow proves criterion #355:
// when placing the tooltip below the anchor would push it past base's
// last line, the tooltip is placed above the anchor instead.
func TestTooltipOverlayFlipsAboveWhenNoRoomBelow(t *testing.T) {
	dt := theme.DarkTheme()
	baseHeight := 10
	base := grid(40, baseHeight)
	tooltip := Tooltip("hi", dt, 0)
	_, th := tooltipSize(tooltip)

	// Anchor near the bottom so default placement (below) would run
	// past base's last line.
	anchorX, anchorY := 5, baseHeight-1

	got := TooltipOverlay(base, "hi", anchorX, anchorY, dt)
	lines := strings.Split(got, "\n")
	tipLines := strings.Split(tooltip, "\n")

	wantTop := anchorY - th
	if wantTop < 0 {
		t.Fatalf("test setup invalid: tooltip height %d too tall for anchorY %d", th, anchorY)
	}

	tw, _ := tooltipSize(tooltip)
	for i := 0; i < th; i++ {
		row := wantTop + i
		gotSegment := ansi.Truncate(ansi.TrimLeftWidth(lines[row], anchorX), tw)
		if gotSegment != tipLines[i] {
			t.Errorf("row %d: got %q, want tooltip line %q (placed above anchor)", row, gotSegment, tipLines[i])
		}
	}

	// And it must not also appear starting at anchorY+1 (which would
	// run past base's last line at anchorY==baseHeight-1).
	if anchorY+1 < len(lines) {
		below := ansi.Truncate(ansi.TrimLeftWidth(lines[anchorY+1], anchorX), tw)
		if below == tipLines[0] {
			t.Errorf("tooltip placed below anchor despite no room; row %d = %q", anchorY+1, below)
		}
	}
}
