package avatar

import (
	"fmt"
	"strings"
	"testing"
)

// The bands of the new traits give the share of names they say, within a
// sampling error.
func TestAccessoriesAndEyeStylesAreSpreadAsTheBandsSay(t *testing.T) {
	const n = 20000
	acc, eyes, eligible := map[string]int{}, map[string]int{}, 0
	for i := 0; i < n; i++ {
		tr := newTraits(fmt.Sprintf("spread-%d", i), false)
		kind := pickShape(tr.at("shape"))
		acc[pickAccessory(tr, kind).String()]++
		eyes[eyeKindOf(tr.at("eye.kind")).name]++
		switch kind {
		case shapeRound, shapeOrganic, shapeBoxy, shapeCapsule, shapeHexagon:
			eligible++
		}
	}
	near := func(what string, got int, of int, want float64) {
		if share := float64(got) / float64(of); share < want-0.03 || share > want+0.03 {
			t.Errorf("%s: %.3f of %d, want about %.2f", what, share, of, want)
		}
	}
	for _, k := range eyeKinds[:] {
		lo := 0.0
		for i, o := range eyeKinds {
			if o.name == k.name && i > 0 {
				lo = eyeKinds[i-1].upTo
			}
		}
		near("eyes "+k.name, eyes[k.name], n, k.upTo-lo)
	}
	worn := 0
	for name, c := range acc {
		if name != "none" {
			worn += c
		}
	}
	near("names that wear something", worn, n, 0.48*float64(eligible)/n)
	for _, a := range []string{"ears", "bunny", "antenna", "tuft", "horns"} {
		if acc[a] == 0 {
			t.Errorf("no name wore %s in %d", a, n)
		}
	}
}

// bounds is the box that holds the body of a figure: its core, its outlines and
// its circles.
func bounds(f figure) (minX, minY, maxX, maxY float64) {
	s := newScene(f, nil)
	minX, minY, maxX, maxY = s.extent()
	return
}

// Whatever it wears, the whole body fits the 100 by 100 frame, so a plate never
// cuts it.
func TestAccessoriesStayInsideTheFrame(t *testing.T) {
	for acc := AccessoryNone; acc <= AccessoryHorns; acc++ {
		for _, sil := range []Silhouette{SilhouetteRound, SilhouetteOrganic, SilhouetteBoxy, SilhouetteCapsule, SilhouetteHexagon} {
			for i := 0; i < 60; i++ {
				m := New(fmt.Sprintf("fit-%d", i))
				m.Accessory, m.Silhouette = acc, sil
				f := layoutFigure(m.traits())
				x0, y0, x1, y1 := bounds(f)
				if x0 < 1 || y0 < 1 || x1 > 99 || y1 > 99 {
					t.Fatalf("%v on %v for %q: body at %.1f,%.1f to %.1f,%.1f", acc, sil, m.Name, x0, y0, x1, y1)
				}
			}
		}
	}
}

func TestAccessoryPinsDrawTheAccessory(t *testing.T) {
	bare := New("pin")
	bare.Silhouette, bare.Accessory = SilhouetteRound, AccessoryNone
	for acc, parts := range map[Accessory]int{AccessoryNone: 0, AccessoryEars: 2, AccessoryBunny: 2, AccessoryAntenna: 2, AccessoryTuft: 3, AccessoryHorns: 2} {
		m := bare
		m.Accessory = acc
		f := layoutFigure(m.traits())
		if got := len(f.petals) + len(f.extra); got != parts {
			t.Errorf("%v: %d outlines and circles, want %d", acc, got, parts)
		}
		if acc != AccessoryNone && m.View() == bare.View() {
			t.Errorf("%v does not change the View", acc)
		}
	}
}

// A body that already has something on top wears nothing more.
func TestOnlyPlainBodiesWearAnAccessory(t *testing.T) {
	for _, sil := range []Silhouette{SilhouetteNub, SilhouetteCloud, SilhouetteDroplet, SilhouetteSun, SilhouetteTriangle} {
		plain, worn := New("rule"), New("rule")
		plain.Silhouette, plain.Accessory = sil, AccessoryNone
		worn.Silhouette, worn.Accessory = sil, AccessoryBunny
		if plain.SVG() != worn.SVG() {
			t.Errorf("%v wears an accessory", sil)
		}
	}
}

func TestEyeStylePinsDrawTheirShape(t *testing.T) {
	for style, want := range map[EyeStyle][2]float64{EyesCapsule: {1.9, 3.2}, EyesDot: {1, 1.2}, EyesOval: {1.35, 1.75}, EyesSlit: {3.4, 4.4}} {
		for i := 0; i < 30; i++ {
			m := New(fmt.Sprintf("eyes-%d", i))
			m.Eyes = style
			e := layoutFigure(m.traits()).eye[0]
			if r := e.ry / e.rx; r < want[0]-0.01 || r > want[1]+0.01 {
				t.Fatalf("%v for %q: height over width %.2f, want %v", style, m.Name, r, want)
			}
		}
	}
}

