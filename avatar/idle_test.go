package avatar

import (
	"context"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/motion"
)

// beat feeds m the tick it is waiting for, without waiting, and returns how
// long the real tick would have taken to arrive.
func beat(t *testing.T, m Model) (Model, time.Duration) {
	t.Helper()
	wait := m.idle().beat
	if m.blink != 0 {
		wait = blinkInterval
	}
	next, cmd := m.Update(tickMsg{owner: m.owner})
	if cmd == nil {
		t.Fatalf("%q stopped scheduling ticks while idling", m.Name)
	}
	return next, wait
}

func idling(name string) Model {
	m := New(name)
	m.Width, m.Height = 24, 12
	return m
}

// Once started, the idle loop schedules a tick after every tick until it is
// stopped; then a tick still on its way does nothing and the avatar rests.
func TestIdleRunsUntilStopped(t *testing.T) {
	m := idling("ada")
	rest := m.View()
	if m.Idling() {
		t.Fatal("a new avatar is idling")
	}
	cmd := m.StartIdle()
	if cmd == nil || !m.Idling() {
		t.Fatal("StartIdle did not start")
	}
	// The Cmd really produces a tick this avatar accepts.
	first := tui.RunCmd(context.Background(), cmd)
	if n, c := m.Update(first); c == nil || n.beat != 1 {
		t.Fatal("the scheduled tick did not advance the loop")
	}
	views := map[string]bool{}
	for range 300 {
		m, _ = beat(t, m)
		views[m.View()] = true
	}
	if len(views) < 4 {
		t.Errorf("300 ticks drew only %d different frames", len(views))
	}
	pending := tickMsg{owner: m.owner}
	m.StopIdle()
	if m.Idling() || m.Blinking() || m.View() != rest {
		t.Error("StopIdle did not return the avatar to rest")
	}
	if n, c := m.Update(pending); c != nil || n.View() != rest {
		t.Error("a tick that arrived after StopIdle moved the avatar")
	}
}

func TestIdleUnderReducedMotionRests(t *testing.T) {
	m := idling("ada")
	rest := m.View()
	m.Motion = motion.Reduced
	if cmd := m.StartIdle(); cmd != nil || m.Idling() || m.View() != rest {
		t.Error("StartIdle under reduced motion should schedule nothing and draw the resting figure")
	}
	// Reduced motion also ends a loop that was running.
	n := idling("ada")
	n.StartIdle()
	n.Motion = motion.Reduced
	if cmd := n.StartIdle(); cmd != nil || n.Idling() {
		t.Error("restarting under reduced motion left the loop running")
	}
}

// blinkTimes returns when, counted from StartIdle, a name's first n idle
// blinks begin.
func blinkTimes(t *testing.T, name string, n int) []time.Duration {
	t.Helper()
	m := idling(name)
	m.StartIdle()
	var at []time.Duration
	elapsed := m.idle().beat
	for len(at) < n {
		was := m.blink
		var wait time.Duration
		m, wait = beat(t, m)
		if was == 0 && m.blink == 1 {
			at = append(at, elapsed)
		}
		elapsed += wait
	}
	return at
}

// Two names side by side do not blink together: none of their first blinks
// begin at the same moment, and a wall of names has many rhythms.
func TestNamesIdleOutOfStep(t *testing.T) {
	names := []string{"ada", "linus", "grace", "ken", "margaret", "dennis", "barbara", "alan", "hedy", "edsger"}
	times := map[string][]time.Duration{}
	rhythms := map[idle]bool{}
	for _, name := range names {
		times[name] = blinkTimes(t, name, 8)
		rhythms[idling(name).idle()] = true
	}
	if len(rhythms) != len(names) {
		t.Errorf("%d names share %d rhythms", len(names), len(rhythms))
	}
	for i, a := range names {
		for _, b := range names[i+1:] {
			for _, ta := range times[a] {
				for _, tb := range times[b] {
					if ta == tb {
						t.Errorf("%s and %s both blink at %v", a, b, ta)
					}
				}
			}
		}
	}
}

// A tick moves only the avatar that scheduled it, so an app can hand every
// Msg to every avatar.
func TestATickMovesOnlyItsOwnAvatar(t *testing.T) {
	a, b := idling("ada"), idling("linus")
	a.StartIdle()
	b.StartIdle()
	if n, cmd := b.Update(tickMsg{owner: a.owner}); cmd != nil || n.beat != 0 {
		t.Error("another avatar's tick advanced this one")
	}
	if n, cmd := New("ken").Update(tickMsg{}); cmd != nil || n.Idling() {
		t.Error("a tick with no owner moved an avatar at rest")
	}
}

