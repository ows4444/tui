package avatar

import (
	"context"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

// wobbling eases m into e and returns it on the first frame of the pose's
// own motion, with the Cmd that carries it on.
func wobbling(t *testing.T, e Expression) (Model, tui.Cmd) {
	t.Helper()
	m := reactor("ada")
	if cmd := m.SetExpression(e); cmd == nil {
		t.Fatalf("SetExpression(%v) did not start", e)
	}
	m, _, cmd := settle(t, m)
	return m, cmd
}

// step feeds m the tick of its held pose's motion, without waiting.
func step(m Model) (Model, tui.Cmd) { return m.Update(wobbleMsg{owner: m.poseOwner}) }

// Mad, scared and sick tremble while they are held: the ticks keep coming
// and the eyes are drawn on alternating sides.
func TestTremblingPosesShake(t *testing.T) {
	for _, e := range []Expression{ExpressionMad, ExpressionScared, ExpressionSick} {
		m, cmd := wobbling(t, e)
		if cmd == nil || m.wobble != 1 {
			t.Fatalf("%v: the ease did not start the tremble", e)
		}
		// The Cmd really produces a tick this avatar accepts.
		if _, ok := tui.RunCmd(context.Background(), cmd).(wobbleMsg); !ok && e == ExpressionMad {
			t.Fatalf("%v: the Cmd did not produce a tick of the tremble", e)
		}
		still := reactor("ada")
		still.Expression = e
		views := map[string]int{}
		for i := 0; i < 2*wobbleCycle; i++ {
			views[m.View()]++
			if m.View() == still.View() {
				t.Fatalf("%v: frame %d is the still pose", e, i)
			}
			if m, cmd = step(m); cmd == nil {
				t.Fatalf("%v: the tremble stopped after %d frames", e, i)
			}
		}
		if len(views) != 2 {
			t.Errorf("%v: a tremble drew %d different frames, want 2", e, len(views))
		}
		for _, n := range views {
			if n != wobbleCycle {
				t.Errorf("%v: the two sides were drawn %v times, want %d each", e, views, wobbleCycle)
			}
		}
		// PNG trembles too, by the pose's own small reach.
		a := string(m.PNG(96))
		m, _ = step(m)
		if string(m.PNG(96)) == a {
			t.Errorf("%v: the PNG does not tremble", e)
		}
	}
}

// Thinking rocks: its two eyes trade heights and are back where they began
// after wobbleCycle frames.
func TestThinkingRocks(t *testing.T) {
	m, cmd := wobbling(t, ExpressionThinking)
	if cmd == nil {
		t.Fatal("the ease did not start the rock")
	}
	still := reactor("ada")
	still.Expression = ExpressionThinking
	if m.View() != still.View() {
		t.Error("the first frame of the rock is not the pose as it is drawn still")
	}
	first := m.View()
	frames := map[string]bool{}
	heights := func(m Model) (float64, float64) {
		_, to, _ := m.poses()
		return to.eye[0].dy, to.eye[1].dy
	}
	l0, r0 := heights(m)
	swapped := false
	for i := 0; i < wobbleCycle; i++ {
		frames[m.View()] = true
		if l, r := heights(m); l > r && l0 < r0 {
			swapped = true
		}
		if m, cmd = step(m); cmd == nil {
			t.Fatal("the rock stopped")
		}
	}
	if !swapped {
		t.Error("the two eyes never traded heights")
	}
	if len(frames) < 4 {
		t.Errorf("a rock drew only %d different frames", len(frames))
	}
	if m.wobble != 1 || m.View() != first {
		t.Errorf("after %d frames the rock is on frame %d, not back where it began", wobbleCycle, m.wobble)
	}
}

// A reaction that lands on a pose that moves is still released when its hold
// is over: the motion's own ticks count it down.
func TestAMovingReactionIsStillReleased(t *testing.T) {
	m := reactor("ada")
	for i := 0; i < 60 && !m.shown().pose().moves(); i++ {
		m.React()
	}
	if !m.shown().pose().moves() {
		t.Fatal("sixty reactions never reached a pose that moves")
	}
	rest := reactor("ada").View()
	m, _, cmd := settle(t, m)
	if cmd == nil || m.wobble != 1 {
		t.Fatal("the reaction did not start moving")
	}
	want := int((ReactHold - (tweenFrames-1)*tweenInterval) / wobbleInterval)
	steps := 0
	for m.Reacting() {
		if steps++; steps > want+2 {
			t.Fatalf("the reaction was not released after %d frames", steps)
		}
		if m, cmd = step(m); cmd == nil {
			t.Fatal("the motion stopped before the release")
		}
	}
	if steps != want {
		t.Errorf("released after %d frames, want %d", steps, want)
	}
	m, _, cmd = settle(t, m)
	if cmd != nil || m.wobble != 0 || m.View() != rest {
		t.Error("the avatar did not return to rest, still, after the release")
	}
	// A release Msg, from a reaction on a still pose, also ends one that
	// moves.
	n := reactor("ada")
	for !n.shown().pose().moves() {
		n.React()
	}
	n, _, _ = settle(t, n)
	if o := release(t, n); o.Reacting() || o.wobble != 0 {
		t.Error("a release Msg did not end a moving reaction")
	}
}

// The ticks stop when the pose changes to one that holds still, whether it
// is eased or assigned, and a stale tick does nothing.
func TestMotionStopsWithThePose(t *testing.T) {
	m, _ := wobbling(t, ExpressionMad)
	stale := wobbleMsg{owner: m.poseOwner}
	// Eased to a still pose: the ease ends with no Cmd.
	eased := m
	eased.SetExpression(ExpressionHappy)
	if eased.wobble != 0 {
		t.Error("starting an ease did not stop the tremble")
	}
	if n, cmd := eased.Update(stale); cmd != nil || n.tween != eased.tween {
		t.Error("the old pose's tick disturbed the ease")
	}
	eased, _, cmd := settle(t, eased)
	if cmd != nil || eased.wobble != 0 {
		t.Error("a still pose left a Cmd or kept moving")
	}
	// Assigned a still pose: the next tick ends the motion.
	assigned := m
	assigned.Expression = ExpressionHappy
	if n, cmd := step(assigned); cmd != nil || n.wobble != 0 {
		t.Error("the tick after an assignment kept the motion going")
	}
	// Eased from one moving pose to another: the motion starts again.
	other := m
	other.SetExpression(ExpressionThinking)
	other, _, cmd = settle(t, other)
	if cmd == nil || other.wobble != 1 {
		t.Error("easing to another moving pose did not start its motion")
	}
	// A tick with no owner, another avatar's, or one for a still avatar.
	for name, msg := range map[string]tui.Msg{"no owner": wobbleMsg{}, "another avatar's": wobbleMsg{owner: other.poseOwner}} {
		if n, cmd := m.Update(msg); cmd != nil || n.wobble != m.wobble {
			t.Errorf("a tick with %s moved this avatar", name)
		}
	}
	if n, cmd := eased.Update(wobbleMsg{owner: eased.poseOwner}); cmd != nil || n.wobble != 0 {
		t.Error("a tick moved an avatar that is still")
	}
}

// Assigning Expression, and reduced motion, draw the pose still and schedule
// nothing.
func TestAssignedAndReducedPosesAreStill(t *testing.T) {
	for _, e := range []Expression{ExpressionMad, ExpressionThinking} {
		assigned := reactor("ada")
		assigned.Expression = e
		if assigned.wobble != 0 {
			t.Errorf("%v: assigning started the motion", e)
		}
		reduced := reactor("ada")
		reduced.Motion = motion.Reduced
		if cmd := reduced.SetExpression(e); cmd != nil || reduced.wobble != 0 {
			t.Errorf("%v: SetExpression under reduced motion scheduled a tick", e)
		}
		reduced.Motion = motion.Normal
		if reduced.View() != assigned.View() {
			t.Errorf("%v: the reduced pose is not the still pose", e)
		}
	}
	// Reduced motion also stops a pose that was moving.
	m, _ := wobbling(t, ExpressionMad)
	m.Motion = motion.Reduced
	if cmd := m.SetExpression(ExpressionThinking); cmd != nil || m.wobble != 0 {
		t.Error("reduced motion left a pose moving")
	}
	n, _ := wobbling(t, ExpressionMad)
	n.Motion = motion.Reduced
	if cmd := n.React(); cmd != nil || n.wobble != 0 {
		t.Error("React under reduced motion left a pose moving")
	}
}

// The cache follows a moving pose frame by frame.
func TestCacheFollowsTheMotion(t *testing.T) {
	for _, e := range []Expression{ExpressionMad, ExpressionThinking} {
		m, _ := wobbling(t, e)
		for i := 0; i < wobbleCycle; i++ {
			fresh := m
			fresh.cache = nil
			if m.View() != fresh.View() {
				t.Fatalf("%v: the cached View is stale on frame %d", e, i)
			}
			m, _ = step(m)
		}
	}
}
