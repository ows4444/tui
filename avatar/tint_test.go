package avatar

import (
	"encoding/json"
	"math"
	"os"
	"testing"

	"github.com/ows4444/tui/ansi"
)

// The reference vectors record the palette of each tinting pose for a few
// names; the tint path must reproduce them exactly.
func TestTintsMatchVectors(t *testing.T) {
	raw, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		Tints []struct {
			Expression    string
			Heat          float64
			Seed          string
			ShapeOverride *float64 `json:"shape_override"`
			Palette       struct{ Head, Eye string }
		}
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	targets := map[string]*tint{"mad": tintHot, "love": tintRose, "shy": tintBlush, "sick": tintBile}
	seen := map[string]int{}
	for _, c := range v.Tints {
		tr := newTraits(c.Seed, false)
		rest := paletteFor(tr.num("hue", 0, 360), tr.at("tone"))
		got := rest.tinted(targets[c.Expression], c.Heat)
		if hex(got.head) != c.Palette.Head || hex(got.eye) != c.Palette.Eye {
			t.Errorf("%s on %q: body %s eyes %s, want %s %s", c.Expression, c.Seed, hex(got.head), hex(got.eye), c.Palette.Head, c.Palette.Eye)
		}
		if got.bg != rest.bg {
			t.Errorf("%s on %q changed the plate colour", c.Expression, c.Seed)
		}
		seen[c.Expression]++
	}
	for name := range targets {
		if seen[name] < 3 {
			t.Errorf("only %d vectors cover the %s tint", seen[name], name)
		}
	}
}

func wcag(a, b ansi.RGB) float64 { return contrast(oklchOf(a), oklchOf(b)) }

// The mad expression flushes the body toward red; the untinted ones leave
// the colours alone; SVG stays in the avatar's own colours.
func TestMadTintsTheBodyAndOthersDoNot(t *testing.T) {
	for _, name := range []string{"alain00", "tove", "kasper", "mdawais", "user-1", "ada", "linus"} {
		m := New(name)
		body, eyes, plate := m.Colors()
		svg := m.SVG()
		for e := ExpressionNone; int(e) < len(poses); e++ {
			m.Expression = e
			b, ey, pl := m.Colors()
			if e == ExpressionMad {
				continue
			}
			if b != body || ey != eyes || pl != plate {
				t.Errorf("%q: %v changed the colours", name, e)
			}
		}
		m.Expression = ExpressionMad
		b, _, pl := m.Colors()
		if b == body || pl != plate {
			t.Errorf("%q: mad did not tint the body, or tinted the plate", name)
		}
		// Redder: closer in hue to the tint's than the resting body is.
		away := func(c ansi.RGB) float64 {
			d := math.Abs(oklchOf(c).h - tintHot.h)
			return math.Min(d, 360-d)
		}
		if oklchOf(body).c > 0.03 && away(b) >= away(body) {
			t.Errorf("%q: mad moved the body's hue from %.0f to %.0f, away from red", name, oklchOf(body).h, oklchOf(b).h)
		}
		red := func(c ansi.RGB) float64 { return float64(c.R) / float64(int(c.R)+int(c.G)+int(c.B)) }
		if red(b) <= red(body) {
			t.Errorf("%q: the mad body %v has no more red in it than %v", name, b, body)
		}
		if m.SVG() != svg {
			t.Errorf("%q: an expression changed the SVG", name)
		}
		// View and PNG are drawn in the tinted colour.
		if code := "38;2;" + sgr(b); !containsSGR(m.View(), code) || containsSGR(m.View(), "38;2;"+sgr(body)) {
			t.Errorf("%q: the View is not drawn in the tinted body colour", name)
		}
	}
}

// Every tint, on every hue and tone, keeps the eyes at 4.5:1 against the
// body, at the heat each pose uses.
func TestTintedEyesStayLegible(t *testing.T) {
	worst := math.Inf(1)
	for name, tn := range map[string]*tint{"hot": tintHot, "rose": tintRose, "blush": tintBlush, "bile": tintBile} {
		for hue := 0.0; hue < 360; hue += 5 {
			for _, tone := range []float64{0.1, 0.3, 0.5, 0.7, 0.9, 0.97} {
				for _, heat := range []float64{0.55, 0.6, 0.62, 1} {
					p := paletteFor(hue, tone).tinted(tn, heat)
					if c := wcag(p.head, p.eye); c < worst {
						worst = c
						if c < 4.5 {
							t.Errorf("%s at hue %v tone %v heat %v: contrast %.2f", name, hue, tone, heat, c)
						}
					}
				}
			}
		}
	}
	t.Logf("the lowest tinted eye contrast is %.2f", worst)
}

// A reaction that lands on a tinting pose tints the avatar while it is held.
func TestAReactionCanTint(t *testing.T) {
	m := New("ada")
	body, _, _ := m.Colors()
	for i := 0; i < 40 && m.shown() != ExpressionMad; i++ {
		m.React()
	}
	if m.shown() != ExpressionMad {
		t.Fatal("forty reactions never reached the mad pose")
	}
	if b, _, _ := m.Colors(); b == body {
		t.Error("a mad reaction did not tint the body")
	}
	n, _ := m.Update(releaseMsg{owner: m.reactOwner})
	if b, _, _ := n.Colors(); b != body {
		t.Error("the tint outlived the reaction")
	}
}

// oklchOf undoes rgb for colours inside sRGB, and mixRGB ends where it says.
func TestColourRoundTrip(t *testing.T) {
	for _, c := range []ansi.RGB{{R: 255}, {G: 128, B: 64}, {R: 12, G: 200, B: 230}, {R: 250, G: 250, B: 250}, {R: 20, G: 20, B: 20}} {
		if got := oklchOf(c).rgb(); got != c {
			t.Errorf("%v round-trips to %v", c, got)
		}
		other := ansi.RGB{R: 90, G: 40, B: 200}
		if mixRGB(c, other, 0) != c || mixRGB(c, other, 1) != other {
			t.Errorf("mixRGB from %v does not end at its ends", c)
		}
	}
}
