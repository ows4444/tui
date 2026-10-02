package datepicker

import (
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
)

func press(x, y int, b tui.MouseButton) tui.MouseEvent {
	return tui.MouseEvent{X: x, Y: y, Button: b, Action: tui.MouseActionPress}
}

// March 2024 starts on a Friday (column 5): row 1 holds the 1st at col 5.
func marchModel() Model {
	m := New(date(2024, time.March, 15))
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 10, Y: 4, W: 20, H: 7}
	return m
}

func TestClickDayMovesCursor(t *testing.T) {
	m := marchModel()
	// 1st: col 5 (x=10+15), row 1 (y=5). The 4th is the next row's Monday (col 1).
	m, cmd := m.Update(press(10+15, 5, tui.MouseButtonLeft))
	if !sameDay(m.Cursor(), date(2024, time.March, 1)) || cmd != nil {
		t.Fatalf("cursor = %v cmd=%v", m.Cursor(), cmd != nil)
	}
	m, _ = m.Update(press(10+3+1, 6, tui.MouseButtonLeft)) // second digit of the day cell
	if !sameDay(m.Cursor(), date(2024, time.March, 4)) {
		t.Errorf("cursor = %v, want March 4", m.Cursor())
	}
}

func TestClickCursorDayConfirms(t *testing.T) {
	m := marchModel()
	// 15th: Friday, week of the 10th: row 3, col 5.
	_, cmd := m.Update(press(10+15, 4+3, tui.MouseButtonLeft))
	if cmd == nil {
		t.Fatal("no Cmd")
	}
	if got, ok := cmd().(SelectedMsg); !ok || !sameDay(got.Date, date(2024, time.March, 15)) {
		t.Errorf("msg = %#v", cmd())
	}
}

func TestClickMissesAreIgnored(t *testing.T) {
	m := marchModel()
	for name, ev := range map[string]tui.MouseEvent{
		"header":      press(10, 4, tui.MouseButtonLeft),
		"separator":   press(10+2, 5, tui.MouseButtonLeft),
		"before 1st":  press(10, 5, tui.MouseButtonLeft),
		"after 31st":  press(10+18, 4+6, tui.MouseButtonLeft),
		"outside":     press(0, 0, tui.MouseButtonLeft),
		"right click": press(10+15, 5, tui.MouseButtonRight),
	} {
		got, cmd := m.Update(ev)
		if !sameDay(got.Cursor(), m.Cursor()) || cmd != nil {
			t.Errorf("%s changed state", name)
		}
	}
}

func TestClickOutOfRangeDayIgnored(t *testing.T) {
	m := marchModel()
	m.MaxDate = date(2024, time.March, 20)
	got, _ := m.Update(press(10+15, 4+4, tui.MouseButtonLeft)) // 22nd
	if !sameDay(got.Cursor(), date(2024, time.March, 15)) {
		t.Errorf("cursor = %v", got.Cursor())
	}
}

func TestWheelMovesByMonthKeepingDay(t *testing.T) {
	m := New(date(2024, time.January, 31))
	m.Mouse = true
	m.Bounds = hittest.Rect{W: 20, H: 7}
	m, _ = m.Update(press(1, 1, tui.MouseButtonWheelDown))
	if !sameDay(m.Cursor(), date(2024, time.February, 29)) {
		t.Errorf("down: %v", m.Cursor())
	}
	m, _ = m.Update(press(1, 1, tui.MouseButtonWheelUp))
	if !sameDay(m.Cursor(), date(2024, time.January, 29)) {
		t.Errorf("up: %v", m.Cursor())
	}
}

func TestMouseOffIgnoresEvents(t *testing.T) {
	m := marchModel()
	m.Mouse = false
	got, _ := m.Update(press(10+15, 5, tui.MouseButtonLeft))
	if !sameDay(got.Cursor(), date(2024, time.March, 15)) {
		t.Error("mouse handled while Mouse is off")
	}
}
