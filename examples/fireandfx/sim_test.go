package main

import (
	"strings"
	"testing"
)

func lit(f fire) int {
	n := 0
	for _, h := range f.heat {
		if h > 0 {
			n++
		}
	}
	return n
}

func TestNewFireIsAlreadyBurning(t *testing.T) {
	f := newFire(40, 20)
	if f.heat[19*40+5] != maxHeat || lit(f) < 40*3 {
		t.Fatalf("lit %d", lit(f))
	}
	for _, h := range f.heat[:40] {
		if h > maxHeat {
			t.Fatal("heat past the maximum")
		}
	}
	if g := newFire(40, 20); string(g.heat) != string(f.heat) {
		t.Fatal("the same start gave different fire")
	}
}

func TestTheFlameStopsShortOfTheTop(t *testing.T) {
	for _, h := range []int{7, 21, 37, 77} {
		f := newFire(60, h)
		for range 100 {
			f = f.step(0)
		}
		top := 0
		for _, v := range f.heat[:60] {
			top += int(v)
		}
		if top != 0 || f.heat[(h-1)*60] != maxHeat {
			t.Errorf("height %d: heat along the top row %d", h, top)
		}
		if lit(f) < 60*h/3 {
			t.Errorf("height %d: only %d cells lit", h, lit(f))
		}
	}
}

func TestStepDoesNotChangeTheFireItWasCalledOn(t *testing.T) {
	f := newFire(30, 12)
	before := string(f.heat)
	_ = f.step(1)
	_ = f.ignite(3, 3, 2)
	if string(f.heat) != before {
		t.Fatal("step or ignite wrote into the original")
	}
}

func TestSourceOffLetsTheFireDie(t *testing.T) {
	f := newFire(30, 12)
	f.on = false
	for range 80 {
		f = f.step(0)
	}
	if lit(f) != 0 {
		t.Fatalf("%d cells still lit", lit(f))
	}
}

func TestWindPushesTheFireSideways(t *testing.T) {
	mean := func(wind int) float64 {
		f := newFire(60, 20)
		f.on = false
		f.heat = make([]uint8, len(f.heat))
		f = f.ignite(30, 19, 1)
		for range 8 {
			f = f.step(wind)
		}
		n, c := 0, 0
		for i, h := range f.heat {
			n += int(h) * (i % f.w)
			c += int(h)
		}
		return float64(n) / float64(c)
	}
	if left, right := mean(-1), mean(1); left >= right-2 {
		t.Fatalf("wind did not lean the fire: left %.1f right %.1f", left, right)
	}
}

func TestIgniteLightsABlobAndClipsAtTheEdge(t *testing.T) {
	f := newFire(40, 20)
	f.on = false
	f.heat = make([]uint8, len(f.heat))
	f = f.ignite(0, 0, 2)
	if f.heat[0] != maxHeat || f.heat[2*40+4] != maxHeat || f.heat[3*40] != 0 {
		t.Fatalf("lit %d", lit(f))
	}
	if g := newFire(0, 0).step(0); len(g.heat) != 0 {
		t.Fatal("an empty fire grew")
	}
}

func TestIsinIsASineAndPlasmaStaysInRange(t *testing.T) {
	for a := -300; a < 300; a++ {
		if v := isin(a); v < -255 || v > 255 || isin(a+256) != v || isin(a+128) != -v {
			t.Fatalf("isin(%d) = %d", a, v)
		}
	}
	if isin(64) < 250 || isin(0) != 0 {
		t.Fatalf("isin(64) = %d", isin(64))
	}
	for x := 0; x < 50; x++ {
		for y := 0; y < 30; y++ {
			if v := plasma(x, y, 17, 5, 5); v < 0 || v > 255 {
				t.Fatalf("plasma(%d,%d) = %d", x, y, v)
			}
		}
	}
}

func TestTypedCutsTheScript(t *testing.T) {
	if len(typed(0)) != 0 {
		t.Fatal("text before typing")
	}
	got := typed(5)
	if len(got) != 1 || got[0] != "tui: " {
		t.Fatalf("%q", got)
	}
	all := typed(scriptLen())
	if strings.Join(all, "\n") != strings.Join(script, "\n") {
		t.Fatalf("%q", all)
	}
	if len(typed(scriptLen()+50)) != len(script) {
		t.Fatal("typed past the end")
	}
	if got := typed(len(script[0]) + 1); len(got) != 1 || got[0] != script[0] {
		t.Fatalf("%q", got)
	}
}

func TestPauseIsLongerAfterFullStops(t *testing.T) {
	stop := strings.Index(script[0], ".")
	if pause(0) != 0 || pause(stop) <= pause(0) || pause(len(script[0])) == 0 {
		t.Fatalf("pause(0)=%d pause(stop)=%d pause(end)=%d", pause(0), pause(stop), pause(len(script[0])))
	}
	if pause(scriptLen()+3) != 0 {
		t.Fatal("pause past the end")
	}
}
