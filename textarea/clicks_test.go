package textarea

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func clickyArea() Model {
	m := New()
	m.SetValue("alpha beta\nsecond line here")
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 0, Y: 0, W: 30, H: 5}
	m.Focus()
	return m
}

func areaClick(m Model, x, y, clicks int) Model {
	m, _ = m.Update(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress, Clicks: clicks})
	m, _ = m.Update(tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionRelease, Clicks: clicks})
	return m
}

func TestAreaDoubleClickSelectsTheWord(t *testing.T) {
	if got := areaClick(clickyArea(), 7, 0, 2).SelectedText(); got != "beta" {
		t.Fatalf("selected %q; want beta", got)
	}
	if got := areaClick(clickyArea(), 8, 1, 2).SelectedText(); got != "line" {
		t.Fatalf("second line: selected %q; want line", got)
	}
}

func TestAreaTripleClickSelectsTheLine(t *testing.T) {
	if got := areaClick(clickyArea(), 3, 1, 3).SelectedText(); got != "second line here" {
		t.Fatalf("selected %q", got)
	}
	if got := areaClick(clickyArea(), 3, 0, 3).SelectedText(); got != "alpha beta" {
		t.Fatalf("first line: selected %q", got)
	}
}

func TestAreaSingleClickAndEmptyLine(t *testing.T) {
	m := areaClick(clickyArea(), 7, 0, 1)
	if m.SelectedText() != "" || m.Cursor() != 7 {
		t.Fatalf("single click: %q at %d", m.SelectedText(), m.Cursor())
	}
	e := New()
	e.SetValue("a\n\nb")
	e.Mouse = true
	e.Bounds = hittest.Rect{W: 10, H: 5}
	e.Focus()
	if got := areaClick(e, 0, 1, 2).SelectedText(); got != "" {
		t.Fatalf("a double click on an empty line selected %q", got)
	}
}
