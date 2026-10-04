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

// settle feeds m the ticks of the ease in progress, without waiting, until
// it rests, and returns it with the Views of the frames between and the Cmd
// the last tick returned.
func settle(t *testing.T, m Model) (Model, []string, tui.Cmd) {
	t.Helper()
	var frames []string
	var cmd tui.Cmd
	for i := 0; m.tween != 0; i++ {
		if i > 2*tweenFrames {
			t.Fatal("the ease does not end")
		}
		frames = append(frames, m.View())
		m, cmd = m.Update(tweenMsg{owner: m.poseOwner})
	}
	return m, frames, cmd
}

// release ends the reaction m holds, as its release Msg would, and settles
// the ease back.
func release(t *testing.T, m Model) Model {
	t.Helper()
	m, cmd := m.Update(releaseMsg{owner: m.poseOwner})
	if cmd == nil {
		t.Fatal("the release did not start the ease back")
	}
	m, _, _ = settle(t, m)
	return m
}

// A reaction eases into an expression other than the avatar's own, holds
// it, and on release eases back; the frames between are neither pose.
func TestReactEasesInHoldsAndEasesBack(t *testing.T) {
	m := reactor("ada")
	rest := m.View()
	cmd := m.React()
	if cmd == nil || !m.Reacting() {
		t.Fatal("React did not start")
	}
	if m.Expression != ExpressionNone {
		t.Errorf("React changed the Model's own Expression to %v", m.Expression)
	}
	// The Cmd is the first tick of the ease; run it for real once.
	if _, ok := tui.RunCmd(context.Background(), cmd).(tweenMsg); !ok {
		t.Fatal("React's Cmd did not produce a tick of the ease")
	}
	m, frames, hold := settle(t, m)
	held := m.View()
	if len(frames) != tweenFrames-1 || held == rest {
		t.Fatalf("the ease drew %d frames between (want %d), or did not reach the reaction", len(frames), tweenFrames-1)
	}
	for i, f := range frames {
		if f == rest || f == held {
			t.Errorf("frame %d of the ease is one of its two ends", i)
		}
	}
	if hold == nil {
		t.Fatal("the end of the ease did not schedule the release")
	}
	back, cmd := m.Update(releaseMsg{owner: m.poseOwner})
	if cmd == nil || back.Reacting() {
		t.Fatal("the release did not start the ease back")
	}
	back, frames, last := settle(t, back)
	if len(frames) != tweenFrames-1 || last != nil || back.View() != rest {
		t.Errorf("the ease back drew %d frames, left a Cmd (%v) or did not rest on the avatar's own pose", len(frames), last != nil)
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
		if len(seen) < 10 {
			t.Errorf("own %v: 60 reactions used only %d expressions", own, len(seen))
		}
	}
}

