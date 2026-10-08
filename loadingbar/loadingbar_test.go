package loadingbar

import (
	"context"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func TestNewIsNotRunning(t *testing.T) {
	m := New(12)
	if m.Running() {
		t.Error("New() should not start running")
	}
}

func TestStartReturnsCmdAndSetsRunning(t *testing.T) {
	m := New(12)
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

func TestSegmentLength(t *testing.T) {
	tests := []struct {
		width int
		want  int
	}{
		{0, 0},
		{1, 1},
		{2, 1},
		{3, 1},
		{9, 3},
		{30, 10},
	}
	for _, tt := range tests {
		m := New(tt.width)
		if got := m.segmentLength(); got != tt.want {
			t.Errorf("New(%d).segmentLength() = %d, want %d", tt.width, got, tt.want)
		}
	}
}

func TestBouncesAtRightEdge(t *testing.T) {
	m := New(9) // segmentLength 3, maxPos 6
	m.Start()

	for i := 0; i < 6; i++ {
		next, _ := m.Update(tickFor(m))
		m = next
	}
	if m.pos != 6 {
		t.Fatalf("pos after 6 ticks = %d, want 6 (at the right edge)", m.pos)
	}
	if m.dir != -1 {
		t.Fatalf("dir at the right edge = %d, want -1 (should have reversed)", m.dir)
	}

	next, _ := m.Update(tickFor(m))
	m = next
	if m.pos != 5 {
		t.Errorf("pos one tick past the right edge = %d, want 5 (moving back left)", m.pos)
	}
}

func TestBouncesAtLeftEdge(t *testing.T) {
	m := New(9) // segmentLength 3, maxPos 6
	m.dir = -1  // start already heading left, as if returning from the right edge
	m.pos = 1
	m.Start()

	next, _ := m.Update(tickFor(m)) // 1 + (-1) = 0
	m = next
	if m.pos != 0 || m.dir != 1 {
		t.Fatalf("after reaching the left edge: pos=%d dir=%d, want pos=0 dir=1", m.pos, m.dir)
	}

	next, _ = m.Update(tickFor(m))
	m = next
	if m.pos != 1 {
		t.Errorf("pos one tick past the left edge = %d, want 1 (moving right again)", m.pos)
	}
}

func TestTickIsNoOpAfterStop(t *testing.T) {
	m := New(9)
	m.Start()
	m.Stop()

	next, cmd := m.Update(tickFor(m))
	if cmd != nil {
		t.Error("a tick after Stop should not reschedule (nil Cmd)")
	}
	if next.pos != 0 {
		t.Error("a tick after Stop should not move the segment")
	}
}

func TestNonTickMsgIgnored(t *testing.T) {
	m := New(9)
	m.Start()
	next, cmd := m.Update(tui.ResizeMsg{Width: 10, Height: 10})
	if cmd != nil || next.pos != m.pos {
		t.Error("a non-tick Msg should be a no-op")
	}
}

func TestViewWidthMatchesTrackWidth(t *testing.T) {
	m := New(20)
	if w := ansi.Width(m.View()); w != 20 {
		t.Errorf("Width(View()) = %d, want 20", w)
	}
	m.Start()
	for i := 0; i < 5; i++ {
		next, _ := m.Update(tickFor(m))
		m = next
		if w := ansi.Width(m.View()); w != 20 {
			t.Fatalf("Width(View()) after tick %d = %d, want 20", i, w)
		}
	}
}

func TestZeroWidthView(t *testing.T) {
	m := New(0)
	if got := m.View(); got != "" {
		t.Errorf("View() with width 0 = %q, want empty", got)
	}
}
