package main

import (
	"testing"

	"github.com/ows4444/tui"
)

func click(x, y int) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}
}

func update(m model, msg tui.Msg) model {
	next, _ := m.Update(msg)
	return next.(model)
}

func TestClickingATabSelectsIt(t *testing.T) {
	m := initialModel()
	if m.tabs.Active() != 0 {
		t.Fatal("starts on the first tab")
	}
	// Click the middle of the second tab's label.
	x := 2 + len(tabLabels[0]) + 3 + 3
	m = update(m, click(x, 2))
	if m.tabs.Active() != 1 {
		t.Errorf("active = %d after clicking the Advanced tab", m.tabs.Active())
	}
	m = update(m, click(4, 2)) // back to General
	if m.tabs.Active() != 0 {
		t.Errorf("active = %d after clicking General", m.tabs.Active())
	}
}

func TestOtherClicksDoNothing(t *testing.T) {
	m := update(initialModel(), click(2+len(tabLabels[0])+3+3, 2)) // select Advanced
	for _, ev := range []tui.MouseEvent{
		click(3, 5),  // body
		click(0, 0),  // border
		click(60, 2), // right of the box
		{X: 4, Y: 2, Button: tui.MouseButtonRight, Action: tui.MouseActionPress},
		{X: 4, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease},
	} {
		if got := update(m, ev); got.tabs.Active() != 1 {
			t.Errorf("%+v changed the active tab to %d", ev, got.tabs.Active())
		}
	}
}

func TestKeyboardTabSwitchingStillWorks(t *testing.T) {
	m := update(initialModel(), tui.Key{Type: tui.KeyRight})
	if m.tabs.Active() != 1 {
		t.Errorf("right arrow: active = %d", m.tabs.Active())
	}
}
