package layout

import (
	"strings"
	"testing"
)

// A box around a child that takes no room is its frame and nothing else.
// Box.Render gives empty content one blank row, which made the drawn box one
// row taller than Measure said; the last row, the bottom border, was clipped.
func TestBoxNodeAroundAnEmptyChildKeepsItsBottomBorder(t *testing.T) {
	unbounded := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	for _, tc := range []struct {
		name string
		box  Box
		size Size // Measure's answer when unbounded; a zero Size asks Measure
		want string
	}{
		{"border", NewBox().Border(NormalBorder()), Size{}, "┌┐\n└┘"},
		{"border and padding", NewBox().Border(NormalBorder()).PaddingAll(1), Size{},
			"┌──┐\n│  │\n│  │\n└──┘"},
		{"border, stretched wide", NewBox().Border(NormalBorder()), Size{W: 6, H: 2}, "┌────┐\n└────┘"},
		{"padding only", NewBox().PaddingAll(1), Size{}, "  \n  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := BoxNode(tc.box, Block(""))
			s := tc.size
			if s == (Size{}) {
				s = n.Measure(unbounded)
			}
			if got := n.Render(s); got != tc.want {
				t.Fatalf("Render(%+v) =\n%s\nwant\n%s", s, got, tc.want)
			}
		})
	}
}

// A child given height but no width (the box is only as wide as its frame)
// still gets its rows, so the bottom border sits on the last row.
func TestBoxNodeWithNoInnerWidthFillsItsHeight(t *testing.T) {
	got := BoxNode(NewBox().Border(NormalBorder()), Block("x")).Render(Size{W: 2, H: 4})
	if want := "┌┐\n││\n││\n└┘"; got != want {
		t.Fatalf("Render =\n%s\nwant\n%s", got, want)
	}
	if rows := strings.Count(got, "\n") + 1; rows != 4 {
		t.Fatalf("rows = %d, want 4", rows)
	}
}
