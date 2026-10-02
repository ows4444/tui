package chart

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// TestLineChartEmpty proves criterion #417 (zero values -> "").
func TestLineChartEmpty(t *testing.T) {
	if got := LineChart(nil, 10, 4, theme.DarkTheme()); got != "" {
		t.Errorf("LineChart(nil, ...) = %q, want empty", got)
	}
	if got := LineChart([]float64{}, 10, 4, theme.DarkTheme()); got != "" {
		t.Errorf("LineChart([]float64{}, ...) = %q, want empty", got)
	}
}

// TestLineChartZeroSize proves LineChart doesn't panic and returns "" when
// given a non-positive width or height, regardless of values.
func TestLineChartZeroSize(t *testing.T) {
	for _, w := range []int{0, -1} {
		if got := LineChart([]float64{1, 2, 3}, w, 4, theme.DarkTheme()); got != "" {
			t.Errorf("LineChart(vals, %d, 4) = %q, want empty", w, got)
		}
	}
	for _, h := range []int{0, -1} {
		if got := LineChart([]float64{1, 2, 3}, 10, h, theme.DarkTheme()); got != "" {
			t.Errorf("LineChart(vals, 10, %d) = %q, want empty", h, got)
		}
	}
}

// TestLineChartSingleValue proves criterion #417 (single value -> one dot,
// no panic).
func TestLineChartSingleValue(t *testing.T) {
	got := LineChart([]float64{5}, 10, 4, theme.DarkTheme())
	if got == "" {
		t.Fatal("LineChart with 1 value returned empty, want a single dot")
	}
	count := 0
	for _, r := range got {
		if isBraille(r) {
			count++
		}
	}
	if count != 1 {
		t.Errorf("got %d braille cells, want exactly 1 for a single value", count)
	}
}

// TestLineChartWithinBounds proves criterion #418: every line's
// ansi.Width stays within width, and there are at most height lines.
func TestLineChartWithinBounds(t *testing.T) {
	cases := []struct {
		values        []float64
		width, height int
	}{
		{[]float64{1, 5, 2, 8, 3, 9, 0, 4}, 12, 6},
		{[]float64{1, 1, 1}, 5, 3},
		{[]float64{-3, 7, -1, 42}, 20, 10},
		{[]float64{1}, 3, 2},
		{[]float64{1, 2}, 1, 1},
	}
	for _, c := range cases {
		out := LineChart(c.values, c.width, c.height, theme.DarkTheme())
		lines := strings.Split(out, "\n")
		if len(lines) != c.height {
			t.Errorf("values=%v width=%d height=%d: got %d lines, want %d", c.values, c.width, c.height, len(lines), c.height)
		}
		for i, l := range lines {
			if w := ansi.Width(l); w > c.width {
				t.Errorf("values=%v width=%d height=%d: line %d width = %d, want <= %d", c.values, c.width, c.height, i, w, c.width)
			}
		}
	}
}

// TestLineChartScaling proves criterion #413: points are placed at a
// column evenly spaced across the plot width, and a row scaled between
// the slice's own min and max (min at bottom, max at top).
func TestLineChartScaling(t *testing.T) {
	// Two extreme values: the min value should light a dot near the
	// bottom-left, the max value a dot near the top-right.
	out := LineChart([]float64{0, 100}, 10, 8, theme.DarkTheme())
	lines := strings.Split(stripANSI(out), "\n")
	if len(lines) != 8 {
		t.Fatalf("got %d lines, want 8", len(lines))
	}
	// Top line should have a braille dot in its rightmost cell.
	firstLineRunes := []rune(lines[0])
	if !isBraille(firstLineRunes[len(firstLineRunes)-1]) {
		t.Errorf("top line last cell = %q, want a braille dot (max value plotted at top-right)", string(firstLineRunes[len(firstLineRunes)-1]))
	}
	// Bottom line should have a braille dot in its leftmost cell.
	lastLineRunes := []rune(lines[len(lines)-1])
	if !isBraille(lastLineRunes[0]) {
		t.Errorf("bottom line first cell = %q, want a braille dot (min value plotted at bottom-left)", string(lastLineRunes[0]))
	}
}

// TestLineChartBrailleEncoding proves criterion #414: LineChart uses the
// braille dot-cell technique (only braille runes and spaces, styled with
// t.Primary), not block characters.
func TestLineChartBrailleEncoding(t *testing.T) {
	out := LineChart([]float64{1, 5, 2, 9, 4}, 12, 5, theme.DarkTheme())
	seq := ansi.NewStyle().Foreground(theme.DarkTheme().Primary).Render("x")
	prefix := strings.SplitN(seq, "x", 2)[0]
	if !strings.Contains(out, prefix) {
		t.Errorf("output not styled with t.Primary: %q", out)
	}
	for _, l := range strings.Split(stripANSI(out), "\n") {
		for _, r := range l {
			if r != ' ' && !isBraille(r) {
				t.Errorf("line %q contains non-braille, non-space rune %q", l, r)
			}
		}
	}
}

// TestLineChartConnectsPoints proves criterion #415: consecutive points
// with a big vertical jump are connected by an interpolated line rather
// than leaving isolated dots (there should be lit dots in intermediate
// rows too, not just the two endpoint rows).
func TestLineChartConnectsPoints(t *testing.T) {
	out := LineChart([]float64{0, 100}, 4, 8, theme.DarkTheme())
	lines := strings.Split(stripANSI(out), "\n")
	// With only 2 values but 8 rows tall, a connected line should touch
	// every row between the top and bottom, not just the first and last.
	litRows := 0
	for _, l := range lines {
		for _, r := range l {
			if isBraille(r) {
				litRows++
				break
			}
		}
	}
	if litRows != len(lines) {
		t.Errorf("got %d lit rows out of %d, want every row touched by the connecting line", litRows, len(lines))
	}
}

// TestLineChartFlatSeries proves criterion #416: a flat series (min ==
// max) renders a horizontal line at the vertical middle.
func TestLineChartFlatSeries(t *testing.T) {
	out := LineChart([]float64{5, 5, 5, 5, 5}, 10, 8, theme.DarkTheme())
	lines := strings.Split(stripANSI(out), "\n")
	if len(lines) != 8 {
		t.Fatalf("got %d lines, want 8", len(lines))
	}
	mid := 8 / 2
	for i, l := range lines {
		hasDot := false
		for _, r := range l {
			if isBraille(r) {
				hasDot = true
				break
			}
		}
		if i == mid {
			if !hasDot {
				t.Errorf("line %d (middle band) has no dots, want the flat line there: %q", i, l)
			}
		} else if hasDot {
			t.Errorf("line %d is outside the vertical middle but has dots: %q", i, l)
		}
	}
}

// stripANSI removes SGR escape sequences so tests can inspect the plain
// braille/space content of a line.
func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
