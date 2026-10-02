package hittest

import "github.com/ows4444/tui/layout"

// HitMap returns a Map of every named node (see layout.Named) of the tree
// under root laid out at size s, keyed by name, with the rectangle each is
// drawn at (as layout.Rects reports it, clipped to its ancestors). A cell
// inside a named node's rectangle reports that name and a cell outside it does
// not. Where nodes overlap, At reports the topmost: the later node in drawing
// order (a nested node over its parent, a higher layout.Layer over a lower
// one), the same answer as layout.NamedAt. Nodes whose rectangle is empty are
// left out. It lives here rather than in package layout because hittest
// imports layout, not the other way round (the import tiers of
// internal/archtest).
func HitMap(root layout.Node, s layout.Size) Map[string] {
	var m Map[string]
	for _, p := range layout.Rects(root, s) {
		if p.Name != "" && !p.Rect.Empty() {
			m = m.Add(p.Name, p.Rect)
		}
	}
	return m
}
