package hittest

import (
	"fmt"
	"testing"

	"github.com/ows4444/tui"
)

func TestRectContainsEdges(t *testing.T) {
	r := Rect{X: 2, Y: 3, W: 4, H: 2} // columns 2-5, rows 3-4
	in := [][2]int{{2, 3}, {5, 3}, {2, 4}, {5, 4}}
	out := [][2]int{{1, 3}, {6, 3}, {2, 2}, {2, 5}, {5, 5}, {-1, -1}}
	for _, c := range in {
		if !r.Contains(c[0], c[1]) {
			t.Errorf("%v should be inside", c)
		}
	}
	for _, c := range out {
		if r.Contains(c[0], c[1]) {
			t.Errorf("%v should be outside", c)
		}
	}
	for _, e := range []Rect{{}, {X: 1, Y: 1, W: 0, H: 5}, {X: 1, Y: 1, W: 5, H: -1}} {
		if !e.Empty() || e.Contains(e.X, e.Y) {
			t.Errorf("%+v is empty and must contain nothing", e)
		}
	}
	if lx, ly := r.Local(4, 4); lx != 2 || ly != 1 {
		t.Errorf("Local = %d,%d", lx, ly)
	}
}

func TestCutsTileTheOriginalExactly(t *testing.T) {
	r := Rect{X: 5, Y: 2, W: 10, H: 6}
	cases := []struct {
		name string
		f    func(int) (Rect, Rect)
	}{{"top", r.CutTop}, {"bottom", r.CutBottom}, {"left", r.CutLeft}, {"right", r.CutRight}}
	for _, c := range cases {
		for _, n := range []int{-3, 0, 1, 4, 6, 10, 99} {
			a, b := c.f(n)
			// Every cell of r is in exactly one part, and no cell outside r is.
			for y := r.Y - 1; y < r.Y+r.H+1; y++ {
				for x := r.X - 1; x < r.X+r.W+1; x++ {
					got := 0
					if a.Contains(x, y) {
						got++
					}
					if b.Contains(x, y) {
						got++
					}
					want := 0
					if r.Contains(x, y) {
						want = 1
					}
					if got != want {
						t.Fatalf("%s(%d): cell (%d,%d) covered %d times, want %d (%+v | %+v)", c.name, n, x, y, got, want, a, b)
					}
				}
			}
		}
	}
	top, rest := r.CutTop(2)
	if top != (Rect{X: 5, Y: 2, W: 10, H: 2}) || rest != (Rect{X: 5, Y: 4, W: 10, H: 4}) {
		t.Errorf("CutTop(2) = %+v %+v", top, rest)
	}
	right, rest := r.CutRight(3)
	if right != (Rect{X: 12, Y: 2, W: 3, H: 6}) || rest != (Rect{X: 5, Y: 2, W: 7, H: 6}) {
		t.Errorf("CutRight(3) = %+v %+v", right, rest)
	}
}

func TestTopmostRegionWinsWithLocalCoordinates(t *testing.T) {
	screen := Rect{X: 0, Y: 0, W: 40, H: 10}
	header, body := screen.CutTop(1)
	dialog := Rect{X: 10, Y: 3, W: 20, H: 4}
	m := Map[string]{}.Add("header", header).Add("body", body).Add("dialog", dialog)

	h, ok := m.At(12, 4)
	if !ok || h.ID != "dialog" || h.LX != 2 || h.LY != 1 || h.Rect != dialog {
		t.Errorf("At(12,4) = %+v ok=%v, want dialog at local (2,1)", h, ok)
	}
	if h, _ := m.At(2, 4); h.ID != "body" {
		t.Errorf("outside the dialog should hit body, got %q", h.ID)
	}
	if h, _ := m.At(39, 0); h.ID != "header" {
		t.Errorf("row 0 should hit header, got %q", h.ID)
	}
	if _, ok := m.At(40, 5); ok {
		t.Error("beyond the screen should miss")
	}
	if m.Len() != 3 {
		t.Errorf("Len = %d", m.Len())
	}
}

func TestAtEventUsesMouseCoordinatesOnly(t *testing.T) {
	m := Map[int]{}.Add(7, Rect{X: 3, Y: 1, W: 5, H: 2})
	ev := tui.MouseEvent{X: 4, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	if h, ok := m.AtEvent(ev); !ok || h.ID != 7 || h.LX != 1 || h.LY != 1 {
		t.Errorf("AtEvent = %+v ok=%v", h, ok)
	}
	if _, ok := m.AtEvent(tui.MouseEvent{X: 0, Y: 0}); ok {
		t.Error("outside every region should report no hit")
	}
	if _, ok := (Map[int]{}).AtEvent(ev); ok {
		t.Error("the zero Map has no regions")
	}
}

func TestMapHasValueSemantics(t *testing.T) {
	base := Map[string]{}.Add("a", Rect{X: 0, Y: 0, W: 5, H: 5})
	one := base.Add("b", Rect{X: 0, Y: 0, W: 2, H: 2})
	two := base.Add("c", Rect{X: 0, Y: 0, W: 2, H: 2}) // a sibling of one, built from the same parent
	if base.Len() != 1 || one.Len() != 2 || two.Len() != 2 {
		t.Fatalf("lens = %d %d %d", base.Len(), one.Len(), two.Len())
	}
	if h, _ := one.At(0, 0); h.ID != "b" {
		t.Errorf("one hit %q, want b (a sibling must not overwrite it)", h.ID)
	}
	if h, _ := two.At(0, 0); h.ID != "c" {
		t.Errorf("two hit %q, want c", h.ID)
	}
	if h, _ := base.At(0, 0); h.ID != "a" {
		t.Errorf("base changed: %q", h.ID)
	}
}

func TestEmptyRectsAreNeverHit(t *testing.T) {
	m := Map[string]{}.Add("ghost", Rect{X: 1, Y: 1, W: 0, H: 3}).Add("real", Rect{X: 0, Y: 0, W: 2, H: 2})
	if h, ok := m.At(1, 1); !ok || h.ID != "real" {
		t.Errorf("empty region on top must not shadow: %+v", h)
	}
}

// A click at (12, 4) lands on the dialog, 2 columns and 1 row into it.
func Example() {
	m := Map[string]{}.
		Add("page", Rect{X: 0, Y: 0, W: 40, H: 10}).
		Add("dialog", Rect{X: 10, Y: 3, W: 20, H: 4})
	h, _ := m.At(12, 4)
	fmt.Println(h.ID, h.LX, h.LY)
	// Output: dialog 2 1
}
