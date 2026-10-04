package avatar

import (
	"math"
	"strconv"
	"strings"
)

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

// Trait names one of the numbers a name chooses for its avatar, for
// Model.Pins.
type Trait int

// The traits that can be pinned. Each is a position from 0 to 1 across the
// range the name would have chosen from, so 0 is the smallest, roundest or
// leftmost an avatar ever is and 1 the largest, squarest or rightmost.
const (
	// TraitBodySize is how much of the frame the body takes.
	TraitBodySize Trait = iota
	// TraitBodyProportion runs from wider than tall to taller than wide.
	TraitBodyProportion
	// TraitBodySquareness runs from an ellipse toward a rounded square.
	TraitBodySquareness
	// TraitEyeSize is the width of the eyes.
	TraitEyeSize
	// TraitEyeRoundness runs from round eyes to tall capsules.
	TraitEyeRoundness
	// TraitEyeSquareness runs from soft ends to square ones.
	TraitEyeSquareness
	// TraitEyeSeparation is how far apart the eyes are.
	TraitEyeSeparation
	// TraitEyeLean runs from leaning one way, through upright, to the other.
	TraitEyeLean
	// TraitGazeX is where the eyes rest, from left to right.
	TraitGazeX
	// TraitGazeY is where the eyes rest, from high to low.
	TraitGazeY

	traitCount
)

// traitKeys holds, per Trait, its name and the key it is hashed under.
var traitKeys = [traitCount]struct{ name, key string }{
	TraitBodySize:       {"body size", "body.r"},
	TraitBodyProportion: {"body proportion", "body.ratio"},
	TraitBodySquareness: {"body squareness", "body.n"},
	TraitEyeSize:        {"eye size", "eye.rx"},
	TraitEyeRoundness:   {"eye roundness", "eye.ratio"},
	TraitEyeSquareness:  {"eye squareness", "eye.n"},
	TraitEyeSeparation:  {"eye separation", "eye.gap"},
	TraitEyeLean:        {"eye lean", "eye.lean"},
	TraitGazeX:          {"gaze x", "gaze.x"},
	TraitGazeY:          {"gaze y", "gaze.y"},
}

// String returns the trait's name, such as "eye size"; a value that is not
// a trait is "unknown".
func (t Trait) String() string {
	if t < 0 || t >= traitCount {
		return "unknown"
	}
	return traitKeys[t].name
}

// pins returns the trait positions the overrides fix: the middle of the
// chosen silhouette's band and of the chosen tone's, and every trait in
// Pins. Everything else still comes from the name.
func (m Model) pins() map[string]float64 {
	var fixed map[string]float64
	set := func(key string, at float64) {
		if fixed == nil {
			fixed = map[string]float64{}
		}
		fixed[key] = at
	}
	pin := func(key string, lo, hi float64) { set(key, (lo+hi)/2) }
	for t, at := range m.Pins {
		if t >= 0 && t < traitCount {
			set(traitKeys[t].key, at)
		}
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

// pinKey is Pins as a string, for the View cache: the pinned traits in order
// with their positions. An unpinned Model gives "".
func (m Model) pinKey() string {
	if len(m.Pins) == 0 {
		return ""
	}
	var b strings.Builder
	for t := Trait(0); t < traitCount; t++ {
		if at, ok := m.Pins[t]; ok {
			b.WriteString(strconv.Itoa(int(t)))
			b.WriteByte('=')
			b.WriteString(strconv.FormatFloat(at, 'g', -1, 64))
			b.WriteByte(';')
		}
	}
	return b.String()
}
