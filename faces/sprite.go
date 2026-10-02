package faces

// fx is an effect at (x, y), in twelfths of the head, measured from the
// head's top-left corner (so x = 13 is just beyond its right edge and a
// negative y is above it).
type fx struct {
	kind fxKind
	x, y int
}

// pose is one animation frame: everything that can change between frames.
type pose struct {
	eye    [2]eyeState
	brow   [2]browKind
	mouth  mouthKind
	gaze   int    // -1/0/+1: eyes and brows look sideways
	dx, dy int    // head offset in twelfths: shake and bob
	blush  bool   // cheek marks
	tear   [2]int // 0 none, 1..4 how far a tear has run down
	ant    bool   // antenna lit (robots)
	fx     []fx
}

// look is what a catalog entry chooses: how its eyes and mouth are shaped.
type look struct {
	eyes  eyeStyle
	mouth mouthStyle
}

func (h *head) render(sz Size, lk look, p pose) string {
	g := sz.geom()
	s := g.head

	body := h.body(s)
	if h.cut != nil {
		h.cut(body, s)
	}
	cutFace(body, sz, lk, p)

	c := newBitmap(g.w, g.h)
	ox, oy := g.ox+g.u(p.dx), g.oy+g.u(p.dy)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			if body.get(x, y) {
				c.set(ox+x, oy+y, true)
			}
		}
	}
	if h.ext != nil {
		h.ext(c, ox, oy, s, p.ant)
	}
	for _, f := range p.fx {
		c.stamp(ox+g.u(f.x), oy+g.u(f.y), fxArts[sz][f.kind], true)
	}
	return c.String()
}

// cutFace punches the eyes, brows, mouth, blush and tears out of the body.
func cutFace(b *bitmap, sz Size, lk look, p pose) {
	g := sz.geom()
	s := g.head
	gx := g.u(p.gaze)

	if p.eye[0] == eyeShades || p.eye[0] == eyeShadesHalf {
		x0, w := g.eyeIn-1, s-2*(g.eyeIn-1)
		rows := pick(s, 2, 3)
		if p.eye[0] == eyeShadesHalf {
			rows = 1
		}
		for j := 0; j < rows; j++ {
			for i := 0; i < w; i++ {
				b.set(x0+i+gx, g.eyeTop+j+(g.eyeMax-pick(s, 2, 3))/2, false)
			}
		}
	} else {
		for i, left := range []bool{true, false} {
			a, dy := eyeShape(sz, lk.eyes, p.eye[i], left)
			x := g.eyeIn - eyeArts[sz][lk.eyes].width()/2
			if !left {
				x = s - x - eyeArts[sz][lk.eyes].width()
			}
			x += (eyeArts[sz][lk.eyes].width() - a.width()) / 2
			b.stamp(x+gx, g.eyeTop+dy, a, false)
		}
	}

	for i, left := range []bool{true, false} {
		k := p.brow[i]
		if k == browNone {
			continue
		}
		a := browArts[sz][k]
		x := g.eyeIn - a.width()/2
		if !left {
			a = a.mirror()
			x = s - x - a.width()
		}
		b.stamp(x+gx, g.eyeTop-len(a)-1, a, false)
	}

	m := mouthArt(sz, lk.mouth, p.mouth)
	b.stamp((s-m.width())/2, g.mouthBottom-len(m)+1, m, false)

	if p.blush {
		a := blushArt[sz]
		y := g.mouthBottom - pick(s, 1, 4)
		b.stamp(g.eyeIn-2, y, a, false)
		b.stamp(s-(g.eyeIn-2)-a.width(), y, a, false)
	}
	for i, left := range []bool{true, false} {
		if p.tear[i] == 0 {
			continue
		}
		x := g.eyeIn - 1 - eyeArts[sz][lk.eyes].width()/2
		if !left {
			x = s - 1 - x
		}
		y := g.eyeTop + g.eyeMax + p.tear[i]*pick(s, 1, 1) - 1
		b.set(x, y, false)
		if s >= 20 {
			b.set(x, y+1, false)
		}
	}
}
