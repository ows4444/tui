package avatar

import "math"

// The ten silhouettes. Which one a name gets, the
// ranges every trait is read into and the tone set together are the frozen
// name-to-look mapping: changing any of them changes somebody's avatar.

type shapeKind int

const (
	shapeRound shapeKind = iota
	shapeOrganic
	shapeBoxy
	shapeCapsule
	shapeNub
	shapeCloud
	shapeDroplet
	shapeHexagon
	shapeSun
	shapeTriangle
)

// shapes holds, per silhouette, its name, the upper edge of its band in
// [0, 1) and how much of the frame its core body takes. The bands are
// weighted: round and organic are everyday, the louder shapes are finds.
var shapes = [...]struct {
	name string
	upTo float64
	core float64
}{
	shapeRound:    {"round", 0.22, 1},
	shapeOrganic:  {"organic", 0.48, 0.98},
	shapeBoxy:     {"boxy", 0.6, 0.86},
	shapeCapsule:  {"capsule", 0.7, 1.02},
	shapeNub:      {"nub", 0.79, 0.88},
	shapeCloud:    {"cloud", 0.86, 0.78},
	shapeDroplet:  {"droplet", 0.915, 0.78},
	shapeHexagon:  {"hexagon", 0.95, 1.05},
	shapeSun:      {"sun", 0.98, 0.7},
	shapeTriangle: {"triangle", 1, 1.15},
}

func pickShape(v float64) shapeKind {
	for k, s := range shapes {
		if v < s.upTo {
			return shapeKind(k)
		}
	}
	return shapeTriangle
}

// ellipse is a region: the body, or the part of it the eyes must stay inside.
type ellipse struct{ cx, cy, rx, ry float64 }

type circle struct{ cx, cy, r float64 }

func (e ellipse) shrunk(k float64) ellipse { return ellipse{e.cx, e.cy, e.rx * k, e.ry * k} }

// figure is one avatar's geometry in the 100 by 100 frame: the body is the
// union of petals, extra and core, and the eyes are drawn over it.
type figure struct {
	kind   shapeKind
	petals []circle
	extra  []path
	core   path
	eyes   [2]path
	// eyeAt is each eye's centre.
	eyeAt [2]point
	// face is the region the eyes were fitted into, and eye holds what each
	// was drawn from, so posed can draw them again somewhere else.
	face ellipse
	eye  [2]eyeSpec
}

// eyeSpec is one eye: a superellipse.
type eyeSpec struct{ cx, cy, rx, ry, n, rot float64 }

// gazeTravel is how far a full look moves the eyes, as a fraction of the
// face's radius on each axis.
const gazeTravel = 0.3

// in returns eye i as pose p shapes it, before any look, blink or boost.
func (p pose) in(f figure, i int) eyeSpec {
	e, q := f.eye[i], p.eye[i]
	out := eyeSpec{
		cx: e.cx + q.dx*f.face.rx, cy: e.cy + q.dy*f.face.ry,
		rx: e.rx * q.sx, ry: e.ry * q.sy, n: e.n, rot: e.rot,
	}
	if p.n > 0 {
		out.n = p.n
	}
	if q.turn {
		out.rot = q.rot
	}
	return out
}

