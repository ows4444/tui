package layout

import "testing"

// A border side that BorderSides switched off takes no room. Measure counted
// two rows and two columns for any border, so a box with sides off measured
// larger than it drew and was padded out with blank rows and columns.
func TestBoxNodeMeasuresOnlyTheBorderSidesItDraws(t *testing.T) {
	unbounded := Constraints{MaxW: Unbounded, MaxH: Unbounded}
	border := NewBox().Border(NormalBorder())
	for _, tc := range []struct {
		name string
		box  Box
		size Size
		want string
	}{
		{"all four", border, Size{W: 3, H: 3}, "┌─┐\n│x│\n└─┘"},
		{"left and right", border.BorderSides(false, true, false, true), Size{W: 3, H: 1}, "│x│"},
		{"top and bottom", border.BorderSides(true, false, true, false), Size{W: 1, H: 3}, "─\nx\n─"},
		{"bottom only", border.BorderSides(false, false, true, false), Size{W: 1, H: 2}, "x\n─"},
		{"left only, padded", border.BorderSides(false, false, false, true).PaddingAll(1), Size{W: 4, H: 3},
			"│   \n│ x \n│   "},
		{"none", border.BorderSides(false, false, false, false), Size{W: 1, H: 1}, "x"},
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
