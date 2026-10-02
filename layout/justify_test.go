package layout

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// TestJoinHorizontalJustifyDistribution proves criteria #615-#621: each
// Justify value distributes width-minus-blocks-width the way its own doc
// comment describes, against two 1-column blocks ("X", "Y") and width=10
// (leftover=8) so leading/between/trailing gaps are easy to count exactly.
func TestJoinHorizontalJustifyDistribution(t *testing.T) {
	tests := []struct {
		name    string
		justify Justify
		want    string
	}{
		{"start: flush left, no gap", JustifyStart, "XY"},
		{"end: flush right, single leading gap", JustifyEnd, strings.Repeat(" ", 8) + "XY"},
		{"center: leftover split before/after, odd extra leading", JustifyCenter, strings.Repeat(" ", 4) + "XY" + strings.Repeat(" ", 4)},
		{"space-between: one gap strictly between, no edges", JustifySpaceBetween, "X" + strings.Repeat(" ", 8) + "Y"},
		{"space-around: edges are half the between-gap", JustifySpaceAround, strings.Repeat(" ", 2) + "X" + strings.Repeat(" ", 4) + "Y" + strings.Repeat(" ", 2)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinHorizontalJustify(10, tt.justify, "X", "Y")
			if got != tt.want {
				t.Errorf("JoinHorizontalJustify(10, %v, \"X\", \"Y\") = %q, want %q", tt.justify, got, tt.want)
			}
		})
	}
}

// TestSpaceEvenlyGapsAreEqual proves criterion #620: leading, every
// between-gap and trailing are all the same width.
func TestSpaceEvenlyGapsAreEqual(t *testing.T) {
	got := JoinHorizontalJustify(10, JustifySpaceEvenly, "X", "Y")
	// leftover=8, n+1=3 slots -> unit=2 (8/3 truncates), remainder unused.
	want := strings.Repeat(" ", 2) + "X" + strings.Repeat(" ", 2) + "Y" + strings.Repeat(" ", 2)
	if got != want {
		t.Errorf("JoinHorizontalJustify(10, JustifySpaceEvenly, \"X\", \"Y\") = %q, want %q", got, want)
	}
}

// TestSpaceVariantsSingleBlockNoPanic proves criterion #621: SpaceBetween/
// SpaceAround/SpaceEvenly with only one block render it without panicking
// or dividing by zero.
func TestSpaceVariantsSingleBlockNoPanic(t *testing.T) {
	for _, justify := range []Justify{JustifySpaceBetween, JustifySpaceAround, JustifySpaceEvenly} {
		got := JoinHorizontalJustify(10, justify, "X")
		if !strings.Contains(got, "X") {
			t.Errorf("JoinHorizontalJustify(10, %v, \"X\") = %q, want it to contain the block", justify, got)
		}
	}
}

// TestOverflowClampsToZeroGap proves criterion #622: when blocks already
// fill or exceed width, no gap is added and the result may exceed width.
func TestOverflowClampsToZeroGap(t *testing.T) {
	for _, justify := range []Justify{JustifyStart, JustifyEnd, JustifyCenter, JustifySpaceBetween, JustifySpaceAround, JustifySpaceEvenly} {
		got := JoinHorizontalJustify(5, justify, "Hello", "World")
		want := "HelloWorld"
		if got != want {
			t.Errorf("JoinHorizontalJustify(5, %v, \"Hello\", \"World\") = %q, want %q (zero gap, overflowing width)", justify, got, want)
		}
		if w := ansi.Width(got); w <= 5 {
			t.Errorf("JoinHorizontalJustify(5, %v, ...) width = %d, want > 5 (target width, not a truncation limit)", justify, w)
		}
	}
}

// TestCrossAxisAlignmentUnaffected proves criterion #623: blocks of
// different heights are still top cross-axis-aligned, the same padding
// behavior joinHorizontalAlign(gap, AlignStart, ...) already has, whatever
// the main-axis Justify is.
func TestCrossAxisAlignmentUnaffected(t *testing.T) {
	tall := "A\nB"
	short := "C"

	got := JoinHorizontalJustify(10, JustifySpaceBetween, tall, short)
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (top-aligned to the taller block)", len(lines))
	}
	if !strings.HasPrefix(lines[0], "A") {
		t.Errorf("row 0 = %q, want to start with the tall block's first line", lines[0])
	}
	if !strings.Contains(lines[0], "C") {
		t.Errorf("row 0 = %q, want the short block's only line on row 0 (top-aligned)", lines[0])
	}
	if !strings.HasPrefix(lines[1], "B") {
		t.Errorf("row 1 = %q, want the tall block's second line", lines[1])
	}
	if strings.Contains(lines[1], "C") {
		t.Errorf("row 1 = %q, the short block has no second line and should not repeat its content", lines[1])
	}
}

func TestJoinHorizontalJustifyEmptyBlocks(t *testing.T) {
	if got := JoinHorizontalJustify(10, JustifyCenter); got != "" {
		t.Errorf("JoinHorizontalJustify with no blocks = %q, want empty", got)
	}
}
