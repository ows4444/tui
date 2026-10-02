package layout

import (
	"math/rand"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// refJoinHorizontalAlign is the original joinHorizontalAlign, kept verbatim
// as the reference the optimised version must match byte for byte.
func refJoinHorizontalAlign(gap int, align Align, blocks ...string) string {
	if len(blocks) == 0 {
		return ""
	}
	split := make([][]string, len(blocks))
	widths := make([]int, len(blocks))
	maxLines := 0
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
	}
	offsets := make([]int, len(blocks))
	for i, lines := range split {
		missing := maxLines - len(lines)
		switch align {
		case AlignCenter:
			offsets[i] = missing / 2
		case AlignEnd:
			offsets[i] = missing
		default:
			offsets[i] = 0
		}
	}
	spacer := strings.Repeat(" ", gap)
	rows := make([]string, maxLines)
	for row := 0; row < maxLines; row++ {
		var line strings.Builder
		for i, lines := range split {
			var cell string
			if r := row - offsets[i]; r >= 0 && r < len(lines) {
				cell = lines[r]
			}
			pad := widths[i] - ansi.Width(cell)
			if pad < 0 {
				pad = 0
			}
			line.WriteString(cell + strings.Repeat(" ", pad))
			if i < len(split)-1 {
				line.WriteString(spacer)
			}
		}
		rows[row] = line.String()
	}
	return strings.Join(rows, "\n")
}

var joinPieces = []string{"", "a", "hello", "你好", "\x1b[1mbold\x1b[0m", "│ x │", "line1\nline2", "a\n\nc", "tall\n1\n2\n3", "\n", "e\u0301"}

func TestJoinHorizontalAlignMatchesReference(t *testing.T) {
	r := rand.New(rand.NewSource(11))
	for i := 0; i < 60000; i++ {
		blocks := make([]string, r.Intn(7))
		for j := range blocks {
			blocks[j] = joinPieces[r.Intn(len(joinPieces))] + joinPieces[r.Intn(len(joinPieces))]
		}
		gap := r.Intn(5) // 0..4; the original panics on a negative gap (see below)
		align := Align(r.Intn(3))
		if got, want := joinHorizontalAlign(gap, align, blocks...), refJoinHorizontalAlign(gap, align, blocks...); got != want {
			t.Fatalf("gap %d align %d blocks %q:\n got %q\nwant %q", gap, align, blocks, got, want)
		}
	}
}

// The original joinHorizontalAlign panicked on a negative gap
// (strings.Repeat with a negative count); joinVertical already documents
// gap <= 0 as "none". A negative gap now means no gap.
func TestJoinHorizontalNegativeGapIsNoGap(t *testing.T) {
	blocks := []string{"ab", "cd\nef", "g"}
	for _, align := range []Align{AlignStart, AlignCenter, AlignEnd} {
		want := joinHorizontalAlign(0, align, blocks...)
		for _, gap := range []int{-1, -100} {
			if got := joinHorizontalAlign(gap, align, blocks...); got != want {
				t.Errorf("gap %d align %d = %q, want the gap-0 result %q", gap, align, got, want)
			}
		}
	}
	if got := joinHorizontal(-3, "x", "y"); got != "xy" {
		t.Errorf("joinHorizontal(-3, x, y) = %q", got)
	}
}
