// Package focus is a small focus-traversal helper: a Ring tracks which of n
// items has focus and moves it with Tab and Shift+Tab, wrapping at the ends
// and skipping disabled items. It replaces the modulo arithmetic every
// multi-field screen otherwise hand-rolls (see examples/focus).
//
// Route and Bind go one step further: they blur the old widget, focus the new
// one and send every other message to the focused widget only, so a screen
// needs no per-widget switch (see examples/focus).
//
// Tab follows index order unless the Ring is given another with WithOrder.
// LayoutOrder derives one from a layout.Node, reading order of where each
// named field is drawn, and Zones gives RouteAuto the matching click regions,
// so neither tab order nor mouse focus is maintained by hand.
//
// Ring is a value type like the widget Models: every method returns the new
// Ring and leaves the receiver alone, so copies held elsewhere are never
// changed behind your back.
package focus

import "github.com/ows4444/tui"

// Ring is a set of n focusable items, numbered 0 to n-1, with one focused.
// The zero Ring has no items.
type Ring struct {
	n        int
	cur      int
	disabled map[int]bool
	// order is the traversal order set by WithOrder, a permutation of 0..n-1,
	// or nil for index order. It is never written after WithOrder builds it,
	// so Ring copies share it safely.
	order []int
	// parent is the scope this one was pushed over (nil at the root). It
	// points at a private copy, so Rings sharing it never see each other.
	parent *Ring
}

// New returns a Ring of n items with item 0 focused. n <= 0 gives an empty
// Ring.
func New(n int) Ring {
	if n < 0 {
		n = 0
	}
	return Ring{n: n}
}

// Push opens a new focus scope of n items over the receiver, with item 0
// focused, and returns it. While the scope is open Next, Prev, Set, Update and
// Route act on its items only, so Tab cannot reach the items of the scopes
// beneath it; this is how a modal traps focus. n <= 0 gives an empty scope.
// Pop returns to the scope the Push was made from, exactly as it was.
func (r Ring) Push(n int) Ring {
	under := r
	s := New(n)
	s.parent = &under
	return s
}

// Pop closes the innermost scope and returns the Ring as it was when Push was
// called, including which item had focus and which were disabled, so focus
// goes back to the widget that had it before the scope opened. Call Sync (or
// focus the item yourself) to tell the widget. Pop on a Ring with no open
// scope returns it unchanged.
func (r Ring) Pop() Ring {
	if r.parent == nil {
		return r
	}
	return *r.parent
}

// Depth is the number of open scopes pushed over the root: 0 for a Ring that
// never had Push called.
func (r Ring) Depth() int {
	d := 0
	for p := r.parent; p != nil; p = p.parent {
		d++
	}
	return d
}

// Len is the number of items, disabled or not.
func (r Ring) Len() int { return r.n }

// Current is the index of the focused item, or -1 for an empty Ring.
func (r Ring) Current() int {
	if r.n == 0 {
		return -1
	}
	return r.cur
}

// Focused reports whether item i has focus.
func (r Ring) Focused(i int) bool { return r.n > 0 && i == r.cur }

// Enabled reports whether item i exists and is not disabled.
func (r Ring) Enabled(i int) bool { return i >= 0 && i < r.n && !r.disabled[i] }

// Next moves focus to the next enabled item, wrapping from the last to the
// first. With no enabled item other than the current one, or none at all,
// focus stays where it is.
func (r Ring) Next() Ring { return r.step(1) }

// Prev moves focus to the previous enabled item, wrapping from the first to
// the last.
func (r Ring) Prev() Ring { return r.step(-1) }

func (r Ring) step(dir int) Ring {
	pos := r.cur
	if r.order != nil {
		for p, i := range r.order {
			if i == r.cur {
				pos = p
				break
			}
		}
	}
	for i := 1; i <= r.n; i++ {
		j := ((pos+dir*i)%r.n + r.n) % r.n
		if r.order != nil {
			j = r.order[j]
		}
		if r.Enabled(j) {
			r.cur = j
			return r
		}
	}
	return r
}

// Set moves focus to item i. An index out of range, or a disabled item,
// leaves focus unchanged.
func (r Ring) Set(i int) Ring {
	if r.Enabled(i) {
		r.cur = i
	}
	return r
}

// SetDisabled marks item i disabled (skipped by Next and Prev) or enabled
// again. Disabling the focused item moves focus to the next enabled one, if
// there is one. An out-of-range i is ignored.
func (r Ring) SetDisabled(i int, disabled bool) Ring {
	if i < 0 || i >= r.n {
		return r
	}
	// Copy on write: a Ring copy elsewhere keeps its own disabled set.
	m := make(map[int]bool, len(r.disabled)+1)
	for k, v := range r.disabled {
		m[k] = v
	}
	if disabled {
		m[i] = true
	} else {
		delete(m, i)
	}
	r.disabled = m
	if disabled && i == r.cur {
		r = r.step(1)
	}
	return r
}

// Update moves focus on Tab (next) and Shift+Tab (previous) and reports
// whether msg was one of those keys, so a caller can stop handling it:
//
//	if ring, moved := m.ring.Update(msg); moved {
//		m.ring = ring
//		return m, nil
//	}
//
// Shift+Tab arrives as a Tab key with the Shift modifier, which the input
// reader decodes from both ESC [ Z and the kitty keyboard protocol.
func (r Ring) Update(msg tui.Msg) (Ring, bool) {
	k, ok := msg.(tui.Key)
	if !ok || k.Type != tui.KeyTab {
		return r, false
	}
	if k.Mod.Shift() {
		return r.Prev(), true
	}
	return r.Next(), true
}
