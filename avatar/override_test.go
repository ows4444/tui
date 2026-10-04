package avatar_test

import (
	"math"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/avatar"
)

var (
	silhouettes = []avatar.Silhouette{
		avatar.SilhouetteRound, avatar.SilhouetteOrganic, avatar.SilhouetteBoxy, avatar.SilhouetteCapsule,
		avatar.SilhouetteNub, avatar.SilhouetteCloud, avatar.SilhouetteDroplet, avatar.SilhouetteHexagon,
		avatar.SilhouetteSun, avatar.SilhouetteTriangle,
	}
	allTones = []avatar.Tone{
		avatar.TonePastel, avatar.TonePale, avatar.ToneMid, avatar.ToneDeep, avatar.ToneBright, avatar.ToneInk,
	}
)

// contrast is the WCAG contrast ratio of two sRGB colours.
func contrast(a, b ansi.RGB) float64 {
	lum := func(c ansi.RGB) float64 {
		lin := func(v uint8) float64 {
			s := float64(v) / 255
			if s <= 0.04045 {
				return s / 12.92
			}
			return math.Pow((s+0.055)/1.055, 2.4)
		}
		return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
	}
	x, y := lum(a), lum(b)
	return (math.Max(x, y) + 0.05) / (math.Min(x, y) + 0.05)
}

// With no override set the avatar is what the name alone draws; a value that
// is not a tone or a silhouette, and a hue that is not a number, set nothing.
func TestNoOverrideIsTheNamesOwnAvatar(t *testing.T) {
	for _, name := range names {
		own := avatar.New(name)
		m := avatar.New(name)
		m.Hue, m.Tone, m.Silhouette = 0, avatar.ToneAuto, avatar.SilhouetteAuto
		if m.View() != own.View() || m.SVG() != own.SVG() {
			t.Errorf("%q: the zero overrides changed the avatar", name)
		}
		m.Hue, m.Tone, m.Silhouette = math.NaN(), 99, -4
		if m.SVG() != own.SVG() || m.Shape() != own.Shape() {
			t.Errorf("%q: invalid overrides changed the avatar", name)
		}
		m.Hue = math.Inf(1)
		if m.SVG() != own.SVG() {
			t.Errorf("%q: an infinite hue changed the avatar", name)
		}
	}
	if avatar.Tone(99).String() != "auto" || avatar.Silhouette(-4).String() != "auto" {
		t.Error("invalid overrides are not named auto")
	}
}

// A pinned hue colours every name alike and leaves each its own silhouette
// and tone: with the tone pinned too, every body is the same colour.
func TestPinnedHueKeepsEachNamesFigure(t *testing.T) {
	bodies := map[ansi.RGB]bool{}
	for _, name := range names {
		own := avatar.New(name)
		m := avatar.New(name)
		m.Hue = 210
		if m.Shape() != own.Shape() {
			t.Errorf("%q: a pinned hue changed the silhouette from %s to %s", name, own.Shape(), m.Shape())
		}
		m.Tone = avatar.ToneMid
		body, _, _ := m.Colors()
		bodies[body] = true
		if code := "38;2;" + rgb(body); !containsCode(m.View(), code) {
			t.Errorf("%q: the View is not drawn in the pinned colour", name)
		}
	}
	if len(bodies) != 1 {
		t.Errorf("one hue and one tone drew %d body colours", len(bodies))
	}
	// Another hue is another colour, in the cool half of the wheel for 250
	// and the warm half for 30.
	cool, warm := avatar.New("ada"), avatar.New("ada")
	cool.Hue, cool.Tone, warm.Hue, warm.Tone = 250, avatar.ToneMid, 30, avatar.ToneMid
	cb, _, _ := cool.Colors()
	wb, _, _ := warm.Colors()
	if cb.B <= cb.R || wb.R <= wb.B {
		t.Errorf("hue 250 drew %v and hue 30 drew %v", cb, wb)
	}
	// 360 is the hue that 0 cannot name; hues wrap in both directions.
	full, wrapped, negative, same := avatar.New("ada"), avatar.New("ada"), avatar.New("ada"), avatar.New("ada")
	full.Hue, wrapped.Hue, negative.Hue, same.Hue = 360, 720, -150, 210
	if full.SVG() != wrapped.SVG() || negative.SVG() != same.SVG() || full.SVG() == avatar.New("ada").SVG() {
		t.Error("hues do not wrap at 360")
	}
}

// A pinned silhouette is the shape every name draws, and a pinned tone the
// swatch; each name still differs in size, placement and eyes.
func TestPinnedSilhouetteAndTone(t *testing.T) {
	for _, s := range silhouettes {
		figures := map[string]bool{}
		for _, name := range names {
			m := avatar.New(name)
			m.Silhouette = s
			if m.Shape() != s.String() {
				t.Errorf("%q pinned to %v draws %s", name, s, m.Shape())
			}
			figures[m.SVG()] = true
		}
		if len(figures) != len(names) {
			t.Errorf("%v: %d names drew %d figures", s, len(names), len(figures))
		}
	}
	want := []string{"pastel", "pale", "mid", "deep", "bright", "ink"}
	swatches := map[ansi.RGB]bool{}
	for i, tone := range allTones {
		m := avatar.New("ada")
		m.Hue, m.Tone = 30, tone
		body, _, _ := m.Colors()
		swatches[body] = true
		if tone.String() != want[i] {
			t.Errorf("tone %d is %q, want %q", i, tone, want[i])
		}
	}
	if len(swatches) != len(allTones) {
		t.Errorf("six tones drew %d body colours", len(swatches))
	}
}

// Whatever is pinned, the eyes keep a contrast of 4.5:1 against the body.
func TestOverridesKeepTheEyesLegible(t *testing.T) {
	worst := math.Inf(1)
	for hue := 1.0; hue <= 360; hue += 3 {
		for _, tone := range append([]avatar.Tone{avatar.ToneAuto}, allTones...) {
			m := avatar.New("ada")
			m.Hue, m.Tone = hue, tone
			body, eyes, _ := m.Colors()
			if c := contrast(body, eyes); c < worst {
				worst = c
			}
		}
	}
	if worst < 4.5 {
		t.Errorf("the lowest eye contrast over every hue and tone is %.2f, want 4.5 or more", worst)
	}
}

// Changing an override redraws a cached Model.
func TestOverridesInvalidateTheCache(t *testing.T) {
	m := avatar.New("ada")
	m.Width, m.Height = 12, 6
	seen := map[string]bool{m.View(): true}
	for _, change := range []func(){
		func() { m.Hue = 120 },
		func() { m.Tone = avatar.ToneInk },
		func() { m.Silhouette = avatar.SilhouetteSun },
	} {
		change()
		if v := m.View(); seen[v] {
			t.Error("an override did not change the cached View")
		} else {
			seen[v] = true
		}
	}
}
