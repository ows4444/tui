package spinner

import (
	"context"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func TestNewIsNotRunning(t *testing.T) {
	m := New()
	if m.Running() {
		t.Error("New() should not start running")
	}
}

func TestStartReturnsCmdAndSetsRunning(t *testing.T) {
	m := New()
	cmd := m.Start()
	if !m.Running() {
		t.Error("Start() should set Running() true")
	}
	if cmd == nil {
		t.Fatal("Start() should return a non-nil Cmd")
	}
	if _, ok := tui.RunCmd(context.Background(), cmd).(tickMsg); !ok {
		t.Errorf("Start()'s Cmd produced %T, want tickMsg", tui.RunCmd(context.Background(), cmd))
	}
}

func TestTickAdvancesFrameAndReschedules(t *testing.T) {
	m := New()
	m.Start()

	next, cmd := m.Update(tickMsg{})
	if next.frame != 1 {
		t.Errorf("frame after one tick = %d, want 1", next.frame)
	}
	if cmd == nil {
		t.Fatal("a tick while running should reschedule (non-nil Cmd)")
	}
}

func TestFrameWrapsAround(t *testing.T) {
	m := New()
	m.Frames = []string{"a", "b", "c"}
	m.Start()

	for i := 0; i < 3; i++ {
		next, _ := m.Update(tickMsg{})
		m = next
	}
	if m.frame != 0 {
		t.Errorf("frame after len(Frames) ticks = %d, want 0 (wrapped)", m.frame)
	}
}

func TestTickIsNoOpAfterStop(t *testing.T) {
	m := New()
	m.Start()
	m.Stop()

	next, cmd := m.Update(tickMsg{})
	if cmd != nil {
		t.Error("a tick after Stop should not reschedule (nil Cmd) — that's what ends the animation")
	}
	if next.frame != 0 {
		t.Error("a tick after Stop should not advance the frame")
	}
}

func TestNonTickMsgIgnored(t *testing.T) {
	m := New()
	m.Start()
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil || next.frame != m.frame {
		t.Error("a non-tick Msg should be a no-op")
	}
}

func TestView(t *testing.T) {
	m := New()
	m.Theme = theme.DarkTheme()
	got := m.View()
	want := ansi.NewStyle().Foreground(theme.DarkTheme().Primary).Render(Frames()[0])
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestViewWithLabel(t *testing.T) {
	m := New()
	m.Label = "Loading..."
	got := m.View()
	frame := ansi.NewStyle().Foreground(m.Theme.Primary).Render(m.Frames[0])
	want := frame + " " + "Loading..."
	if got != want {
		t.Errorf("View() = %q, want %q", got, want)
	}
}

func TestEmptyFramesDoesNotPanic(t *testing.T) {
	m := New()
	m.Frames = nil
	m.Label = "still loading"
	if got := m.View(); got != "still loading" {
		t.Errorf("View() with no Frames = %q, want just the Label", got)
	}
	m.Start()
	next, cmd := m.Update(tickMsg{})
	if cmd != nil || next.frame != 0 {
		t.Error("ticking with no Frames should be a safe no-op")
	}
}

func TestFrameStaysInBoundsIfFramesShrinks(t *testing.T) {
	m := New() // 10 default frames
	m.Start()
	for i := 0; i < 7; i++ {
		next, _ := m.Update(tickMsg{})
		m = next
	}
	m.Frames = []string{"x"} // shrink after frame has advanced past index 0
	if got := m.View(); got == "" {
		t.Fatal("View() should not panic when Frames shrinks below the current frame index")
	}
}