// Reacting during the hold picks another expression and starts again: the
// ticks and the release of the reaction it replaced do nothing.
func TestReactingAgainRestartsTheHold(t *testing.T) {
	m := reactor("ada")
	m.React()
	m, _, _ = settle(t, m)
	first, oldRelease, oldTick := m.shown(), releaseMsg{owner: m.poseOwner}, tweenMsg{owner: m.poseOwner}
	m.React()
	if m.shown() == first {
		t.Fatal("the second reaction repeated the first")
	}
	easing := m.View()
	for name, stale := range map[string]tui.Msg{"release": oldRelease, "tick": oldTick} {
		if n, cmd := m.Update(stale); cmd != nil || !n.Reacting() || n.View() != easing {
			t.Errorf("the replaced reaction's %s disturbed the new one", name)
		}
	}
	m, _, _ = settle(t, m)
	if n := release(t, m); n.Reacting() {
		t.Error("the current release did not end the reaction")
	}
	// A Msg reaches only the avatar that scheduled it, and one with no
	// owner reaches none.
	other := reactor("linus")
	other.React()
	for name, msg := range map[string]tui.Msg{
		"another avatar's release": releaseMsg{owner: m.poseOwner},
		"a release with no owner":  releaseMsg{},
		"a tick with no owner":     tweenMsg{},
	} {
		if n, cmd := other.Update(msg); cmd != nil || !n.Reacting() || n.tween != other.tween {
			t.Errorf("%s moved this avatar", name)
		}
	}
	if n, cmd := reactor("ken").Update(releaseMsg{}); cmd != nil || n.Reacting() {
		t.Error("a release moved an avatar that was not reacting")
	}
	// A release that arrives when nothing is held does nothing.
	done := release(t, func() Model { x := reactor("ken"); x.React(); x, _, _ = settle(t, x); return x }())
	if n, cmd := done.Update(releaseMsg{owner: done.poseOwner}); cmd != nil || n.Reacting() {
		t.Error("a second release did something")
	}
	if n, cmd := done.Update(tweenMsg{owner: done.poseOwner}); cmd != nil || n.tween != 0 {
		t.Error("a tick after the ease ended did something")
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
	if cmd := n.React(); cmd != nil || n.Reacting() || n.tween != 0 {
		t.Error("reacting under reduced motion left a reaction or an ease in progress")
	}
}

// SetExpression eases into the new pose: frames between the two, then rest
// on the new one, with no Cmd left over.
func TestSetExpressionEases(t *testing.T) {
	m := reactor("ada")
	rest := m.View()
	target := reactor("ada")
	target.Expression = ExpressionSurprised
	cmd := m.SetExpression(ExpressionSurprised)
	if cmd == nil || m.Expression != ExpressionSurprised {
		t.Fatal("SetExpression did not start an ease")
	}
	m, frames, last := settle(t, m)
	if len(frames) != tweenFrames-1 || last != nil || m.View() != target.View() {
		t.Fatalf("the ease drew %d frames, left a Cmd (%v) or did not rest on the new pose", len(frames), last != nil)
	}
	seen := map[string]bool{rest: true, target.View(): true}
	for i, f := range frames {
		if seen[f] {
			t.Errorf("frame %d of the ease repeats an earlier frame or an end", i)
		}
		seen[f] = true
	}
	// Setting the pose it already has, or a value that is no pose when it
	// has none, starts nothing.
	if cmd := m.SetExpression(ExpressionSurprised); cmd != nil || m.tween != 0 {
		t.Error("setting the same expression started an ease")
	}
	none := reactor("ada")
	if cmd := none.SetExpression(99); cmd != nil {
		t.Error("an invalid expression on a resting avatar started an ease")
	}
	// Assigning the field is still an instant change.
	m.Expression = ExpressionSad
	sad := reactor("ada")
	sad.Expression = ExpressionSad
	if m.View() != sad.View() {
		t.Error("assigning Expression did not change the pose at once")
	}
}

// An ease into or out of a tinting pose passes through colours between the
// two, and one into a dome changes the eye's kind half way.
func TestEaseBlendsTintAndShape(t *testing.T) {
	m := reactor("ada")
	body, _, _ := m.Colors()
	m.SetExpression(ExpressionMad)
	mid, _, _ := m.Colors()
	m, _, _ = settle(t, m)
	end, _, _ := m.Colors()
	if mid == body || mid == end || end == body {
		t.Errorf("the body went %v, %v, %v: no colour between", body, mid, end)
	}
	h := reactor("ada")
	h.SetExpression(ExpressionHappy)
	a, b, at := h.poses()
	if a.dome || !b.dome || at >= 0.5 {
		t.Fatalf("the ease starts at %v from dome=%v to dome=%v", at, a.dome, b.dome)
	}
	first := h.View()
	h, _ = h.Update(tweenMsg{owner: h.poseOwner})
	h, _ = h.Update(tweenMsg{owner: h.poseOwner})
	if _, _, at := h.poses(); at < 0.5 || h.View() == first {
		t.Errorf("the ease is at %v and has not moved", at)
	}
}

func TestSetExpressionUnderReducedMotionIsInstant(t *testing.T) {
	m := reactor("ada")
	m.Motion = motion.Reduced
	target := reactor("ada")
	target.Expression, target.Motion = ExpressionLove, motion.Reduced
	if cmd := m.SetExpression(ExpressionLove); cmd != nil || m.tween != 0 || m.View() != target.View() {
		t.Error("SetExpression under reduced motion should draw the new pose at once and schedule nothing")
	}
}

// While a reaction is held, SetExpression changes what the avatar returns
// to and leaves the reaction, and its release, alone.
func TestSetExpressionDuringAReaction(t *testing.T) {
	m := reactor("ada")
	m.React()
	m, _, _ = settle(t, m)
	held, owner := m.View(), m.poseOwner
	if cmd := m.SetExpression(ExpressionSleepy); cmd != nil || m.View() != held || m.poseOwner != owner {
		t.Error("SetExpression disturbed a held reaction")
	}
	sleepy := reactor("ada")
	sleepy.Expression = ExpressionSleepy
	if n := release(t, m); n.View() != sleepy.View() {
		t.Error("the reaction did not release to the new expression")
	}
}

// A reaction is drawn by PNG as well, leaves SVG and Linearize alone, and
// carries on through a blink and the idle loop.
func TestReactionComposes(t *testing.T) {
	m := reactor("ada")
	svg, spoken, png := m.SVG(), m.Linearize(), string(m.PNG(48))
	m.StartIdle()
	m.React()
	m, _, _ = settle(t, m)
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
	// The cache also follows an ease frame by frame.
	e := reactor("ada")
	e.SetExpression(ExpressionSad)
	for e.tween != 0 {
		fresh := e
		fresh.cache = nil
		if e.View() != fresh.View() {
			t.Fatal("the cached View is stale during an ease")
		}
		e, _ = e.Update(tweenMsg{owner: e.poseOwner})
	}
}
