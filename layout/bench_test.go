package layout

import (
	"fmt"
	"testing"
)

// BenchmarkBoxJoin measures joinHorizontal composing a large number of
// blocks — the pattern a wide row of cells (a table row, a toolbar, a
// grid row) goes through on every render.
func BenchmarkBoxJoin(b *testing.B) {
	const n = 50
	blocks := make([]string, n)
	for i := range blocks {
		blocks[i] = NewBox().PaddingAll(1).Border(NormalBorder()).Render(fmt.Sprintf("cell %d\nsecond line", i))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		joinHorizontal(1, blocks...)
	}
}

// BenchmarkNodeTree measures and renders a dashboard-shaped tree of Row,
// Column, BoxNode and GridNode over 120x40 cells: the cost of one frame of
// a nested Node layout, to compare against the string-based BoxJoin.
func BenchmarkNodeTree(b *testing.B) {
	cells := make([]Node, 24)
	for i := range cells {
		cells[i] = Block("cell " + string(rune('a'+i%26)))
	}
	ui := Column(0,
		FlexChild{Node: BoxNode(NewBox().Border(NormalBorder()), Block("header"))},
		FlexChild{Grow: 1, Node: Row(1,
			FlexChild{Node: BoxNode(NewBox().Border(NormalBorder()), Block("nav\nnav\nnav")), Basis: 20},
			FlexChild{Grow: 1, Node: GridNode([]Track{{}, {Grow: 1}, {Size: 8}}, 1, cells...)},
		)},
		FlexChild{Node: RowJustify(2, JustifySpaceBetween,
			FlexChild{Node: Block("status")}, FlexChild{Node: Block("q: quit"), CrossAlign: CrossEnd})},
	)
	c := Constraints{MinW: 120, MaxW: 120, MinH: 40, MaxH: 40}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Draw(ui, c)
	}
}
