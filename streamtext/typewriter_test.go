package streamtext

import (
	"context"
	"github.com/ows4444/tui"
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// step delivers one tick to m and returns the updated model and whether a
// follow-up tick was scheduled.
func step(m Model) (Model, bool) {
	m, cmd := m.Update(tickFor(m))
	return m, cmd != nil
}

func TestNewTypewriterPresets(t *testing.T) {
	tw, base := NewTypewriter(), New()
	if tw.Running() || tw.Text() != "" {
		t.Error("NewTypewriter should be idle and empty")
	}
	if tw.CharsPerTick != 1 {
		t.Errorf("CharsPerTick = %d, want 1", tw.CharsPerTick)
	}
	if tw.Interval <= base.Interval {
		t.Errorf("Interval %v should be slower than New()'s %v", tw.Interval, base.Interval)
	}
	if !tw.CursorWhenDone || tw.BlinkInterval <= 0 {
		t.Errorf("want persistent blinking cursor, got CursorWhenDone=%v BlinkInterval=%v", tw.CursorWhenDone, tw.BlinkInterval)
	}
	if tw.Cursor == "" || tw.Cursor == base.Cursor {
		t.Errorf("Cursor = %q, want a distinct non-empty cursor (New() uses %q)", tw.Cursor, base.Cursor)
	}
	if tw.PauseAfter == "" || tw.PauseTicks <= 0 {
		t.Error("want pauses after sentence-ending characters")
	}
	if !strings.ContainsAny(tw.PauseAfter, ".!?") {
		t.Errorf("PauseAfter %q should include sentence enders", tw.PauseAfter)
	}
	if tw.id == base.id || tw.id == NewTypewriter().id {
		t.Error("each model needs its own id")
	}
}

func TestPauseAfterSentenceEnder(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.CharsPerTick = 1
	m.PauseAfter = "."
	m.PauseTicks = 3
	m.SetText("ab.cd")
	m.Start()

	var views []string
	for i := 0; i < 9; i++ {
		m, _ = step(m)
		views = append(views, m.View())
	}
	want := []string{
		"a", "ab", "ab.", // '.' revealed, pause begins
		"ab.", "ab.", "ab.", // three ticks reveal nothing
		"ab.c", "ab.cd", "ab.cd",
	}
	for i := range want {
		if views[i] != want[i] {
			t.Errorf("tick %d: View = %q, want %q (all: %q)", i+1, views[i], want[i], views)
			break
		}
	}
}

func TestNoPauseAfterFinalCharacter(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.CharsPerTick = 1
	m.PauseAfter = "."
	m.PauseTicks = 5
	m.SetText("a.")
	m.Start()
	m, _ = step(m)
	m, more := step(m) // reveals '.', which is last
	if !m.Done() {
		t.Fatal("expected done")
	}
	if more {
		t.Error("no pause and no further ticks should follow the final character")
	}
}

// A pause charged to the final character would leak into later text.
func TestNoLeftoverPauseForAppendedText(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.CharsPerTick = 1
	m.PauseAfter = "."
	m.PauseTicks = 5
	m.SetText("a.")
	m.Start()
	m, _ = step(m)
	m, _ = step(m) // '.' revealed, done
	if cmd := m.Append("b"); cmd == nil {
		t.Fatal("Append after done should restart ticking")
	}
	m, _ = step(m)
	if got := m.View(); got != "a.b" {
		t.Errorf("View = %q, want %q on the very next tick (no leftover pause)", got, "a.b")
	}
}

func TestNoPausesWhenDisabled(t *testing.T) {
	for name, cfg := range map[string]func(*Model){
		"zero ticks": func(m *Model) { m.PauseAfter = "."; m.PauseTicks = 0 },
		"empty set":  func(m *Model) { m.PauseAfter = ""; m.PauseTicks = 4 },
	} {
		m := New()
		m.Cursor = ""
		m.CharsPerTick = 1
		cfg(&m)
		m.SetText("a.b")
		m.Start()
		for i := 0; i < 3; i++ {
			m, _ = step(m)
		}
		if m.View() != "a.b" {
			t.Errorf("%s: View = %q after 3 ticks, want full text (no pauses)", name, m.View())
		}
	}
}

func TestMultiCharTickStopsAtPauseCharacter(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.CharsPerTick = 5
	m.PauseAfter = "!"
	m.PauseTicks = 1
	m.SetText("ab!cdefgh")
	m.Start()
	m, _ = step(m)
	if got := m.View(); got != "ab!" {
		t.Errorf("first tick View = %q, want reveal to stop at the pause char: %q", got, "ab!")
	}
	m, _ = step(m) // pause tick
	if got := m.View(); got != "ab!" {
		t.Errorf("pause tick View = %q, want unchanged", got)
	}
	m, _ = step(m)
	if got := m.View(); got != "ab!cdefg" {
		t.Errorf("after pause View = %q, want 5 more columns", got)
	}
}

func TestPauseCharacterInsideStyledAndWideText(t *testing.T) {
	m := New()
	m.Cursor = ""
	m.CharsPerTick = 1
	m.PauseAfter = "."
	m.PauseTicks = 2
	m.SetText("你." + ansi.NewStyle().Bold().Render("b"))
	m.Start()
	// Columns: 你 (2), '.', b -> pause must land after the '.' at column 3.
	m, _ = step(m)
	m, _ = step(m)
	m, _ = step(m)
	if got := ansi.StripANSI(m.View()); got != "你." {
		t.Fatalf("View = %q, want %q", got, "你.")
	}
	m, _ = step(m)
	m, _ = step(m)
	if got := ansi.StripANSI(m.View()); got != "你." {
		t.Errorf("View during pause = %q, want unchanged", got)
	}
	m, _ = step(m)
	if got := ansi.StripANSI(m.View()); got != "你.b" {
		t.Errorf("View after pause = %q", got)
	}
}

func TestCursorRemainsWhenDoneOnlyIfAsked(t *testing.T) {
	th := theme.DarkTheme()
	cursor := ansi.NewStyle().Foreground(th.Primary).Render("▌")

	m := New()
	m.Theme = th
	m.SetText("hi")
	m.Skip()
	if got := m.View(); got != "hi" {
		t.Errorf("default: done View = %q, want no cursor", got)
	}
	m.CursorWhenDone = true
	if got := m.View(); got != "hi"+cursor {
		t.Errorf("CursorWhenDone: done View = %q, want cursor kept", got)
	}
}

func TestBlinkTogglesWhenDone(t *testing.T) {
	m := NewTypewriter()
	m.Theme = theme.DarkTheme()
	m.PauseAfter = ""
	m.SetText("hi")
	m.Start()
	for !m.Done() {
		m, _ = step(m)
	}
	shown := m.View()
	cursor := ansi.NewStyle().Foreground(theme.DarkTheme().Primary).Render(m.Cursor)
	if shown != "hi"+cursor {
		t.Fatalf("done View = %q, want text plus solid cursor", shown)
	}

	m2, cmd := m.Update(tickFor(m))
	if cmd == nil {
		t.Fatal("a blink tick should reschedule while running")
	}
	if got := m2.View(); got != "hi" {
		t.Errorf("after 1 blink tick View = %q, want cursor hidden", got)
	}
	m3, cmd := m2.Update(tickFor(m2))
	if cmd == nil || m3.View() != "hi"+cursor {
		t.Errorf("after 2 blink ticks View = %q (cmd nil=%v), want cursor back", m3.View(), cmd == nil)
	}
}

func TestBlinkReschedulesAtBlinkInterval(t *testing.T) {
	m := NewTypewriter()
	m.PauseAfter = ""
	m.Interval = time.Millisecond
	m.BlinkInterval = 60 * time.Millisecond
	m.SetText("a")
	m.Start()
	m, _ = step(m) // reveals "a": done

	start := time.Now()
	_, cmd := m.Update(tickFor(m))
	if cmd == nil {
		t.Fatal("blink tick should reschedule")
	}
	tui.RunCmd(context.Background(), cmd)
	if d := time.Since(start); d < 50*time.Millisecond {
		t.Errorf("blink tick fired after %v, want ~BlinkInterval (60ms)", d)
	}
}

func TestCursorAlwaysVisibleWhileRevealing(t *testing.T) {
	m := NewTypewriter()
	m.PauseAfter = ""
	m.SetText("abcdef")
	m.Start()
	for i := 0; !m.Done(); i++ {
		if !strings.Contains(m.View(), m.Cursor) {
			t.Fatalf("tick %d: cursor hidden while revealing: %q", i, m.View())
		}
		m, _ = step(m)
	}
}

func TestNoBlinkWithoutBlinkInterval(t *testing.T) {
	m := New()
	m.CursorWhenDone = true
	m.BlinkInterval = 0
	m.CharsPerTick = 10
	m.SetText("hi")
	m.Start()
	m, more := step(m)
	if !m.Done() || more {
		t.Fatalf("expected done with no further ticks; done=%v more=%v", m.Done(), more)
	}
	// Even a stray tick doesn't blink or reschedule.
	m2, more := step(m)
	if more || m2.View() != m.View() {
		t.Errorf("stray tick: more=%v View changed=%v", more, m2.View() != m.View())
	}
}

func TestSetTextAndAppendAfterDoneResumeSolid(t *testing.T) {
	m := NewTypewriter()
	m.PauseAfter = ""
	m.Theme = theme.DarkTheme()
	m.SetText("a")
	m.Start()
	m, _ = step(m) // done
	m, _ = step(m) // blink off
	if strings.Contains(m.View(), m.Cursor) {
		t.Fatal("setup: cursor should be blinked off")
	}

	// Append while the blink chain's tick is in flight: no second chain.
	if cmd := m.Append("b"); cmd != nil {
		t.Error("Append during the blink chain must not start a second chain")
	}
	if !strings.Contains(m.View(), m.Cursor) {
		t.Error("cursor should be solid again as soon as there is new text")
	}
	m, more := step(m) // the in-flight tick now reveals
	if got := ansi.StripANSI(m.View()); !strings.HasPrefix(got, "ab") {
		t.Errorf("View = %q, want reveal to resume with %q", got, "ab")
	}
	if !more {
		t.Error("chain should continue (blink) after reveal completes")
	}
	if !strings.Contains(m.View(), m.Cursor) {
		t.Error("cursor should be solid when the resumed reveal completes, not left hidden by the earlier blink")
	}

	// SetText restarts the reveal from zero: only the cursor is visible.
	m.SetText("xyz")
	if got := ansi.StripANSI(m.View()); got != m.Cursor {
		t.Errorf("after SetText View = %q, want only the cursor %q", got, m.Cursor)
	}
}

func TestStopEndsBlinkChain(t *testing.T) {
	m := NewTypewriter()
	m.PauseAfter = ""
	m.SetText("a")
	m.Start()
	m, _ = step(m) // done, blink chain scheduled
	m.Stop()
	m2, more := step(m)
	if more {
		t.Error("a tick after Stop must not reschedule")
	}
	if m2.View() != m.View() {
		t.Error("a tick after Stop must not toggle the cursor")
	}
}
