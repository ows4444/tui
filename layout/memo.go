package layout

import "sync"

// memoizer is implemented by container nodes that can return a copy of
// themselves whose children memoise Measure. It is internal: Draw applies it
// once per call, so the cache lives exactly one frame and is never shared
// between goroutines.
type memoizer interface {
	memoize(a *arena) Node
}

// memoNode caches the Measure results of one node occurrence, keyed by
// Constraints, so a parent that probes a child several times (flex basis,
// cross placement, grid track sizing) reaches the child's Measure once per
// distinct constraint. Render is passed through.
type memoNode struct {
	n     Node
	a     *arena
	first memoEnt // inline: most nodes see one constraint
	used  bool
}

// memoEnt is one cached Measure; further ones chain through next, taken from
// the frame's arena.
type memoEnt struct {
	c    Constraints
	s    Size
	next *memoEnt
}

func (m *memoNode) Measure(c Constraints) Size {
	if m.used {
		for e := &m.first; e != nil; e = e.next {
			if e.c == c {
				return e.s
			}
		}
	}
	s := m.n.Measure(c)
	if !m.used {
		m.first, m.used = memoEnt{c: c, s: s}, true
		return s
	}
	e := &m.a.ents.take(1)[0]
	e.c, e.s = c, s
	e.next, m.first.next = m.first.next, e
	return s
}

func (m *memoNode) Render(s Size) string { return m.n.Render(s) }

func (m *memoNode) DrawCells(dst CellSurface, r Rect) { drawNode(m.n, dst, r) }

// memoized wraps n for one frame, taking its memo slot (and, for a container,
// its memoised copy) from arena a. A Windowed node is returned unwrapped so a
// Scroll parent can still discover that interface.
func memoized(n Node, a *arena) Node {
	if n == nil {
		return nil
	}
	if _, ok := n.(Windowed); ok {
		return n
	}
	if mz, ok := n.(memoizer); ok {
		n = mz.memoize(a)
	}
	m := &a.memos.take(1)[0]
	m.n, m.a = n, a
	return m
}

func (f flexNode) memoize(a *arena) Node {
	kids := a.kids.take(len(f.kids))
	for i, k := range f.kids {
		k.Node = memoized(k.Node, a)
		kids[i] = k
	}
	f.kids, f.ar = kids, a
	p := &a.flexes.take(1)[0]
	*p = f
	return p
}

func (g gridNode) memoize(a *arena) Node {
	cells := a.nodes.take(len(g.cells))
	for i, c := range g.cells {
		cells[i] = memoized(c, a)
	}
	g.cells, g.ar = cells, a
	p := &a.grids.take(1)[0]
	*p = g
	return p
}

func (b boxNode) memoize(a *arena) Node {
	b.child = memoized(b.child, a)
	p := &a.boxes.take(1)[0]
	*p = b
	return p
}

func (n scrollNode) memoize(a *arena) Node {
	n.child = memoized(n.child, a)
	p := &a.scrolls.take(1)[0]
	*p = n
	return p
}

func (o overlayNode) memoize(a *arena) Node {
	o.base = memoized(o.base, a)
	o.over = memoized(o.over, a)
	p := &a.overlays.take(1)[0]
	*p = o
	return p
}

// slab hands out zeroed runs of T from chunks that are never reallocated, so
// a pointer into one stays valid until reset.
type slab[T any] struct{ chunks [][]T }

func (s *slab[T]) take(n int) []T {
	for i, c := range s.chunks {
		if cap(c)-len(c) >= n {
			s.chunks[i] = c[:len(c)+n]
			return c[len(c) : len(c)+n : len(c)+n]
		}
	}
	c := make([]T, n, max(n, 64))
	s.chunks = append(s.chunks, c)
	return c[:n:n]
}

func (s *slab[T]) reset() {
	for i, c := range s.chunks {
		clear(c)
		s.chunks[i] = c[:0]
	}
}

// arena holds everything one Draw allocates for its memoised tree and its
// sizing scratch. Arenas are pooled, so after warm-up a Draw of a built-in
// container tree allocates nothing for them.
type arena struct {
	memos    slab[memoNode]
	ents     slab[memoEnt]
	flexes   slab[flexNode]
	grids    slab[gridNode]
	boxes    slab[boxNode]
	scrolls  slab[scrollNode]
	overlays slab[overlayNode]
	kids     slab[FlexChild]
	nodes    slab[Node]
	ints     slab[int]
	specs    slab[flexSpec]
}

func (a *arena) reset() {
	a.memos.reset()
	a.ents.reset()
	a.flexes.reset()
	a.grids.reset()
	a.boxes.reset()
	a.scrolls.reset()
	a.overlays.reset()
	a.kids.reset()
	a.nodes.reset()
	a.ints.reset()
	a.specs.reset()
}

var arenas = sync.Pool{New: func() any { return new(arena) }}

func getArena() *arena { return arenas.Get().(*arena) }

func putArena(a *arena) {
	a.reset()
	arenas.Put(a)
}

// intsFrom returns n zeroed ints from a, or from the heap when a is nil (a node
// used outside Draw).
func intsFrom(a *arena, n int) []int {
	if a == nil {
		return make([]int, n)
	}
	return a.ints.take(n)
}

func specsFrom(a *arena, n int) []flexSpec {
	if a == nil {
		return make([]flexSpec, n)
	}
	return a.specs.take(n)
}
