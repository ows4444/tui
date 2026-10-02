package layout

import "testing"

// countNode is a leaf that counts Measure calls per constraint.
type countNode struct {
	calls map[Constraints]int
}

func (n *countNode) Measure(c Constraints) Size {
	n.calls[c]++
	return c.Constrain(Size{W: 5, H: 1})
}

func (n *countNode) Render(s Size) string { return Block("x").Render(s) }

func TestDrawMeasuresOncePerConstraint(t *testing.T) {
	a, b, c := &countNode{map[Constraints]int{}}, &countNode{map[Constraints]int{}}, &countNode{map[Constraints]int{}}
	ui := Column(0,
		FlexChild{Node: a},
		FlexChild{Grow: 1, Node: Row(1,
			FlexChild{Node: BoxNode(NewBox().Border(NormalBorder()), b), Basis: 10},
			FlexChild{Grow: 1, Node: GridNode([]Track{{}, {Grow: 1}}, 1, c, c)},
		)},
	)
	_ = Draw(ui, Constraints{MinW: 40, MaxW: 40, MinH: 10, MaxH: 10})
	for name, n := range map[string]*countNode{"a": a, "b": b, "c": c} {
		for cs, k := range n.calls {
			// c appears in two cells, so allow one call per occurrence.
			limit := 1
			if name == "c" {
				limit = 2
			}
			if k > limit {
				t.Errorf("%s measured %d times under %+v", name, k, cs)
			}
		}
	}
}