// posed returns f with its eyes at the fraction at of the way from pose a to
// pose b (1 is b itself), moved by (lx, ly), each in [-1, 1], and sideways
// by shift frame units, closed to open of their height, and enlarged boost
// times about their centres. The body is unchanged, and the renderer draws an
// eye only where it is over the body.
func (f figure) posed(a, b pose, at, lx, ly, shift, open, boost float64) figure {
	dx, dy := lx*gazeTravel*f.face.rx+shift, ly*gazeTravel*f.face.ry
	mix := func(x, y float64) float64 { return x + (y-x)*at }
	// A dome and a capsule have no shape between them: the eye changes
	// kind half way.
	domed := b.dome
	if at < 0.5 {
		domed = a.dome
	}
	for i := range f.eye {
		from, to := a.in(f, i), b.in(f, i)
		cx, cy := mix(from.cx, to.cx)+dx, mix(from.cy, to.cy)+dy
		rx, ry := mix(from.rx, to.rx)*boost, mix(from.ry, to.ry)*open*boost
		n, rot := mix(from.n, to.n), mix(from.rot, to.rot)
		f.eyeAt[i] = point{cx, cy}
		f.eyes[i] = superellipse(cx, cy, rx, ry, n, rot)
		if domed {
			// The centre of a dome is on its flat edge; a third of the
			// way up is inside it.
			f.eyeAt[i] = point{cx, cy - ry/3}
			f.eyes[i] = dome(cx, cy+ry/2, rx, ry, rot)
		}
	}
	return f
}

func layoutFigure(t traits) figure {
	kind := pickShape(t.at("shape"))
	r := t.num("body.r", 31, 38) * shapes[kind].core
	b := ellipse{
		cx: 50 + t.jitter("body.x", 1.5),
		cy: 50 + t.jitter("body.y", 1.5),
		rx: r,
		ry: r * t.num("body.ratio", 0.92, 1.08),
	}
	n := t.num("body.n", 1.9, 2.5)
	radii := make([]float64, t.count("body.pts", 6, 8))
	for i := range radii {
		radii[i] = 1 + t.jitter("body.r"+string(rune('0'+i)), 0.16)
	}

	f := figure{kind: kind}
	face := b
	switch kind {
	case shapeRound:
		f.core = superellipse(b.cx, b.cy, b.rx, b.ry, n, 0)
	case shapeOrganic:
		face = b.shrunk(minOf(radii) * 0.95)
		f.core = splinePath(b.cx, b.cy, b.rx, b.ry, radii, 0)
	case shapeBoxy:
		// round, squared off and tilted
		f.core = superellipse(b.cx, b.cy, b.rx, b.ry, t.num("body.n", 3.4, 6), t.num("body.rot", -20, 20))
	case shapeCapsule:
		b.ry *= t.num("capsule.squat", 0.55, 0.68)
		face = b.shrunk(0.94)
		for _, s := range []float64{-1, 1} {
			f.petals = append(f.petals, circle{b.cx + s*(b.rx-b.ry), b.cy, b.ry})
		}
		f.core = box(b.cx, b.cy, b.rx-b.ry, b.ry)
	case shapeNub:
		for i := 0; i < t.count("nub.n", 1, 2); i++ {
			d := string(rune('0' + i))
			a := t.num("nub.a"+d, 0, 2*math.Pi)
			f.petals = append(f.petals, circle{
				b.cx + math.Cos(a)*b.rx*0.88,
				b.cy + math.Sin(a)*b.rx*0.88,
				b.rx * t.num("nub.r"+d, 0.24, 0.4),
			})
		}
		f.core = superellipse(b.cx, b.cy, b.rx, b.ry, n, 0)
	case shapeCloud:
		// organic, with lobes on the upper half
		face = b.shrunk(minOf(radii) * 0.95)
		count := t.count("cloud.n", 4, 6)
		for i := 0; i < count; i++ {
			a := math.Pi + math.Pi*(float64(i)+0.5)/float64(count)
			f.petals = append(f.petals, circle{
				b.cx + math.Cos(a)*b.rx*0.8,
				b.cy + math.Sin(a)*b.rx*0.5,
				b.rx * t.num("cloud.r"+string(rune('0'+i)), 0.44, 0.62),
			})
		}
		f.core = splinePath(b.cx, b.cy, b.rx, b.ry, radii, 0)
	case shapeDroplet:
		// Shifted down by what the taper adds above, so head and point
		// together sit centred. The taper is tangent to a true ellipse.
		b.cy += 0.22 * b.ry
		face = ellipse{b.cx, b.cy + b.ry*0.05, b.rx * 0.88, b.ry * 0.88}
		f.extra = append(f.extra, taper(b.cx, b.cy, b.rx, b.ry, t.num("droplet.tip", 1.4, 1.65)))
		f.core = superellipse(b.cx, b.cy, b.rx, b.ry, 2, 0)
	case shapeHexagon:
		face = b.shrunk(0.84)
		f.core = polygon(b.cx, b.cy, b.rx, b.ry, 6, t.num("poly.round", 0.24, 0.5), t.num("body.rot", -12, 12))
	case shapeSun:
		count := t.count("sun.n", 6, 9)
		dist := b.rx * t.num("sun.dist", 1.0, 1.08)
		pr := b.rx * t.num("sun.r", 0.2, 0.26)
		off := t.num("sun.rot", 0, 2*math.Pi)
		for i := 0; i < count; i++ {
			a := off + 2*math.Pi*float64(i)/float64(count)
			f.petals = append(f.petals, circle{b.cx + math.Cos(a)*dist, b.cy + math.Sin(a)*dist, pr})
		}
		f.core = superellipse(b.cx, b.cy, b.rx, b.ry, n, 0)
	case shapeTriangle:
		// a tighter tilt than the hexagon, so it rests on its base
		face = ellipse{b.cx, b.cy + b.ry*0.1, b.rx * 0.54, b.ry * 0.36}
		f.core = polygon(b.cx, b.cy, b.rx, b.ry, 3, t.num("poly.round", 0.24, 0.5), t.num("body.rot", -5, 5))
	}
	fitEyes(&f, t, b.rx, face)
	return f
}

