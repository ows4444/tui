package avatar

import (
	"math"
	"strconv"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

// ReactHold is how long a reaction lasts, from the click to the moment the
// avatar starts back to its own Expression. The ease into the reaction is
// part of it.
const ReactHold = 1500 * time.Millisecond

// A pose change is eased over tweenFrames frames, tweenInterval apart: the
// eyes are drawn part of the way from the old pose to the new one on each.
const (
	tweenFrames   = 4
	tweenInterval = 45 * time.Millisecond
)

// A pose that moves while it is held is redrawn every wobbleInterval. A
// tremble alternates sides each frame; a rock takes wobbleCycle frames to
// come round. trembleReach is how far a full-strength tremble moves the
// eyes, as a fraction of the face's radius.
const (
	wobbleInterval = 90 * time.Millisecond
	wobbleCycle    = 10
	trembleReach   = 0.05
)

// tweenMsg advances an ease one frame, wobbleMsg advances a held pose's own
// motion one frame, and releaseMsg ends a reaction. owner is the token of
// the pose change that scheduled it, so a change that was replaced is not
// disturbed by what the one before it left behind, and one avatar's Msgs
// leave the others alone.
type (
	tweenMsg   struct{ owner *int }
	wobbleMsg  struct{ owner *int }
	releaseMsg struct{ owner *int }
)

func (m Model) wobbleTick() tui.Cmd {
	owner := m.poseOwner
	return tui.FromCtx(motion.After(wobbleInterval, func(time.Time) tui.Msg { return wobbleMsg{owner: owner} }))
}

// tremble is how far sideways, in frame units, the eyes of pose p are drawn
// on this frame of its motion: one way on odd frames and the other on even,
// and never less than floor either way, so that it shows in a few cells.
func (m Model) tremble(p pose, f figure, floor float64) float64 {
	if m.wobble == 0 || m.tween != 0 || p.shake == 0 {
		return 0
	}
	reach := math.Max(p.shake*trembleReach*f.face.rx, floor)
	if m.wobble%2 == 0 {
		return -reach
	}
	return reach
}

// ease starts the ease from pose from to the pose now shown and returns its
// first tick.
func (m *Model) ease(from Expression) tui.Cmd {
	owner := new(int)
	m.from, m.tween, m.poseOwner, m.wobble = from, 1, owner, 0
	return tui.FromCtx(motion.After(tweenInterval, func(time.Time) tui.Msg { return tweenMsg{owner: owner} }))
}

// React makes the avatar pull a face: an expression other than the one it is
// showing, eased into, held until ReactHold has passed and then eased back
// to Expression. Return the Cmd from your Update, for example on a mouse
// click, and pass the Msgs that follow to Update. Reacting again during the
// hold picks another expression and starts the hold again.
//
// The expression is not random: it is drawn from the name and from how many
// times this Model has reacted, so the same name reacts the same way every
// run, which keeps a recorded session reproducible, while a wall of names
// each reacts in its own order. A reaction that lands on a pose that moves
// trembles or rocks while it is held. Under motion.Reduced it returns nil and
// the expression does not change.
func (m *Model) React() tui.Cmd {
	if m.Motion.Reduced() {
		m.reaction, m.tween, m.wobble, m.poseOwner = ExpressionNone, 0, 0, nil
		return nil
	}
	shown := m.shown()
	var pool []Expression
	for e := ExpressionNone + 1; int(e) < len(poses); e++ {
		if e != shown {
			pool = append(pool, e)
		}
	}
	m.reacts++
	pick := m.traits().at("react." + strconv.Itoa(m.reacts))
	m.reaction = pool[int(pick*float64(len(pool)))]
	return m.ease(shown)
}

// Reacting reports whether a reaction is being held.
func (m Model) Reacting() bool { return m.reaction != ExpressionNone }

// SetExpression changes Expression and eases the eyes into the new pose over
// a few frames; return the Cmd from your Update. A pose that moves (mad,
// scared and sick tremble, thinking rocks) then keeps moving until the
// expression changes again, for as long as its Msgs are passed to Update.
// Assigning the field changes the pose at once and draws it still, as does
// this under motion.Reduced, where it returns nil. While a reaction is held
// the change shows when the reaction is released.
func (m *Model) SetExpression(e Expression) tui.Cmd {
	from := m.shown()
	m.Expression = e
	if m.Motion.Reduced() {
		m.tween, m.wobble = 0, 0
		return nil
	}
	if m.reaction != ExpressionNone || m.shown() == from {
		return nil
	}
	return m.ease(from)
}

// shown is the expression the avatar is in, or is easing into: the reaction
// being held, or else the Model's own Expression.
func (m Model) shown() Expression {
	if m.reaction != ExpressionNone {
		return m.reaction
	}
	if m.Expression < 0 || int(m.Expression) >= len(poses) {
		return ExpressionNone
	}
	return m.Expression
}

// easing is the pose being eased away from, or the pose shown when no ease
// is in progress.
func (m Model) easing() Expression {
	if m.tween == 0 {
		return m.shown()
	}
	return m.from
}

// poses returns what to draw: the pose eased from, the pose eased to, and
// how far along the ease is; 1 when the avatar rests on the pose shown.
func (m Model) poses() (from, to pose, at float64) {
	to = m.shown().pose()
	if m.tween == 0 {
		to = to.rocked(m.wobble)
		return to, to, 1
	}
	return m.from.pose(), to, float64(m.tween) / tweenFrames
}

// posing handles the Msgs of a pose change: a tick of the ease, a tick of a
// held pose's own motion, and the release of a reaction. It reports whether msg was one of them, with the
// Cmd that carries the change on.
func (m *Model) posing(msg tui.Msg) (tui.Cmd, bool) {
	switch msg := msg.(type) {
	case tweenMsg:
		if m.poseOwner == nil || msg.owner != m.poseOwner || m.tween == 0 {
			return nil, true
		}
		if m.tween++; m.tween < tweenFrames {
			owner := m.poseOwner
			return tui.FromCtx(motion.After(tweenInterval, func(time.Time) tui.Msg { return tweenMsg{owner: owner} })), true
		}
		m.tween = 0
		hold := ReactHold - (tweenFrames-1)*tweenInterval
		if m.shown().pose().moves() {
			// The pose trembles or rocks: keep ticking, and count a
			// reaction's hold in frames.
			m.wobble, m.holdLeft = 1, int(hold/wobbleInterval)
			return m.wobbleTick(), true
		}
		if m.reaction == ExpressionNone {
			return nil, true
		}
		// The ease into a reaction is over: wait out the rest of the hold.
		owner := m.poseOwner
		return tui.FromCtx(motion.After(hold, func(time.Time) tui.Msg { return releaseMsg{owner: owner} })), true
	case wobbleMsg:
		if m.poseOwner == nil || msg.owner != m.poseOwner || m.wobble == 0 {
			return nil, true
		}
		if !m.shown().pose().moves() {
			// Expression was assigned a pose that holds still.
			m.wobble = 0
			return nil, true
		}
		if m.reaction != ExpressionNone {
			if m.holdLeft--; m.holdLeft <= 0 {
				return m.letGo(), true
			}
		}
		m.wobble = m.wobble%wobbleCycle + 1
		return m.wobbleTick(), true
	case releaseMsg:
		if m.poseOwner == nil || msg.owner != m.poseOwner || m.reaction == ExpressionNone {
			return nil, true
		}
		return m.letGo(), true
	}
	return nil, false
}

// letGo ends the reaction being held and starts the ease back to the Model's
// own Expression.
func (m *Model) letGo() tui.Cmd {
	held := m.reaction
	m.reaction = ExpressionNone
	return m.ease(held)
}
