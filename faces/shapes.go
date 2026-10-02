package faces

import "math"

// A head is a solid silhouette; the face is cut out of it afterwards. Some
// heads add a texture (a groove, a dither), and some add things stuck on
// the outside (ears, an antenna, whiskers).
type head struct {
	body func(s int) *bitmap
	cut  func(m *bitmap, s int)                   // texture cut into the body
	ext  func(c *bitmap, ox, oy, s int, lit bool) // dots outside the body
}

func superellipse(s int, n float64) *bitmap {
	b := newBitmap(s, s)
	h := float64(s) / 2
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			u, v := (float64(x)+0.5-h)/h, (float64(y)+0.5-h)/h
			b.set(x, y, math.Pow(math.Abs(u), n)+math.Pow(math.Abs(v), n) <= 1)
		}
	}
	return b
}

func roundRect(s, r int) *bitmap {
	b := newBitmap(s, s)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			cx, cy := x, y
			if x >= s-r {
				cx = s - 1 - x
			}
			if y >= s-r {
				cy = s - 1 - y
			}
			ok := true
			if cx < r && cy < r {
				dx, dy := float64(r)-float64(cx)-0.5, float64(r)-float64(cy)-0.5
				ok = dx*dx+dy*dy <= float64(r*r)
			}
			b.set(x, y, ok)
		}
	}
	return b
}

// pick returns small or large depending on the head size.
func pick(s, small, large int) int {
	if s >= 20 {
		return large
	}
	return small
}

func octagon(s int) *bitmap {
	b := newBitmap(s, s)
	c := pick(s, 3, 5)
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			dx, dy := min(x, s-1-x), min(y, s-1-y)
			b.set(x, y, dx >= c || dy >= c || dx+dy >= c-1)
		}
	}
	return b
}

// shield has a flat top and narrows to a chin.
func shield(s int) *bitmap {
	b := newBitmap(s, s)
	top := roundRect(s, pick(s, 2, 3))
	flat := int(float64(s) * 0.55)
	for y := 0; y < s; y++ {
		half := float64(s) / 2
		if y > flat {
			t := float64(y-flat) / float64(s-1-flat)
			half = half - t*(half-float64(pick(s, 2, 3)))
		}
		for x := 0; x < s; x++ {
			ok := math.Abs(float64(x)+0.5-float64(s)/2) <= half
			if y <= flat {
				ok = top.get(x, y)
			}
			b.set(x, y, ok)
		}
	}
	return b
}

// ghost is round on top with a scalloped hem.
func ghost(s int) *bitmap {
	b := superellipse(s, 2.2)
	for y := s / 2; y < s; y++ {
		for x := 0; x < s; x++ {
			b.set(x, y, math.Abs(float64(x)+0.5-float64(s)/2) <= float64(s)/2)
		}
	}
	if s < 20 {
		for x := 0; x < s; x++ {
			if x%4 == 1 || x%4 == 2 {
				b.set(x, s-1, false)
			}
		}
	} else {
		for x := 0; x < s; x++ {
			if m := x % 5; m >= 1 && m <= 3 {
				b.set(x, s-1, false)
			}
			if x%5 == 2 {
				b.set(x, s-2, false)
			}
		}
	}
	return b
}

func spiky(s int) *bitmap {
	b := superellipse(s, 2.6)
	if s < 20 {
		for x := 0; x < s; x++ {
			if x%4 == 0 || x%4 == 3 {
				b.set(x, 0, false)
			}
		}
		return b
	}
	for x := 0; x < s; x++ {
		if m := x % 5; m != 2 {
			b.set(x, 0, false)
		}
		if m := x % 5; m == 0 || m == 4 {
			b.set(x, 1, false)
		}
	}
	return b
}

// egg is wide at the brow and narrow at the chin.
func egg(s int) *bitmap {
	b := newBitmap(s, s)
	h := float64(s) / 2
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			u, v := (float64(x)+0.5-h)/h, (float64(y)+0.5-h)/h
			u /= 1 - 0.32*math.Pow((v+1)/2, 1.6)
			b.set(x, y, math.Pow(math.Abs(u), 2.2)+math.Pow(math.Abs(v), 2.2) <= 1)
		}
	}
	return b
}

// groove cuts a one-dot outline inset from the edge; dotted alternates.
func groove(m *bitmap, s, inset int, dotted bool) {
	for i := inset; i < s-inset; i++ {
		for _, p := range [][2]int{{i, inset}, {i, s - 1 - inset}, {inset, i}, {s - 1 - inset, i}} {
			if !dotted || (p[0]+p[1])%2 == 0 {
				m.set(p[0], p[1], false)
			}
		}
	}
}

