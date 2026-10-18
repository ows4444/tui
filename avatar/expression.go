package avatar

import "math"

// Expression is a pose the avatar holds until it is changed. A pose reshapes
// and moves the two eyes, and may tint the body; it never adds a mark, so no
// avatar grows a mouth. No two poses differ by tint alone, so they stay
// apart with colour stripped.
type Expression int

// The expressions. ExpressionNone, the zero value, is the figure at rest.
const (
	ExpressionNone Expression = iota
	// ExpressionHappy is the eyes closed in a smile: two domes.
	ExpressionHappy
	// ExpressionSad is flat eyes dropped low, their outer ends down.
	ExpressionSad
	// ExpressionMad is wide flat bars, their inner ends down, and the body
	// flushed toward red. Eased into, it trembles.
	ExpressionMad
	// ExpressionSurprised is the only pose that grows the eyes: wide, tall
	// and round.
	ExpressionSurprised
	// ExpressionWink is one eye shut and the other open.
	ExpressionWink
	// ExpressionSleepy is both eyes nearly shut, and low.
	ExpressionSleepy
	// ExpressionThinking is the eyes at two heights, looking aside. Eased
	// into, the two trade heights on a loop: a loader with a face.
	ExpressionThinking
	// ExpressionSmug is both eyes half shut and tilted the same way, one
	// held higher than the other.
	ExpressionSmug
	// ExpressionUnsure is one eye open and the other squeezed and tilted.
	ExpressionUnsure
	// ExpressionScared is small eyes, lifted and drawn together. Eased into,
	// it trembles.
	ExpressionScared
	// ExpressionLove is tall eyes leaning together, and the body flushed
	// rose.
	ExpressionLove
	// ExpressionShy is small eyes, low and drawn together, and a pale blush.
	ExpressionShy
	// ExpressionSick is two slumped bars at different heights, and the body
	// turned green. Eased into, it trembles faintly.
	ExpressionSick
)

// String returns the expression's name, such as "happy"; ExpressionNone and
// any value that is not an expression are "none".
func (e Expression) String() string { return e.pose().name }

// eyePose is what a pose does to one eye: its width and height are scaled by
// sx and sy, its centre moves by dx and dy (fractions of the face's radius on
// each axis), and with turn set its own lean is replaced by rot degrees.
type eyePose struct {
	sx, sy, dx, dy, rot float64
	turn                bool
}

// pose is one expression: its name, what it does to the left and the right
// eye, the squareness it gives them (0 keeps each eye's own), whether they
// are drawn as domes, flat side down, the tint it gives the body with how
// far it goes toward it (nil leaves the colours alone), and how it moves
// while it is held: shake is how hard the eyes tremble, and with rock the two
// eyes trade heights on a loop. lift moves the whole figure, in frame units:
// a lifted body (negative) reads as light, a sunk one as heavy.
type pose struct {
	name  string
	eye   [2]eyePose
	n     float64
	dome  bool
	tint  *tint
	heat  float64
	shake float64
	rock  bool
	lift  float64
	// brow is what the pose does to each eyebrow and mouth to the mouth. A
	// rest figure has neither; a pose adds them, and they show only where the
	// picture is big enough to hold them (see markGate).
	brow  [2]browPose
	mouth mouthPose
}

// browPose is one eyebrow: on draws it, and dx and dy move it from where it
// sits over its eye (fractions of the face's radius on each axis), rot turns it
// in degrees, clockwise, and long scales its length.
type browPose struct {
	on          bool
	dx, dy, rot float64
	long        float64
}

type mouthKind int

// The mouths. A smile is the lower half of an ellipse, a frown the upper half,
// an O a small round hole and a flat mouth a short bar.
const (
	mouthNone mouthKind = iota
	mouthSmile
	mouthFrown
	mouthO
	mouthFlat
)

// mouthPose is the mouth: its kind, its size against the usual, how far it is
// moved sideways (a fraction of the face's radius) and turned (degrees,
// clockwise).
type mouthPose struct {
	kind      mouthKind
	scale, dx float64
	rot       float64
}

// brow returns a pose's eyebrow, shorthand for the table.
func brow(dx, dy, rot, long float64) browPose { return browPose{true, dx, dy, rot, long} }

// moves reports whether the pose moves while it is held.
func (p pose) moves() bool { return p.shake > 0 || p.rock }

// rocked returns p as it is on frame wob of its loop: a pose that rocks has
// its two eyes part of the way to each other's height, back where they began
// every wobbleCycle frames. Frame 1 is the pose as it is drawn still.
func (p pose) rocked(wob int) pose {
	if !p.rock || wob == 0 {
		return p
	}
	r := math.Cos(2 * math.Pi * float64(wob-1) / wobbleCycle)
	mid, half := (p.eye[0].dy+p.eye[1].dy)/2, (p.eye[1].dy-p.eye[0].dy)/2
	p.eye[0].dy, p.eye[1].dy = mid-half*r, mid+half*r
	return p
}

var still = eyePose{sx: 1, sy: 1}

