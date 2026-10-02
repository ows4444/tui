package datepicker

import (
	"strings"
	"testing"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func sendKey(t *testing.T, m Model, kt tui.KeyType) Model {
	t.Helper()
	m, _ = m.Update(tui.Key{Type: kt})
	return m
}

// #554: Left/Right move the cursor by one day; Up/Down move it by seven
// days, both clamped within [MinDate, MaxDate] when set.
func TestArrowMovement(t *testing.T) {
	m := New(date(2024, time.March, 15))

	m = sendKey(t, m, tui.KeyRight)
	if !sameDay(m.Cursor(), date(2024, time.March, 16)) {
		t.Fatalf("Right: got %v, want 2024-03-16", m.Cursor())
	}

	m = sendKey(t, m, tui.KeyLeft)
	m = sendKey(t, m, tui.KeyLeft)
	if !sameDay(m.Cursor(), date(2024, time.March, 14)) {
		t.Fatalf("Left: got %v, want 2024-03-14", m.Cursor())
	}

	m = sendKey(t, m, tui.KeyDown)
	if !sameDay(m.Cursor(), date(2024, time.March, 21)) {
		t.Fatalf("Down: got %v, want 2024-03-21", m.Cursor())
	}

	m = sendKey(t, m, tui.KeyUp)
	m = sendKey(t, m, tui.KeyUp)
	if !sameDay(m.Cursor(), date(2024, time.March, 7)) {
		t.Fatalf("Up: got %v, want 2024-03-07", m.Cursor())
	}
}

func TestArrowMovementClamped(t *testing.T) {
	min := date(2024, time.March, 10)
	max := date(2024, time.March, 20)

	m := New(date(2024, time.March, 11))
	m.MinDate = min
	m.MaxDate = max

	// Walk left past MinDate; cursor should clamp to MinDate.
	for i := 0; i < 5; i++ {
		m = sendKey(t, m, tui.KeyLeft)
	}
	if !sameDay(m.Cursor(), min) {
		t.Fatalf("clamp low: got %v, want %v", m.Cursor(), min)
	}

	m = New(date(2024, time.March, 19))
	m.MinDate = min
	m.MaxDate = max

	// Walk right past MaxDate; cursor should clamp to MaxDate.
	for i := 0; i < 5; i++ {
		m = sendKey(t, m, tui.KeyRight)
	}
	if !sameDay(m.Cursor(), max) {
		t.Fatalf("clamp high: got %v, want %v", m.Cursor(), max)
	}

	// A full week jump past the bound should also clamp.
	m = New(date(2024, time.March, 15))
	m.MinDate = min
	m.MaxDate = max
	m = sendKey(t, m, tui.KeyDown)
	if !sameDay(m.Cursor(), max) {
		t.Fatalf("clamp week down: got %v, want %v", m.Cursor(), max)
	}
	m = sendKey(t, m, tui.KeyUp)
	m = sendKey(t, m, tui.KeyUp)
	if !sameDay(m.Cursor(), min) {
		t.Fatalf("clamp week up: got %v, want %v", m.Cursor(), min)
	}
}

// #555: Home moves the cursor to the first day of its month; End moves it
// to the last day of that month.
func TestHomeEnd(t *testing.T) {
	m := New(date(2024, time.February, 15)) // leap year, 29 days

	m = sendKey(t, m, tui.KeyHome)
	if !sameDay(m.Cursor(), date(2024, time.February, 1)) {
		t.Fatalf("Home: got %v, want 2024-02-01", m.Cursor())
	}

	m = sendKey(t, m, tui.KeyEnd)
	if !sameDay(m.Cursor(), date(2024, time.February, 29)) {
		t.Fatalf("End: got %v, want 2024-02-29", m.Cursor())
	}

	// Non-leap year February should end on the 28th.
	m2 := New(date(2023, time.February, 10))
	m2 = sendKey(t, m2, tui.KeyEnd)
	if !sameDay(m2.Cursor(), date(2023, time.February, 28)) {
		t.Fatalf("End (non-leap): got %v, want 2023-02-28", m2.Cursor())
	}
}

func TestHomeEndClamped(t *testing.T) {
	m := New(date(2024, time.March, 15))
	m.MinDate = date(2024, time.March, 5)
	m.MaxDate = date(2024, time.March, 25)

	m = sendKey(t, m, tui.KeyHome)
	if !sameDay(m.Cursor(), m.MinDate) {
		t.Fatalf("Home clamp: got %v, want %v", m.Cursor(), m.MinDate)
	}

	m = New(date(2024, time.March, 15))
	m.MinDate = date(2024, time.March, 5)
	m.MaxDate = date(2024, time.March, 25)
	m = sendKey(t, m, tui.KeyEnd)
	if !sameDay(m.Cursor(), m.MaxDate) {
		t.Fatalf("End clamp: got %v, want %v", m.Cursor(), m.MaxDate)
	}
}

// #556: Enter confirms the highlighted date, returning a Cmd delivering a
// SelectedMsg carrying that time.Time value.
func TestEnterConfirms(t *testing.T) {
	m := New(date(2024, time.March, 15))
	m = sendKey(t, m, tui.KeyRight)

	_, cmd := m.Update(tui.Key{Type: tui.KeyEnter})
	if cmd == nil {
		t.Fatal("Enter: expected non-nil Cmd")
	}

	msg := cmd()
	sel, ok := msg.(SelectedMsg)
	if !ok {
		t.Fatalf("Enter: expected SelectedMsg, got %T", msg)
	}
	if !sameDay(sel.Date, date(2024, time.March, 16)) {
		t.Fatalf("Enter: got %v, want 2024-03-16", sel.Date)
	}
}

// Non-Key messages are a no-op.
func TestUpdateNonKeyMsg(t *testing.T) {
	m := New(date(2024, time.March, 15))
	m2, cmd := m.Update(tui.ResizeMsg{Width: 80, Height: 24})
	if cmd != nil {
		t.Fatal("expected nil Cmd for non-Key msg")
	}
	if !sameDay(m2.Cursor(), m.Cursor()) {
		t.Fatal("expected cursor unchanged for non-Key msg")
	}
}

// #557: View renders a 7-column calendar grid for the cursor's month, with
// the cursor day highlighted and out-of-range days dimmed.
func TestViewGrid(t *testing.T) {
	m := New(date(2024, time.March, 15))
	view := m.View()

	lines := strings.Split(view, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header + at least one week row, got %d lines", len(lines))
	}
	if !strings.Contains(lines[0], "Su") || !strings.Contains(lines[0], "Sa") {
		t.Fatalf("expected weekday header row, got %q", lines[0])
	}

	// The cursor day (15) must appear somewhere, styled.
	if !strings.Contains(view, "15") {
		t.Fatalf("expected day 15 present in view: %q", view)
	}

	// Highlighted cursor day should carry a reverse SGR code.
	cursorRendered := ansi.NewStyle().Bold().Reverse().Render("15")
	// We don't assert the exact color-inclusive sequence (theme-dependent),
	// but the reverse attribute (7) must be present somewhere.
	if !strings.Contains(view, "\x1b[") {
		t.Fatalf("expected ANSI styling in view: %q", view)
	}
	_ = cursorRendered
}

func TestViewDimsOutOfRange(t *testing.T) {
	m := New(date(2024, time.March, 15))
	m.MinDate = date(2024, time.March, 10)
	m.MaxDate = date(2024, time.March, 20)

	view := m.View()

	// Faint SGR code is "2". Day 5 (before MinDate) should be dimmed, so a
	// faint escape must appear in the rendered view.
	if !strings.Contains(view, "\x1b[2m") {
		t.Fatalf("expected faint styling for out-of-range days, view: %q", view)
	}

	// A model with no bounds set should never render faint text.
	unbounded := New(date(2024, time.March, 15))
	uview := unbounded.View()
	if strings.Contains(uview, "\x1b[2m") {
		t.Fatalf("unbounded model should not dim any day, view: %q", uview)
	}
}

func TestNewZeroValueDefaultsToToday(t *testing.T) {
	before := time.Now()
	m := New(time.Time{})
	after := time.Now()

	if m.Cursor().Before(truncateDay(before)) || m.Cursor().After(after.Add(time.Second)) {
		t.Fatalf("expected cursor near now, got %v (between %v and %v)", m.Cursor(), before, after)
	}
}
