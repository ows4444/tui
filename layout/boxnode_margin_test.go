package layout

import "testing"

// A box's margin is part of what it draws, so it is part of what it
// measures. Measure left it out, and the drawn box was clipped to the
// smaller size: the right and bottom borders were lost.
func TestBoxNodeMeasuresItsMargin(t *testing.T) {
	unbounded := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	for _, tc := range []struct {
		name string
		box  Box
		size Size
		want string
	}{
		{"every side", NewBox().Border(NormalBorder()).Margin(1, 1, 1, 1), Size{W: 5, H: 5},
			"     \n ┌─┐ \n │x│ \n └─┘ \n     "},
		{"top and left", NewBox().Border(NormalBorder()).Margin(1, 0, 0, 2), Size{W: 5, H: 4},
			"     \n  ┌─┐\n  │x│\n  └─┘"},
		{"no border", NewBox().Margin(0, 1, 1, 0), Size{W: 2, H: 2}, "x \n  "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := BoxNode(tc.box, Block("x"))
			s := n.Measure(unbounded)
			if s != tc.size {
				t.Errorf("Measure = %+v, want %+v", s, tc.size)
			}
			if got := n.Render(tc.size); got != tc.want {
				t.Fatalf("Render(%+v) =\n%s\nwant\n%s", tc.size, got, tc.want)
			}
		})
	}
}

// The child gets what is left after the margin too, so a box given a fixed
// size keeps its frame inside it.
func TestBoxNodeWithAMarginFitsTheSizeItIsGiven(t *testing.T) {
	n := BoxNode(NewBox().Border(NormalBorder()).Margin(0, 1, 0, 1), Block("abcdef"))
	if got, want := n.Render(Size{W: 7, H: 3}), " ┌───┐ \n │abc│ \n └───┘ "; got != want {
		t.Fatalf("Render =\n%s\nwant\n%s", got, want)
	}
}