func minOf(v []float64) float64 {
	m := v[0]
	for _, x := range v[1:] {
		m = math.Min(m, x)
	}
	return m
}

// fitEyes places the two capsule eyes inside face. Their size is a fraction
// of the body radius rx, and the whole cluster is scaled down when it would
// otherwise reach past 90% of the face on either axis.
func fitEyes(f *figure, t traits, rx float64, face ellipse) {
	er0 := t.num("eye.rx", 0.075, 0.105) * rx
	ratio := t.num("eye.ratio", 1.9, 3.2)
	scale := t.num("eye.scale", 0.78, 1.24)
	stretch := t.num("eye.stretch", 0.85, 1.18)
	clearance := t.num("eye.gap", 0.1, 0.24) * rx
	wide := er0 * math.Max(1, scale)
	tall := er0 * ratio * math.Max(1, scale*stretch)
	gap0 := wide + rx*0.03 + clearance

	gx := t.jitter("gaze.x", 0.09) * face.rx
	gy := t.num("gaze.y", -0.2, 0.08) * face.ry
	dy := t.jitter("eye.dy", 0.04) * face.ry
	reach := math.Hypot(wide, tall)
	need := math.Hypot(
		(math.Abs(gx)+gap0+reach)/face.rx,
		(math.Abs(gy)+math.Abs(dy)+reach)/face.ry,
	)
	fit := 1.0
	if need > 0.9 {
		fit = 0.9 / need
	}

	er := er0 * fit
	eyeRy := er * ratio
	gap := gap0 * fit
	room := math.Max(0, math.Min(1, clearance/tall))
	bound := math.Min(12, math.Asin(room)*180/math.Pi)
	lean := t.num("eye.lean", -1, 1) * bound
	lean2 := math.Max(-12, math.Min(12, lean+t.jitter("eye.lean2", 3.5)))

	cx := face.cx + gx*fit
	cy := face.cy + gy*fit
	en := t.num("eye.n", 3.5, 6)
	f.face = face
	f.eye = [2]eyeSpec{
		{cx - gap, cy, er, eyeRy, en, lean},
		{cx + gap, cy + dy*fit, er * scale, eyeRy * scale * stretch, en, lean2},
	}
	for i, e := range f.eye {
		f.eyeAt[i] = point{e.cx, e.cy}
		f.eyes[i] = superellipse(e.cx, e.cy, e.rx, e.ry, e.n, e.rot)
	}
}
