// Package hittest does mouse hit-testing: which region of the screen did a click
// land on? A Rect is a rectangle of terminal cells in the same 0-indexed
// coordinates as input.MouseEvent (column X, row Y, origin top-left), and a Map
// is an ordered set of named Rects that answers "what is at this cell".
//
// Draw returns a string, not positions, but layout.Rects and layout.RectOf
// report where every node of a layout.Node tree is drawn: label the widgets
// with layout.Named and use the rectangles directly (hittest.Rect is layout.Rect). For a string
// layout, build the Map from the same sizes you gave the layout (Rect.CutTop,
// CutLeft and friends make that arithmetic exact). In inline mode (WithAltScreen(false)) mouse coordinates are absolute in the
// terminal, so offset your rectangles by where the live region starts.
package hittest

import (
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

// Rect is a rectangle of cells: X columns and Y rows from the top-left of the
// screen, W wide and H tall. A Rect with W <= 0 or H <= 0 is empty. It is an
// alias of layout.Rect, so a rectangle from layout.Rects or layout.RectOf
// needs no conversion and the two types have the same methods.
type Rect = layout.Rect

type region[ID comparable] struct {
	id   ID
	rect Rect
}

// Map is an ordered set of identified Rects. Regions added later are drawn on
// top, so they win where regions overlap (a dialog over a page). It is a
// value type like the widget Models: Add returns a new Map and never changes
// the receiver or a copy of it. The zero Map is empty and ready to use.
type Map[ID comparable] struct {
	regions []region[ID]
}

// Add returns a Map with r registered under id, on top of everything already
// in m. Empty rectangles are accepted but can never be hit.
func (m Map[ID]) Add(id ID, r Rect) Map[ID] {
	// A fresh slice, so two Maps built from the same parent never share
	// (and overwrite) a backing array.
	regions := make([]region[ID], len(m.regions), len(m.regions)+1)
	copy(regions, m.regions)
	m.regions = append(regions, region[ID]{id: id, rect: r})
	return m
}

// Len is the number of registered regions.
func (m Map[ID]) Len() int { return len(m.regions) }

// Hit is a successful hit test: which region, and the cell's position inside
// it.
type Hit[ID comparable] struct {
	ID     ID
	Rect   Rect
	LX, LY int // the cell relative to Rect's top-left corner
}

// At returns the topmost region containing cell (x, y), and false if there is
// none.
func (m Map[ID]) At(x, y int) (Hit[ID], bool) {
	for i := len(m.regions) - 1; i >= 0; i-- {
		if r := m.regions[i].rect; r.Contains(x, y) {
			lx, ly := r.Local(x, y)
			return Hit[ID]{ID: m.regions[i].id, Rect: r, LX: lx, LY: ly}, true
		}
	}
	return Hit[ID]{}, false
}

// AtEvent hit-tests the position of a mouse event. It ignores the button and
// action, so the caller decides whether a press, a release or motion counts.
func (m Map[ID]) AtEvent(ev input.MouseEvent) (Hit[ID], bool) { return m.At(ev.X, ev.Y) }
