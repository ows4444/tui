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

// Accessory pins what the body wears on top: ears, a bunny's ears, an
// antenna, a tuft of hair or horns. AccessoryAuto, the zero value, lets the
// name choose, which is none for about half of all names. Only the round,
// organic, boxy, capsule and hexagon silhouettes wear one: on any other the pin
// changes nothing.
type Accessory int

// The accessories.
const (
	AccessoryAuto Accessory = iota
	AccessoryNone
	AccessoryEars
	AccessoryBunny
	AccessoryAntenna
	AccessoryTuft
	AccessoryHorns
)

// String returns the accessory's name, such as "ears"; AccessoryAuto and any
// value that is not an accessory are "auto".
func (a Accessory) String() string {
	if a <= AccessoryAuto || int(a) > len(accessories) {
		return "auto"
	}
	return accessories[a-1].name
}

// EyeStyle pins how the eyes are drawn: tall capsules, round dots, ovals or
// thin slits. EyesAuto, the zero value, lets the name choose, which is a capsule
// for just over half of all names.
type EyeStyle int

// The eye styles.
const (
	EyesAuto EyeStyle = iota
	EyesCapsule
	EyesDot
	EyesOval
	EyesSlit
)

// String returns the style's name, such as "dot"; EyesAuto and any value that
// is not a style are "auto".
func (s EyeStyle) String() string {
	if s <= EyesAuto || int(s) > len(eyeKinds) {
		return "auto"
	}
	return eyeKinds[s-1].name
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

// traitID names one of the numbers a name chooses for its avatar, for
// Model.traitPins.
type traitID int

// The traits that can be pinned. Each is a position from 0 to 1 across the
// range the name would have chosen from, so 0 is the smallest, roundest,
// fewest or leftmost an avatar ever is and 1 the largest, squarest, most or
// rightmost.
const (
	// traitBodySize is how much of the frame the body takes.
	traitBodySize traitID = iota
	// traitBodyProportion runs from wider than tall to taller than wide.
	traitBodyProportion
	// traitBodySquareness runs from an ellipse toward a rounded square.
	traitBodySquareness
	// traitEyeSize is the width of the eyes.
	traitEyeSize
	// traitEyeRoundness runs from round eyes to tall capsules.
	traitEyeRoundness
	// traitEyeSquareness runs from soft ends to square ones.
	traitEyeSquareness
	// traitEyeSeparation is how far apart the eyes are.
	traitEyeSeparation
	// traitEyeLean runs from leaning one way, through upright, to the other.
	traitEyeLean
	// traitGazeX is where the eyes rest, from left to right.
	traitGazeX
	// traitGazeY is where the eyes rest, from high to low.
	traitGazeY

	// The traits below belong to one or a few silhouettes; on any other
	// they pin nothing.

	// traitTilt is how far a boxy, hexagon or triangle body is turned, from
	// one way, through level, to the other.
	traitTilt
	// traitSquat is how flat a capsule is.
	traitSquat
	// traitCornerRounding runs from sharp corners to none at all, on a
	// hexagon or a triangle.
	traitCornerRounding
	// traitPetals is how many petals a sun has.
	traitPetals
	// traitPetalDistance is how far a sun's petals sit from its centre.
	traitPetalDistance
	// traitPetalSize is the size of a sun's petals.
	traitPetalSize
	// traitPetalRotation turns a sun's ring of petals.
	traitPetalRotation
	// traitLobes is how many lobes a cloud has.
	traitLobes
	// traitNubs is how many nubs a nub body has.
	traitNubs
	// traitNubAngle is where the first nub sits around the body.
	traitNubAngle
	// traitNubSize is the size of the nubs.
	traitNubSize
	// traitTipLength is how tall a droplet's point is.
	traitTipLength

	traitCount
)

// traitKeys holds, per traitID, its name and the keys it is hashed under: one,
// except that the size of the nubs is a key for each nub.
var traitKeys = [traitCount]struct {
	name string
	keys []string
}{
	traitBodySize:       {"body size", []string{"body.r"}},
	traitBodyProportion: {"body proportion", []string{"body.ratio"}},
	traitBodySquareness: {"body squareness", []string{"body.n"}},
	traitEyeSize:        {"eye size", []string{"eye.rx"}},
	traitEyeRoundness:   {"eye roundness", []string{"eye.ratio"}},
	traitEyeSquareness:  {"eye squareness", []string{"eye.n"}},
	traitEyeSeparation:  {"eye separation", []string{"eye.gap"}},
	traitEyeLean:        {"eye lean", []string{"eye.lean"}},
	traitGazeX:          {"gaze x", []string{"gaze.x"}},
	traitGazeY:          {"gaze y", []string{"gaze.y"}},
	traitTilt:           {"tilt", []string{"body.rot"}},
	traitSquat:          {"squat", []string{"capsule.squat"}},
	traitCornerRounding: {"corner rounding", []string{"poly.round"}},
	traitPetals:         {"petals", []string{"sun.n"}},
	traitPetalDistance:  {"petal distance", []string{"sun.dist"}},
	traitPetalSize:      {"petal size", []string{"sun.r"}},
	traitPetalRotation:  {"petal rotation", []string{"sun.rot"}},
	traitLobes:          {"lobes", []string{"cloud.n"}},
	traitNubs:           {"nubs", []string{"nub.n"}},
	traitNubAngle:       {"nub angle", []string{"nub.a0"}},
	traitNubSize:        {"nub size", []string{"nub.r0", "nub.r1"}},
	traitTipLength:      {"tip length", []string{"droplet.tip"}},
}

// String returns the trait's name, such as "eye size"; a value that is not
// a trait is "unknown".
func (t traitID) String() string {
	if t < 0 || t >= traitCount {
		return "unknown"
	}
	return traitKeys[t].name
}

// pins returns the trait positions the overrides fix: the middle of the
// chosen silhouette's band and of the chosen tone's, and every trait in
// traitPins. Everything else still comes from the name.
func (m Model) pins() map[string]float64 {
	var fixed map[string]float64
	set := func(key string, at float64) {
		if fixed == nil {
			fixed = map[string]float64{}
		}
		fixed[key] = at
	}
	pin := func(key string, lo, hi float64) { set(key, (lo+hi)/2) }
	for t, at := range m.traitPins {
		if t >= 0 && t < traitCount {
			for _, key := range traitKeys[t].keys {
				set(key, at)
			}
		}
	}
	if k, ok := m.Silhouette.kind(); ok {
		lo := 0.0
		if k > 0 {
			lo = shapes[k-1].upTo
		}
		pin("shape", lo, shapes[k].upTo)
	}
	if a := int(m.Accessory) - 1; a >= 0 && a < len(accessories) {
		lo := 0.0
		if a > 0 {
			lo = accessories[a-1].upTo
		}
		pin("acc.kind", lo, accessories[a].upTo)
	}
	if e := int(m.Eyes) - 1; e >= 0 && e < len(eyeKinds) {
		lo := 0.0
		if e > 0 {
			lo = eyeKinds[e-1].upTo
		}
		pin("eye.kind", lo, eyeKinds[e].upTo)
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
	if m.Hue == nil || math.IsNaN(*m.Hue) || math.IsInf(*m.Hue, 0) {
		return 0, false
	}
	h := math.Mod(*m.Hue, 360)
	if h < 0 {
		h += 360
	}
	return h, true
}

// pinKey is traitPins as a string, for the View cache: the pinned traits in order
// with their positions. An unpinned Model gives "".
func (m Model) pinKey() string {
	if len(m.traitPins) == 0 && m.Accessory == AccessoryAuto && m.Eyes == EyesAuto {
		return ""
	}
	var b strings.Builder
	b.WriteString("a" + strconv.Itoa(int(m.Accessory)) + "e" + strconv.Itoa(int(m.Eyes)) + ";")
	for t := traitID(0); t < traitCount; t++ {
		if at, ok := m.traitPins[t]; ok {
			b.WriteString(strconv.Itoa(int(t)))
			b.WriteByte('=')
			b.WriteString(strconv.FormatFloat(at, 'g', -1, 64))
			b.WriteByte(';')
		}
	}
	return b.String()
}
