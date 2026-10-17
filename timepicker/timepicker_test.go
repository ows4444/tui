package timepicker

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
)

var day = time.Date(2026, 10, 17, 14, 30, 45, 0, time.UTC)

func ready() Model {
	m := New(day)
	m.Focus()
	return m
}

func k(t tui.KeyType) tui.Key { return tui.Key{Type: t} }

func rune_(s string) tui.Key { return tui.Key{Type: tui.KeyRunes, Text: s} }

func press(m Model, msgs ...tui.Msg) Model {
	for _, msg := range msgs {
		m, _ = m.Update(msg)
	}
	return m
}

func TestViewAndFieldVisits(t *testing.T) {
	m := ready()
	if v := ansi.StripANSI(m.View()); v != "14:30" {
		t.Fatalf("view %q", v)
	}
	m.ShowSeconds = true
	if v := ansi.StripANSI(m.View()); v != "14:30:45" {
		t.Fatalf("view %q", v)
	}
	m = press(m, k(tui.KeyRight), k(tui.KeyRight), k(tui.KeyRight)) // stops at the last field
	if m.field != fieldSecond {
		t.Fatalf("field %d", m.field)
	}
	m = press(m, k(tui.KeyLeft), k(tui.KeyLeft), k(tui.KeyLeft))
	if m.field != fieldHour {
		t.Fatalf("field %d", m.field)
	}
}

func TestUpAndDownWrapAndKeepTheDate(t *testing.T) {
	m := ready()
	m = press(m, k(tui.KeyUp))
	if m.Value().Hour() != 15 {
		t.Fatalf("hour %d", m.Value().Hour())
	}
	m = press(m, k(tui.KeyRight), k(tui.KeyDown))
	if m.Value().Minute() != 29 {
		t.Fatalf("minute %d", m.Value().Minute())
	}
	m.SetValue(time.Date(2026, 10, 17, 23, 59, 0, 0, time.UTC))
	m = press(m, k(tui.KeyUp)) // 23 -> 0, still the same day
	if m.Value().Hour() != 0 || m.Value().Day() != 17 {
		t.Fatalf("%v", m.Value())
	}
	m = press(m, k(tui.KeyRight), k(tui.KeyUp)) // 59 -> 0, hour unchanged
	if m.Value().Minute() != 0 || m.Value().Hour() != 0 {
		t.Fatalf("%v", m.Value())
	}
}

func TestStepMovesMinutes(t *testing.T) {
	m := ready()
	m.Step = 15
	m = press(m, k(tui.KeyRight), k(tui.KeyUp))
	if m.Value().Minute() != 45 {
		t.Fatalf("minute %d", m.Value().Minute())
	}
	m = press(m, k(tui.KeyUp)) // 45 + 15 = 60 wraps to 0
	if m.Value().Minute() != 0 {
		t.Fatalf("minute %d", m.Value().Minute())
	}
}

func TestTypingDigits(t *testing.T) {
	m := ready()
	m = press(m, rune_("0"), rune_("9")) // hours 09, then on to the minutes
	if m.Value().Hour() != 9 || m.field != fieldMinute {
		t.Fatalf("%v field %d", m.Value(), m.field)
	}
	m = press(m, rune_("7")) // 7 cannot start a longer minute: 07 at once
	if m.Value().Minute() != 7 {
		t.Fatalf("minute %d", m.Value().Minute())
	}
	m = ready()
	m = press(m, rune_("3")) // 3 cannot start an hour: 03, on
	if m.Value().Hour() != 3 || m.field != fieldMinute {
		t.Fatalf("%v field %d", m.Value(), m.field)
	}
	m = ready()
	m = press(m, rune_("9"), rune_("9")) // clamped
	if m.Value().Hour() != 9 {
		t.Fatalf("hour %d", m.Value().Hour())
	}
	m = ready()
	m = press(m, rune_("2"), rune_("9"))
	if m.Value().Hour() != 23 {
		t.Fatalf("29 should clamp to 23, got %d", m.Value().Hour())
	}
	m = ready()
	if m = press(m, rune_("x")); m.Value() != day {
		t.Fatal("a letter changed the time")
	}
}

func TestTwelveHourClock(t *testing.T) {
	m := ready()
	m.Hour12 = true
	if v := ansi.StripANSI(m.View()); v != "02:30 PM" {
		t.Fatalf("view %q", v)
	}
	m = press(m, k(tui.KeyRight), k(tui.KeyRight)) // the period
	m = press(m, k(tui.KeyUp))
	if m.Value().Hour() != 2 || ansi.StripANSI(m.View()) != "02:30 AM" {
		t.Fatalf("%v %q", m.Value(), ansi.StripANSI(m.View()))
	}
	m = press(m, rune_("p"))
	if m.Value().Hour() != 14 {
		t.Fatalf("p should set PM: %d", m.Value().Hour())
	}
	m = press(m, rune_("a"))
	if m.Value().Hour() != 2 {
		t.Fatalf("a should set AM: %d", m.Value().Hour())
	}
	m.SetValue(time.Date(2026, 1, 1, 0, 5, 0, 0, time.UTC))
	if v := ansi.StripANSI(m.View()); v != "12:05 AM" {
		t.Fatalf("midnight reads %q", v)
	}
	m = press(m, rune_("1"), rune_("2")) // 12 AM stays hour 0
	if m.Value().Hour() != 0 {
		t.Fatalf("12 in the morning is hour 0, got %d", m.Value().Hour())
	}
	m.AM, m.PM = "vorm.", "nachm."
	m.t = time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC)
	if !strings.HasSuffix(ansi.StripANSI(m.View()), "nachm.") {
		t.Fatal("custom PM label ignored")
	}
}

func TestEnterConfirms(t *testing.T) {
	m := ready()
	_, cmd := m.Update(k(tui.KeyEnter))
	if sel, ok := cmd().(SelectedMsg); !ok || !sel.Value.Equal(day) {
		t.Fatalf("%+v", sel)
	}
	u := New(day)
	if _, cmd := u.Update(k(tui.KeyUp)); cmd != nil {
		t.Fatal("an unfocused picker reacted")
	}
}

func TestMouse(t *testing.T) {
	m := ready()
	m.Mouse = true
	m.Bounds = hittest.Rect{X: 5, Y: 2, W: 20, H: 1}
	m = press(m, tui.MouseEvent{X: 5 + 3, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress}) // the minutes
	if m.field != fieldMinute {
		t.Fatalf("field %d", m.field)
	}
	m = press(m, tui.MouseEvent{X: 5 + 4, Y: 2, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.Value().Minute() != 31 {
		t.Fatalf("minute %d", m.Value().Minute())
	}
	m = press(m, tui.MouseEvent{X: 50, Y: 2, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
	if m.field != fieldMinute {
		t.Fatal("a click outside moved the cursor")
	}
}

func TestLinearizeAndHelpers(t *testing.T) {
	m := ready()
	if got := m.Linearize(); got != "time picker, 14:30, hours" {
		t.Fatalf("%q", got)
	}
	m.Hour12 = true
	if got := m.Linearize(); got != "time picker, 02:30 PM, hours" {
		t.Fatalf("%q", got)
	}
	_ = m.LayoutNode()
	_ = m.Tokens()
	_ = m.SetTheme(m.Theme).WithTokens(m.Tokens())
	if len(m.Bindings()) != 5 {
		t.Fatal("bindings")
	}
	m.Blur()
	if m.Focused() {
		t.Fatal("Blur")
	}
}
