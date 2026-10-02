package faces

import "time"

// anim is one animation script: a named loop of 4, 8 or 12 poses and the
// pace it reads best at. Positions in fx and offsets in dx/dy are in
// twelfths of the head, so a script draws correctly at either Size.
type anim struct {
	name     string
	interval time.Duration
	frames   func(look) []pose
}

func pair(e eyeState) [2]eyeState { return [2]eyeState{e, e} }

func (l look) rest() pose { return pose{eye: pair(eyeOpen), mouth: mIdle} }

func (p pose) eyes(s eyeState) pose     { p.eye = pair(s); return p }
func (p pose) brows(l, r browKind) pose { p.brow = [2]browKind{l, r}; return p }
func (p pose) says(k mouthKind) pose    { p.mouth = k; return p }
func (p pose) look(g int) pose          { p.gaze = g; return p }
func (p pose) at(dx, dy int) pose       { p.dx, p.dy = dx, dy; return p }
func (p pose) cheeks(on bool) pose      { p.blush = on; return p }
func (p pose) tears(l, r int) pose      { p.tear = [2]int{l, r}; return p }
func (p pose) tip(lit bool) pose        { p.ant = lit; return p }
func (p pose) with(f ...fx) pose        { p.fx = append([]fx(nil), f...); return p }

func seq(n int, f func(i int) pose) []pose {
	out := make([]pose, n)
	for i := range out {
		out[i] = f(i)
	}
	return out
}

