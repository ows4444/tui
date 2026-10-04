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

// tickMsg advances one avatar. owner is the token of the animation that
// scheduled it, so an app can hand every Msg to every avatar it holds and
// each tick still moves only its own.
type tickMsg struct{ owner *int }

func (m Model) tick(d time.Duration) tui.Cmd {
	owner := m.owner
	return tui.FromCtx(motion.After(d, func(time.Time) tui.Msg { return tickMsg{owner: owner} }))
}

// Blink closes and reopens the eyes once; return the Cmd from your Update,
// for example on a key press. Blinking again mid-blink restarts it: ticks
// left over from before are ignored. While the avatar idles, the idle loop
// carries on after the blink. Under motion.Reduced it returns nil and the
// eyes stay open.
func (m *Model) Blink() tui.Cmd {
	m.owner = new(int)
	if m.Motion.Reduced() {
		m.blink = 0
		return nil
	}
	m.blink = 1
	return m.tick(blinkInterval)
}

// Blinking reports whether a blink is playing.
func (m Model) Blinking() bool { return m.blink != 0 }

// idle is one name's idle rhythm. Every number is drawn from the name, so a
// wall of avatars does not breathe, blink or glance in step.
type idle struct {
	beat        time.Duration // the time between beats
	blinkEvery  int           // a blink starts every this many beats...
	blinkAt     int           // ...on this beat of them
	glanceEvery int           // a glance starts every this many beats...
	glanceAt    int           // ...on this beat of them
	glanceX     float64       // where a glance looks, as a LookX
}

// breathBeats is the number of beats in one breath, glanceBeats how many a
// glance is held for, and breathDepth how much a breath grows and shrinks
// the body.
const (
	breathBeats = 8
	glanceBeats = 2
	breathDepth = 0.035
)

func (m Model) idle() idle {
	t := m.traits()
	i := idle{
		beat:        time.Duration(t.num("idle.beat", 360, 520)) * time.Millisecond,
		blinkEvery:  t.count("idle.blink", 7, 13),
		glanceEvery: t.count("idle.glance", 11, 19),
		glanceX:     0.75,
	}
	i.blinkAt = t.count("idle.blink.at", 0, i.blinkEvery-1)
	i.glanceAt = t.count("idle.glance.at", 0, i.glanceEvery-glanceBeats)
	if t.at("idle.glance.x") < 0.5 {
		i.glanceX = -i.glanceX
	}
	return i
}

// StartIdle begins the idle loop: the avatar breathes, blinks now and then
// and glances aside, at a rhythm drawn from its name. Return the Cmd from
// your Init or Update; the loop runs until StopIdle. Restarting is safe at
// any moment. Under motion.Reduced it returns nil and the avatar rests.
func (m *Model) StartIdle() tui.Cmd {
	m.owner = new(int)
	m.blink, m.beat = 0, 0
	if m.Motion.Reduced() {
		m.idling = false
		return nil
	}
	m.idling = true
	return m.tick(m.idle().beat)
}

// StopIdle ends the idle loop and returns the avatar to rest. A tick still
// on its way is ignored.
func (m *Model) StopIdle() {
	m.owner = nil
	m.idling, m.blink, m.beat = false, 0, 0
}

// Idling reports whether the idle loop is running.
func (m Model) Idling() bool { return m.idling }

// Update advances a blink one frame, the idle loop one beat, or the ease
// between two poses one frame, per tick, and starts a reaction back when its
// hold is over. A blink rests when it is done unless
// the avatar idles. It ignores a tick or a release that another avatar, or
// an animation since replaced, scheduled, and every other Msg.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if cmd, ok := m.posing(msg); ok {
		return m, cmd
	}
	if t, ok := msg.(tickMsg); !ok || m.owner == nil || t.owner != m.owner || m.blink == 0 && !m.idling {
		return m, nil
	}
	if m.blink != 0 {
		m.blink++
		if m.blink < len(blinkOpen) {
			return m, m.tick(blinkInterval)
		}
		m.blink = 0
		if !m.idling {
			return m, nil
		}
		return m, m.tick(m.idle().beat)
	}
	i := m.idle()
	m.beat++
	if m.beat%i.blinkEvery == i.blinkAt {
		m.blink = 1
		return m, m.tick(blinkInterval)
	}
	return m, m.tick(i.beat)
}

// moved returns what the idle loop adds to the figure on this beat: a look
// to add to LookX, and how much larger than at rest to draw the body.
func (m Model) moved() (lookX, zoom float64) {
	if !m.idling {
		return 0, 1
	}
	i := m.idle()
	if at := m.beat % i.glanceEvery; at >= i.glanceAt && at < i.glanceAt+glanceBeats {
		lookX = i.glanceX
	}
	return lookX, 1 + breathDepth*breath[m.beat%breathBeats]
}

// breath is one breath, sampled once per beat: a sine from rest, in and out.
var breath = [breathBeats]float64{0, 0.71, 1, 0.71, 0, -0.71, -1, -0.71}

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
