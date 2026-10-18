package main

import (
	"strconv"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

func palette(th theme.Theme, i int) ansi.Color {
	p := []ansi.Color{th.Primary, th.Secondary, th.Success, th.Warning, th.Error, th.Info}
	return p[i%len(p)]
}

// box makes tile i for the current mode; it is selected when the mode's own
// selector (child, column, cell) points at it.
func (m model) box(i int, th theme.Theme, w, h int, sel bool) layout.Node {
	return layout.Named("c"+strconv.Itoa(i), tile{label: strconv.Itoa(i + 1), w: w, h: h, col: palette(th, i), sel: sel})
}

// stage is the layout under test, before it is given its frame.
func (m model) stage(th theme.Theme) layout.Node {
	v := m.vals
	switch m.mode {
	case mFlex:
		var kids []layout.FlexChild
		for i := 0; i < v["count"]; i++ {
			k := strconv.Itoa(i)
			kids = append(kids, layout.FlexChild{
				Node: m.box(i, th, 8, 3, i == v["child"]), Basis: v["basis."+k], Grow: v["grow."+k], Shrink: v["shrink."+k],
				Min: v["min."+k], Max: v["max."+k], CrossAlign: layout.CrossAlign(v["cross."+k]),
			})
		}
		if v["dir"] == 1 {
			return layout.ColumnJustify(v["gap"], layout.Justify(v["justify"]), kids...)
		}
		return layout.RowJustify(v["gap"], layout.Justify(v["justify"]), kids...)
	case mGrid:
		tracks := make([]layout.Track, v["cols"])
		for j := range tracks {
			tracks[j] = layout.Track{Size: v["size."+strconv.Itoa(j)], Grow: v["grow."+strconv.Itoa(j)]}
		}
		cells := make([]layout.Node, v["cells"])
		for i := range cells {
			cells[i] = m.box(i, th, 6, 3, false)
		}
		cells[0] = layout.Span(cells[0], v["spanc"], v["spanr"])
		return layout.GridNodeGaps(tracks, v["colgap"], v["rowgap"], cells...)
	case mWrap:
		chips := make([]layout.Node, v["count"])
		for i := range chips {
			chips[i] = m.box(i, th, 6+(i%4)*3, 3, false)
		}
		return layout.Wrap(v["colgap"], v["rowgap"], chips...)
	}
	panel := layout.Named("panel", tile{label: "panel", w: v["width"], h: v["height"], col: th.Muted})
	badge := layout.Named("badge", tile{label: "badge", w: 9, h: 3, col: th.Warning})
	return layout.Stack(panel, layout.Anchor(layout.Corner(v["corner"]), v["dx"], v["dy"], badge))
}

// frame is the size of the stage: the settings, cut to the room there is.
func (m model) frame(availW, availH int) layout.Size {
	return layout.Size{W: min(m.vals["width"], availW), H: min(m.vals["height"], availH)}
}

// boxes are the named rectangles of the stage, in tree order.
func (m model) boxes(th theme.Theme, s layout.Size) []layout.Placed {
	var out []layout.Placed
	for _, p := range layout.Rects(layout.Fixed(m.stage(th), s), s) {
		if p.Name != "" {
			out = append(out, p)
		}
	}
	return out
}
