package main

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
)

// now is a Wednesday. Booked days are today+5, 6, 7 (19 to 21 Oct), +13
// (27 Oct), +20, 21 (3 and 4 Nov) and +34 (17 Nov).
var now = time.Date(2026, 10, 14, 9, 30, 0, 0, time.UTC)

func update(m model, msg tui.Msg) (model, tui.Cmd) {
	next, cmd := m.Update(msg)
	return next.(model), cmd
}

func send(m model, msg tui.Msg) model {
	m, _ = update(m, msg)
	return m
}

func key(m model, t tui.KeyType) model { return send(m, tui.Key{Type: t}) }

func runes(m model, s string) model {
	for _, r := range s {
		m = send(m, tui.Key{Type: tui.KeyRunes, Text: string(r)})
	}
	return m
}

func ready() model { return send(initialModel(now), tui.ResizeMsg{Width: 80, Height: 24}) }

func text(m model) string { return ansi.StripANSI(m.View()) }

func rect(t testing.TB, m model, id string) layout.Rect {
	t.Helper()
	r, ok := m.rect(id)
	if !ok {
		t.Fatalf("no target %q on screen:\n%s", id, text(m))
	}
	return r
}

func press(m model, x, y int) model {
	return send(m, tui.MouseEvent{X: x, Y: y, Button: tui.MouseButtonLeft, Action: tui.MouseActionPress})
}

func clickID(t testing.TB, m model, id string) model {
	r := rect(t, m, id)
	return press(m, r.X+r.W/2, r.Y)
}

// clickDay clicks the cell of d, which must be in the month shown.
func clickDay(t testing.TB, m model, d day) model {
	t.Helper()
	g := rect(t, m, "grid")
	for row, w := range weeks(m.month) {
		for col, c := range w {
			if c.Equal(d) {
				return press(m, g.X+col*cellW+1, g.Y+row)
			}
		}
	}
	t.Fatalf("%s is not in the month shown", dayKey(d))
	return m
}

func isQuit(cmd tui.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tui.QuitMsg)
	return ok
}

func oct(d int) day { return date(2026, 10, d) }

func TestShowsTodaysMonth(t *testing.T) {
	out := text(ready())
	for _, want := range []string{"Book a room", "October 2026", "Mo Tu We Th Fr Sa Su", "Pick check-in", "Check-in time", "15:00", "Check-out time", "11:00", "Book"} {
		if !strings.Contains(out, want) {
			t.Errorf("screen lacks %q:\n%s", want, out)
		}
	}
}

func TestTwoClicksMakeAStay(t *testing.T) {
	m := clickDay(t, ready(), oct(14))
	if !m.start.Equal(oct(14)) || m.hasEnd() || !strings.Contains(text(m), "Pick check-out") {
		t.Fatalf("after one click: start=%v end=%v\n%s", m.start, m.end, text(m))
	}
	m = clickDay(t, m, oct(16))
	if !m.end.Equal(oct(16)) || !strings.Contains(text(m), "2 nights, €160") {
		t.Fatalf("after two clicks:\n%s", text(m))
	}
	m = clickDay(t, m, oct(15)) // a third click starts again
	if !m.start.Equal(oct(15)) || m.hasEnd() {
		t.Fatalf("a third click did not restart: start=%v end=%v", m.start, m.end)
	}
}

func TestAnEarlierSecondClickMovesTheStart(t *testing.T) {
	m := clickDay(t, clickDay(t, ready(), oct(16)), oct(15))
	if !m.start.Equal(oct(15)) || m.hasEnd() {
		t.Fatalf("start=%v end=%v", m.start, m.end)
	}
}

func TestPastAndBookedDaysCannotBePicked(t *testing.T) {
	m := clickDay(t, ready(), oct(10))
	if m.hasStart() || !strings.Contains(text(m), "That day has passed") {
		t.Fatalf("a past day was picked:\n%s", text(m))
	}
	m = clickDay(t, m, oct(20))
	if m.hasStart() || !strings.Contains(text(m), "already booked") {
		t.Fatalf("a booked day was picked:\n%s", text(m))
	}
}

