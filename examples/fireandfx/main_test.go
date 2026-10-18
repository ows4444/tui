package main

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

// ready is a moving model: the environment's motion preference is ignored.
func ready() model {
	m := newModel()
	m.pref, m.still = motion.Normal, false
	return send(m, tui.ResizeMsg{Width: 100, Height: 30})
}

func stillModel() model {
	m := newModel()
	m.pref, m.still = motion.Reduced, true
	return send(m, tui.ResizeMsg{Width: 100, Height: 30})
}

func frames(m model, n int) model {
	for range n {
		m = send(m, tickMsg{})
	}
	return m
}

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := layout.RectOf(m.screen(), m.size(), id)
	if !ok {
		t.Fatalf("no %q on screen:\n%s", id, text(m))
	}
	return r
}

func mouse(m model, x, y int, b tui.MouseButton, a tui.MouseAction) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: b, Action: a})
}

func click(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return mouse(m, r.X+1, r.Y, tui.MouseButtonLeft, tui.MouseActionPress)
}

func TestFirstScreenShowsTheFire(t *testing.T) {
	m := frames(ready(), 12)
	out := text(m)
	for _, want := range []string{"fireandfx", "▸1 Fire", "2 Plasma", "3 Typewriter", "[e Dissolve]", "playing", "speed 1/4", "@", "[1-3] scene"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestTicksAdvanceTheFireAndKeepTheLoopGoing(t *testing.T) {
	m := ready()
	before := string(m.fire.heat)
	m, cmd := update(m, tickMsg{})
	if m.frame != 1 || cmd == nil || string(m.fire.heat) == before {
		t.Fatalf("frame %d cmd %v", m.frame, cmd)
	}
	if ready().Init() == nil {
		t.Fatal("no first tick")
	}
}

func TestSceneKeysTabAndClicks(t *testing.T) {
	m := runes(ready(), "2")
	if m.scene != sPlasma || !strings.Contains(text(m), "▸2 Plasma") {
		t.Fatalf("scene %v", m.scene)
	}
	m = key(m, tui.KeyTab)
	if m.scene != sType {
		t.Fatalf("Tab: scene %v", m.scene)
	}
	m = key(m, tui.KeyTab)
	if m.scene != sFire {
		t.Fatalf("Tab wraps: scene %v", m.scene)
	}
	if m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift}); m.scene != sType {
		t.Fatalf("Shift+Tab: scene %v", m.scene)
	}
	if m = click(t, m, "scene-1"); m.scene != sPlasma {
		t.Fatalf("click: scene %v", m.scene)
	}
}

func TestTransitionRunsForAboutSixHundredMilliseconds(t *testing.T) {
	m := frames(ready(), 30)
	if m.progress() != 1 {
		t.Fatalf("progress %v at rest", m.progress())
	}
	m = runes(m, "2")
	if p := m.progress(); p != 0 || !strings.Contains(text(m), "Dissolve 0%") {
		t.Fatalf("progress %v:\n%s", p, text(m))
	}
	m = frames(m, 7)
	if p := m.progress(); p <= 0 || p >= 1 {
		t.Fatalf("progress %v midway", p)
	}
	mid := text(m)
	m = frames(m, 10)
	if m.progress() != 1 || text(m) == mid {
		t.Fatalf("progress %v after the transition", m.progress())
	}
}

func TestEveryEffectDrawsMidTransition(t *testing.T) {
	m := ready()
	for i := range effects {
		m.fx = i
		m = frames(runes(runes(m, "3"), "1"), 4)
		out := text(m)
		if len(strings.Split(out, "\n")) > 30 || strings.TrimSpace(out) == "" {
			t.Errorf("%s: bad frame:\n%s", effects[i].name, out)
		}
		m = frames(m, 20)
	}
}

func TestEffectKeyAndClickCycleAndRestartTheTransition(t *testing.T) {
	m := frames(ready(), 30)
	m = runes(m, "e")
	if m.fx != 1 || !strings.Contains(text(m), "[e Fade]") || m.progress() != 0 {
		t.Fatalf("fx %d progress %v", m.fx, m.progress())
	}
	m = click(t, m, "effect")
	if m.fx != 2 || !strings.Contains(text(m), "[e Slide]") {
		t.Fatalf("fx %d", m.fx)
	}
	for range 2 {
		m = runes(m, "e")
	}
	if m.fx != 0 {
		t.Fatalf("fx %d did not wrap", m.fx)
	}
}

func TestPauseFreezesTheSceneButNotTheTransition(t *testing.T) {
	m := runes(ready(), " ")
	if !m.paused || !strings.Contains(text(m), "paused") {
		t.Fatal("not paused")
	}
	heat := string(m.fire.heat)
	m = frames(m, 5)
	if string(m.fire.heat) != heat || m.frame != 5 {
		t.Fatalf("paused fire moved, frame %d", m.frame)
	}
}

