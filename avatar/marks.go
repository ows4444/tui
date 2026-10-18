package avatar

import "math"

// A mark is a stroke the pose draws over the body in the eyes' colour: an
// eyebrow or a mouth. A rest figure has none, so an avatar stays what its name
// made it; an expression adds them, because two eyes alone leave some poses
// (angry, smug, sleepy) too alike. A mark is thin, so it is drawn only where a
// pixel is small enough to hold it: in a few cells the picture stays two eyes.
type mark struct {
	p path
	// thick is the mark's smallest width in frame units.
	thick float64
}

// markShare is how much of a pixel a mark's thickness must reach for the mark
// to be drawn at all.
const markShare = 0.9

// marksFor returns the eyebrows and the mouth of pose p on figure f, moved
// (dx, dy) by the look, so they follow the face when it turns.
func (f figure) marksFor(p pose, dx, dy float64) []mark {
	var out []mark
	for i, b := range p.brow {
		if !b.on {
			continue
		}
		e := p.in(f, i)
		u := e.rx
		long := b.long
		if long == 0 {
			long = 1
		}
		half, thick := 1.5*u*long, 0.34*u
		cx := e.cx + dx + b.dx*f.face.rx
		cy := e.cy + dy - math.Max(e.ry, 0.6*u) - u + b.dy*f.face.ry
		out = append(out, mark{superellipse(cx, cy, half, thick, 4, b.rot), 2 * thick})
	}
	if p.mouth.kind != mouthNone {
		out = append(out, f.mouth(p.mouth, dx))
	}
	return out
}

// mouth draws m low on the face, with the look moving it half as far as the
// eyes.
func (f figure) mouth(m mouthPose, dx float64) mark {
	k := m.scale
	if k == 0 {
		k = 1
	}
	cx := f.face.cx + m.dx*f.face.rx + dx/2
	cy := f.face.cy + 0.62*f.face.ry
	switch m.kind {
	case mouthSmile:
		rx, ry := 0.26*f.face.rx*k, 0.17*f.face.ry*k
		return mark{dome(cx, cy-ry/2, rx, ry, m.rot), ry}
	case mouthFrown:
		rx, ry := 0.26*f.face.rx*k, 0.17*f.face.ry*k
		return mark{dome(cx, cy+ry/2, rx, ry, m.rot), ry}
	case mouthO:
		rx, ry := 0.11*f.face.rx*k, 0.15*f.face.ry*k
		return mark{superellipse(cx, cy, rx, ry, 2, m.rot), 2 * rx}
	}
	ry := math.Max(0.05*f.face.ry*k, 0.8)
	return mark{superellipse(cx, cy, 0.22*f.face.rx*k, ry, 4, m.rot), 2 * ry}
}