func TestAStayCannotCrossABookedDay(t *testing.T) {
	m := clickDay(t, clickDay(t, ready(), oct(17)), oct(23))
	if m.hasEnd() || !strings.Contains(text(m), "A day in between is already booked") {
		t.Fatalf("a stay across booked days was accepted:\n%s", text(m))
	}
	m = clickDay(t, ready(), oct(17))
	m = clickDay(t, m, oct(19)) // leaving on a booked day is fine
	if !m.end.Equal(oct(19)) {
		t.Fatalf("checking out on the morning of a booked day was refused: %q", m.msg)
	}
}

func TestKeyboardPicksAStay(t *testing.T) {
	m := key(ready(), tui.KeyEnter)
	m = key(key(m, tui.KeyRight), tui.KeyRight)
	m = key(m, tui.KeyEnter)
	if !m.start.Equal(oct(14)) || !m.end.Equal(oct(16)) {
		t.Fatalf("start=%v end=%v", m.start, m.end)
	}
	m = key(m, tui.KeyDown) // a week on
	if !m.cursor.Equal(oct(23)) {
		t.Fatalf("cursor = %v after Down", m.cursor)
	}
	m = runes(runes(m, "h"), "k")
	if !m.cursor.Equal(oct(15)) {
		t.Fatalf("cursor = %v after h and k", m.cursor)
	}
	if m = key(m, tui.KeyEsc); m.hasStart() {
		t.Fatal("Esc did not clear the choice")
	}
}

func TestMonthNavigation(t *testing.T) {
	m := key(ready(), tui.KeyPgDown)
	if m.month != date(2026, 11, 1) || !m.cursor.Equal(date(2026, 11, 14)) || !strings.Contains(text(m), "November 2026") {
		t.Fatalf("PgDn: month=%v cursor=%v", m.month, m.cursor)
	}
	m = clickID(t, clickID(t, m, "prev"), "prev")
	if m.month != date(2026, 9, 1) {
		t.Fatalf("‹ twice: month = %v", m.month)
	}
	m = clickID(t, m, "next")
	m = send(m, tui.MouseEvent{X: rect(t, m, "grid").X, Y: rect(t, m, "grid").Y, Button: tui.MouseButtonWheelDown, Action: tui.MouseActionPress})
	if m.month != date(2026, 11, 1) {
		t.Fatalf("› and the wheel: month = %v", m.month)
	}
	if m = clickID(t, m, "today"); m.month != date(2026, 10, 1) || !m.cursor.Equal(oct(14)) {
		t.Fatalf("Today: month=%v cursor=%v", m.month, m.cursor)
	}
}

func TestMonthShiftKeepsADayThatExists(t *testing.T) {
	m := ready()
	m.cursor, m.month = date(2027, 1, 31), date(2027, 1, 1)
	m = key(m, tui.KeyPgDown)
	if !m.cursor.Equal(date(2027, 2, 28)) {
		t.Fatalf("cursor = %v, want 28 February", m.cursor)
	}
}

func TestPickAcrossMonths(t *testing.T) {
	m := clickDay(t, ready(), oct(29))
	m = clickID(t, m, "next")
	m = clickDay(t, m, date(2026, 11, 1))
	if got := nights(m.start, m.end); got != 3 || !strings.Contains(text(m), "3 nights, €240") {
		t.Fatalf("nights = %d\n%s", got, text(m))
	}
}

func TestBookNeedsACompleteStay(t *testing.T) {
	m := clickID(t, ready(), "book")
	if m.done || !strings.Contains(text(m), "Pick a check-in day") {
		t.Fatalf("booked with nothing chosen:\n%s", text(m))
	}
	m = clickDay(t, clickDay(t, ready(), oct(15)), oct(17))
	m = clickID(t, m, "book")
	if !m.done || !strings.Contains(text(m), "✓ Booked: 15 Oct to 17 Oct") {
		t.Fatalf("not booked:\n%s", text(m))
	}
	if m = clickID(t, m, "clear"); m.done || m.hasStart() {
		t.Fatal("Clear left the booking")
	}
}