var (
	animBlink = anim{"blink", 140 * time.Millisecond, func(l look) []pose {
		r := l.rest()
		return []pose{r, r, r, r.eyes(eyeHalf), r.eyes(eyeShut), r.eyes(eyeHalf), r.says(mSmile), r}
	}}

	animTalk = anim{"talk", 120 * time.Millisecond, func(l look) []pose {
		mouths := []mouthKind{mIdle, mOh, mBig, mOh, mSmile, mBig, mOh, mIdle}
		return seq(8, func(i int) pose {
			p := l.rest().says(mouths[i])
			if i == 4 {
				p = p.eyes(eyeShut)
			}
			return p
		})
	}}

	animLaugh = anim{"laugh", 110 * time.Millisecond, func(l look) []pose {
		type beat struct {
			dy    int
			eye   eyeState
			mouth mouthKind
			tear  int
			gaze  int
		}
		beats := []beat{
			{0, eyeHappy, mGrin, 0, 0},
			{2, eyeHappy, mBig, 0, 0},
			{0, eyeShut, mGrin, 1, 0},
			{2, eyeHappy, mBig, 1, 0},
			{0, eyeHappy, mGrin, 0, -1},
			{2, eyeShut, mBig, 0, 0},
			{0, eyeHappy, mBig, 1, 1},
			{2, eyeHappy, mGrin, 1, 0},
		}
		return seq(8, func(i int) pose {
			b := beats[i]
			return l.rest().eyes(b.eye).says(b.mouth).cheeks(true).tears(b.tear, b.tear).look(b.gaze).at(0, b.dy)
		})
	}}

	animSleep = anim{"sleep", 260 * time.Millisecond, func(l look) []pose {
		return seq(12, func(i int) pose {
			p := l.rest().eyes(eyeShut)
			if i >= 6 {
				p = p.says(mOh)
			}
			var zs []fx
			for k := 0; k < 3; k++ {
				age := (i - 4*k + 12) % 12
				if age < 8 {
					kind := fxZ
					if age >= 4 {
						kind = fxZBig
					}
					zs = append(zs, fx{kind, 13 + age/4, 4 - age})
				}
			}
			return p.with(zs...)
		})
	}}

	animLook = anim{"look", 150 * time.Millisecond, func(l look) []pose {
		gaze := []int{0, -1, -1, 0, 1, 1, 0, 0}
		return seq(8, func(i int) pose {
			p := l.rest().look(gaze[i])
			switch i {
			case 3:
				p = p.says(mSmile)
			case 7:
				p = p.eyes(eyeShut)
			}
			return p
		})
	}}

	animSurprise = anim{"surprise", 220 * time.Millisecond, func(l look) []pose {
		wide := l.rest().eyes(eyeWide).brows(browUp, browUp)
		return []pose{
			l.rest(),
			wide.says(mOh).with(fx{fxBang, 13, -2}),
			wide.says(mBig).with(fx{fxBang, 13, -2}, fx{fxBang, 15, -2}),
			wide.says(mOh).with(fx{fxBang, 13, 0}),
		}
	}}

	animAngry = anim{"angry", 90 * time.Millisecond, func(l look) []pose {
		dx := []int{0, -1, 1, -1, 1, -1, 1, 0}
		return seq(8, func(i int) pose {
			p := l.rest().brows(browAngry, browAngry).says(mFrown).at(dx[i], 0)
			if i%2 == 1 {
				p = p.says(mFlat)
			}
			if i >= 2 && i <= 6 {
				kind := fxSteam
				if i%2 == 0 {
					kind = fxSpark
				}
				p = p.with(fx{kind, 0, -4}, fx{kind, 8, -4})
			}
			return p
		})
	}}

	animCry = anim{"cry", 170 * time.Millisecond, func(l look) []pose {
		tear := []int{0, 1, 2, 3, 3, 4, 0, 0}
		return seq(8, func(i int) pose {
			p := l.rest().brows(browSad, browSad).says(mFrown).tears(tear[i], tear[i])
			if i == 4 || i == 5 {
				p = p.says(mOh)
			}
			if i == 5 {
				p = p.with(fx{fxSweat, 2, 13}, fx{fxSweat, 9, 13})
			}
			return p
		})
	}}

	animLove = anim{"love", 140 * time.Millisecond, func(l look) []pose {
		return seq(12, func(i int) pose {
			p := l.rest().eyes(eyeHappy).cheeks(true).says(mSmile)
			if i%4 >= 2 {
				p = p.says(mGrin)
			}
			var hs []fx
			for k := 0; k < 2; k++ {
				if age := (i + 6*k) % 12; age < 6 {
					hs = append(hs, fx{fxHeart, 13, 3 - age})
				}
			}
			return p.with(hs...)
		})
	}}

	animWink = anim{"wink", 200 * time.Millisecond, func(l look) []pose {
		r := l.rest().says(mSmile)
		w := r
		w.eye[1] = eyeShut
		return []pose{r, w, w.says(mGrin), r}
	}}

	animDizzy = anim{"dizzy", 100 * time.Millisecond, func(l look) []pose {
		orbit := [][2]int{{1, -3}, {5, -3}, {9, -3}, {13, 0}, {14, 5}, {13, 10},
			{9, 13}, {5, 13}, {1, 13}, {-4, 10}, {-4, 5}, {-4, 0}}
		gaze := []int{-1, 0, 1, 0}
		return seq(12, func(i int) pose {
			a, b := orbit[i], orbit[(i+6)%12]
			p := l.rest().eyes(eyeSpin0 + eyeState(i%4)).look(gaze[i%4])
			p = p.says([]mouthKind{mWave, mFlat}[i%2])
			return p.with(fx{fxSpark, a[0], a[1]}, fx{fxDot, b[0], b[1]})
		})
	}}

	animScan = anim{"scan", 110 * time.Millisecond, func(l look) []pose {
		gaze := []int{-1, -1, 0, 1, 1, 0, -1, -1, 0, 1, 1, 0}
		return seq(12, func(i int) pose {
			return l.rest().look(gaze[i]).tip(i%4 < 2)
		})
	}}

	animYawn = anim{"yawn", 190 * time.Millisecond, func(l look) []pose {
		r := l.rest()
		return []pose{
			r,
			r.eyes(eyeHalf).says(mOh),
			r.eyes(eyeShut).says(mBig),
			r.eyes(eyeShut).says(mBig),
			r.eyes(eyeShut).says(mBig),
			r.eyes(eyeShut).says(mOh),
			r.eyes(eyeHalf).says(mFlat),
			r,
		}
	}}

	animNervous = anim{"nervous", 100 * time.Millisecond, func(l look) []pose {
		gaze := []int{-1, 1, -1, 1, -1, 1, 0, 0}
		drop := []int{-1, -1, 1, 3, 5, 7, -1, -1} // y of the sweat drop; -1 = none
		return seq(8, func(i int) pose {
			p := l.rest().brows(browSad, browSad).look(gaze[i]).says([]mouthKind{mFlat, mFrown}[i%2])
			if drop[i] >= 0 {
				p = p.with(fx{fxSweat, 13, drop[i]})
			}
			return p
		})
	}}

	animThink = anim{"think", 200 * time.Millisecond, func(l look) []pose {
		return seq(12, func(i int) pose {
			p := l.rest().brows(browFlat, browUp).says(mFlat)
			if i >= 1 && i <= 9 {
				p = p.look(1)
			}
			if i == 7 {
				p = p.says(mOh)
			}
			if i == 10 {
				p = p.eyes(eyeShut)
			}
			var bs []fx
			if i >= 1 && i <= 10 {
				bs = append(bs, fx{fxBubbleS, 13, 4})
			}
			if i >= 4 && i <= 10 {
				bs = append(bs, fx{fxBubbleM, 14, 1})
			}
			if i >= 7 && i <= 10 {
				bs = append(bs, fx{fxBubbleL, 13, -4})
			}
			return p.with(bs...)
		})
	}}

	animCool = anim{"cool", 150 * time.Millisecond, func(l look) []pose {
		r := l.rest()
		return []pose{
			r,
			r.says(mSmile),
			r.eyes(eyeShadesHalf).says(mSmile),
			r.eyes(eyeShades).says(mSmile),
			r.eyes(eyeShades).says(mGrin),
			r.eyes(eyeShades).says(mFlat).brows(browFlat, browFlat),
			r.eyes(eyeShades).says(mSmile).look(1),
			r.eyes(eyeShadesHalf).says(mSmile),
		}
	}}

	animBounce = anim{"bounce", 110 * time.Millisecond, func(l look) []pose {
		// Crouch, squash, spring up, hang, land: two hops per loop.
		sparkA := []fx{{fxSpark, -4, 1}, {fxSpark, 13, 3}}
		sparkB := []fx{{fxSpark, -4, 4}, {fxSpark, 13, 0}}
		r := l.rest()
		return []pose{
			r.says(mSmile).at(0, 2),
			r.eyes(eyeHalf).says(mGrin).at(0, 2),
			r.eyes(eyeWide).says(mBig).with(sparkA...),
			r.eyes(eyeHappy).says(mGrin).with(sparkB...),
			r.eyes(eyeHappy).says(mSmile).at(0, 2),
			r.eyes(eyeHalf).says(mFlat).at(0, 2),
			r.eyes(eyeWide).says(mBig).with(sparkB...),
			r.eyes(eyeHappy).says(mGrin).with(sparkA...),
		}
	}}

	animBored = anim{"bored", 220 * time.Millisecond, func(l look) []pose {
		gaze := []int{0, 0, -1, -1, 0, 0, 1, 1}
		return seq(8, func(i int) pose {
			p := l.rest().eyes(eyeHalf).says(mFlat).look(gaze[i])
			if i == 4 {
				p = p.eyes(eyeShut)
			}
			switch i {
			case 5:
				p = p.with(fx{fxSteam, 13, 6})
			case 6:
				p = p.with(fx{fxSteam, 13, 6}, fx{fxSteam, 13, 3})
			}
			return p
		})
	}}

	animGlitch = anim{"glitch", 90 * time.Millisecond, func(l look) []pose {
		r := l.rest()
		return []pose{
			r,
			r.at(1, 0),
			r.eyes(eyeGlitch).says(mNoise),
			r.at(-1, 0),
			r.eyes(eyeGlitch),
			r,
			r.at(1, 0).eyes(eyeGlitch).says(mNoise),
			r,
			r.brows(browFlat, browFlat),
			r.at(-1, 0),
			r.eyes(eyeGlitch).cheeks(true),
			r,
		}
	}}

	animShiver = anim{"shiver", 70 * time.Millisecond, func(l look) []pose {
		flakes := [][]fx{{{fxFlake, -4, 2}}, {{fxFlake, 13, 4}}, {{fxFlake, -4, 5}}, {{fxFlake, 13, 7}}}
		mouth := []mouthKind{mZig, mZag, mWave, mNoise}
		return seq(4, func(i int) pose {
			return l.rest().at([]int{-1, 1, -1, 1}[i], 0).eyes(eyeSquint).says(mouth[i]).with(flakes[i]...)
		})
	}}

	animSmug = anim{"smug", 240 * time.Millisecond, func(l look) []pose {
		r := l.rest().eyes(eyeHalf).says(mSmile)
		return []pose{r, r.brows(browUp, browNone), r.says(mGrin), r.look(1)}
	}}

	animEvil = anim{"evil", 150 * time.Millisecond, func(l look) []pose {
		gaze := []int{0, -1, 0, 1, 0, -1, 0, 1}
		return seq(8, func(i int) pose {
			p := l.rest().brows(browAngry, browAngry).look(gaze[i]).says([]mouthKind{mGrin, mBig}[i%2])
			if i%2 == 1 {
				p = p.eyes(eyeHalf)
			}
			return p
		})
	}}
)
