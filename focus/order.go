package focus

import (
	"sort"

	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

// LayoutOrder returns the indices of names (item i is names[i]) in the order
// a reader meets them on screen: by where root, laid out at size s, draws the
// node labelled with that name (see layout.Named), top to bottom and then
// left to right. Items whose name is not drawn (absent, or clipped to
// nothing) follow the drawn ones in index order, so every index appears once.
// Pass the result to WithOrder:
//
//	ring = ring.WithOrder(focus.LayoutOrder(root, size, "name", "email", "submit"))
//
// Recompute it when the layout changes shape (a resize past a
// layout.Responsive breakpoint, say); focus itself is kept by WithOrder.
func LayoutOrder(root layout.Node, s layout.Size, names ...string) []int {
	at := placements(root, s, names)
	order := make([]int, len(names))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ra, oka := at[names[order[a]]]
		rb, okb := at[names[order[b]]]
		switch {
		case oka != okb:
			return oka // drawn items first
		case !oka:
			return false // both undrawn: keep index order
		case ra.Y != rb.Y:
			return ra.Y < rb.Y
		}
		return ra.X < rb.X
	})
	return order
}

// Zones returns the click regions of names, laid out as LayoutOrder does,
// with each region's ID the item's index: the Map RouteAuto takes, so a
// left click focuses the item under it. Where named nodes overlap the later
// one in drawing order wins, as in hittest.HitMap. Names not drawn have no
// region.
func Zones(root layout.Node, s layout.Size, names ...string) hittest.Map[int] {
	index := make(map[string]int, len(names))
	for i, n := range names {
		if _, dup := index[n]; !dup {
			index[n] = i
		}
	}
	var m hittest.Map[int]
	for _, p := range layout.Rects(root, s) {
		if i, ok := index[p.Name]; ok && p.Name != "" && !p.Rect.Empty() {
			m = m.Add(i, p.Rect)
		}
	}
	return m
}

// placements maps each of names that root draws at s to the first non-empty
// rectangle labelled with it, in drawing order.
func placements(root layout.Node, s layout.Size, names []string) map[string]layout.Rect {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	at := make(map[string]layout.Rect, len(names))
	for _, p := range layout.Rects(root, s) {
		if _, seen := at[p.Name]; want[p.Name] && !seen && !p.Rect.Empty() {
			at[p.Name] = p.Rect
		}
	}
	return at
}

// WithOrder returns r with Next and Prev (and so Tab, Shift+Tab, Update,
// Route and RouteAuto) moving along order instead of index order: order[0] is
// first, and the last wraps to it. Indices out of range and repeats are
// ignored, and items order leaves out are visited after the listed ones in
// index order, so every item stays reachable. A nil or empty order restores
// index order. Which item has focus, and which are disabled, do not change.
// LayoutOrder computes an order from a layout. A scope opened with Push
// starts in index order; Pop brings back the order of the scope beneath.
func (r Ring) WithOrder(order []int) Ring {
	if len(order) == 0 {
		r.order = nil
		return r
	}
	seen := make([]bool, r.n)
	norm := make([]int, 0, r.n)
	for _, i := range order {
		if i >= 0 && i < r.n && !seen[i] {
			seen[i] = true
			norm = append(norm, i)
		}
	}
	for i := 0; i < r.n; i++ {
		if !seen[i] {
			norm = append(norm, i)
		}
	}
	r.order = norm
	return r
}

// Order returns the traversal order Next follows: the order given to
// WithOrder, completed and cleaned, or 0..Len()-1 when none was set. The
// slice is a copy.
func (r Ring) Order() []int {
	out := make([]int, r.n)
	if r.order != nil {
		copy(out, r.order)
		return out
	}
	for i := range out {
		out[i] = i
	}
	return out
}