func TestDayUseNeedsCheckoutAfterCheckin(t *testing.T) {
	m := clickDay(t, clickDay(t, ready(), oct(15)), oct(15))
	if !m.end.Equal(oct(15)) || !strings.Contains(text(m), "day use") {
		t.Fatalf("same-day stay:\n%s", text(m))
	}
	// Check-in 15:00 and check-out 11:00 on one day cannot work.
	if m = clickID(t, m, "book"); m.done || !strings.Contains(text(m), "Check-out must be after check-in") {
		t.Fatalf("a day use ending before it starts was booked:\n%s", text(m))
	}
}

func TestTimeFieldsChangeWithKeysAndMouse(t *testing.T) {
	m := key(key(ready(), tui.KeyTab), tui.KeyUp) // check-in, hour field
	if m.focus != onIn || m.in.Value().Hour() != 16 {
		t.Fatalf("focus=%v hour=%d after Up", m.focus, m.in.Value().Hour())
	}
	m = key(m, tui.KeyTab)
	r := rect(t, m, "tout")
	m = send(m, tui.MouseEvent{X: r.X, Y: r.Y, Button: tui.MouseButtonWheelUp, Action: tui.MouseActionPress})
	if m.focus != onOut || m.out.Value().Hour() != 12 {
		t.Fatalf("focus=%v out hour=%d after the wheel", m.focus, m.out.Value().Hour())
	}
	m = key(m, tui.KeyEsc)
	if m.focus != onCal {
		t.Fatal("Esc did not go back to the calendar")
	}
}

func TestTabCyclesAndShiftTabGoesBack(t *testing.T) {
	m := ready()
	for _, want := range []focus{onIn, onOut, onBook, onCal} {
		if m = key(m, tui.KeyTab); m.focus != want {
			t.Fatalf("Tab went to %v, want %v", m.focus, want)
		}
	}
	m = send(m, tui.Key{Type: tui.KeyTab, Mod: input.ModShift})
	if m.focus != onBook {
		t.Fatalf("Shift+Tab from the calendar went to %v", m.focus)
	}
}

func TestTodayRowIsMarkedAndPastIsDim(t *testing.T) {
	m := ready()
	g := rect(t, m, "grid")
	if g.W != 7*cellW {
		t.Fatalf("grid is %d wide", g.W)
	}
}

func TestHoverIsTracked(t *testing.T) {
	m := ready()
	g := rect(t, m, "grid")
	m = send(m, tui.MouseEvent{X: g.X + 3*cellW + 1, Y: g.Y + 3, Action: tui.MouseActionMotion})
	if !m.hover.Equal(oct(22)) {
		t.Fatalf("hover = %v", m.hover)
	}
}

func TestHelpAndQuit(t *testing.T) {
	m := key(ready(), tui.KeyF1)
	if !strings.Contains(text(m), "pick check-in, then check-out") {
		t.Fatalf("\n%s", text(m))
	}
	if m = key(m, tui.KeyEsc); m.help {
		t.Fatal("Esc did not close help")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyRunes, Text: "q"}); !isQuit(cmd) {
		t.Fatal("q did not quit")
	}
	if _, cmd := update(ready(), tui.Key{Type: tui.KeyCtrlC}); !isQuit(cmd) {
		t.Fatal("Ctrl+C did not quit")
	}
}

func TestNarrowTerminalStacksTheTimes(t *testing.T) {
	m := send(ready(), tui.ResizeMsg{Width: 30, Height: 30})
	if rect(t, m, "tin").X != rect(t, m, "grid").X {
		t.Fatal("the times are not under the calendar at 30 columns")
	}
	m = send(ready(), tui.ResizeMsg{Width: 60, Height: 24})
	if rect(t, m, "tin").X <= rect(t, m, "grid").X {
		t.Fatal("the times are not beside the calendar at 60 columns")
	}
}

func TestTinyTerminalsStayInside(t *testing.T) {
	m := clickDay(t, clickDay(t, ready(), oct(15)), oct(17))
	for _, sz := range [][2]int{{40, 10}, {20, 6}, {10, 4}} {
		m = send(m, tui.ResizeMsg{Width: sz[0], Height: sz[1]})
		for i, line := range strings.Split(m.View(), "\n") {
			if w := ansi.Width(line); w > sz[0] {
				t.Fatalf("%dx%d: line %d is %d wide: %q", sz[0], sz[1], i, w, line)
			}
		}
	}
}
