package edit

import "sync/atomic"

// DefaultHistory is the number of undo steps kept when a widget does not
// configure one.
const DefaultHistory = 100

// Step is one reversible edit in rune offsets: replace Del runes at Pos with
// Ins, then put the cursor at Cursor. A Step on the undo stack describes how
// to undo an edit (Del is the length of what the edit inserted, Ins is what it
// removed); applying a Step yields the Step that reverses it, which goes on
// the other stack. A pure insertion therefore stores no text at all: the
// inserted runes are read back from the document when the undo happens.
type Step struct {
	Pos, Del int
	Ins      string
	Cursor   int
	// Join says the step below this one on its stack belongs with it: a widget
	// undoes or redoes steps until it meets one without Join. History sets it on
	// the steps recorded inside a group (see BeginGroup); a widget that undoes
	// a group sets it on the steps it pushes back, all but the first it applied.
	Join bool
}

type node struct {
	step  Step
	prev  *node
	depth int // number of nodes in the chain ending here
}

// History is a bounded undo/redo stack of Steps. The zero value is an empty
// history with the default bound. It is a persistent structure (immutable
// nodes), so copies of a widget Model that hold a History do not disturb one
// another. Recording a step costs one small allocation, and a run of typed
// characters is merged into a single step.
type History struct {
	undo, redo *node
	floor      int // undo nodes of depth <= floor are out of reach
	pool       *slab
	grouping   bool // Record joins steps into the group begun by BeginGroup
	fresh      bool // the next Record opens the group
}

// slab hands out nodes in batches so recording a step does not allocate every
// time. Slots are claimed atomically, so Model copies that share a slab never
// get the same node.
type slab struct {
	nodes [32]node
	used  atomic.Int32
}

func (h *History) alloc() *node {
	if s := h.pool; s != nil {
		if i := int(s.used.Add(1)) - 1; i < len(s.nodes) {
			return &s.nodes[i]
		}
	}
	s := new(slab)
	s.used.Store(1)
	h.pool = s
	return &s.nodes[0]
}

func bound(max int) int {
	if max == 0 {
		return DefaultHistory
	}
	return max
}

// Record pushes undo step s (and drops the redo stack). max bounds the number
// of undo steps kept: 0 means DefaultHistory and a negative value disables
// history. When coalesce is set and s is a pure insertion directly after the
// pure insertion on top of the stack, the two merge into one step, so a typing
// run undoes at once.
func (h *History) Record(s Step, coalesce bool, max int) {
	if max < 0 {
		h.Clear()
		return
	}
	max = bound(max)
	h.redo = nil
	if h.grouping {
		s.Join = !h.fresh
		coalesce = coalesce && !h.fresh
		h.fresh = false
	} else if h.fresh {
		coalesce, h.fresh = false, false
	}
	// A step with Ins == "" undoes a pure insertion of Del runes at Pos.
	if t := h.undo; coalesce && t != nil && t.depth > h.floor && s.Ins == "" && t.step.Ins == "" &&
		t.step.Pos+t.step.Del == s.Pos {
		m := t.step
		m.Del += s.Del
		n := h.alloc()
		*n = node{step: m, prev: t.prev, depth: t.depth}
		h.undo = n
		return
	}
	h.push(&h.undo, s, max)
}

// push puts s on the stack at *top and keeps the stack within 2*max nodes by
// rebuilding it from its newest max nodes when it grows past that, which makes
// the cost of trimming amortized constant.
func (h *History) push(top **node, s Step, max int) {
	d := 1
	if *top != nil {
		d = (*top).depth + 1
	}
	n := h.alloc()
	*n = node{step: s, prev: *top, depth: d}
	*top = n
	if top == &h.undo {
		if d-h.floor > max {
			h.floor = d - max
		}
		if d > 2*max {
			h.trim(max)
		}
	}
}

// trim rebuilds the undo chain keeping only its newest max nodes.
func (h *History) trim(max int) {
	keep := make([]Step, 0, max)
	for n := h.undo; n != nil && len(keep) < max; n = n.prev {
		keep = append(keep, n.step)
	}
	h.undo, h.floor = nil, 0
	for i := len(keep) - 1; i >= 0; i-- {
		d := 1
		if h.undo != nil {
			d = h.undo.depth + 1
		}
		n := h.alloc()
		*n = node{step: keep[i], prev: h.undo, depth: d}
		h.undo = n
	}
}

// BeginGroup makes the steps recorded from now until EndGroup one unit: the
// first starts a new unit (it never merges into a typing run before it) and
// the rest join it. A widget that undoes steps until one has no Join then
// undoes the whole group at once.
func (h *History) BeginGroup() { h.grouping, h.fresh = true, true }

// EndGroup ends the group begun by BeginGroup. The next step starts a unit of
// its own and does not merge into the group's last step.
func (h *History) EndGroup() { h.grouping, h.fresh = false, true }

// CanUndo reports whether an undo step is available.
func (h *History) CanUndo() bool { return h.undo != nil && h.undo.depth > h.floor }

// CanRedo reports whether a redo step is available.
func (h *History) CanRedo() bool { return h.redo != nil }

// Len is the number of undo steps currently reachable.
func (h *History) Len() int {
	if h.undo == nil {
		return 0
	}
	return h.undo.depth - h.floor
}

// Clear forgets all history.
func (h *History) Clear() { h.undo, h.redo, h.floor = nil, nil, 0 }

// PopUndo removes and returns the newest undo step.
func (h *History) PopUndo() (Step, bool) {
	if !h.CanUndo() {
		return Step{}, false
	}
	s := h.undo.step
	h.undo = h.undo.prev
	return s, true
}

// PopRedo removes and returns the newest redo step.
func (h *History) PopRedo() (Step, bool) {
	if h.redo == nil {
		return Step{}, false
	}
	s := h.redo.step
	h.redo = h.redo.prev
	return s, true
}

// PushRedo pushes the step that reverses an undo that was just applied.
func (h *History) PushRedo(s Step) { h.push(&h.redo, s, 0) }

// PushUndo pushes the step that reverses a redo that was just applied; unlike
// Record it keeps the redo stack. max is as for Record.
func (h *History) PushUndo(s Step, max int) {
	if max < 0 {
		return
	}
	h.push(&h.undo, s, bound(max))
}
