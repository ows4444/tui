package main

import "testing"

func TestNewGameHasAThreeCellSnakeAndFoodOffIt(t *testing.T) {
	g := newGame(20, 10, 7)
	if len(g.body) != 3 || g.body[0] != (pt{10, 5}) || g.dir != right {
		t.Fatalf("body %v dir %v", g.body, g.dir)
	}
	if g.on(g.food) || g.food.x < 0 || g.food.x >= 20 || g.food.y < 0 || g.food.y >= 10 {
		t.Fatalf("food %v", g.food)
	}
	if newGame(20, 10, 7).food != g.food {
		t.Fatal("the same seed gave different food")
	}
}

func TestStepMovesTheHeadAndKeepsTheLength(t *testing.T) {
	g := newGame(20, 10, 7).turn(right).step(1)
	if g.body[0] != (pt{11, 5}) || len(g.body) != 3 || g.body[2] != (pt{9, 5}) {
		t.Fatalf("%v", g.body)
	}
}

func TestEatingGrowsScoresTheLevelAndMovesTheFood(t *testing.T) {
	g := newGame(20, 10, 7)
	g.food = pt{11, 5}
	g = g.turn(right).step(3)
	if len(g.body) != 4 || g.score != 3 || g.food == (pt{11, 5}) || g.on(g.food) {
		t.Fatalf("len %d score %d food %v", len(g.body), g.score, g.food)
	}
}

func TestReversingIsIgnored(t *testing.T) {
	g := newGame(20, 10, 7).turn(left)
	if g.next != right {
		t.Fatalf("next = %v", g.next)
	}
	g = g.turn(down).step(1)
	if g.dir != down {
		t.Fatalf("dir = %v", g.dir)
	}
}

func TestWallEndsTheGame(t *testing.T) {
	g := newGame(6, 5, 7).turn(right)
	for i := 0; i < 5 && g.state == running; i++ {
		g = g.step(1)
	}
	if g.state != over {
		t.Fatalf("state %v body %v", g.state, g.body)
	}
	if again := g.step(1); again.body[0] != g.body[0] {
		t.Fatal("a finished game moved")
	}
}

func TestHittingTheBodyEndsTheGame(t *testing.T) {
	g := newGame(20, 10, 7)
	g.body = []pt{{5, 5}, {5, 6}, {4, 6}, {4, 5}, {4, 4}, {5, 4}}
	g.dir, g.next, g.food, g.state = up, up, pt{0, 0}, running
	g = g.turn(left).step(1) // into (4, 5), which is body
	if g.state != over {
		t.Fatalf("state %v", g.state)
	}
}

func TestTheTailCellIsFreeToEnter(t *testing.T) {
	g := newGame(20, 10, 7)
	g.body = []pt{{5, 5}, {5, 6}, {4, 6}, {4, 5}}
	g.dir, g.next, g.food, g.state = up, up, pt{0, 0}, running
	g = g.turn(left).step(1) // (4, 5) is the tail, which moves away
	if g.state != running || g.body[0] != (pt{4, 5}) {
		t.Fatalf("state %v body %v", g.state, g.body)
	}
}

func TestSteerTurnsAlongTheFurtherAxis(t *testing.T) {
	g := newGame(20, 10, 7)
	if s := g.steer(pt{10, 0}); s.next != up {
		t.Fatalf("above: %v", s.next)
	}
	if s := g.steer(pt{10, 9}); s.next != down {
		t.Fatalf("below: %v", s.next)
	}
	if s := g.steer(pt{19, 6}); s.next != right {
		t.Fatalf("right: %v", s.next)
	}
	if s := g.steer(pt{0, 5}); s.next != right { // left would reverse into the body
		t.Fatalf("left reversed: %v", s.next)
	}
	if s := g.steer(g.body[0]); s.state != ready {
		t.Fatal("steering at the head started the game")
	}
}

func TestFillingTheBoardWins(t *testing.T) {
	g := newGame(4, 1, 7)
	g.body = []pt{{2, 0}, {1, 0}, {0, 0}}
	g.food = pt{3, 0}
	g = g.turn(right).step(1)
	if g.state != won || len(g.body) != 4 {
		t.Fatalf("state %v len %d", g.state, len(g.body))
	}
}
