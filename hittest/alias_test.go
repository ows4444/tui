package hittest_test

import (
	"testing"

	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
)

// TestLayoutRectIsHitRect proves criterion #43: a layout.Rect is accepted
// where a hittest.Rect is expected, with no conversion, and the two share methods.
func TestLayoutRectIsHitRect(t *testing.T) {
	var lr layout.Rect = layout.Rect{X: 2, Y: 1, W: 6, H: 3}
	var hr hittest.Rect = lr
	m := hittest.Map[string]{}.Add("box", lr)
	if h, ok := m.At(3, 2); !ok || h.ID != "box" || h.LX != 1 || h.LY != 1 {
		t.Fatalf("At(3,2) = %+v, %v", h, ok)
	}
	top, rest := lr.CutTop(1)
	if top != (layout.Rect{X: 2, Y: 1, W: 6, H: 1}) || rest != (hittest.Rect{X: 2, Y: 2, W: 6, H: 2}) {
		t.Fatalf("CutTop = %v, %v", top, rest)
	}
	if lx, ly := hr.Local(4, 3); lx != 2 || ly != 2 {
		t.Fatalf("Local = %d,%d", lx, ly)
	}
}
