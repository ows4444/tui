package avatar

import "math"

// Silhouette pins the avatar's shape. SilhouetteAuto, the zero value, lets
// the name choose.
type Silhouette int

// The silhouettes, in the order of how often a name draws them.
const (
	SilhouetteAuto Silhouette = iota
	SilhouetteRound
	SilhouetteOrganic
	SilhouetteBoxy
	SilhouetteCapsule
	SilhouetteNub
	SilhouetteCloud
	SilhouetteDroplet
	SilhouetteHexagon
	SilhouetteSun
	SilhouetteTriangle
)

// String returns the silhouette's name, such as "round"; SilhouetteAuto and
// any value that is not a silhouette are "auto".
func (s Silhouette) String() string {
	if k, ok := s.kind(); ok {
		return shapes[k].name
	}
	return "auto"
}

func (s Silhouette) kind() (shapeKind, bool) {
	if s < SilhouetteRound || s > SilhouetteTriangle {
		return 0, false
	}
	return shapeKind(s - SilhouetteRound), true
}

// Tone pins the avatar's swatch: how light and how saturated its body is.
// ToneAuto, the zero value, lets the name choose.
type Tone int

// The tones, pale to ink.
const (
	ToneAuto Tone = iota
	TonePastel
	TonePale
	ToneMid
	ToneDeep
	ToneBright
	ToneInk
)

var toneNames = [...]string{"auto", "pastel", "pale", "mid", "deep", "bright", "ink"}

// String returns the tone's name, such as "pastel"; ToneAuto and any value
// that is not a tone are "auto".
func (t Tone) String() string {
	if t < TonePastel || t > ToneInk {
		return toneNames[ToneAuto]
	}
	return toneNames[t]
}

// pins returns the trait positions the overrides fix: the middle of the
// chosen silhouette's band and of the chosen tone's. Everything else still
// comes from the name.
func (m Model) pins() map[string]float64 {
	var fixed map[string]float64
	pin := func(key string, lo, hi float64) {
		if fixed == nil {
			fixed = map[string]float64{}
		}
		fixed[key] = (lo + hi) / 2
	}
	if k, ok := m.Silhouette.kind(); ok {
		lo := 0.0
		if k > 0 {
			lo = shapes[k-1].upTo
		}
		pin("shape", lo, shapes[k].upTo)
	}
	if m.Tone >= TonePastel && m.Tone <= ToneInk {
		i := int(m.Tone - TonePastel)
		lo := 0.0
		if i > 0 {
			lo = tones[i-1].upTo
		}
		pin("tone", lo, tones[i].upTo)
	}
	return fixed
}

// hue returns the pinned hue in [0, 360), and whether one is pinned.
func (m Model) hue() (float64, bool) {
	if m.Hue == 0 || math.IsNaN(m.Hue) || math.IsInf(m.Hue, 0) {
		return 0, false
	}
	h := math.Mod(m.Hue, 360)
	if h < 0 {
		h += 360
	}
	return h, true
}
