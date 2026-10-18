package main

// The three scenes are simulations with no screen in them. They use integers
// only, so a frame is the same on every machine and the golden files hold.

const maxHeat = 36

// fire is the doom-fire effect: heat rises from the bottom row, loses a little
// as it goes, and drifts sideways at random (and with the wind).
type fire struct {
	w, h int
	heat []uint8
	seed uint64
	on   bool // the bottom row burns
}

func newFire(w, h int) fire {
	f := fire{w: w, h: h, heat: make([]uint8, max(0, w*h)), seed: 0x9e3779b97f4a7c15, on: true}
	for range 40 {
		f = f.step(0)
	}
	return f
}

func (f *fire) rand() uint64 {
	f.seed ^= f.seed << 13
	f.seed ^= f.seed >> 7
	f.seed ^= f.seed << 17
	return f.seed
}

// step spreads the heat one row up. wind is -1, 0 or 1.
func (f fire) step(wind int) fire {
	f.heat = append([]uint8(nil), f.heat...)
	if f.w == 0 || f.h == 0 {
		return f
	}
	for x := 0; x < f.w; x++ {
		bottom := &f.heat[(f.h-1)*f.w+x]
		switch {
		case f.on:
			*bottom = maxHeat
		case *bottom > 0:
			*bottom--
		}
	}
	// The flame should reach about seven tenths of the way up whatever the
	// height, so the heat lost per row (in thousandths) depends on it.
	rate := max(300, maxHeat*10000/(7*f.h))
	for y := 1; y < f.h; y++ {
		for x := 0; x < f.w; x++ {
			p := int(f.heat[y*f.w+x])
			r := int(f.rand() & 3)
			dx := max(0, min(f.w-1, x-r+1+wind))
			loss := rate / 1000
			if int(f.rand()%1000) < rate%1000 {
				loss++
			}
			f.heat[(y-1)*f.w+dx] = uint8(max(0, min(maxHeat, p-loss)))
		}
	}
	return f
}

// ignite lights a blob of radius r around (cx, cy).
func (f fire) ignite(cx, cy, r int) fire {
	f.heat = append([]uint8(nil), f.heat...)
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - 2*r; x <= cx+2*r; x++ {
			if x >= 0 && y >= 0 && x < f.w && y < f.h {
				f.heat[y*f.w+x] = maxHeat
			}
		}
	}
	return f
}

// isin is a sine with a period of 256 and a range of -255 to 255, from a
// parabola: close enough for a plasma, and exact on every machine.
func isin(a int) int {
	a &= 255
	if a >= 128 {
		return -isin(a - 128)
	}
	return a * (128 - a) * 255 / 4096
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// plasma is the value 0 to 255 at cell (x, y) at time t, with a ripple that
// is centred on (mx, my).
func plasma(x, y, t, mx, my int) int {
	v := isin(x*7+t*3) + isin(y*9-t*2) + isin((x+y)*5+t*4) + isin((abs(x-mx)+2*abs(y-my))*8-t*6)
	return (v + 4*255) * 255 / (8 * 255)
}

// script is what the typewriter types.
var script = []string{
	"tui: a toolkit for terminal programs.",
	"",
	"This screen is three small simulations drawn into one",
	"region. Switching scenes wraps the region in an effect:",
	"a dissolve, a fade, a slide or a sweep, each a function",
	"of one number that runs from 0 to 1.",
	"",
	"With reduced motion on, that number is 1 at once and the",
	"scenes stand still. Press n to step them by hand.",
}

func scriptLen() int {
	n := 0
	for _, l := range script {
		n += len([]rune(l)) + 1
	}
	return n
}

// typed returns the script cut to its first n characters, one string a line.
func typed(n int) []string {
	var out []string
	for _, l := range script {
		r := []rune(l)
		if n <= 0 {
			break
		}
		if n >= len(r)+1 {
			out = append(out, l)
			n -= len(r) + 1
			continue
		}
		out = append(out, string(r[:n]))
		n = 0
	}
	return out
}

// pause is how many extra frames the typewriter waits after the character at
// index n: longer after a full stop or a line end.
func pause(n int) int {
	i := 0
	for _, l := range script {
		r := []rune(l)
		if n < i+len(r) {
			switch r[n-i] {
			case '.', ':':
				return 6
			case ',':
				return 3
			}
			return 0
		}
		if n == i+len(r) {
			return 4
		}
		i += len(r) + 1
	}
	return 0
}