func dither(m *bitmap, s int) {
	band := pick(s, 2, 3)
	for y := s - band; y < s; y++ {
		for x := 0; x < s; x++ {
			if (x+y)%2 == 0 {
				m.set(x, y, false)
			}
		}
	}
}

var (
	headRound    = &head{body: func(s int) *bitmap { return superellipse(s, 2.2) }}
	headSquare   = &head{body: func(s int) *bitmap { return roundRect(s, pick(s, 1, 2)) }}
	headSquircle = &head{body: func(s int) *bitmap { return superellipse(s, 4) }}
	headOctagon  = &head{body: octagon}
	headDither   = &head{
		body: func(s int) *bitmap { return roundRect(s, pick(s, 2, 3)) },
		cut:  dither,
	}
	headGhost  = &head{body: ghost}
	headDashed = &head{
		body: func(s int) *bitmap { return roundRect(s, pick(s, 2, 3)) },
		cut:  func(m *bitmap, s int) { groove(m, s, pick(s, 1, 2), true) },
	}
	headShield = &head{body: shield}
	headCRT    = &head{
		body: func(s int) *bitmap { return roundRect(s, pick(s, 2, 3)) },
		cut:  func(m *bitmap, s int) { groove(m, s, pick(s, 1, 2), false) },
	}
	headSpiky = &head{body: spiky}

	headRobot = &head{
		body: func(s int) *bitmap { return roundRect(s, pick(s, 1, 2)) },
		ext: func(c *bitmap, ox, oy, s int, lit bool) {
			mid := ox + s/2 - 1
			if s < 20 {
				c.stamp(mid, oy-1, art{"##"}, true)
				if lit {
					c.stamp(mid, oy-3, art{"##", "##"}, true)
				} else {
					c.stamp(mid, oy-2, art{"##"}, true)
				}
				c.stamp(ox-1, oy+4, art{"#", "#", "#"}, true)
				c.stamp(ox+s, oy+4, art{"#", "#", "#"}, true)
				return
			}
			c.stamp(mid, oy-2, art{"##", "##"}, true)
			if lit {
				c.stamp(mid-1, oy-5, art{".##.", "####", ".##."}, true)
			} else {
				c.stamp(mid-1, oy-5, art{".##.", "#..#", ".##."}, true)
			}
			c.stamp(ox-2, oy+7, art{"##", "##", "##", "##", "##"}, true)
			c.stamp(ox+s, oy+7, art{"##", "##", "##", "##", "##"}, true)
		},
	}
	headCat = &head{
		body: func(s int) *bitmap { return superellipse(s, 2.6) },
		ext: func(c *bitmap, ox, oy, s int, _ bool) {
			if s < 20 {
				ear := art{"#..", "##.", "###"}
				c.stamp(ox+1, oy-3, ear, true)
				c.stamp(ox+s-4, oy-3, ear.mirror(), true)
				for _, y := range []int{6, 8} {
					c.stamp(ox-3, oy+y, art{"##"}, true)
					c.stamp(ox+s+1, oy+y, art{"##"}, true)
				}
				return
			}
			ear := art{"#....", "##...", "###..", "####.", "#####"}
			c.stamp(ox+2, oy-5, ear, true)
			c.stamp(ox+s-7, oy-5, ear.mirror(), true)
			for _, y := range []int{10, 13} {
				c.stamp(ox-4, oy+y, art{"###"}, true)
				c.stamp(ox+s+1, oy+y, art{"###"}, true)
			}
		},
	}
	headAlien = &head{
		body: egg,
		ext: func(c *bitmap, ox, oy, s int, _ bool) {
			if s < 20 {
				for _, x := range []int{3, s - 4} {
					c.stamp(ox+x, oy-3, art{"#", ".", "#"}, true)
					c.stamp(ox+x, oy-3, art{"#"}, true)
				}
				return
			}
			for _, x := range []int{4, s - 6} {
				c.stamp(ox+x, oy-5, art{"##", "##", "#.", "#.", "#."}, true)
			}
		},
	}
	headBear = &head{
		body: func(s int) *bitmap { return superellipse(s, 2.2) },
		ext: func(c *bitmap, ox, oy, s int, _ bool) {
			if s < 20 {
				ear := art{".#.", "###", "###"}
				c.stamp(ox, oy-2, ear, true)
				c.stamp(ox+s-3, oy-2, ear, true)
				return
			}
			ear := art{".###.", "#####", "#####", "#####", ".###."}
			c.stamp(ox, oy-3, ear, true)
			c.stamp(ox+s-5, oy-3, ear, true)
		},
	}
)
