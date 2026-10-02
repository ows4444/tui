package appshell

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

// SetBounds records the rectangle child id occupies, in the same coordinates
// as the mouse events the Router receives. Unknown ids are recorded too and
// take effect once a child with that id is added.
func (r Router) SetBounds(id string, rect layout.Rect) Router {
	next := make(map[string]layout.Rect, len(r.bounds)+1)
	for k, v := range r.bounds {
		next[k] = v
	}
	next[id] = rect
	r.bounds = next
	return r
}

// SetLayout takes every child's rectangle from root laid out at size: the
// node named with the child's id (layout.Named) gives its rectangle. Children
// with no such node lose their bounds. The rectangles are relative to root's
// top-left, so the mouse coordinates must be too.
func (r Router) SetLayout(root layout.Node, size layout.Size) Router {
	next := make(map[string]layout.Rect, len(r.ids))
	for _, p := range layout.Rects(root, size) {
		if p.Name == "" || r.index(p.Name) < 0 {
			continue
		}
		if _, dup := next[p.Name]; !dup {
			next[p.Name] = p.Rect
		}
	}
	r.bounds = next
	return r
}

// routeMouse delivers ev to the topmost child (the one added last) whose
// bounds contain the pointer, with X and Y made local to those bounds. A left
// press also focuses that child; wheel, release and motion events leave focus
// alone. An event outside every child's bounds goes nowhere and changes
// nothing.
func (r Router) routeMouse(ev tui.MouseEvent) (Router, tui.Cmd) {
	for i := len(r.ids) - 1; i >= 0; i-- {
		rect, ok := r.bounds[r.ids[i]]
		if !ok || !rect.Contains(ev.X, ev.Y) {
			continue
		}
		if ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress {
			r.cur = i
		}
		ev.X, ev.Y = rect.Local(ev.X, ev.Y)
		return r.updateAt(i, ev)
	}
	return r, nil
}
