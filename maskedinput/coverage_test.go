package maskedinput

import (
	"strings"
	"testing"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
)

func TestLayoutNodeMeasure(t *testing.T) {
	loose := layout.Constraints{MaxW: 100, MaxH: 10}
	m := New()
	m.Prompt = "Card: "
	m.Placeholder = "number"

	// Empty value measures the placeholder: prompt + placeholder + cursor cell.
	want := ansi.Width(m.Prompt) + ansi.Width(m.Placeholder) + 1
	if got := m.LayoutNode().Measure(loose); got != (layout.Size{W: want, H: 1}) {
		t.Errorf("empty Measure = %v, want %dx1", got, want)
	}

	// A typed value measures its rune count, not the placeholder.
	m.SetValue("12345678901")
	want = ansi.Width(m.Prompt) + 11 + 1
	if got := m.LayoutNode().Measure(loose); got != (layout.Size{W: want, H: 1}) {
		t.Errorf("value Measure = %v, want %dx1", got, want)
	}

	// Constraints clamp the result.
	if got := m.LayoutNode().Measure(layout.Constraints{MaxW: 5, MaxH: 10}); got.W != 5 {
		t.Errorf("clamped Measure width = %d, want 5", got.W)
	}
}

func TestLayoutNodeRenderEmptySize(t *testing.T) {
	n := New().LayoutNode()
	for _, s := range []layout.Size{{W: 0, H: 1}, {W: 10, H: 0}, {W: -1, H: -1}} {
		if got := n.Render(s); got != "" {
			t.Errorf("Render(%v) = %q, want empty", s, got)
		}
	}
}

func TestViewEmptyShowsPlaceholderUnmasked(t *testing.T) {
	m := New()
	m.Placeholder = "secret here"
	if got := ansi.StripANSI(m.View()); !strings.Contains(got, "secret here") {
		t.Errorf("View() = %q, want the placeholder", got)
	}
}

func TestLinearizeBranches(t *testing.T) {
	m := New()
	if got := m.Linearize(); got != "Masked field, empty" {
		t.Errorf("no prompt, empty: %q", got)
	}
	m.SetValue("x")
	if got := m.Linearize(); got != "Masked field, 1 character entered" {
		t.Errorf("one char: %q", got)
	}
	m.Prompt = "PIN: "
	m.Focus()
	if got := m.Linearize(); got != "PIN, masked field, focused, 1 character entered" {
		t.Errorf("focused: %q", got)
	}
}
