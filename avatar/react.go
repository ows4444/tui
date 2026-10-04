package avatar

import (
	"strconv"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

// ReactHold is how long a reaction is held before the avatar returns to its
// own Expression.
const ReactHold = 1500 * time.Millisecond

// releaseMsg ends a reaction. owner is the token of the reaction that
// scheduled it, so a reaction that was replaced is not cut short by the
// release of the one before it, and one avatar's release leaves the others
// alone.
type releaseMsg struct{ owner *int }

// React makes the avatar pull a face: an expression other than the one it is
// showing, held for ReactHold and then released back to Expression. Return
// the Cmd from your Update, for example on a mouse click. Reacting again
// during the hold picks another expression and starts the hold again.
//
// The expression is not random: it is drawn from the name and from how many
// times this Model has reacted, so the same name reacts the same way every
// run, which keeps a recorded session reproducible, while a wall of names
// each reacts in its own order. Under motion.Reduced it returns nil and the
// expression does not change.
func (m *Model) React() tui.Cmd {
	if m.Motion.Reduced() {
		m.reaction, m.reactOwner = ExpressionNone, nil
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
	owner := new(int)
	m.reactOwner = owner
	return tui.FromCtx(motion.After(ReactHold, func(time.Time) tui.Msg { return releaseMsg{owner: owner} }))
}

// Reacting reports whether a reaction is being held.
func (m Model) Reacting() bool { return m.reaction != ExpressionNone }

// shown is the expression that is drawn: the reaction being held, or else
// the Model's own Expression.
func (m Model) shown() Expression {
	if m.reaction != ExpressionNone {
		return m.reaction
	}
	if m.Expression < 0 || int(m.Expression) >= len(poses) {
		return ExpressionNone
	}
	return m.Expression
}

// release ends the reaction msg belongs to, and reports whether msg was a
// release at all.
func (m *Model) release(msg tui.Msg) bool {
	r, ok := msg.(releaseMsg)
	if ok && m.reactOwner != nil && r.owner == m.reactOwner {
		m.reaction, m.reactOwner = ExpressionNone, nil
	}
	return ok
}
