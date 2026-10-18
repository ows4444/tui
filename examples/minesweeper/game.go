package main

type level struct {
	name       string
	w, h, mine int
}

var levels = [3]level{{"Beginner", 9, 9, 10}, {"Intermediate", 16, 16, 40}, {"Expert", 30, 16, 99}}

type phase int

const (
	fresh phase = iota // no cell opened yet; the mines are not placed
	playing
	won
	lost
)

type cell struct {
	mine, open, flag bool
	adj              int  // mines among the eight neighbours
	boom             bool // the mine that was stepped on
}

// game is the rules. cells is row-major; every method that changes it copies
// it first, so a game is a value.
type game struct {
	lv    level
	cells []cell
	state phase
	seed  uint64
	left  int // closed cells that are not mines
}

func newGame(lv level, seed uint64) game {
	return game{lv: lv, cells: make([]cell, lv.w*lv.h), seed: seed | 1, left: lv.w*lv.h - lv.mine}
}

func (g game) in(x, y int) bool { return x >= 0 && y >= 0 && x < g.lv.w && y < g.lv.h }

func (g game) at(x, y int) cell { return g.cells[y*g.lv.w+x] }

func (g *game) rand() uint64 {
	g.seed ^= g.seed << 13
	g.seed ^= g.seed >> 7
	g.seed ^= g.seed << 17
	return g.seed
}

func (g game) clone() game {
	g.cells = append([]cell(nil), g.cells...)
	return g
}

// neighbours calls f with each of the up to eight cells around (x, y).
func (g game) neighbours(x, y int, f func(nx, ny int)) {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			if (dx != 0 || dy != 0) && g.in(x+dx, y+dy) {
				f(x+dx, y+dy)
			}
		}
	}
}

// lay puts the mines down, keeping (sx, sy) and its neighbours clear so the
// first click opens an area.
func (g game) lay(sx, sy int) game {
	safe := func(x, y int) bool { return abs(x-sx) <= 1 && abs(y-sy) <= 1 }
	if g.lv.w*g.lv.h-g.lv.mine < 9 {
		safe = func(x, y int) bool { return x == sx && y == sy }
	}
	for n := 0; n < g.lv.mine; {
		x, y := int(g.rand()%uint64(g.lv.w)), int(g.rand()%uint64(g.lv.h)) // #nosec G115 -- the sizes are small positive numbers
		if i := y*g.lv.w + x; !g.cells[i].mine && !safe(x, y) {
			g.cells[i].mine = true
			n++
		}
	}
	return g.count()
}

// count fills in every cell's neighbour count.
func (g game) count() game {
	for y := 0; y < g.lv.h; y++ {
		for x := 0; x < g.lv.w; x++ {
			n := 0
			g.neighbours(x, y, func(nx, ny int) {
				if g.at(nx, ny).mine {
					n++
				}
			})
			g.cells[y*g.lv.w+x].adj = n
		}
	}
	return g
}

func (g game) flags() int {
	n := 0
	for _, c := range g.cells {
		if c.flag {
			n++
		}
	}
	return n
}

func (g game) minesLeft() int { return g.lv.mine - g.flags() }

// reveal opens (x, y); a zero opens its neighbours too. Opening a mine loses.
func (g game) reveal(x, y int) game {
	if !g.in(x, y) || g.state == won || g.state == lost || g.at(x, y).flag || g.at(x, y).open {
		return g
	}
	g = g.clone()
	if g.state == fresh {
		g = g.lay(x, y)
		g.state = playing
	}
	if g.at(x, y).mine {
		g.cells[y*g.lv.w+x].boom = true
		return g.lose()
	}
	g = g.flood(x, y)
	if g.left == 0 {
		g.state = won
		for i := range g.cells {
			if g.cells[i].mine {
				g.cells[i].flag = true
			}
		}
	}
	return g
}

func (g game) flood(x, y int) game {
	stack := []int{y*g.lv.w + x}
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		c := &g.cells[i]
		if c.open || c.flag || c.mine {
			continue
		}
		c.open = true
		g.left--
		if c.adj == 0 {
			g.neighbours(i%g.lv.w, i/g.lv.w, func(nx, ny int) { stack = append(stack, ny*g.lv.w+nx) })
		}
	}
	return g
}

func (g game) lose() game {
	g.state = lost
	for i := range g.cells {
		if g.cells[i].mine && !g.cells[i].flag {
			g.cells[i].open = true
		}
	}
	return g
}

// toggleFlag flags or unflags a closed cell.
func (g game) toggleFlag(x, y int) game {
	if !g.in(x, y) || g.state == won || g.state == lost || g.at(x, y).open {
		return g
	}
	g = g.clone()
	g.cells[y*g.lv.w+x].flag = !g.at(x, y).flag
	return g
}

// chord opens the neighbours of an opened number whose flags match it. A wrong
// flag makes that open a mine, which loses.
func (g game) chord(x, y int) game {
	if !g.in(x, y) || g.state != playing || !g.at(x, y).open || g.at(x, y).adj == 0 {
		return g
	}
	flags := 0
	g.neighbours(x, y, func(nx, ny int) {
		if g.at(nx, ny).flag {
			flags++
		}
	})
	if flags != g.at(x, y).adj {
		return g
	}
	g.neighbours(x, y, func(nx, ny int) { g = g.reveal(nx, ny) })
	return g
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
