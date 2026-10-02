package viewport

import (
	"fmt"
	"strings"
	"testing"
)

func filled(n, w, h int) Model {
	m := New(w, h)
	for i := 0; i < n; i++ {
		m.AppendLine(fmt.Sprintf("l%d", i))
	}
	return m
}

// Criteria #88, #90: TrimFront drops the oldest lines and shifts the offset.
func TestTrimFrontDropsOldestAndShiftsOffset(t *testing.T) {
	m := filled(10, 5, 2)
	m.GotoBottom()
	m.LineUp(3) // offset 5
	if n := m.TrimFront(6); n != 4 {
		t.Fatalf("dropped %d, want 4", n)
	}
	if m.LineCount() != 6 {
		t.Errorf("LineCount = %d, want 6", m.LineCount())
	}
	if got, want := m.View(), "l5\nl6"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
	if n := m.TrimFront(100); n != 0 || m.TrimFront(-1) != 0 {
		t.Error("trimming to a larger or negative max must drop nothing")
	}
	// Offset clamps at 0 when more is dropped than the offset.
	m.TrimFront(2)
	if !m.AtTop() || m.View() != "l8\nl9" {
		t.Errorf("view = %q, want top of l8,l9", m.View())
	}
}

// Criterion #89: a bottom-pinned view stays at the bottom through a trim.
func TestTrimFrontKeepsBottomPinned(t *testing.T) {
	m := filled(10, 5, 3)
	m.GotoBottom()
	m.TrimFront(4)
	if !m.AtBottom() || m.View() != "l7\nl8\nl9" {
		t.Errorf("view = %q, atBottom=%v", m.View(), m.AtBottom())
	}
}

// Criterion #91: repeated trimming stays correct across compaction and does
// not retain the dropped lines' storage unboundedly.
func TestTrimFrontAmortisedCompaction(t *testing.T) {
	m := New(5, 2)
	for i := 0; i < 1000; i++ {
		m.AppendLine(fmt.Sprintf("l%d", i))
		m.TrimFront(4)
		if m.LineCount() > 4 {
			t.Fatalf("LineCount = %d", m.LineCount())
		}
		if len(m.lines) > 3*4+2 {
			t.Fatalf("backing slice grew to %d for 4 live lines", len(m.lines))
		}
	}
	m.GotoBottom()
	if got, want := m.View(), "l998\nl999"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

// Criterion #92: without TrimFront, behaviour is unchanged (SetContent resets
// any trimmed state).
func TestSetContentAfterTrimFront(t *testing.T) {
	m := filled(10, 5, 2)
	m.TrimFront(3)
	m.SetContent(strings.Join([]string{"a", "b", "c"}, "\n"))
	if m.LineCount() != 3 || m.View() != "a\nb" {
		t.Errorf("LineCount=%d view=%q", m.LineCount(), m.View())
	}
}

// After compaction the widest-line cache reflects the remaining lines.
func TestTrimFrontRecomputesWidthOnCompaction(t *testing.T) {
	m := New(3, 2)
	m.AppendLine("very wide line")
	m.AppendLine("a")
	m.AppendLine("b")
	m.TrimFront(1) // head=2 > live=1: compacts
	m.LineRight(100)
	if m.xOffset != 0 {
		t.Errorf("xOffset = %d, want 0 once the wide line is gone", m.xOffset)
	}
}
