package focus_test

import (
	"reflect"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/hittest"
)

func TestRouteAutoTabWrapsThreeWidgets(t *testing.T) {
	_, fs, _ := setup(3)
	r := focus.New(3)
	var seen []int
	for i := 0; i < 3; i++ {
		r, _ = r.RouteAuto(tab, hittest.Map[int]{}, fs...)
		seen = append(seen, r.Current()+1)
	}
	if !reflect.DeepEqual(seen, []int{2, 3, 1}) {
		t.Errorf("Tab from 1 visits %v, want [2 3 1]", seen)
	}
	r, _ = r.RouteAuto(shiftTab, hittest.Map[int]{}, fs...)
	if r.Current() != 2 {
		t.Errorf("Shift+Tab wrapped to %d, want 2", r.Current())
	}
}

func TestRouteAutoConsumedTabKeepsFocus(t *testing.T) {
	ws, fs, log := setup(3)
	fs[0].Consumes = func(m tui.Msg) bool { return true }
	r, cmd := focus.New(3).RouteAuto(tab, hittest.Map[int]{}, fs...)
	if r.Current() != 0 {
		t.Errorf("focus moved to %d", r.Current())
	}
	if got := msgOf(t, cmd); got != updatedMsg(ws[0].name) {
		t.Errorf("msg = %v", got)
	}
	if !reflect.DeepEqual(*log, []string{"update a"}) {
		t.Errorf("log = %v", *log)
	}
}

func TestRouteAutoClickFocuses(t *testing.T) {
	_, fs, log := setup(3)
	zones := hittest.Map[int]{}.Add(0, hittest.Rect{X: 0, Y: 0, W: 10, H: 1}).Add(2, hittest.Rect{X: 0, Y: 2, W: 10, H: 1})
	click := tui.MouseEvent{X: 3, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	r, cmd := focus.New(3).RouteAuto(click, zones, fs...)
	if r.Current() != 2 {
		t.Fatalf("current = %d, want 2", r.Current())
	}
	if cmd == nil {
		t.Fatal("no Cmd")
	}
	if !reflect.DeepEqual(*log, []string{"blur a", "focus c", "update c"}) {
		t.Errorf("log = %v", *log)
	}
	// A click outside every region, or with another button, does not move focus.
	miss := tui.MouseEvent{X: 50, Y: 50, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
	if r2, _ := r.RouteAuto(miss, zones, fs...); r2.Current() != 2 {
		t.Error("miss moved focus")
	}
	right := tui.MouseEvent{X: 3, Y: 0, Button: tui.MouseButtonRight, Action: tui.MouseActionPress}
	if r2, _ := r.RouteAuto(right, zones, fs...); r2.Current() != 2 {
		t.Error("right click moved focus")
	}
}
