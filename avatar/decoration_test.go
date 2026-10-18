package avatar

import "testing"

var decorations = []traitID{
	traitTilt, traitSquat, traitCornerRounding, traitPetals, traitPetalDistance, traitPetalSize,
	traitPetalRotation, traitLobes, traitNubs, traitNubAngle, traitNubSize, traitTipLength,
}

// readers lists, per decoration trait, the silhouettes that read it.
var readers = map[traitID][]Silhouette{
	traitTilt:           {SilhouetteBoxy, SilhouetteHexagon, SilhouetteTriangle},
	traitSquat:          {SilhouetteCapsule},
	traitCornerRounding: {SilhouetteHexagon, SilhouetteTriangle},
	traitPetals:         {SilhouetteSun},
	traitPetalDistance:  {SilhouetteSun},
	traitPetalSize:      {SilhouetteSun},
	traitPetalRotation:  {SilhouetteSun},
	traitLobes:          {SilhouetteCloud},
	traitNubs:           {SilhouetteNub},
	traitNubAngle:       {SilhouetteNub},
	traitNubSize:        {SilhouetteNub},
	traitTipLength:      {SilhouetteDroplet},
}

// A decoration trait pinned on a silhouette that reads it comes out the same
// for every name and redraws the figure; on any other silhouette it changes
// nothing.
func TestDecorationPinsBelongToTheirSilhouettes(t *testing.T) {
	if len(readers) != len(decorations) {
		t.Fatalf("%d decoration traits, %d listed with their readers", len(decorations), len(readers))
	}
	for _, tr := range decorations {
		reads := map[Silhouette]bool{}
		for _, s := range readers[tr] {
			reads[s] = true
		}
		for s := SilhouetteRound; s <= SilhouetteTriangle; s++ {
			changed := 0
			positions := map[float64]bool{}
			for _, name := range pinNames {
				own := New(name)
				own.Silhouette = s
				m := own
				m.traitPins = map[traitID]float64{tr: 0.07}
				if m.SVG() != own.SVG() {
					changed++
				}
				for _, key := range traitKeys[tr].keys {
					positions[m.traits().at(key)] = true
				}
				// A body or eye trait is left to the name.
				for other, v := range measure(own) {
					if measure(m)[other] != v {
						t.Fatalf("%v on %v %q moved %v", tr, s, name, other)
					}
				}
			}
			if len(positions) != 1 {
				t.Errorf("%v pinned on %v came out at %d positions", tr, s, len(positions))
			}
			switch {
			case reads[s] && changed == 0:
				t.Errorf("%v pinned on %v, which reads it, changed no avatar", tr, s)
			case !reads[s] && changed != 0:
				t.Errorf("%v pinned on %v, which does not read it, changed %d avatars", tr, s, changed)
			}
		}
	}
}

// With every decoration trait at either end, alone and all together, on
// every silhouette, the eyes stay inside the body and the body inside the
// frame.
func TestPinnedDecorationKeepsTheFigureWhole(t *testing.T) {
	check := func(s Silhouette, name string, pins map[traitID]float64) {
		t.Helper()
		m := New(name)
		m.Silhouette, m.traitPins = s, pins
		f := layoutFigure(m.traits())
		sc := newScene(f, nil)
		for i, eye := range f.eyes {
			for _, p := range eye.flatten() {
				if !sc.inBody(p.x, p.y) {
					t.Fatalf("%v %q pins %v: eye %d leaves the body", s, name, pins, i)
				}
			}
		}
		if minX, minY, maxX, maxY := sc.extent(); minX < 0 || minY < 0 || maxX > 100 || maxY > 100 {
			t.Fatalf("%v %q pins %v: the body spans (%.1f, %.1f) to (%.1f, %.1f)", s, name, pins, minX, minY, maxX, maxY)
		}
	}
	for s := SilhouetteRound; s <= SilhouetteTriangle; s++ {
		for _, name := range pinNames[:6] {
			for _, end := range []float64{0, 1} {
				all := map[traitID]float64{traitBodySize: 1, traitBodyProportion: end}
				for _, tr := range decorations {
					check(s, name, map[traitID]float64{tr: end})
					check(s, name, map[traitID]float64{tr: end, traitBodySize: 1, traitEyeSize: 1, traitEyeSeparation: 1})
					all[tr] = end
				}
				check(s, name, all)
			}
		}
	}
}

// The number of petals, lobes and nubs follows the pin from the fewest to
// the most.
func TestPinnedCounts(t *testing.T) {
	count := func(s Silhouette, tr traitID, at float64) int {
		m := New("ada")
		m.Silhouette, m.traitPins = s, map[traitID]float64{tr: at}
		return len(layoutFigure(m.traits()).petals)
	}
	for _, c := range []struct {
		s         Silhouette
		tr        traitID
		low, high int
	}{
		{SilhouetteSun, traitPetals, 6, 9},
		{SilhouetteCloud, traitLobes, 4, 6},
		{SilhouetteNub, traitNubs, 1, 2},
	} {
		if lo, hi := count(c.s, c.tr, 0), count(c.s, c.tr, 1); lo != c.low || hi != c.high {
			t.Errorf("%v on %v: %d to %d, want %d to %d", c.tr, c.s, lo, hi, c.low, c.high)
		}
	}
}
