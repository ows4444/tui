package avatar

// Expression is a pose the avatar holds until it is changed. A pose reshapes
// and moves the two eyes; it never adds a mark, so no avatar grows a mouth.
type Expression int

// The expressions. ExpressionNone, the zero value, is the figure at rest.
const (
	ExpressionNone Expression = iota
	// ExpressionHappy is the eyes closed in a smile: two domes.
	ExpressionHappy
	// ExpressionSad is flat eyes dropped low, their outer ends down.
	ExpressionSad
	// ExpressionMad is wide flat bars, their inner ends down.
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
// eye, the squareness it gives them (0 keeps each eye's own), and whether
// they are drawn as domes, flat side down.
type pose struct {
	name string
	eye  [2]eyePose
	n    float64
	dome bool
}

var still = eyePose{sx: 1, sy: 1}

// poses is indexed by Expression. Every pose differs from its neighbours in
// more than one of size, height and tilt, so they stay apart in a few cells
// and with colour stripped.
var poses = [...]pose{
	ExpressionNone:      {name: "none", eye: [2]eyePose{still, still}},
	ExpressionHappy:     {name: "happy", eye: [2]eyePose{{sx: 2.1, sy: 1.05, dy: 0.14, turn: true}, {sx: 2.1, sy: 1.05, dy: 0.14, turn: true}}, dome: true},
	ExpressionSad:       {name: "sad", eye: [2]eyePose{{sx: 1.6, sy: 0.36, dy: 0.3, rot: -24, turn: true}, {sx: 1.6, sy: 0.36, dy: 0.3, rot: 24, turn: true}}},
	ExpressionMad:       {name: "mad", eye: [2]eyePose{{sx: 2.1, sy: 0.34, dx: 0.04, rot: 26, turn: true}, {sx: 2.1, sy: 0.34, dx: -0.04, rot: -26, turn: true}}},
	ExpressionSurprised: {name: "surprised", eye: [2]eyePose{{sx: 1.9, sy: 1.3, dy: -0.08}, {sx: 1.9, sy: 1.3, dy: -0.08}}, n: 2},
	ExpressionWink:      {name: "wink", eye: [2]eyePose{{sx: 1.9, sy: 0.2, turn: true}, still}},
	ExpressionSleepy:    {name: "sleepy", eye: [2]eyePose{{sx: 1.9, sy: 0.2, dy: 0.2, turn: true}, {sx: 1.9, sy: 0.2, dy: 0.2, turn: true}}},
	ExpressionThinking:  {name: "thinking", eye: [2]eyePose{{sx: 1, sy: 0.7, dx: 0.2, dy: -0.26}, {sx: 1, sy: 0.7, dx: 0.2, dy: 0.14}}},
}

func (e Expression) pose() pose {
	if e < 0 || int(e) >= len(poses) {
		return poses[ExpressionNone]
	}
	return poses[e]
}
