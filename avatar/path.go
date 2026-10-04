package avatar

import (
	"math"
	"strconv"
	"strings"
)

// A path is an SVG path as data: the commands that go into the markup, with every coordinate already rounded to two decimals. String gives
// the markup; flatten gives the outline the cell renderer fills.
type path []seg

// seg is one path command: M, L, H, V, Q, C or Z, with its n coordinates.
type seg struct {
	op byte
	v  [6]float64
	n  int
}

type point struct{ x, y float64 }

// round2 rounds to two decimals: to nearest, a tie going up (toward positive
// infinity), which is not what math.Round does for negative values.
func round2(v float64) float64 {
	x := v * 100
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	return r / 100
}

// num formats a rounded coordinate in the shortest form that reads back
// exactly, with no exponent.
func num(v float64) string {
	if v == 0 {
		return "0" // also negative zero
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (p *path) add(op byte, v ...float64) {
	s := seg{op: op, n: len(v)}
	for i, x := range v {
		s.v[i] = round2(x)
	}
	*p = append(*p, s)
}

// String returns the path data, the d attribute of an SVG path.
func (p path) String() string {
	var b strings.Builder
	for _, s := range p {
		b.WriteByte(s.op)
		for i := 0; i < s.n; i++ {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(num(s.v[i]))
		}
	}
	return b.String()
}

// Curves are flattened into this many straight pieces. The cell grid is far
// coarser than the 100-unit frame, so a handful is plenty.
const (
	cubicSteps = 8
	quadSteps  = 6
)

// flatten returns the closed outline as a polygon.
func (p path) flatten() []point {
	var out []point
	var cur point
	for _, s := range p {
		switch s.op {
		case 'M', 'L':
			cur = point{s.v[0], s.v[1]}
			out = append(out, cur)
		case 'H':
			cur.x = s.v[0]
			out = append(out, cur)
		case 'V':
			cur.y = s.v[0]
			out = append(out, cur)
		case 'Q':
			c, e := point{s.v[0], s.v[1]}, point{s.v[2], s.v[3]}
			for i := 1; i <= quadSteps; i++ {
				t := float64(i) / quadSteps
				u := 1 - t
				out = append(out, point{
					u*u*cur.x + 2*u*t*c.x + t*t*e.x,
					u*u*cur.y + 2*u*t*c.y + t*t*e.y,
				})
			}
			cur = e
		case 'C':
			c1, c2, e := point{s.v[0], s.v[1]}, point{s.v[2], s.v[3]}, point{s.v[4], s.v[5]}
			for i := 1; i <= cubicSteps; i++ {
				t := float64(i) / cubicSteps
				u := 1 - t
				out = append(out, point{
					u*u*u*cur.x + 3*u*u*t*c1.x + 3*u*t*t*c2.x + t*t*t*e.x,
					u*u*u*cur.y + 3*u*u*t*c1.y + 3*u*t*t*c2.y + t*t*t*e.y,
				})
			}
			cur = e
		}
	}
	return out
}

func rad(deg float64) float64 { return deg * math.Pi / 180 }

// superellipse traces |x/rx|^n + |y/ry|^n = 1, turned rot degrees clockwise,
// with one cubic Bezier per quadrant. n=2 is an ellipse, n near 4 a squircle.
// The control offset makes the curve pass through the 45-degree point; it is
// capped at the radius, past which the corner would bulge outside its box.
func superellipse(cx, cy, rx, ry, n, rot float64) path {
	k := math.Min(1, (8*math.Pow(2, -1/n)-4)/3)
	ak, bk := rx*k, ry*k
	pts := [13]point{
		{rx, 0},
		{rx, bk}, {ak, ry}, {0, ry},
		{-ak, ry}, {-rx, bk}, {-rx, 0},
		{-rx, -bk}, {-ak, -ry}, {0, -ry},
		{ak, -ry}, {rx, -bk}, {rx, 0},
	}
	// Sin and Cos are called separately: Sincos may differ from them in the
	// last bit, and the vectors pin the rounded coordinates.
	sin, cos := math.Sin(rad(rot)), math.Cos(rad(rot))
	at := func(i int) (float64, float64) {
		q := pts[i]
		return cx + q.x*cos - q.y*sin, cy + q.x*sin + q.y*cos
	}
	var p path
	x, y := at(0)
	p.add('M', x, y)
	for i := 1; i < 13; i += 3 {
		x1, y1 := at(i)
		x2, y2 := at(i + 1)
		x3, y3 := at(i + 2)
		p.add('C', x1, y1, x2, y2, x3, y3)
	}
	p.add('Z')
	return p
}

// splinePath is an organic closed curve: radii, as multiples of the base
// radius, sampled around a circle and joined by a closed Catmull-Rom spline.
// The spline passes through every point, so the radii mean what they say.
func splinePath(cx, cy, rx, ry float64, radii []float64, rot float64) path {
	n := len(radii)
	t0 := rad(rot)
	pts := make([]point, n)
	for i, m := range radii {
		a := t0 + 2*math.Pi*float64(i)/float64(n)
		pts[i] = point{cx + rx*m*math.Cos(a), cy + ry*m*math.Sin(a)}
	}
	at := func(i int) point { return pts[((i%n)+n)%n] }
	var p path
	p.add('M', at(0).x, at(0).y)
	for i := 0; i < n; i++ {
		p0, p1, p2, p3 := at(i-1), at(i), at(i+1), at(i+2)
		p.add('C',
			p1.x+(p2.x-p0.x)/6, p1.y+(p2.y-p0.y)/6,
			p2.x-(p3.x-p1.x)/6, p2.y-(p3.y-p1.y)/6,
			p2.x, p2.y)
	}
	p.add('Z')
	return p
}

// polygon is a regular polygon with a vertex at the top and rounded corners.
// Each corner is cut back along both of its edges by round (0 sharp, 1 cut to
// the edge midpoints) and joined by a quadratic through the vertex, which
// keeps the outline inside the polygon.
func polygon(cx, cy, rx, ry float64, sides int, round, rot float64) path {
	k := 0.0
	if round > 0 {
		k = math.Min(round, 1) / 2
	}
	t0 := rad(rot) - math.Pi/2
	v := make([]point, sides)
	for i := range v {
		a := t0 + 2*math.Pi*float64(i)/float64(sides)
		v[i] = point{cx + rx*math.Cos(a), cy + ry*math.Sin(a)}
	}
	at := func(i int) point { return v[((i%sides)+sides)%sides] }
	// cut is the cut point on the edge leaving vertex i toward vertex j.
	cut := func(i, j int) (float64, float64) {
		a, b := at(i), at(j)
		return a.x + (b.x-a.x)*k, a.y + (b.y-a.y)*k
	}
	var p path
	x, y := cut(0, -1)
	p.add('M', x, y)
	for i := 0; i < sides; i++ {
		c := at(i)
		x, y = cut(i, i+1)
		p.add('Q', c.x, c.y, x, y)
		if k < 0.5 {
			x, y = cut(i+1, i)
			p.add('L', x, y)
		}
	}
	p.add('Z')
	return p
}

// box is the straight run of a capsule; its two cap circles are drawn
// separately and the union is an exact stadium.
func box(cx, cy, rx, ry float64) path {
	var p path
	p.add('M', cx-rx, cy-ry)
	p.add('H', cx+rx)
	p.add('V', cy+ry)
	p.add('H', cx-rx)
	p.add('Z')
	return p
}

// taper is the point of a droplet: the two tangents from an apex, tip radii
// above the centre, to the body ellipse. The apex is eased with a quadratic,
// so the drawn tip stops just short of it.
func taper(cx, cy, rx, ry, tip float64) path {
	t := math.Max(1.05, tip)
	tx := rx * math.Sqrt(1-1/(t*t))
	ty := cy - ry/t
	apex := cy - t*ry
	px := tx * 0.14
	py := ty + 0.86*(apex-ty)
	var p path
	p.add('M', cx-tx, ty)
	p.add('L', cx-px, py)
	p.add('Q', cx, apex, cx+px, py)
	p.add('L', cx+tx, ty)
	p.add('Z')
	return p
}