// poses is indexed by Expression. Every pose differs from its neighbours in
// more than one of size, height and tilt, so they stay apart in a few cells
// and with colour stripped.
var poses = [...]pose{
	ExpressionNone:      {name: "none", eye: [2]eyePose{still, still}},
	ExpressionHappy:     {name: "happy", eye: [2]eyePose{{sx: 2.1, sy: 1.05, dy: 0.14, turn: true}, {sx: 2.1, sy: 1.05, dy: 0.14, turn: true}}, dome: true, lift: -2.2, mouth: mouthPose{kind: mouthSmile, scale: 1.3, rot: 180}},
	ExpressionSad:       {name: "sad", eye: [2]eyePose{{sx: 1.6, sy: 0.36, dy: 0.3, rot: -24, turn: true}, {sx: 1.6, sy: 0.36, dy: 0.3, rot: 24, turn: true}}, lift: 2.6, brow: [2]browPose{brow(0, 0.02, -16, 1), brow(0, 0.02, 16, 1)}, mouth: mouthPose{kind: mouthFrown, scale: 0.9}},
	ExpressionMad:       {name: "mad", eye: [2]eyePose{{sx: 2.1, sy: 0.34, dx: 0.04, rot: 26, turn: true}, {sx: 2.1, sy: 0.34, dx: -0.04, rot: -26, turn: true}}, tint: tintHot, heat: 0.62, shake: 0.55, lift: 0.8, brow: [2]browPose{brow(0.02, 0.06, 24, 1.1), brow(-0.02, 0.06, -24, 1.1)}, mouth: mouthPose{kind: mouthFrown, scale: 0.8}},
	ExpressionSurprised: {name: "surprised", eye: [2]eyePose{{sx: 1.9, sy: 1.3, dy: -0.08}, {sx: 1.9, sy: 1.3, dy: -0.08}}, n: 2, lift: -1.4, brow: [2]browPose{brow(0, -0.1, 0, 0.9), brow(0, -0.1, 0, 0.9)}, mouth: mouthPose{kind: mouthO, scale: 1}},
	ExpressionWink:      {name: "wink", eye: [2]eyePose{{sx: 1.9, sy: 0.2, turn: true}, still}, lift: -1.1, mouth: mouthPose{kind: mouthSmile, scale: 0.9, dx: 0.06, rot: 172}},
	ExpressionSleepy:    {name: "sleepy", eye: [2]eyePose{{sx: 1.9, sy: 0.2, dy: 0.2, turn: true}, {sx: 1.9, sy: 0.2, dy: 0.2, turn: true}}, lift: 1.2, mouth: mouthPose{kind: mouthFlat, scale: 0.6}},
	ExpressionThinking:  {name: "thinking", eye: [2]eyePose{{sx: 1, sy: 0.7, dx: 0.2, dy: -0.26}, {sx: 1, sy: 0.7, dx: 0.2, dy: 0.14}}, rock: true, lift: -0.4, brow: [2]browPose{brow(0, -0.08, -8, 0.9), {}}, mouth: mouthPose{kind: mouthFlat, scale: 0.7, dx: 0.08, rot: 10}},
	ExpressionSmug:      {name: "smug", eye: [2]eyePose{{sx: 1.6, sy: 0.5, dy: -0.16, rot: 20, turn: true}, {sx: 1.6, sy: 0.5, dy: 0.04, rot: 20, turn: true}}, lift: -1, brow: [2]browPose{brow(0, -0.1, -12, 1), {}}, mouth: mouthPose{kind: mouthSmile, scale: 0.8, dx: 0.1, rot: 166}},
	ExpressionUnsure:    {name: "unsure", eye: [2]eyePose{still, {sx: 1.5, sy: 0.4, dy: 0.12, rot: -24, turn: true}}, brow: [2]browPose{brow(0, -0.1, -10, 1), brow(0, 0.04, 14, 1)}, mouth: mouthPose{kind: mouthFlat, scale: 0.7, dx: -0.06, rot: -12}},
	ExpressionScared:    {name: "scared", eye: [2]eyePose{{sx: 0.8, sy: 0.5, dx: 0.16, dy: -0.22}, {sx: 0.8, sy: 0.5, dx: -0.16, dy: -0.22}}, n: 2, shake: 0.35, lift: -0.6, brow: [2]browPose{brow(0.04, -0.1, -12, 0.9), brow(-0.04, -0.1, 12, 0.9)}, mouth: mouthPose{kind: mouthO, scale: 0.8}},
	ExpressionLove:      {name: "love", eye: [2]eyePose{{sx: 1.5, sy: 1.4, dx: 0.08, rot: -8, turn: true}, {sx: 1.5, sy: 1.4, dx: -0.08, rot: 8, turn: true}}, n: 2, tint: tintRose, heat: 0.6, lift: -1.6, mouth: mouthPose{kind: mouthSmile, scale: 1.1, rot: 180}},
	ExpressionShy:       {name: "shy", eye: [2]eyePose{{sx: 0.85, sy: 0.5, dx: 0.14, dy: 0.26}, {sx: 0.85, sy: 0.5, dx: -0.14, dy: 0.26}}, tint: tintBlush, heat: 0.55, lift: 0.9, brow: [2]browPose{brow(0, 0.02, -10, 0.8), brow(0, 0.02, 10, 0.8)}, mouth: mouthPose{kind: mouthSmile, scale: 0.6, rot: 180}},
	ExpressionSick:      {name: "sick", eye: [2]eyePose{{sx: 2, sy: 0.3, dy: -0.04, rot: -20, turn: true}, {sx: 1.5, sy: 0.3, dy: 0.22, rot: 20, turn: true}}, tint: tintBile, heat: 0.6, shake: 0.18, lift: 1.4, brow: [2]browPose{brow(0, 0.02, -14, 1), brow(0, 0.02, 14, 1)}, mouth: mouthPose{kind: mouthFrown, scale: 0.7, rot: 8}},
}

func (e Expression) pose() pose {
	if e < 0 || int(e) >= len(poses) {
		return poses[ExpressionNone]
	}
	return poses[e]
}
