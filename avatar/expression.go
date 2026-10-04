package avatar

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
	// flushed toward red.
	ExpressionMad
	// ExpressionSurprised is the only pose that grows the eyes: wide, tall
	// and round.
	ExpressionSurprised
	// ExpressionWink is one eye shut and the other open.
	ExpressionWink
	// ExpressionSleepy is both eyes nearly shut, and low.
	ExpressionSleepy
	// ExpressionThinking is the eyes at two heights, looking aside.
	ExpressionThinking
	// ExpressionSmug is both eyes half shut and tilted the same way, one
	// held higher than the other.
	ExpressionSmug
	// ExpressionUnsure is one eye open and the other squeezed and tilted.
	ExpressionUnsure
	// ExpressionScared is small eyes, lifted and drawn together.
	ExpressionScared
	// ExpressionLove is tall eyes leaning together, and the body flushed
	// rose.
	ExpressionLove
	// ExpressionShy is small eyes, low and drawn together, and a pale blush.
	ExpressionShy
	// ExpressionSick is two slumped bars at different heights, and the body
	// turned green.
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
// are drawn as domes, flat side down, and the tint it gives the body with how
// far it goes toward it (nil leaves the colours alone).
type pose struct {
	name string
	eye  [2]eyePose
	n    float64
	dome bool
	tint *tint
	heat float64
}

var still = eyePose{sx: 1, sy: 1}

// poses is indexed by Expression. Every pose differs from its neighbours in
// more than one of size, height and tilt, so they stay apart in a few cells
// and with colour stripped.
var poses = [...]pose{
	ExpressionNone:      {name: "none", eye: [2]eyePose{still, still}},
	ExpressionHappy:     {name: "happy", eye: [2]eyePose{{sx: 2.1, sy: 1.05, dy: 0.14, turn: true}, {sx: 2.1, sy: 1.05, dy: 0.14, turn: true}}, dome: true},
	ExpressionSad:       {name: "sad", eye: [2]eyePose{{sx: 1.6, sy: 0.36, dy: 0.3, rot: -24, turn: true}, {sx: 1.6, sy: 0.36, dy: 0.3, rot: 24, turn: true}}},
	ExpressionMad:       {name: "mad", eye: [2]eyePose{{sx: 2.1, sy: 0.34, dx: 0.04, rot: 26, turn: true}, {sx: 2.1, sy: 0.34, dx: -0.04, rot: -26, turn: true}}, tint: tintHot, heat: 0.62},
	ExpressionSurprised: {name: "surprised", eye: [2]eyePose{{sx: 1.9, sy: 1.3, dy: -0.08}, {sx: 1.9, sy: 1.3, dy: -0.08}}, n: 2},
	ExpressionWink:      {name: "wink", eye: [2]eyePose{{sx: 1.9, sy: 0.2, turn: true}, still}},
	ExpressionSleepy:    {name: "sleepy", eye: [2]eyePose{{sx: 1.9, sy: 0.2, dy: 0.2, turn: true}, {sx: 1.9, sy: 0.2, dy: 0.2, turn: true}}},
	ExpressionThinking:  {name: "thinking", eye: [2]eyePose{{sx: 1, sy: 0.7, dx: 0.2, dy: -0.26}, {sx: 1, sy: 0.7, dx: 0.2, dy: 0.14}}},
	ExpressionSmug:      {name: "smug", eye: [2]eyePose{{sx: 1.6, sy: 0.5, dy: -0.16, rot: 20, turn: true}, {sx: 1.6, sy: 0.5, dy: 0.04, rot: 20, turn: true}}},
	ExpressionUnsure:    {name: "unsure", eye: [2]eyePose{still, {sx: 1.5, sy: 0.4, dy: 0.12, rot: -24, turn: true}}},
	ExpressionScared:    {name: "scared", eye: [2]eyePose{{sx: 0.8, sy: 0.5, dx: 0.16, dy: -0.22}, {sx: 0.8, sy: 0.5, dx: -0.16, dy: -0.22}}, n: 2},
	ExpressionLove:      {name: "love", eye: [2]eyePose{{sx: 1.2, sy: 1.25, dx: 0.1, rot: -16, turn: true}, {sx: 1.2, sy: 1.25, dx: -0.1, rot: 16, turn: true}}, tint: tintRose, heat: 0.6},
	ExpressionShy:       {name: "shy", eye: [2]eyePose{{sx: 0.85, sy: 0.5, dx: 0.14, dy: 0.26}, {sx: 0.85, sy: 0.5, dx: -0.14, dy: 0.26}}, tint: tintBlush, heat: 0.55},
	ExpressionSick:      {name: "sick", eye: [2]eyePose{{sx: 2, sy: 0.3, dy: -0.04, rot: -20, turn: true}, {sx: 1.5, sy: 0.3, dy: 0.22, rot: 20, turn: true}}, tint: tintBile, heat: 0.6},
}

func (e Expression) pose() pose {
	if e < 0 || int(e) >= len(poses) {
		return poses[ExpressionNone]
	}
	return poses[e]
}
