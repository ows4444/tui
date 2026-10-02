package logview

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Criterion #87: lines are stored once, in the viewport; Model holds no
// line slice of its own.
func TestLinesStoredOnlyInTheViewport(t *testing.T) {
	rt := reflect.TypeOf(Model{})
	for i := 0; i < rt.NumField(); i++ {
		if rt.Field(i).Type.Kind() == reflect.Slice {
			t.Errorf("field %s is a slice: lines must live only in the viewport", rt.Field(i).Name)
		}
	}
	m := New(10, 2)
	m.Append("a")
	m.Append("b")
	if got := m.Viewport.LineCount(); got != 2 {
		t.Errorf("viewport holds %d lines, want 2", got)
	}
}

// Criterion #88: with Max set, an append past it drops the oldest lines.
func TestMaxDropsOldestLines(t *testing.T) {
	m := New(10, 3)
	m.Max = 3
	for i := 0; i < 10; i++ {
		m.Append(fmt.Sprintf("l%d", i))
		if got := m.Viewport.LineCount(); got > 3 {
			t.Fatalf("after append %d: %d lines, want at most 3", i, got)
		}
	}
	if got, want := m.View(), "l7\nl8\nl9"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

// Criterion #89: a view pinned to the bottom stays pinned as lines drop.
func TestMaxKeepsPinnedToBottom(t *testing.T) {
	m := New(10, 2)
	m.Max = 5
	for i := 0; i < 50; i++ {
		m.Append(fmt.Sprintf("l%d", i))
		if !m.Viewport.AtBottom() {
			t.Fatalf("drifted off the bottom after append %d", i)
		}
	}
	if got, want := m.View(), "l48\nl49"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

// Criterion #90: dropping above a scrolled-back view keeps the same visible
// content, moving the offset by the dropped count, clamped at 0.
func TestMaxKeepsScrolledBackContentInPlace(t *testing.T) {
	m := New(10, 2)
	m.Max = 10
	for i := 0; i < 10; i++ {
		m.Append(fmt.Sprintf("l%d", i))
	}
	m.Viewport.LineUp(3) // showing l5,l6
	if got, want := m.View(), "l5\nl6"; got != want {
		t.Fatalf("setup view = %q, want %q", got, want)
	}
	m.Append("l10") // drops l0
	m.Append("l11") // drops l1
	if got, want := m.View(), "l5\nl6"; got != want {
		t.Errorf("view after drops = %q, want %q", got, want)
	}
	// Scrolled to the top: the content scrolls out from under the reader, offset clamps at 0.
	m.Viewport.GotoTop()
	m.Append("l12")
	if !m.Viewport.AtTop() || m.View() != "l3\nl4" {
		t.Errorf("at top after drop: view = %q", m.View())
	}
	// Reader further back than the dropped count is pushed to the top.
	m.Viewport.LineDown(1) // l4,l5
	m.Append("l13")        // drops l3 -> l4,l5 still
	m.Append("l14")        // drops l4 -> l5,l6? offset 0 clamps
	if got, want := m.View(), "l5\nl6"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

// Criterion #91: with a cap, n appends cost O(n) in total; no O(n) work per
// append once the cap is reached. Count-based via a large cap, time-bounded
// generously like TestAppendScalesLinearly.
func TestMaxAppendScalesLinearly(t *testing.T) {
	const n = 200000
	m := New(120, 40)
	m.Max = 50000
	start := time.Now()
	for i := 0; i < n; i++ {
		m.Append(fmt.Sprintf("2026-09-29T12:00:00Z INFO request %d handled", i))
	}
	if el := time.Since(start); el > 3*time.Second {
		t.Fatalf("%d capped appends took %v, want well under 3s (O(n) per append?)", n, el)
	}
	if got := m.Viewport.LineCount(); got != m.Max {
		t.Errorf("LineCount = %d, want %d", got, m.Max)
	}
	if !strings.HasSuffix(m.View(), fmt.Sprintf("request %d handled", n-1)) {
		t.Errorf("last line missing from view")
	}
}

// Criterion #92: with no cap set, nothing is dropped.
func TestNoMaxKeepsEverything(t *testing.T) {
	m := New(10, 2)
	for i := 0; i < 1000; i++ {
		m.Append(fmt.Sprintf("l%d", i))
	}
	if got := m.Viewport.LineCount(); got != 1000 {
		t.Errorf("LineCount = %d, want 1000", got)
	}
	m.Viewport.GotoTop()
	if got, want := m.View(), "l0\nl1"; got != want {
		t.Errorf("top view = %q, want %q", got, want)
	}
}
