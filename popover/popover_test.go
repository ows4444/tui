package popover

import (
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func key(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

// grid returns a w x h rectangle of '.' characters, one row per line, a
// fixed-size base to test Popover placement against.
func grid(w, h int) string {
	row := strings.Repeat(".", w)
	rows := make([]string, h)
	for i := range rows {
		rows[i] = row
	}
	return strings.Join(rows, "\n")
}

// TestPopoverLifecycleMirrorsDialog proves criterion #356: New returns an
// already-open Model, Show/Hide/Open work as in dialog.Model, and Update
// dismisses on Enter/Esc while open and no-ops when closed.
func TestPopoverLifecycleMirrorsDialog(t *testing.T) {
	m := New("content", 0, 0)
	if !m.Open() {
		t.Fatal("New() should return an already-open popover")
	}

	m.Hide()
	if m.Open() {
		t.Error("Hide() should close the popover")
	}
	m.Show()
	if !m.Open() {
		t.Error("Show() should open the popover")
	}

	next, cmd := m.Update(key(tui.KeyEnter))
	if next.Open() {
		t.Error("Enter should close the popover")
	}
	if cmd == nil {
		t.Fatal("Enter should return a non-nil Cmd")
	}
	if _, ok := cmd().(DismissedMsg); !ok {
		t.Errorf("Cmd produced %T, want DismissedMsg", cmd())
	}

	m2 := New("content", 0, 0)
	next2, cmd2 := m2.Update(key(tui.KeyEsc))
	if next2.Open() {
		t.Error("Esc should close the popover")
	}
	if cmd2 == nil {
		t.Fatal("Esc should return a non-nil Cmd")
	}

	m3 := New("content", 0, 0)
	next3, cmd3 := m3.Update(key(tui.KeyDown))
	if !next3.Open() {
		t.Error("an unrelated key should not close the popover")
	}
	if cmd3 != nil {
		t.Error("an unrelated key should not return a Cmd")
	}

	m4 := New("content", 0, 0)
	m4.Hide()
	next4, cmd4 := m4.Update(key(tui.KeyEnter))
	if next4.Open() {
		t.Error("Update on a closed popover should stay closed")
	}
	if cmd4 != nil {
		t.Error("Update on a closed popover should not return a Cmd, even for Enter/Esc")
	}
}

// TestRenderReturnsBaseUnchangedWhenClosed proves criterion #357: Render
// returns base unchanged while closed.
func TestRenderReturnsBaseUnchangedWhenClosed(t *testing.T) {
	base := "some\nbackground\ncontent"
	m := New("hi", 1, 1)
	m.Hide()
	if got := m.Render(base); got != base {
		t.Errorf("Render() with a closed popover = %q, want base unchanged", got)
	}
}

// TestRenderCompositesContentNearAnchor proves criterion #358: Render
// composites a bordered box holding arbitrary, possibly multi-line
// Content near a given anchor point (not centered over the whole base).
func TestRenderCompositesContentNearAnchor(t *testing.T) {
	base := grid(40, 20)
	m := New("line one\nline two", 5, 3)
	got := m.Render(base)

	visible := ansi.StripANSI(got)
	if !strings.Contains(visible, "line one") || !strings.Contains(visible, "line two") {
		t.Fatalf("Render() should contain multi-line Content: %q", visible)
	}

	box := layout.NewBox().Border(m.Theme.Border).BorderColor(m.Theme.BorderColor).PaddingAll(1).Render(m.Content)
	want := layout.Overlay(base, box, 5, 4)
	if got != want {
		t.Errorf("Render() did not composite the box anchored below-right of (5,3)")
	}

	// Not centered: the center of a 40x20 base would put a small box
	// well away from anchor (5,3).
	centerBase := layout.Overlay(base, box, 17, 8) // roughly center for a small box
	if got == centerBase {
		t.Error("Render() should not center the popover over base")
	}
}

// TestRenderShiftsLeftWhenClipped proves criterion #359: when default
// below-anchor placement would push the box's right edge past base's
// width, it's shifted left to stay within bounds.
func TestRenderShiftsLeftWhenClipped(t *testing.T) {
	content := "a fairly long popover content string"
	probe := New(content, 0, 0)
	box := layout.NewBox().Border(probe.Theme.Border).BorderColor(probe.Theme.BorderColor).PaddingAll(1).Render(content)
	bw, _ := extent(box)

	baseWidth := bw + 5
	base := grid(baseWidth, 20)

	anchorX, anchorY := baseWidth-2, 3
	m := New(content, anchorX, anchorY)
	got := m.Render(base)

	for _, line := range strings.Split(got, "\n") {
		if w := ansi.Width(line); w > baseWidth {
			t.Fatalf("overlaid line width = %d, want <= base width %d: %q", w, baseWidth, line)
		}
	}
	if bw >= baseWidth {
		t.Fatalf("test setup invalid: box width %d >= base width %d", bw, baseWidth)
	}
}

// TestRenderFlipsAboveWhenNoRoomBelow proves criterion #360: when placing
// the box below the anchor would push it past base's last line, it's
// placed above the anchor instead.
func TestRenderFlipsAboveWhenNoRoomBelow(t *testing.T) {
	baseHeight := 10
	base := grid(40, baseHeight)
	content := "hi"
	probe := New(content, 0, 0)
	box := layout.NewBox().Border(probe.Theme.Border).BorderColor(probe.Theme.BorderColor).PaddingAll(1).Render(content)
	_, bh := extent(box)

	anchorX, anchorY := 5, baseHeight-1
	m := New(content, anchorX, anchorY)
	got := m.Render(base)

	wantTop := anchorY - bh
	if wantTop < 0 {
		t.Fatalf("test setup invalid: box height %d too tall for anchorY %d", bh, anchorY)
	}
	want := layout.Overlay(base, box, anchorX, wantTop)
	if got != want {
		t.Errorf("Render() should place the box above the anchor when there's no room below")
	}

	belowPlacement := layout.Overlay(base, box, anchorX, anchorY+1)
	if got == belowPlacement {
		t.Errorf("popover placed below anchor despite no room")
	}
}