// Over one round of its rhythm an idling avatar glances aside, breathes and
// blinks, and the cached View always equals a fresh drawing.
func TestIdleGlancesBreathesAndBlinks(t *testing.T) {
	m := idling("ada")
	m.StartIdle()
	i := m.idle()
	glanced, zooms, blinked := false, map[float64]bool{}, false
	for range i.blinkEvery * i.glanceEvery {
		m, _ = beat(t, m)
		lookX, zoom := m.moved()
		if lookX != 0 {
			glanced = true
			still := m
			still.idling = false
			if m.View() == still.View() {
				t.Fatal("a glance did not move the eyes")
			}
		}
		zooms[zoom] = true
		blinked = blinked || m.Blinking()
		fresh := m
		fresh.cache = nil
		if m.View() != fresh.View() {
			t.Fatal("the cached View is stale while idling")
		}
	}
	if !glanced || !blinked || len(zooms) < 3 {
		t.Errorf("glanced=%v blinked=%v breath steps=%d", glanced, blinked, len(zooms))
	}
}

// A blink asked for while idling plays, and the loop carries on after it.
func TestBlinkWhileIdlingResumesTheLoop(t *testing.T) {
	m := idling("ada")
	m.StartIdle()
	old := tickMsg{owner: m.owner}
	if cmd := m.Blink(); cmd == nil || !m.Blinking() || !m.Idling() {
		t.Fatal("Blink while idling did not start")
	}
	if n, cmd := m.Update(old); cmd != nil || n.blink != 1 {
		t.Error("the idle tick from before the blink was not ignored")
	}
	for range len(blinkOpen) - 1 {
		m, _ = beat(t, m)
	}
	if m.Blinking() || !m.Idling() {
		t.Error("the loop did not carry on after the blink")
	}
	m, _ = beat(t, m)
	if m.beat != 1 {
		t.Errorf("the loop is on beat %d after the blink, want 1", m.beat)
	}
}

// The idle loop has no effect over a plate on the size of the figure, and
// SVG stays at rest.
func TestIdleLeavesThePlateAndSVGAlone(t *testing.T) {
	m := idling("ada")
	rest := m.SVG()
	m.Background = BackgroundCircle
	m.StartIdle()
	plate := m.View()
	for range breathBeats {
		m, _ = beat(t, m)
		if _, zoom := m.moved(); zoom != 1 {
			lookX, _ := m.moved()
			if lookX == 0 && !m.Blinking() && m.View() != plate {
				t.Fatal("a breath resized the figure over its plate")
			}
		}
	}
	m.Background = BackgroundNone
	if m.SVG() != rest {
		t.Error("idling changed the SVG")
	}
}

// Hover runs the idle loop only while the pointer is over the avatar: it
// starts on arrival, restarts nothing while the pointer stays, and stops,
// back at rest, when it leaves.
func TestHoverIdlesOnlyUnderThePointer(t *testing.T) {
	m := idling("ada")
	rest := m.View()
	if cmd := m.Hover(false); cmd != nil || m.Idling() {
		t.Error("a pointer that is elsewhere started the loop")
	}
	cmd := m.Hover(true)
	if cmd == nil || !m.Idling() {
		t.Fatal("the pointer arriving did not start the loop")
	}
	for range 5 {
		m, _ = beat(t, m)
	}
	at, owner := m.beat, m.owner
	if cmd := m.Hover(true); cmd != nil || m.beat != at || m.owner != owner {
		t.Error("the pointer staying restarted the loop")
	}
	pending := tickMsg{owner: m.owner}
	if cmd := m.Hover(false); cmd != nil || m.Idling() || m.View() != rest {
		t.Error("the pointer leaving did not stop the loop and return the avatar to rest")
	}
	if n, cmd := m.Update(pending); cmd != nil || n.View() != rest {
		t.Error("a tick that arrived after the pointer left moved the avatar")
	}
	if cmd := m.Hover(false); cmd != nil || m.Idling() {
		t.Error("the pointer staying away did something")
	}
}

func TestHoverUnderReducedMotionSchedulesNothing(t *testing.T) {
	m := idling("ada")
	rest := m.View()
	m.Motion = motion.Reduced
	for range 3 {
		if cmd := m.Hover(true); cmd != nil || m.Idling() || m.View() != rest {
			t.Fatal("Hover under reduced motion should schedule nothing and leave the avatar at rest")
		}
	}
}
