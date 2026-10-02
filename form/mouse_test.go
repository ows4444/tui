package form

import (
	"testing"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func press(x, y int, b tui.MouseButton) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress}
}

func mouseForm() Model {
	m := New(
		Field{Name: "name", Label: "Name", Validators: []Validator{Required()}},
		Field{Name: "plan", Label: "Plan", Kind: FieldSelect, Options: []string{"Free", "Pro", "Team"}},
		Field{Name: "tos", Label: "Agree", Kind: FieldCheckbox},
	)
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 2, Y: 3, W: 40, H: 10}
	m.Focus()
	return m
}

func TestClickFocusesFieldAndTogglesCheckbox(t *testing.T) {
	m := mouseForm()
	m, _ = m.Update(press(5, 3+2, tui.MouseButtonLeft))
	if m.Current() != "tos" || m.Values()["tos"] != "true" {
		t.Fatalf("current=%q tos=%q", m.Current(), m.Values()["tos"])
	}
	m, _ = m.Update(press(5, 3+2, tui.MouseButtonLeft))
	if m.Values()["tos"] != "false" {
		t.Error("second click did not untick")
	}
	m, _ = m.Update(press(5, 3, tui.MouseButtonLeft))
	if m.Current() != "name" || m.Values()["tos"] != "false" {
		t.Errorf("text click: current=%q", m.Current())
	}
}

func TestClickSelectHalvesChoosePrevAndNext(t *testing.T) {
	m := mouseForm()
	// "Plan: < Free >": prompt 6 cells, choice 8 cells, middle at col 10.
	m, _ = m.Update(press(2+13, 3+1, tui.MouseButtonLeft))
	if m.Values()["plan"] != "Pro" || m.Current() != "plan" {
		t.Fatalf("right half: plan=%q current=%q", m.Values()["plan"], m.Current())
	}
	m, _ = m.Update(press(2+7, 3+1, tui.MouseButtonLeft))
	if m.Values()["plan"] != "Free" {
		t.Errorf("left half: plan=%q", m.Values()["plan"])
	}
	m, _ = m.Update(press(2+7, 3+1, tui.MouseButtonLeft))
	if m.Values()["plan"] != "Free" {
		t.Error("left half must stop at the first option")
	}
}

func TestWheelOverSelectChangesOption(t *testing.T) {
	m := mouseForm()
	m, _ = m.Update(press(5, 3+1, tui.MouseButtonWheelDown))
	if m.Values()["plan"] != "Pro" {
		t.Fatalf("plan = %q", m.Values()["plan"])
	}
	m, _ = m.Update(press(5, 3+1, tui.MouseButtonWheelUp))
	if m.Values()["plan"] != "Free" {
		t.Errorf("plan = %q", m.Values()["plan"])
	}
	before := m.Current()
	m, _ = m.Update(press(5, 3, tui.MouseButtonWheelDown)) // over the text field
	if m.Current() != before || m.Values()["plan"] != "Free" {
		t.Error("wheel over a text field changed state")
	}
}

func TestClickRowsAccountForErrorLines(t *testing.T) {
	m := mouseForm()
	m, _ = m.Submit() // "name" is required: an error line appears under it
	if m.Err("name") == "" {
		t.Fatal("setup: no error")
	}
	// Rows: 0 name, 1 error, 2 plan, 3 agree.
	m, _ = m.Update(press(5, 3+3, tui.MouseButtonLeft))
	if m.Current() != "tos" {
		t.Errorf("current = %q, want tos", m.Current())
	}
	m, _ = m.Update(press(5, 3+1, tui.MouseButtonLeft)) // error line belongs to name
	if m.Current() != "name" {
		t.Errorf("current = %q, want name", m.Current())
	}
}

func TestMouseIgnoredWhenOffOutsideOrUnfocused(t *testing.T) {
	for name, fn := range map[string]func(Model) Model{
		"off":       func(m Model) Model { m.Mouse = false; return m },
		"unfocused": func(m Model) Model { m.Blur(); return m },
	} {
		m := fn(mouseForm())
		got, _ := m.Update(press(5, 3+2, tui.MouseButtonLeft))
		if got.Values()["tos"] != "false" {
			t.Errorf("%s: handled the click", name)
		}
	}
	m := mouseForm()
	got, _ := m.Update(press(5, 3+9, tui.MouseButtonLeft)) // below the last field
	if got.Values()["tos"] != "false" || got.Current() != "name" {
		t.Error("click below the fields changed state")
	}
	got, _ = m.Update(press(0, 3+2, tui.MouseButtonLeft)) // left of Bounds
	if got.Values()["tos"] != "false" {
		t.Error("click outside Bounds changed state")
	}
}
