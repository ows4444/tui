package appshell

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func click(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

func routerWithBounds(log *[]string) Router {
	return three(log).
		SetBounds("a", layout.Rect{X: 0, Y: 0, W: 10, H: 5}).
		SetBounds("b", layout.Rect{X: 10, Y: 0, W: 10, H: 5}).
		SetBounds("c", layout.Rect{X: 0, Y: 5, W: 20, H: 5})
}

func TestRouterClickFocusesAndLocalizes(t *testing.T) {
	var log []string
	r := routerWithBounds(&log)
	r, _ = r.Update(click(12, 3))
	if r.Focused() != "b" {
		t.Fatal(r.Focused())
	}
	want := fmt.Sprintf("b:%v", tui.MouseEvent{X: 2, Y: 3, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if len(log) != 1 || log[0] != want {
		t.Fatalf("log=%v want %s", log, want)
	}
}

func TestRouterClickOutsideDoesNothing(t *testing.T) {
	var log []string
	r := routerWithBounds(&log).Focus("c")
	r, _ = r.Update(click(50, 50))
	if r.Focused() != "c" || len(log) != 0 {
		t.Fatalf("%s %v", r.Focused(), log)
	}
}

func TestRouterWheelKeepsFocus(t *testing.T) {
	var log []string
	r := routerWithBounds(&log)
	r, _ = r.Update(tui.MouseEvent{X: 4, Y: 7, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if r.Focused() != "a" || len(log) != 1 || !strings.HasPrefix(log[0], "c:") {
		t.Fatalf("%s %v", r.Focused(), log)
	}
}

func TestRouterTopmostWins(t *testing.T) {
	var log []string
	r := three(&log).
		SetBounds("a", layout.Rect{X: 0, Y: 0, W: 10, H: 10}).
		SetBounds("b", layout.Rect{X: 2, Y: 2, W: 3, H: 3})
	r, _ = r.Update(click(3, 3))
	if r.Focused() != "b" {
		t.Fatal(r.Focused())
	}
}

func TestRouterSetLayoutUsesNamedRects(t *testing.T) {
	var log []string
	root := layout.Row(0,
		layout.Fill(layout.Named("a", layout.Text("a"))),
		layout.Fill(layout.Named("b", layout.Text("b"))),
	)
	r := three(&log).SetLayout(root, layout.Size{W: 20, H: 4})
	r, _ = r.Update(click(15, 1))
	if r.Focused() != "b" || len(log) != 1 {
		t.Fatalf("%s %v", r.Focused(), log)
	}
}
