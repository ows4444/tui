package skeleton

import (
	"context"
	"strings"
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

func TestViewRowsHaveExactWidth(t *testing.T) {
	m := New()
	m.Width = 12
	m.Lines = 4
	got := m.View()
	rows := strings.Split(got, "\n")
	if len(rows) != 4 {
		t.Fatalf("got %d rows, want 4", len(rows))
	}
	for i, row := range rows {
		if w := ansi.Width(row); w != 12 {
			t.Errorf("row %d ansi.Width = %d, want 12", i, w)
		}
	}
}

func TestViewStyledWithMuted(t *testing.T) {
	m := New()
	m.Width = 10
	m.Lines = 1
	m.Theme = theme.DarkTheme()
	got := m.View()
	mutedSeq := ansi.NewStyle().Foreground(theme.DarkTheme().Muted).Render(fillRune)
	if !strings.Contains(got, mutedSeq) {
		t.Errorf("View() = %q, want it to contain a Muted-styled fill rune %q", got, mutedSeq)
	}
	// The shimmer highlight must not be styled identically to plain Muted
	// fill — some part of the row uses a distinct style.
	plainMuted := ansi.NewStyle().Foreground(theme.DarkTheme().Muted).Render(strings.Repeat(fillRune, 10))
	if got == plainMuted {
		t.Error("View() should not be identical to a flat, unhighlighted Muted row")
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

func TestUpdateNoOpWhileNotRunning(t *testing.T) {
	m := New()
	next, cmd := m.Update(tickMsg{})
	if cmd != nil || next.frame != 0 {
		t.Error("a tick while not running should be a no-op")
	}
}

func TestViewAnimatesAcrossFrames(t *testing.T) {
	m := New()
	m.Width = 20
	m.Lines = 1
	m.Start()

	first := m.View()

	// Advance a few frames so the shimmer moves to a different column.
	for i := 0; i < 3; i++ {
		next, _ := m.Update(tickMsg{})
		m = next
	}
	second := m.View()

	if first == second {
		t.Error("View() at two different frame positions should differ (shimmer should move)")
	}
}

func TestViewEmptyWidthOrLinesDoesNotPanic(t *testing.T) {
	cases := []Model{
		{Width: 0, Lines: 5},
		{Width: -1, Lines: 5},
		{Width: 10, Lines: 0},
		{Width: 10, Lines: -1},
	}
	for _, m := range cases {
		got := m.View()
		if got != "" {
			t.Errorf("View() with Width=%d Lines=%d = %q, want empty", m.Width, m.Lines, got)
		}
	}
}