func TestSpeedRunsMoreStepsPerFrame(t *testing.T) {
	m := runes(runes(runes(runes(ready(), "2"), "+"), "+"), "+")
	if m.speed != 4 || !strings.Contains(text(m), "speed 4/4") {
		t.Fatalf("speed %d", m.speed)
	}
	m = runes(m, "+")
	if m.speed != 4 {
		t.Fatal("speed past 4")
	}
	m = frames(m, 5)
	if m.sim != 20 {
		t.Fatalf("sim %d, want 20", m.sim)
	}
	if m = runes(runes(runes(runes(m, "-"), "-"), "-"), "-"); m.speed != 1 {
		t.Fatalf("speed %d", m.speed)
	}
}

func TestFireControls(t *testing.T) {
	m := runes(ready(), "i")
	if m.fire.on {
		t.Fatal("source still on")
	}
	m = frames(runes(m, "i"), 3)
	if !m.fire.on {
		t.Fatal("source did not come back")
	}
	m = key(key(m, tui.KeyRight), tui.KeyRight)
	if m.wind != 1 {
		t.Fatalf("wind %d", m.wind)
	}
	m = runes(runes(runes(m, "h"), "h"), "h")
	if m.wind != -1 {
		t.Fatalf("wind %d", m.wind)
	}
}

func TestMouseLightsTheFire(t *testing.T) {
	m := ready()
	m.fire.heat = make([]uint8, len(m.fire.heat))
	r := rect(t, m, "body")
	m = mouse(m, r.X+10, r.Y+5, tui.MouseButtonLeft, tui.MouseActionPress)
	if m.fire.heat[5*m.fire.w+10] != maxHeat {
		t.Fatal("click did not ignite")
	}
	m.fire.heat = make([]uint8, len(m.fire.heat))
	m = mouse(m, r.X+20, r.Y+7, tui.MouseButtonLeft, tui.MouseActionMotion)
	if m.fire.heat[7*m.fire.w+20] != maxHeat {
		t.Fatal("dragging did not ignite")
	}
	m.fire.heat = make([]uint8, len(m.fire.heat))
	if m = mouse(m, r.X+30, r.Y+7, tui.MouseButtonNone, tui.MouseActionMotion); lit(m.fire) != 0 {
		t.Fatal("hovering ignited")
	}
}

func TestPlasmaRippleFollowsThePointer(t *testing.T) {
	m := runes(ready(), "2")
	m = frames(m, 30)
	a := text(m)
	r := rect(t, m, "body")
	m = mouse(m, r.X+50, r.Y+10, tui.MouseButtonNone, tui.MouseActionMotion)
	if m.mx != 50 || m.my != 10 || text(m) == a {
		t.Fatalf("pointer %d,%d", m.mx, m.my)
	}
	if !strings.ContainsAny(text(m), "@%#") {
		t.Fatal("plasma has no bright cells")
	}
}

func TestTypewriterTypesPausesAndRestarts(t *testing.T) {
	m := frames(runes(ready(), "3"), 30)
	out := text(m)
	if !strings.Contains(out, "tui") || strings.Contains(out, "reduced motion") {
		t.Fatalf("not typing yet:\n%s", out)
	}
	if m.chars == 0 || m.chars >= scriptLen() {
		t.Fatalf("chars %d", m.chars)
	}
	m = click(t, m, "body")
	if m.chars != scriptLen() || !strings.Contains(text(m), "type it again") {
		t.Fatalf("click did not finish:\n%s", text(m))
	}
	m = click(t, m, "body")
	if m.chars != 0 {
		t.Fatalf("second click: chars %d", m.chars)
	}
	m = frames(m, 1500)
	if m.chars != scriptLen() {
		t.Fatalf("chars %d after 1500 frames", m.chars)
	}
	if m = runes(m, "r"); m.chars != 0 {
		t.Fatalf("r: chars %d", m.chars)
	}
}

func TestReducedMotionShowsFinishedStillFrames(t *testing.T) {
	m := stillModel()
	if m.Init() != nil {
		t.Fatal("a tick loop under reduced motion")
	}
	if m.progress() != 1 || !strings.Contains(text(m), "Reduced motion") {
		t.Fatalf("progress %v:\n%s", m.progress(), text(m))
	}
	m = runes(m, "3")
	if m.progress() != 1 || m.chars != scriptLen() || !strings.Contains(text(m), "Press n to step") {
		t.Fatalf("typewriter should be finished:\n%s", text(m))
	}
	m = runes(runes(m, "e"), "1")
	if m.progress() != 1 {
		t.Fatal("transition animated")
	}
	heat := string(m.fire.heat)
	if m = runes(m, "n"); string(m.fire.heat) == heat {
		t.Fatal("n did not step the fire")
	}
}

func TestResizeRebuildsTheFireAndKeepsTheSource(t *testing.T) {
	m := runes(ready(), "i")
	m = send(m, tui.ResizeMsg{Width: 60, Height: 20})
	if m.fire.w != 60 || m.fire.h != 17 || m.fire.on {
		t.Fatalf("fire %dx%d on %v", m.fire.w, m.fire.h, m.fire.on)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := runes(ready(), "?")
	if !strings.Contains(text(m), "dissolve, fade, slide, sweep") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyCtrlC}); c == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, c := ready().Update(tui.Key{Type: tui.KeyRunes, Text: "q"}); c == nil {
		t.Fatal("q did not quit")
	}
}
