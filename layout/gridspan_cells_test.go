package layout_test

import (
	"testing"

	. "github.com/ows4444/tui/layout"
)

// DrawCells and Render must show the same screen for a grid with spans.
func TestSpanGridDrawsLikeItRenders(t *testing.T) {
	g := GridNode([]Track{{Size: 10}, {Size: 10}}, 1,
		Span(Block("WIDE"), 2, 1), Block("L"), Block("R"))
	sameScreen(t, "wide over two", g, Tight(Size{W: 21, H: 3}))
	sameScreen(t, "mixed", GridNodeGaps([]Track{{}, {Grow: 1}, {Size: 3}}, 1, 0,
		Block("head"), Span(Block("wide header"), 2, 1), Span(Block("1\n2\n3"), 1, 3),
		Block("a"), Block("b"), Block("c"), Block("d"), Block("e")), Tight(Size{W: 30, H: 8}))
}
