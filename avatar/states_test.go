package avatar

import (
	"github.com/ows4444/tui/motion"
	"testing"

	"github.com/ows4444/tui"
)

// Each test below is named after one arrow of the diagram in states.go.

func TestRestToBlinkingToRest(t *testing.T) {
	m := reactor("ada")
	if cmd := m.Blink(); cmd == nil || !m.Blinking() {
		t.Fatal("Blink did not start")
	}
	for i := 0; i < len(blinkOpen)+1 && m.Blinking(); i++ {
		m, _ = m.Update(tickMsg{owner: m.owner})
	}
	if m.Blinking() || m.Idling() {
		t.Error("the blink did not end at rest")
	}
}

func TestBlinkingRestartsOnBlink(t *testing.T) {
	m := reactor("ada")
	m.Blink()
	old := m.owner
	m, _ = m.Update(tickMsg{owner: m.owner})
	m.Blink()
	if m.blink != 1 {
		t.Errorf("blink frame %d after a restart", m.blink)
	}
	if next, _ := m.Update(tickMsg{owner: old}); next.blink != m.blink {
		t.Error("a tick from before the restart advanced the blink")
	}
}

func TestRestToIdlingAndBackByHover(t *testing.T) {
	m := reactor("ada")
	if cmd := m.Hover(true); cmd == nil || !m.Idling() {
		t.Fatal("Hover(true) did not start the idle loop")
	}
	if cmd := m.Hover(true); cmd != nil {
		t.Error("hovering twice started a second loop")
	}
	m.Hover(false)
	if m.Idling() || m.blink != 0 || m.beat != 0 {
		t.Error("Hover(false) did not return to rest")
	}
}

func TestIdlingBeatsAndBlinksThenIdlesOn(t *testing.T) {
	m := reactor("ada")
	m.StartIdle()
	sawBlink := false
	for i := 0; i < 400 && !sawBlink; i++ {
		var cmd tui.Cmd
		m, cmd = m.Update(tickMsg{owner: m.owner})
		if cmd == nil {
			t.Fatal("the idle loop stopped on its own")
		}
		sawBlink = m.Blinking()
	}
	if !sawBlink {
		t.Fatal("the idle loop never blinked")
	}
	for m.Blinking() {
		m, _ = m.Update(tickMsg{owner: m.owner})
	}
	if !m.Idling() {
		t.Error("the blink ended the idle loop")
	}
}

func TestReducedMotionKeepsEverythingAtRest(t *testing.T) {
	m := reactor("ada")
	m.Motion = motion.Reduced
	if m.Blink() != nil || m.StartIdle() != nil || m.React() != nil || m.SetExpression(ExpressionMad) != nil {
		t.Error("reduced motion started an animation")
	}
	if m.Blinking() || m.Idling() || m.Reacting() || m.tween != 0 {
		t.Error("reduced motion left a state running")
	}
}

func TestStillToEasingToStill(t *testing.T) {
	m := reactor("ada")
	if cmd := m.SetExpression(ExpressionHappy); cmd == nil || m.tween != 1 {
		t.Fatal("SetExpression did not start an ease")
	}
	m, _, _ = settle(t, m)
	if m.tween != 0 || m.wobble != 0 {
		t.Error("a still pose did not come to rest")
	}
}

func TestEasingToMovingAndBackToEasing(t *testing.T) {
	m, cmd := wobbling(t, ExpressionMad)
	if cmd == nil || m.wobble != 1 {
		t.Fatal("a trembling pose did not start moving")
	}
	m, _ = step(m)
	if m.wobble != 2 {
		t.Errorf("wobble frame %d after one tick", m.wobble)
	}
	if cmd := m.SetExpression(ExpressionHappy); cmd == nil || m.tween != 1 || m.wobble != 0 {
		t.Error("changing to a still pose did not ease")
	}
}

func TestStillToReactionHeldThenReleased(t *testing.T) {
	m := reactor("ada")
	if cmd := m.React(); cmd == nil || !m.Reacting() {
		t.Fatal("React did not start")
	}
	held, _, cmd := settle(t, m)
	if !held.Reacting() {
		t.Fatal("the ease into the reaction ended it")
	}
	_ = cmd
	rested := release(t, held)
	if rested.Reacting() || rested.tween != 0 {
		t.Error("the release did not return to the Expression")
	}
}

func TestHeldReactionReactsAgainAndRestartsTheHold(t *testing.T) {
	m := reactor("ada")
	m.React()
	first := m.reaction
	m, _, _ = settle(t, m)
	owner := m.poseOwner
	m.React()
	if m.reacts != 2 || m.poseOwner == owner {
		t.Error("a second React did not start a new hold")
	}
	if next, _ := m.Update(releaseMsg{owner: owner}); !next.Reacting() {
		t.Error("the first hold's release ended the second reaction")
	}
	_ = first
}

func TestHeldReactionDefersSetExpressionToTheRelease(t *testing.T) {
	m := reactor("ada")
	m.React()
	m, _, _ = settle(t, m)
	shown := m.shown()
	if cmd := m.SetExpression(ExpressionSleepy); cmd != nil || m.shown() != shown {
		t.Error("SetExpression changed what a held reaction shows")
	}
	m = release(t, m)
	if m.shown() != ExpressionSleepy {
		t.Errorf("after the release the avatar shows %v", m.shown())
	}
}

func TestMovingReactionIsReleasedByCountingFrames(t *testing.T) {
	m := reactor("ada")
	m.reaction = ExpressionMad
	m.poseOwner = new(int)
	m.wobble, m.holdLeft = 1, 2
	var cmd tui.Cmd
	m, cmd = step(m)
	if !m.Reacting() || cmd == nil {
		t.Fatal("released a frame early")
	}
	m, _ = step(m)
	if m.Reacting() && m.tween == 0 {
		t.Error("the hold ran out without easing back")
	}
}

func TestTicksOfAReplacedAnimationAreDropped(t *testing.T) {
	m := reactor("ada")
	m.SetExpression(ExpressionHappy)
	old := m.poseOwner
	m.SetExpression(ExpressionSad)
	if next, cmd := m.Update(tweenMsg{owner: old}); next.tween != m.tween || cmd != nil {
		t.Error("a tick of the replaced ease advanced the new one")
	}
}