func TestAccessoryAndEyeStyleNames(t *testing.T) {
	for in, want := range map[fmt.Stringer]string{
		AccessoryAuto: "auto", AccessoryNone: "none", AccessoryEars: "ears", AccessoryBunny: "bunny", AccessoryAntenna: "antenna", AccessoryTuft: "tuft", AccessoryHorns: "horns", Accessory(99): "auto", Accessory(-1): "auto",
		EyesAuto: "auto", EyesCapsule: "capsule", EyesDot: "dot", EyesOval: "oval", EyesSlit: "slit", EyeStyle(99): "auto",
	} {
		if got := in.String(); got != want {
			t.Errorf("%#v = %q, want %q", in, got, want)
		}
	}
	if accessory(99).String() != "none" || accessory(-1).String() != "none" {
		t.Error("an unknown accessory is not none")
	}
}

func TestPinsChangeTheCacheKey(t *testing.T) {
	m := New("cache")
	first := m.View()
	m.Accessory = AccessoryBunny
	m.Silhouette = SilhouetteRound
	second := m.View()
	m.Eyes = EyesSlit
	if third := m.View(); third == second || second == first {
		t.Error("changing an accessory or an eye style did not redraw")
	}
}

// expectedMarks is the eyebrows and mouth of each expression.
var expectedMarks = map[Expression]struct {
	brows int
	mouth mouthKind
}{
	ExpressionNone: {0, mouthNone}, ExpressionHappy: {0, mouthSmile}, ExpressionSad: {2, mouthFrown}, ExpressionMad: {2, mouthFrown}, ExpressionSurprised: {2, mouthO},
	ExpressionWink: {0, mouthSmile}, ExpressionSleepy: {0, mouthFlat}, ExpressionThinking: {1, mouthFlat}, ExpressionSmug: {1, mouthSmile}, ExpressionUnsure: {2, mouthFlat},
	ExpressionScared: {2, mouthO}, ExpressionLove: {0, mouthSmile}, ExpressionShy: {2, mouthSmile}, ExpressionSick: {2, mouthFrown},
}

func TestEveryExpressionHasItsEyebrowsAndMouth(t *testing.T) {
	if len(expectedMarks) != len(poses) {
		t.Fatalf("%d expectations for %d poses", len(expectedMarks), len(poses))
	}
	f := layoutFigure(New("marks").traits())
	for e, want := range expectedMarks {
		p := e.pose()
		brows := 0
		for _, b := range p.brow {
			if b.on {
				brows++
			}
		}
		if brows != want.brows || p.mouth.kind != want.mouth {
			t.Errorf("%v: %d brows and mouth %v, want %d and %v", e, brows, p.mouth.kind, want.brows, want.mouth)
		}
		n := len(f.posed(p, p, 1, 0, 0, 0, 1, 1).marks)
		if wantN := want.brows + map[bool]int{true: 1}[want.mouth != mouthNone]; n != wantN {
			t.Errorf("%v: %d marks, want %d", e, n, wantN)
		}
	}
}

// A rest figure has no marks, so a name's avatar stays what the name made.
func TestARestFigureHasNoMarks(t *testing.T) {
	f := layoutFigure(New("rest").traits())
	if n := len(f.posed(poses[ExpressionNone], poses[ExpressionNone], 1, 0, 0, 0, 1, 1).marks); n != 0 {
		t.Fatalf("%d marks", n)
	}
}

// Marks are drawn only where a pixel is thin enough to hold them.
func TestMarksNeedRoom(t *testing.T) {
	marks := []mark{{thick: 2}, {thick: 5}}
	var s scene
	for pw, want := range map[float64]int{1: 2, 2.2: 2, 2.3: 1, 5.5: 1, 5.6: 0, 20: 0} {
		s.setMarks(marks, pw)
		if len(s.marks) != want {
			t.Errorf("pixel %.1f wide: %d marks drawn, want %d", pw, len(s.marks), want)
		}
	}
}

func TestSVGDrawsThePoseItsMarksAndItsLift(t *testing.T) {
	rest := New("svg")
	rest.Silhouette, rest.Accessory = SilhouetteRound, AccessoryNone
	sad := rest
	sad.Expression = ExpressionSad
	if !strings.Contains(sad.SVG(), `transform="translate(0 2.6)"`) {
		t.Errorf("a sad figure sinks: %s", sad.SVG())
	}
	if strings.Contains(rest.SVG(), "transform") {
		t.Error("a rest figure is lifted")
	}
	if got, want := strings.Count(sad.SVG(), "<path"), strings.Count(rest.SVG(), "<path")+3; got != want {
		t.Errorf("%d paths, want %d (two brows and a mouth more than rest)", got, want)
	}
	// the middle of an ease and a blink are not drawn
	sad.Blink()
	if sad.SVG() == "" {
		t.Fatal("empty SVG")
	}
}
