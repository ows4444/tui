package avatar

import (
	"math"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

// blinkInterval is the time between the frames of a blink.
const blinkInterval = 70 * time.Millisecond

// blinkOpen is how open the eyes are, as a fraction of their height, at each
// frame of a blink. Frame 0 is the avatar at rest.
var blinkOpen = [...]float64{1, 0.45, 0.08, 0.45}

type tickMsg struct{ gen int }

func (m Model) tick() tui.Cmd {
	gen := m.gen
	return tui.FromCtx(motion.After(blinkInterval, func(time.Time) tui.Msg { return tickMsg{gen: gen} }))
}

// Blink closes and reopens the eyes once; return the Cmd from your Update,
// for example on a key press. Blinking again mid-blink restarts it: ticks
// left over from before are ignored. Under motion.Reduced it returns nil and
// the eyes stay open.
func (m *Model) Blink() tui.Cmd {
	m.gen++
	if m.Motion.Reduced() {
		m.blink = 0
		return nil
	}
	m.blink = 1
	return m.tick()
}

// Blinking reports whether a blink is playing.
func (m Model) Blinking() bool { return m.blink != 0 }

// Update advances a blink one frame per tick and rests when it is done; it
// ignores every other Msg.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if t, ok := msg.(tickMsg); !ok || m.blink == 0 || t.gen != m.gen {
		return m, nil
	}
	m.blink++
	if m.blink == len(blinkOpen) {
		m.blink = 0
		return m, nil
	}
	return m, m.tick()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// LookAt turns the eyes toward a target dx columns right of and dy rows
// below the centre of the avatar: for the mouse pointer, the pointer's cell
// minus the centre cell. The further the target, the further the eyes turn,
// reaching a full look at one and a half avatar widths; LookAt(0, 0) returns
// them to rest. It sets LookX and LookY.
func (m *Model) LookAt(dx, dy int) {
	// A row is about two columns tall.
	x, y := float64(dx), 2*float64(dy)
	d := math.Hypot(x, y)
	if d == 0 {
		m.LookX, m.LookY = 0, 0
		return
	}
	k := math.Min(1, d/(1.5*float64(max(m.Width, 1)))) / d
	m.LookX, m.LookY = x*k, y*k
}

// unit clamps a look to [-1, 1]; NaN is no look.
func unit(v float64) float64 {
	if v != v {
		return 0
	}
	return math.Max(-1, math.Min(1, v))
}
