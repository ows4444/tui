package main

type pt struct{ x, y int }

type dir int

const (
	up dir = iota
	right
	down
	left
)

var delta = [4]pt{up: {0, -1}, right: {1, 0}, down: {0, 1}, left: {-1, 0}}

func (d dir) opposite() dir { return (d + 2) % 4 }

type phase int

const (
	ready phase = iota // waiting for the first move
	running
	paused
	over
	won
)

// game is the rules, with no screen in it. body[0] is the head.
type game struct {
	w, h  int
	body  []pt
	dir   dir
	next  dir // the turn to take on the next step
	food  pt
	score int
	state phase
	seed  uint64
}

func newGame(w, h int, seed uint64) game {
	g := game{w: w, h: h, dir: right, next: right, seed: seed | 1}
	mid := pt{w / 2, h / 2}
	g.body = []pt{mid, {mid.x - 1, mid.y}, {mid.x - 2, mid.y}}
	g.food = g.place()
	return g
}

// rand is a xorshift step: the same seed always plays the same game.
func (g *game) rand() uint64 {
	g.seed ^= g.seed << 13
	g.seed ^= g.seed >> 7
	g.seed ^= g.seed << 17
	return g.seed
}

func (g *game) on(p pt) bool {
	for _, b := range g.body {
		if b == p {
			return true
		}
	}
	return false
}

// place picks a free cell for the food, or the first free one when the random
// pick is taken; (-1, -1) when the board is full.
func (g *game) place() pt {
	free := g.w*g.h - len(g.body)
	if free <= 0 {
		return pt{-1, -1}
	}
	n := int(g.rand() % uint64(free))
	for y := 0; y < g.h; y++ {
		for x := 0; x < g.w; x++ {
			if p := (pt{x, y}); !g.on(p) {
				if n == 0 {
					return p
				}
				n--
			}
		}
	}
	return pt{-1, -1}
}

// turn asks for a new heading; a reversal into the body is ignored.
func (g game) turn(d dir) game {
	if d != g.dir.opposite() {
		g.next = d
	}
	if g.state == ready {
		g.state = running
	}
	return g
}

// steer turns toward the cell p: along the axis it is furthest from the head.
func (g game) steer(p pt) game {
	h := g.body[0]
	dx, dy := p.x-h.x, p.y-h.y
	switch {
	case dx == 0 && dy == 0:
		return g
	case abs(dx) >= abs(dy) && dx > 0:
		return g.turn(right)
	case abs(dx) >= abs(dy):
		return g.turn(left)
	case dy > 0:
		return g.turn(down)
	}
	return g.turn(up)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// step moves the snake one cell; points per apple is the level.
func (g game) step(level int) game {
	if g.state != running {
		return g
	}
	g.dir = g.next
	h := g.body[0]
	n := pt{h.x + delta[g.dir].x, h.y + delta[g.dir].y}
	eats := n == g.food
	tail := g.body
	if !eats {
		tail = g.body[:len(g.body)-1] // the tail moves out of the way
	}
	hit := n.x < 0 || n.y < 0 || n.x >= g.w || n.y >= g.h
	for _, b := range tail {
		hit = hit || b == n
	}
	if hit {
		g.state = over
		return g
	}
	body := append([]pt{n}, tail...)
	g.body = body
	if eats {
		g.score += level
		if g.food = g.place(); g.food.x < 0 {
			g.state = won
		}
	}
	return g
}
