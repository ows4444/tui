package focus_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/layout"
)

// twoColumns draws
//
//	name   city
//	email  submit
//
// as two columns, so reading order differs from column order.
func twoColumns() layout.Node {
	cell := func(name string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Named(name, layout.Block(name)), Basis: 1}
	}
	return layout.Row(2,
		layout.FlexChild{Node: layout.Column(0, cell("name"), cell("email")), Basis: 8},
		layout.FlexChild{Node: layout.Column(0, cell("city"), cell("submit")), Basis: 8},
	)
}

var size = layout.Size{W: 20, H: 2}

// Criterion #107: reading order, undrawn items last in index order.
func TestLayoutOrderIsReadingOrder(t *testing.T) {
	// Index order is column-major plus two names the layout never draws.
	names := []string{"name", "email", "missing", "city", "submit", "gone"}
	got := focus.LayoutOrder(twoColumns(), size, names...)
	want := []int{0, 3, 1, 4, 2, 5} // name, city, email, submit, then missing, gone
	if !slices.Equal(got, want) {
		t.Fatalf("LayoutOrder = %v, want %v", got, want)
	}
	if got := focus.LayoutOrder(twoColumns(), size); len(got) != 0 {
		t.Errorf("no names: %v, want empty", got)
	}
}

// Criterion #108: Next and Prev follow the order, wrap, and skip disabled items.
func TestWithOrderDrivesTraversal(t *testing.T) {
	r := focus.New(4).WithOrder([]int{0, 2, 1, 3})
	var seen []int
	for range 5 {
		seen = append(seen, r.Current())
		r = r.Next()
	}
	if want := []int{0, 2, 1, 3, 0}; !slices.Equal(seen, want) {
		t.Fatalf("Next visits %v, want %v", seen, want)
	}
	r = r.Set(0)
	if r.Prev().Current() != 3 {
		t.Errorf("Prev from the first wraps to %d, want 3", r.Prev().Current())
	}
	r = r.SetDisabled(2, true)
	if r.Next().Current() != 1 {
		t.Errorf("Next skips disabled 2: got %d, want 1", r.Next().Current())
	}
	tab, _ := r.Update(tui.Key{Type: tui.KeyTab})
	if tab.Current() != 1 {
		t.Errorf("Tab follows the order: got %d, want 1", tab.Current())
	}
	if r.WithOrder(nil).Next().Current() != 1 || !slices.Equal(r.WithOrder(nil).Order(), []int{0, 1, 2, 3}) {
		t.Error("a nil order does not restore index order")
	}
}

// Criterion #110: bad entries are ignored and omitted items come last.
func TestWithOrderNormalises(t *testing.T) {
	r := focus.New(5).WithOrder([]int{3, 3, -1, 9, 1})
	if got, want := r.Order(), []int{3, 1, 0, 2, 4}; !slices.Equal(got, want) {
		t.Fatalf("Order = %v, want %v", got, want)
	}
	if r.Current() != 0 {
		t.Errorf("WithOrder moved focus to %d", r.Current())
	}
	o := r.Order()
	o[0] = 4
	if r.Order()[0] != 3 {
		t.Error("Order returned the Ring's own slice")
	}
}

// Criterion #109: Zones IDs are item indices, and RouteAuto focuses on click.
func TestZonesFocusTheClickedItem(t *testing.T) {
	names := []string{"name", "email", "city", "submit"}
	zones := focus.Zones(twoColumns(), size, names...)
	if zones.Len() != 4 {
		t.Fatalf("%d zones, want 4", zones.Len())
	}
	if h, ok := zones.At(10, 1); !ok || names[h.ID] != "submit" {
		t.Fatalf("At(10,1) = %v %v, want submit", h, ok)
	}
	var updated []int
	fields := make([]focus.Field, len(names))
	for i := range fields {
		i := i
		fields[i] = focus.Field{Update: func(tui.Msg) tui.Cmd { updated = append(updated, i); return nil }}
	}
	r := focus.New(len(names))
	click := tui.MouseEvent{X: 10, Y: 0, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	r, _ = r.RouteAuto(click, zones, fields...)
	if names[r.Current()] != "city" || !slices.Equal(updated, []int{2}) {
		t.Fatalf("focused %s, updates %v; want city focused and given the click", names[r.Current()], updated)
	}
}

// A scope pushed over an ordered Ring starts in index order; Pop restores the order.
func TestOrderSurvivesPushAndPop(t *testing.T) {
	r := focus.New(3).WithOrder([]int{2, 1, 0})
	s := r.Push(2)
	if !slices.Equal(s.Order(), []int{0, 1}) {
		t.Errorf("pushed scope order %v, want index order", s.Order())
	}
	if !slices.Equal(s.Pop().Order(), []int{2, 1, 0}) {
		t.Errorf("Pop lost the order: %v", s.Pop().Order())
	}
}

// Tab order from the layout: fields are numbered in the order the model
// keeps them, and Tab visits them in the order they appear on screen.
func ExampleLayoutOrder() {
	cell := func(name string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Named(name, layout.Block(name)), Basis: 1}
	}
	form := layout.Row(2,
		layout.FlexChild{Node: layout.Column(0, cell("name"), cell("email")), Basis: 8},
		layout.FlexChild{Node: layout.Column(0, cell("city"), cell("submit")), Basis: 8},
	)
	names := []string{"name", "email", "city", "submit"}
	size := layout.Size{W: 20, H: 2}

	ring := focus.New(len(names)).WithOrder(focus.LayoutOrder(form, size, names...))
	for range names {
		fmt.Print(names[ring.Current()], " ")
		ring = ring.Next()
	}
	fmt.Println()
	// Output: name city email submit
}
