package logview

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func TestNextAndPrevMatchMoveAndLeaveFollow(t *testing.T) {
	m := New(10, 2)
	for i := 0; i < 30; i++ {
		m.Append(fmt.Sprintf("row %d hit", i))
	}
	m.SetFollow(true)
	m.Viewport.GotoTop() // moved by code; Follow stays on
	if n := m.Search("hit"); n != 30 {
		t.Fatalf("matches = %d, want 30", n)
	}
	if m.Viewport.CurrentMatch() != 1 || m.Follow {
		t.Fatalf("Search: current=%d follow=%v, want 1 and follow off (match is at the top)", m.Viewport.CurrentMatch(), m.Follow)
	}
	if !m.NextMatch() || m.Viewport.CurrentMatch() != 2 {
		t.Errorf("NextMatch: current = %d, want 2", m.Viewport.CurrentMatch())
	}
	if !m.PrevMatch() || m.Viewport.CurrentMatch() != 1 {
		t.Errorf("PrevMatch: current = %d, want 1", m.Viewport.CurrentMatch())
	}
	// Wrapping back to the last match shows the bottom: follow mode survives.
	m.Follow = true
	if !m.PrevMatch() || m.Viewport.CurrentMatch() != 30 || !m.Follow {
		t.Errorf("PrevMatch wrap: current=%d follow=%v, want 30 and follow kept", m.Viewport.CurrentMatch(), m.Follow)
	}
	// Wrapping forward to the first match leaves it.
	if !m.NextMatch() || m.Viewport.CurrentMatch() != 1 || m.Follow {
		t.Errorf("NextMatch wrap: current=%d follow=%v, want 1 and follow off", m.Viewport.CurrentMatch(), m.Follow)
	}
}

func TestNextMatchWithoutQueryIsFalse(t *testing.T) {
	m := New(10, 2)
	m.Append("a")
	if m.NextMatch() || m.PrevMatch() {
		t.Error("no query: NextMatch and PrevMatch must report false")
	}
}

func TestLayoutNodeRendersNothingForAnEmptySize(t *testing.T) {
	m := New(10, 2)
	m.Append("x")
	node := m.LayoutNode()
	for _, s := range []layout.Size{{W: 0, H: 3}, {W: 4, H: 0}, {W: -1, H: -1}} {
		if got := node.Render(s); got != "" {
			t.Errorf("Render(%+v) = %q, want empty", s, got)
		}
	}
	if got := ansi.StripANSI(node.Render(layout.Size{W: 5, H: 1})); !strings.Contains(got, "x") {
		t.Errorf("Render at a real size = %q", got)
	}
}
