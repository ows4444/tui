package avatar

import (
	"context"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

func reactor(name string) Model {
	m := New(name)
	m.Width, m.Height = 24, 12
	return m
}

// A reaction shows an expression other than the avatar's own, and the Msg
// its Cmd produces releases it.
func TestReactHoldsAnotherExpressionUntilReleased(t *testing.T) {
	m := reactor("ada")
	rest := m.View()
	cmd := m.React()
	if cmd == nil || !m.Reacting() {
		t.Fatal("React did not start")
	}
	if m.shown() == ExpressionNone || m.View() == rest {
		t.Error("the reaction did not change the face")
	}
	if m.Expression != ExpressionNone {
		t.Errorf("React changed the Model's own Expression to %v", m.Expression)
	}
	// The Cmd waits ReactHold and then releases; run it for real once.
	msg := tui.RunCmd(context.Background(), cmd)
	if _, ok := msg.(releaseMsg); !ok {
		t.Fatalf("the Cmd produced %T, want a release", msg)
	}
	n, next := m.Update(msg)
	if next != nil || n.Reacting() || n.View() != rest {
		t.Error("the release did not return the avatar to its own expression")
	}
}

// A reaction never repeats the expression showing: not the avatar's own, and
// not the reaction it replaces.
func TestReactNeverShowsTheSameExpressionTwice(t *testing.T) {
	for _, own := range []Expression{ExpressionNone, ExpressionHappy, ExpressionThinking, 99} {
		m := reactor("linus")
		m.Expression = own
		before := m.shown()
		seen := map[Expression]bool{}
		for i := 0; i < 60; i++ {
			m.React()
			if m.shown() == before || m.shown() == ExpressionNone {
				t.Fatalf("own %v, reaction %d: showed %v after %v", own, i, m.shown(), before)
			}
			before = m.shown()
			seen[before] = true
		}
		if len(seen) < 6 {
			t.Errorf("own %v: 60 reactions used only %d expressions", own, len(seen))
		}
	}
}

// Reacting during the hold picks another expression and starts the hold
// again: the release of the reaction it replaced does nothing.
func TestReactingAgainRestartsTheHold(t *testing.T) {
	m := reactor("ada")
	m.React()
	first, old := m.shown(), releaseMsg{owner: m.reactOwner}
	m.React()
	if m.shown() == first {
		t.Fatal("the second reaction repeated the first")
	}
	held := m.View()
	if n, cmd := m.Update(old); cmd != nil || !n.Reacting() || n.View() != held {
		t.Error("the replaced reaction's release ended the new one")
	}
	if n, _ := m.Update(releaseMsg{owner: m.reactOwner}); n.Reacting() {
		t.Error("the current release did not end the reaction")
	}
	// A release reaches only the avatar that reacted, and one with no owner
	// reaches none.
	other := reactor("linus")
	other.React()
	if n, _ := other.Update(releaseMsg{owner: m.reactOwner}); !n.Reacting() {
		t.Error("another avatar's release ended this reaction")
	}
	if n, _ := other.Update(releaseMsg{}); !n.Reacting() {
		t.Error("a release with no owner ended a reaction")
	}
	if n, cmd := reactor("ken").Update(releaseMsg{}); cmd != nil || n.Reacting() {
		t.Error("a release moved an avatar that was not reacting")
	}
}

// The same name reacts the same way every time, and two names differently.
func TestReactionsAreReproduciblePerName(t *testing.T) {
	order := func(name string) []Expression {
		m := reactor(name)
		var out []Expression
		for i := 0; i < 12; i++ {
			m.React()
			out = append(out, m.shown())
		}
		return out
	}
	a, b := order("ada"), order("ada")
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("reaction %d differs between two runs: %v and %v", i, a[i], b[i])
		}
	}
	differ := false
	for i, e := range order("linus") {
		differ = differ || e != a[i]
	}
	if !differ {
		t.Error("two names react in the same order")
	}
}

func TestReactUnderReducedMotionDoesNothing(t *testing.T) {
	m := reactor("ada")
	m.Expression = ExpressionHappy
	rest := m.View()
	m.Motion = motion.Reduced
	if cmd := m.React(); cmd != nil || m.Reacting() || m.View() != rest {
		t.Error("React under reduced motion should schedule nothing and leave the expression alone")
	}
	// Reduced motion also drops a reaction that was being held.
	n := reactor("ada")
	n.React()
	n.Motion = motion.Reduced
	if cmd := n.React(); cmd != nil || n.Reacting() {
		t.Error("reacting under reduced motion left a reaction held")
	}
}

// A reaction is drawn by PNG as well, leaves SVG and Linearize alone, and
// carries on through a blink and the idle loop.
func TestReactionComposes(t *testing.T) {
	m := reactor("ada")
	svg, spoken, png := m.SVG(), m.Linearize(), string(m.PNG(48))
	m.StartIdle()
	m.React()
	if string(m.PNG(48)) == png {
		t.Error("PNG does not draw the reaction")
	}
	if m.SVG() != svg || m.Linearize() != spoken {
		t.Error("a reaction changed the SVG or the spoken text")
	}
	shown := m.shown()
	m, _ = beat(t, m)
	m.Blink()
	if !m.Reacting() || m.shown() != shown {
		t.Error("an idle beat or a blink ended the reaction")
	}
	m.StopIdle()
	if !m.Reacting() {
		t.Error("StopIdle ended the reaction")
	}
	fresh := m
	fresh.cache = nil
	if m.View() != fresh.View() {
		t.Error("the cached View is stale during a reaction")
	}
}
