package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/tuitest"
)

// fakeClock is a clock the test moves by hand.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func key(m model, t tui.KeyType) (model, tui.Cmd) { return update(m, tui.Key{Type: t}) }

func runes(m model, s string) (model, tui.Cmd) {
	return update(m, tui.Key{Type: tui.KeyRunes, Text: s})
}

func ready() (model, *fakeClock) {
	c := &fakeClock{t: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)}
	m := newModelAt(c.now)
	m, _ = update(m, tui.ResizeMsg{Width: 80, Height: 24})
	return m, c
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

func click(t testing.TB, m model, id string) (model, tui.Cmd) {
	r := rect(t, m, id)
	return update(m, tui.MouseEvent{X: r.X + min(1, r.W-1), Y: r.Y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: 1})
}

func TestFirstScreenIsAStoppedStopwatch(t *testing.T) {
	m, _ := ready()
	out := text(m)
	for _, want := range []string{"stopwatch", "▸1 Stopwatch", "2 Timer", "3 Clock", "Space Start", "l Lap", "r Reset", "No laps yet", "0 laps", "[space] start/stop", "┌─┐"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestSpaceStartsStopsAndTheTimeFollowsTheClock(t *testing.T) {
	m, c := ready()
	m, cmd := key(m, tui.KeySpace)
	if !m.w.running || cmd == nil || !strings.Contains(text(m), "Space Stop") {
		t.Fatal("not started")
	}
	c.advance(1234 * time.Millisecond)
	if m.w.elapsed(c.now()) != 1234*time.Millisecond {
		t.Fatalf("%v", m.w.elapsed(c.now()))
	}
	m, _ = runes(m, " ")
	c.advance(time.Hour)
	if m.w.running || m.w.elapsed(c.now()) != 1234*time.Millisecond || !strings.Contains(text(m), "Space Start") {
		t.Fatal("did not stop")
	}
}

func TestTicksKeepTheLoopGoingOnlyWhileRunning(t *testing.T) {
	m, _ := ready()
	m, _ = key(m, tui.KeySpace)
	seq := m.seq
	if _, cmd := update(m, tickMsg{seq}); cmd == nil {
		t.Fatal("a live tick should schedule the next")
	}
	if _, cmd := update(m, tickMsg{seq - 1}); cmd != nil {
		t.Fatal("a stale tick scheduled another")
	}
	m, _ = key(m, tui.KeySpace)
	if _, cmd := update(m, tickMsg{m.seq}); cmd != nil {
		t.Fatal("a stopped watch kept ticking")
	}
	m, _ = key(m, tui.KeySpace)
	m, _ = key(m, tui.KeySpace)
	m, _ = key(m, tui.KeySpace)
	if m.seq < seq+3 {
		t.Fatal("each start and stop should begin a new chain")
	}
}

func TestLapsAreListedNewestFirstWithFastestAndSlowest(t *testing.T) {
	m, c := ready()
	m, _ = key(m, tui.KeySpace)
	for _, d := range []time.Duration{1230 * time.Millisecond, 2000 * time.Millisecond, 500 * time.Millisecond} {
		c.advance(d)
		m, _ = runes(m, "l")
	}
	out := text(m)
	i3, i2, i1 := strings.Index(out, "\n  3 "), strings.Index(out, "\n  2 "), strings.Index(out, "\n  1 ")
	if !(i3 > 0 && i3 < i2 && i2 < i1) {
		t.Fatalf("order wrong:\n%s", out)
	}
	for _, want := range []string{"00:01.23", "00:02.00", "00:00.50", "00:03.73", "slowest", "fastest", "3 laps"} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
}

func TestLapBeforeStartingSaysSo(t *testing.T) {
	m, _ := ready()
	m, _ = runes(m, "l")
	if len(m.w.laps) != 0 || !strings.Contains(m.msg, "Start the stopwatch first") {
		t.Fatalf("laps %d msg %q", len(m.w.laps), m.msg)
	}
}

func TestResetClearsEverything(t *testing.T) {
	m, c := ready()
	m, _ = key(m, tui.KeySpace)
	c.advance(time.Second)
	m, _ = runes(m, "l")
	m, _ = runes(m, "r")
	if m.w.running || len(m.w.laps) != 0 || m.w.elapsed(c.now()) != 0 {
		t.Fatal("not reset")
	}
}

func TestButtonsByMouse(t *testing.T) {
	m, c := ready()
	m, _ = click(t, m, "btn-start")
	c.advance(2 * time.Second)
	m, _ = click(t, m, "btn-lap")
	if !m.w.running || len(m.w.laps) != 1 {
		t.Fatalf("running %v laps %d", m.w.running, len(m.w.laps))
	}
	m, _ = click(t, m, "btn-start")
	if m.w.running {
		t.Fatal("Stop did not stop")
	}
	m, _ = click(t, m, "btn-reset")
	if len(m.w.laps) != 0 {
		t.Fatal("Reset left laps")
	}
}

func TestManyLapsScrollByWheelAndKeys(t *testing.T) {
	m, c := ready()
	m, _ = update(m, tui.ResizeMsg{Width: 80, Height: 16})
	m, _ = key(m, tui.KeySpace)
	for i := 0; i < 20; i++ {
		c.advance(time.Second)
		m, _ = runes(m, "l")
	}
	if !strings.Contains(text(m), "\n  20 ") || strings.Contains(text(m), "\n  1 ") {
		t.Fatalf("newest first and cut by the window:\n%s", text(m))
	}
	r := rect(t, m, "laps")
	m, _ = update(m, tui.MouseEvent{X: r.X + 2, Y: r.Y, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.lapAt != 1 || strings.Contains(text(m), "\n  20 ") {
		t.Fatalf("lapAt %d", m.lapAt)
	}
	m, _ = key(m, tui.KeyUp)
	if m.lapAt != 0 {
		t.Fatalf("lapAt %d", m.lapAt)
	}
}

func TestTabsByKeyTabAndClick(t *testing.T) {
	m, _ := ready()
	m, _ = runes(m, "2")
	if m.tab != tTimer || !strings.Contains(text(m), "▸2 Timer") {
		t.Fatalf("tab %v", m.tab)
	}
	m, _ = key(m, tui.KeyTab)
	if m.tab != tClock {
		t.Fatalf("tab %v", m.tab)
	}
	m, _ = key(m, tui.KeyTab)
	if m.tab != tStopwatch {
		t.Fatalf("Tab wraps: %v", m.tab)
	}
	m, _ = update(m, tui.Key{Type: tui.KeyTab, Mod: 1})
	if m.tab != tClock {
		t.Fatalf("Shift+Tab: %v", m.tab)
	}
	m, _ = click(t, m, "tab-1")
	if m.tab != tTimer {
		t.Fatalf("click: %v", m.tab)
	}
}

func fastTimer(m model, d time.Duration) model {
	m.timer.Interval = time.Millisecond
	m.timer.Duration = d
	return m
}

func TestTimerStartsOnTheTimerTab(t *testing.T) {
	m, _ := ready()
	m, _ = runes(m, "2")
	if m.timer.View() != "00:05:00" || !strings.Contains(text(m), "ready") {
		t.Fatalf("%s\n%s", m.timer.View(), text(m))
	}
}

// The timer is a clockview, whose ticks need the Program to run them, so this
// one test runs a real session with a timer that takes milliseconds.
func TestTimerCountsDownAndFinishes(t *testing.T) {
	m, _ := ready()
	m.tab = tTimer
	m = fastTimer(m, 3*time.Millisecond)
	s := tuitest.New(m, 80, 24)
	defer s.Close()
	s.Keys("space")
	if !s.WaitForText("Time's up", 5*time.Second) {
		t.Fatalf("the timer never finished:\n%s", strings.Join(s.Screen(), "\n"))
	}
	s.Keys("space") // after the end, Space starts again from the top
	if !s.WaitForText("Space Pause", 5*time.Second) && !s.WaitForText("Time's up", 5*time.Second) {
		t.Fatal("restart after the end failed")
	}
}

func TestTimerPausesResumesAndResets(t *testing.T) {
	m, _ := ready()
	m, _ = runes(m, "2")
	m, cmd := key(m, tui.KeySpace)
	if cmd == nil || !strings.Contains(text(m), "running") || !strings.Contains(text(m), "Space Pause") {
		t.Fatalf("\n%s", text(m))
	}
	m, _ = key(m, tui.KeySpace)
	if m.timer.Running() || !strings.Contains(text(m), "paused") {
		t.Fatalf("\n%s", text(m))
	}
	m, _ = runes(m, "r")
	if m.started || !strings.Contains(text(m), "ready") || m.timer.Running() {
		t.Fatal("reset failed")
	}
}

func TestTimerLengthByKeysButtonsAndPresets(t *testing.T) {
	m, _ := ready()
	m, _ = runes(m, "2")
	m, _ = key(m, tui.KeyUp)
	if m.minutes != 6 || m.timer.Duration != 6*time.Minute || m.timer.View() != "00:06:00" {
		t.Fatalf("minutes %d", m.minutes)
	}
	m, _ = click(t, m, "minus")
	m, _ = click(t, m, "minus")
	if m.minutes != 4 {
		t.Fatalf("minutes %d", m.minutes)
	}
	m, _ = click(t, m, "plus")
	m, _ = click(t, m, "preset-25")
	if m.minutes != 25 || m.timer.View() != "00:25:00" {
		t.Fatalf("minutes %d", m.minutes)
	}
	m, _ = runes(m, "p")
	if m.minutes != 1 {
		t.Fatalf("p wraps to the first: %d", m.minutes)
	}
	m, _ = runes(m, "p")
	if m.minutes != 5 {
		t.Fatalf("p: %d", m.minutes)
	}
	for i := 0; i < 120; i++ {
		m, _ = key(m, tui.KeyUp)
	}
	if m.minutes != 99 {
		t.Fatalf("top %d", m.minutes)
	}
	for i := 0; i < 120; i++ {
		m, _ = key(m, tui.KeyDown)
	}
	if m.minutes != 1 {
		t.Fatalf("bottom %d", m.minutes)
	}
}

func TestChangingTheLengthStopsARunningTimer(t *testing.T) {
	m, _ := ready()
	m, _ = runes(m, "2")
	m, _ = key(m, tui.KeySpace)
	m, _ = key(m, tui.KeyUp)
	if m.timer.Running() || m.started {
		t.Fatal("the timer kept running on the old length")
	}
}

func TestClockTabShowsEachZoneFromTheSameInstant(t *testing.T) {
	m, c := ready()
	m, _ = runes(m, "3")
	out := text(m)
	for _, want := range []string{"Here", "12:00:00", "UTC", "New York", "07:00:00", "Tokyo", "21:00:00", "Sunday, 18 October 2026"} {
		if !strings.Contains(out, want) {
			t.Errorf("lacks %q:\n%s", want, out)
		}
	}
	c.advance(90 * time.Minute)
	if !strings.Contains(text(m), "08:30:00") {
		t.Fatalf("the clocks should follow the injected time:\n%s", text(m))
	}
}

func TestInitStartsTheClocksAndTheyKeepTicking(t *testing.T) {
	m, _ := ready()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("no Init command")
	}
	for _, z := range m.zones {
		if !z.clk.Running() {
			t.Fatalf("%s clock is not running", z.name)
		}
	}
}

func TestSmallWindowFallsBackToPlainDigitsAndKeepsTheFooter(t *testing.T) {
	m, _ := ready()
	m, _ = update(m, tui.ResizeMsg{Width: 30, Height: 10})
	lines := strings.Split(text(m), "\n")
	if len(lines) != 10 || !strings.Contains(lines[9], "[1-3]") || !strings.Contains(text(m), "00:00.00") {
		t.Fatalf("\n%s", text(m))
	}
}

func TestHelpAndQuit(t *testing.T) {
	m, _ := ready()
	m, _ = runes(m, "?")
	if !strings.Contains(text(m), "take a lap") {
		t.Fatalf("no help:\n%s", text(m))
	}
	if m, _ = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, c := key(m, tui.KeyCtrlC); c == nil {
		t.Fatal("Ctrl+C did not quit")
	}
	if _, c := runes(m, "q"); c == nil {
		t.Fatal("q did not quit")
	}
}
