package layout

// Aspect makes n as large as fits in the room it is given while its width is
// w/h times its height, in cells, centred in what is left. A terminal cell is
// about twice as tall as it is wide, so Aspect(2, 1, n) is square on the
// screen and Aspect(16, 9, n) a wide picture would be Aspect(32, 9, n). With an
// unbounded room it takes n's natural size, grown on the short side to the ratio.
// A w or h under 1 is 1.
func Aspect(w, h int, n Node) Node { return aspectNode{w: max(w, 1), h: max(h, 1), node: n} }

type aspectNode struct {
	w, h int
	node Node
}

// fit is the largest size with the ratio inside room.
func (a aspectNode) fit(room Size) Size {
	if room.W <= 0 || room.H <= 0 {
		return Size{}
	}
	w := min(room.W, room.H*a.w/a.h)
	h := w * a.h / a.w
	if h > room.H {
		h = room.H
		w = h * a.w / a.h
	}
	return Size{W: w, H: h}
}

func (a aspectNode) Measure(c Constraints) Size {
	bw, bh := c.MaxW < Unbounded, c.MaxH < Unbounded
	switch {
	case bw && bh:
		return c.Constrain(a.fit(Size{W: c.MaxW, H: c.MaxH}))
	case bw:
		return c.Constrain(Size{W: c.MaxW, H: c.MaxW * a.h / a.w})
	case bh:
		return c.Constrain(Size{W: c.MaxH * a.w / a.h, H: c.MaxH})
	}
	nat := a.node.Measure(c)
	w := max(nat.W, (nat.H*a.w+a.h-1)/a.h)
	return c.Constrain(Size{W: w, H: (w*a.h + a.w - 1) / a.w})
}

// inner is the rectangle of the ratio centred in r.
func (a aspectNode) inner(r Size) Rect {
	f := a.fit(r)
	return Rect{X: (r.W - f.W) / 2, Y: (r.H - f.H) / 2, W: f.W, H: f.H}
}

func (a aspectNode) Render(s Size) string {
	if s.W <= 0 || s.H <= 0 {
		return ""
	}
	in := a.inner(s)
	if in.W <= 0 || in.H <= 0 {
		return Block("").Render(s)
	}
	return Overlay(Block("").Render(s), a.node.Render(Size{W: in.W, H: in.H}), in.X, in.Y)
}

func (a aspectNode) DrawCells(dst CellSurface, r Rect) {
	in := a.inner(Size{W: r.W, H: r.H})
	if in.W <= 0 || in.H <= 0 {
		return
	}
	prev := pushClip(dst, r)
	drawNode(a.node, dst, Rect{X: r.X + in.X, Y: r.Y + in.Y, W: in.W, H: in.H})
	dst.SetClip(prev)
}

func (a aspectNode) children(s Size) []child {
	in := a.inner(s)
	if in.W <= 0 || in.H <= 0 {
		return nil
	}
	return []child{{node: a.node, rect: in}}
}
