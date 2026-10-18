package avatar

import (
	"fmt"
	"math"
	"testing"
)

var allTraits = []traitID{
	traitBodySize, traitBodyProportion, traitBodySquareness, traitEyeSize, traitEyeRoundness,
	traitEyeSquareness, traitEyeSeparation, traitEyeLean, traitGazeX, traitGazeY,
}

var pinNames = []string{"ada", "linus", "grace", "ken", "margaret", "dennis", "barbara", "alan", "hedy", "edsger", "user-1", "user-12", "user-21"}

// With nothing pinned, nil or an empty map, the avatar is the name's own.
func TestNoPinsIsTheNamesOwnAvatar(t *testing.T) {
	for _, name := range pinNames {
		own := New(name)
		m := New(name)
		m.traitPins = map[traitID]float64{}
		if m.View() != own.View() || m.SVG() != own.SVG() {
			t.Errorf("%q: an empty traitPins changed the avatar", name)
		}
		// A key that is not a trait pins nothing.
		m.traitPins = map[traitID]float64{-1: 0.5, traitCount: 0.5, 99: 0.5}
		if m.SVG() != own.SVG() {
			t.Errorf("%q: an unknown trait changed the avatar", name)
		}
	}
	if traitID(99).String() != "unknown" || traitID(-1).String() != "unknown" {
		t.Error("an unknown trait is not named unknown")
	}
}

// measure returns what each trait came out as in a figure: the number the
// trait's position is read into.
func measure(m Model) map[traitID]float64 {
	t := m.traits()
	return map[traitID]float64{
		traitBodySize:       t.num("body.r", 31, 38),
		traitBodyProportion: t.num("body.ratio", 0.92, 1.08),
		traitBodySquareness: t.num("body.n", 1.9, 2.5),
		traitEyeSize:        t.num("eye.rx", 0.075, 0.105),
		traitEyeRoundness:   t.num("eye.ratio", 1.9, 3.2),
		traitEyeSquareness:  t.num("eye.n", 3.5, 6),
		traitEyeSeparation:  t.num("eye.gap", 0.1, 0.24),
		traitEyeLean:        t.num("eye.lean", -1, 1),
		traitGazeX:          t.jitter("gaze.x", 0.09),
		traitGazeY:          t.num("gaze.y", -0.2, 0.08),
	}
}

// A pinned trait is the same for every name, moves the figure, and leaves
// every other trait to the name.
func TestAPinnedTraitIsTheSameForEveryName(t *testing.T) {
	for _, tr := range allTraits {
		values := map[float64]bool{}
		changed := 0
		for _, name := range pinNames {
			own := New(name)
			m := New(name)
			m.traitPins = map[traitID]float64{tr: 0.9}
			before, after := measure(own), measure(m)
			values[after[tr]] = true
			for _, other := range allTraits {
				if other != tr && before[other] != after[other] {
					t.Errorf("%q: pinning %v moved %v", name, tr, other)
				}
			}
			if m.SVG() != own.SVG() {
				changed++
			}
			if m.Shape() != own.Shape() {
				t.Errorf("%q: pinning %v changed the silhouette", name, tr)
			}
			ownBody, _, _ := own.Colors()
			if body, _, _ := m.Colors(); body != ownBody {
				t.Errorf("%q: pinning %v changed the colour", name, tr)
			}
		}
		if len(values) != 1 {
			t.Errorf("%v pinned at 0.9 came out as %d different values", tr, len(values))
		}
		// Body squareness is read only by the round silhouettes and eye lean
		// can be bounded to nothing, so not every name must change.
		if changed == 0 {
			t.Errorf("pinning %v changed no avatar", tr)
		}
	}
	if got := fmt.Sprint(allTraits); got != "[body size body proportion body squareness eye size eye roundness eye squareness eye separation eye lean gaze x gaze y]" {
		t.Errorf("trait names: %s", got)
	}
	if got := fmt.Sprint(decorations); got != "[tilt squat corner rounding petals petal distance petal size petal rotation lobes nubs nub angle nub size tip length]" {
		t.Errorf("decoration trait names: %s", got)
	}
}

// A position outside 0 to 1 is clamped to the nearer end, and one that is
// not a number to 0.
func TestPinsAreClamped(t *testing.T) {
	for _, tr := range allTraits {
		at := func(v float64) string {
			m := New("ada")
			m.traitPins = map[traitID]float64{tr: v}
			return m.SVG()
		}
		if at(-3) != at(0) {
			t.Errorf("%v: -3 is not drawn as 0", tr)
		}
		if at(7) != at(0.999999) || at(1) != at(0.999999) {
			t.Errorf("%v: a position of 1 or more is not drawn as the top of the range", tr)
		}
		if at(math.NaN()) != at(0) {
			t.Errorf("%v: NaN is not drawn as 0", tr)
		}
	}
}

// Whatever is pinned, at either end of every trait at once and for every
// silhouette, the eyes stay inside the body and the body inside the frame.
func TestPinnedEyesStayInsideTheBody(t *testing.T) {
	ends := []float64{0, 0.5, 1}
	for s := SilhouetteRound; s <= SilhouetteTriangle; s++ {
		for _, name := range pinNames[:6] {
			for combo := 0; combo < 81; combo++ {
				// Vary the four traits that place and size the eyes; hold
				// the rest at an end chosen by the combination.
				c := combo
				pins := map[traitID]float64{}
				for _, tr := range []traitID{traitEyeSize, traitEyeRoundness, traitEyeSeparation, traitGazeX} {
					pins[tr] = ends[c%3]
					c /= 3
				}
				for i, tr := range []traitID{traitBodySize, traitBodyProportion, traitBodySquareness, traitEyeSquareness, traitEyeLean, traitGazeY} {
					pins[tr] = ends[(combo+i)%3]
				}
				m := New(name)
				m.Silhouette, m.traitPins = s, pins
				f := layoutFigure(m.traits())
				sc := newScene(f, nil)
				for i, eye := range f.eyes {
					for _, p := range eye.flatten() {
						if !sc.inBody(p.x, p.y) {
							t.Fatalf("%v %q pins %v: eye %d leaves the body at (%.1f, %.1f)", s, name, pins, i, p.x, p.y)
						}
					}
				}
				for _, r := range sc.body {
					if r.minX < 0 || r.minY < 0 || r.maxX > 100 || r.maxY > 100 {
						t.Fatalf("%v %q pins %v: the body leaves the frame", s, name, pins)
					}
				}
			}
		}
	}
}

// Changing a pin, or giving a copy a map of its own, redraws a cached Model.
func TestPinsInvalidateTheCache(t *testing.T) {
	m := New("ada")
	m.Width, m.Height = 24, 12
	own := m.View()
	m.traitPins = map[traitID]float64{traitEyeSize: 1}
	big := m.View()
	if big == own {
		t.Fatal("a pin did not change the View")
	}
	m.traitPins[traitEyeSize] = 0
	if small := m.View(); small == big || small == own {
		t.Error("changing a pin in place did not redraw")
	}
	if m.pinKey() != "a0e0;3=0;" || New("ada").pinKey() != "" {
		t.Errorf("pinKey = %q", m.pinKey())
	}
	m.traitPins = nil
	if m.View() != own {
		t.Error("removing the pins did not return the name's own avatar")
	}
}
