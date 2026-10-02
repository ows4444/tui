package splitpane

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func TestNilPanesAreEmpty(t *testing.T) {
	m := New(nil, nil)
	m.SetTotal(9)
	m.SetPos(4)
	if got := m.LayoutNode().Measure(layout.Unconstrained()); got != (layout.Size{W: 1, H: 0}) {
		t.Errorf("Measure with nil panes = %v, want just the divider", got)
	}
	for _, d := range []Direction{Columns, Rows} {
		m.Direction = d
		lines := strings.Split(m.LayoutNode().Render(layout.Size{W: 9, H: 3}), "\n")
		if len(lines) != 3 {
			t.Errorf("dir %d: %d rows, want 3", d, len(lines))
		}
		s := newSurf(9, 3)
		m.LayoutNode().(layout.CellNode).DrawCells(s, layout.Rect{W: 9, H: 3}) // nil panes draw nothing, no panic
	}
}

func TestDraggingChangesTheDividerStyle(t *testing.T) {
	m := New(layout.Block("a"), layout.Block("b"))
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 9, H: 3}
	m.SetTotal(9)
	m.SetPos(4)
	idle := m.LayoutNode().Render(layout.Size{W: 9, H: 1})
	m, _ = m.Update(tui.MouseEvent{X: 4, Y: 1, Action: tui.MouseActionPress, Button: tui.MouseButtonLeft})
	if !m.Dragging() {
		t.Fatal("press on the divider did not start a drag")
	}
	dragging := m.LayoutNode().Render(layout.Size{W: 9, H: 1})
	if idle == dragging {
		t.Error("the divider looks the same while dragging")
	}
	if ansi.StripANSI(idle) != ansi.StripANSI(dragging) {
		t.Error("dragging must change only the style")
	}
}

func TestDividerRectAndViewByDirection(t *testing.T) {
	m := New(layout.Block("a"), layout.Block("b"))
	if r := m.DividerRect(); r != (hittest.Rect{}) {
		t.Errorf("empty Bounds: DividerRect = %+v, want zero", r)
	}
	m.Bounds = hittest.Rect{X: 2, Y: 3, W: 9, H: 5}
	m.SetTotal(9)
	m.SetPos(4)
	if r := m.DividerRect(); r != (hittest.Rect{X: 6, Y: 3, W: 1, H: 5}) {
		t.Errorf("columns DividerRect = %+v", r)
	}
	m.Direction = Rows
	m.SetTotal(5)
	m.SetPos(2)
	if r := m.DividerRect(); r != (hittest.Rect{X: 2, Y: 5, W: 9, H: 1}) {
		t.Errorf("rows DividerRect = %+v", r)
	}
	// View with empty Bounds uses Total, one cell across, in both directions.
	for _, d := range []Direction{Columns, Rows} {
		v := New(layout.Block("a"), layout.Block("b"))
		v.Direction, v.Total = d, 7
		lines := strings.Split(v.View(), "\n")
		if d == Rows && len(lines) != 7 || d == Columns && (len(lines) != 1 || ansi.Width(lines[0]) != 7) {
			t.Errorf("dir %d: View = %q", d, v.View())
		}
	}
}

func TestTokensOverrideTheDividerColour(t *testing.T) {
	m := New(layout.Block("a"), layout.Block("b"))
	base := m.Tokens()
	red := theme.Tokens{Border: ansi.Red}
	m2 := m.WithTokens(red)
	if m2.Tokens().Border == base.Border {
		t.Error("WithTokens did not change the border colour token")
	}
	if m.Tokens() != base {
		t.Error("WithTokens changed the receiver's tokens")
	}
	m.SetTotal(5)
	m2.SetTotal(5)
	if m.LayoutNode().Render(layout.Size{W: 5, H: 1}) == m2.LayoutNode().Render(layout.Size{W: 5, H: 1}) {
		t.Error("the divider colour did not follow the token")
	}
}

// nodeOnly is a layout.Node that is not a CellNode, so drawPane renders it to
// a string and puts the rows, clipped, onto the surface.
type nodeOnly struct{ text string }

func (n nodeOnly) Measure(layout.Constraints) layout.Size { return layout.Size{W: len(n.text), H: 1} }
func (n nodeOnly) Render(s layout.Size) string {
	return strings.TrimRight(strings.Repeat(n.text+"\n", s.H+2), "\n") // more rows than fit
}

func TestDrawPaneFallbackClipsRows(t *testing.T) {
	m := New(nodeOnly{"AAAA"}, nodeOnly{"BBBB"})
	m.SetTotal(9)
	m.SetPos(4)
	s := newSurf(12, 6)
	m.LayoutNode().(layout.CellNode).DrawCells(s, layout.Rect{X: 1, Y: 1, W: 9, H: 2})
	if s.g[1][1] != "A" || s.g[1][6] != "B" || s.g[2][1] != "A" {
		t.Errorf("panes not drawn: %q %q %q", s.g[1][1], s.g[1][6], s.g[2][1])
	}
	if s.g[3][1] != "" || s.g[0][1] != "" {
		t.Error("a pane drew outside its rectangle")
	}
	// Under a narrower clip, nothing outside it is written.
	s2 := newSurf(12, 6)
	s2.clip = layout.Rect{X: 0, Y: 0, W: 3, H: 6}
	m.LayoutNode().(layout.CellNode).DrawCells(s2, layout.Rect{X: 1, Y: 1, W: 9, H: 2})
	if s2.g[1][6] != "" {
		t.Error("drew outside the surface's own clip")
	}
	if got := clipTo(layout.Rect{W: 2, H: 2}, layout.Rect{X: 5, Y: 5, W: 2, H: 2}); got != (layout.Rect{}) {
		t.Errorf("disjoint clipTo = %+v, want empty", got)
	}
}
