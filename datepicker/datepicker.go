// Package datepicker is a keyboard-navigable calendar widget operating
// directly on time.Time (stdlib time only, no external date library).
package datepicker

import (
	"fmt"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is a single-month calendar grid. Value is the last confirmed date
// (set by New and left untouched by navigation — only Enter updates it, via
// the caller applying SelectedMsg). MinDate and MaxDate, when non-zero,
// bound both cursor movement and which days View renders as enabled.
type Model struct {
	Value   time.Time
	MinDate time.Time
	MaxDate time.Time
	Theme   theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with
	// DefaultKeyMap; a Model built as a struct literal with a zero KeyMap
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on a
	// day moves the cursor to it (a click on the highlighted day confirms it,
	// like Enter) and the wheel moves the cursor by a month. Days outside
	// [MinDate, MaxDate] ignore clicks. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the calendar (its
	// first row is the weekday header); mouse events outside it are ignored.
	Bounds hittest.Rect

	cursor time.Time
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	PrevDay    keymap.Binding // move the cursor back one day
	NextDay    keymap.Binding // move the cursor forward one day
	PrevWeek   keymap.Binding // move the cursor back one week
	NextWeek   keymap.Binding // move the cursor forward one week
	MonthStart keymap.Binding // move the cursor to the first of the month
	MonthEnd   keymap.Binding // move the cursor to the last of the month
	Select     keymap.Binding // confirm the highlighted date
}

// DefaultKeyMap returns the keys a Model used before KeyMap existed.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		PrevDay:    keymap.NewBinding("previous day", "left"),
		NextDay:    keymap.NewBinding("next day", "right"),
		PrevWeek:   keymap.NewBinding("previous week", "up"),
		NextWeek:   keymap.NewBinding("next week", "down"),
		MonthStart: keymap.NewBinding("start of month", "home"),
		MonthEnd:   keymap.NewBinding("end of month", "end"),
		Select:     keymap.NewBinding("select date", "enter"),
	}
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.PrevDay, km.NextDay, km.PrevWeek, km.NextWeek, km.MonthStart, km.MonthEnd, km.Select}
}

func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.list() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the actions the picker currently honours, with
// descriptions, for help text.
func (m Model) Bindings() []keymap.Binding { return m.keys().list() }

// New builds a Model with the cursor (and Value) on value, or today if
// value is the zero time.Time.
func New(value time.Time) Model {
	if value.IsZero() {
		value = time.Now()
	}
	return Model{Value: value, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), cursor: value}
}

// Cursor returns the day currently highlighted.
func (m Model) Cursor() time.Time { return m.cursor }

// SelectedMsg is delivered (via the Cmd Update returns) when the
// highlighted date is confirmed with Enter.
type SelectedMsg struct {
	Date time.Time
}

// Update moves the cursor by day (Left/Right), by week (Up/Down), to the
// start/end of its month (Home/End), and confirms it on Enter, returning a
// Cmd that delivers SelectedMsg. Movement is clamped to [MinDate, MaxDate]
// whenever those are set (non-zero). With Mouse on it also handles
// tui.MouseEvent (see Model.Mouse). Any other Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.PrevDay):
		m.cursor = m.cursor.AddDate(0, 0, -1)
		m.clamp()
	case keymap.Matches(msg, km.NextDay):
		m.cursor = m.cursor.AddDate(0, 0, 1)
		m.clamp()
	case keymap.Matches(msg, km.PrevWeek):
		m.cursor = m.cursor.AddDate(0, 0, -7)
		m.clamp()
	case keymap.Matches(msg, km.NextWeek):
		m.cursor = m.cursor.AddDate(0, 0, 7)
		m.clamp()
	case keymap.Matches(msg, km.MonthStart):
		y, mo, _ := m.cursor.Date()
		m.cursor = time.Date(y, mo, 1, 0, 0, 0, 0, m.cursor.Location())
		m.clamp()
	case keymap.Matches(msg, km.MonthEnd):
		y, mo, _ := m.cursor.Date()
		m.cursor = time.Date(y, mo+1, 0, 0, 0, 0, 0, m.cursor.Location())
		m.clamp()
	case keymap.Matches(msg, km.Select):
		date := m.cursor
		return m, func() tui.Msg { return SelectedMsg{Date: date} }
	}
	return m, nil
}

// updateMouse applies a mouse press inside Bounds when Mouse is on.
func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.shiftMonth(-1)
	case tui.MouseButtonWheelDown:
		m.shiftMonth(1)
	case tui.MouseButtonLeft:
		col, row := m.Bounds.Local(ev.X, ev.Y)
		d, ok := m.dayAt(col, row)
		if !ok || !m.inRange(d) {
			return m, nil
		}
		if sameDay(d, m.cursor) {
			date := m.cursor
			return m, func() tui.Msg { return SelectedMsg{Date: date} }
		}
		m.cursor = d
	}
	return m, nil
}

// dayAt maps a cell of the rendered grid (row 0 is the weekday header) to the
// day of the cursor's month shown there. The separator column between two
// days, cells before the 1st and after the last day, and the header all miss.
func (m Model) dayAt(col, row int) (time.Time, bool) {
	if row < 1 || col < 0 || col%3 == 2 {
		return time.Time{}, false
	}
	y, mo, _ := m.cursor.Date()
	first := time.Date(y, mo, 1, 0, 0, 0, 0, m.cursor.Location())
	day := (row-1)*7 + col/3 - int(first.Weekday()) + 1
	last := time.Date(y, mo+1, 0, 0, 0, 0, 0, m.cursor.Location()).Day()
	if col/3 > 6 || day < 1 || day > last {
		return time.Time{}, false
	}
	return time.Date(y, mo, day, 0, 0, 0, 0, m.cursor.Location()), true
}

// shiftMonth moves the cursor by n months, keeping the day of month or, in a
// shorter month, the last day, then clamps it to [MinDate, MaxDate].
func (m *Model) shiftMonth(n int) {
	y, mo, d := m.cursor.Date()
	last := time.Date(y, mo+time.Month(n)+1, 0, 0, 0, 0, 0, m.cursor.Location()).Day()
	if d > last {
		d = last
	}
	m.cursor = time.Date(y, mo+time.Month(n), d, 0, 0, 0, 0, m.cursor.Location())
	m.clamp()
}

// clamp keeps m.cursor within [MinDate, MaxDate] whenever those bounds are
// set (non-zero time.Time).
func (m *Model) clamp() {
	if !m.MinDate.IsZero() && m.cursor.Before(m.MinDate) {
		m.cursor = m.MinDate
	}
	if !m.MaxDate.IsZero() && m.cursor.After(m.MaxDate) {
		m.cursor = m.MaxDate
	}
}

// inRange reports whether d falls within [MinDate, MaxDate], treating an
// unset (zero) bound as unbounded on that side. Comparisons are done on the
// calendar day, ignoring time-of-day.
func (m Model) inRange(d time.Time) bool {
	day := truncateDay(d)
	if !m.MinDate.IsZero() && day.Before(truncateDay(m.MinDate)) {
		return false
	}
	if !m.MaxDate.IsZero() && day.After(truncateDay(m.MaxDate)) {
		return false
	}
	return true
}

func truncateDay(t time.Time) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, t.Location())
}

// View renders a 7-column calendar grid for the cursor's month: a weekday
// header row, then one row per week. The cursor day is highlighted
// (reverse+bold); any day outside [MinDate, MaxDate] (when set) is dimmed
// (Faint).
func (m Model) View() string {
	var b strings.Builder

	cursorStyle := m.themed().ResolvedStates().Selected.Bold()
	dimStyle := ansi.NewStyle().Faint()

	weekdays := []string{"Su", "Mo", "Tu", "We", "Th", "Fr", "Sa"}
	b.WriteString(strings.Join(weekdays, " "))
	b.WriteByte('\n')

	y, mo, _ := m.cursor.Date()
	first := time.Date(y, mo, 1, 0, 0, 0, 0, m.cursor.Location())
	lastDay := time.Date(y, mo+1, 0, 0, 0, 0, 0, m.cursor.Location()).Day()

	// Pad to align the first day under its weekday column.
	b.WriteString(strings.Repeat("   ", int(first.Weekday())))

	for day := 1; day <= lastDay; day++ {
		d := time.Date(y, mo, day, 0, 0, 0, 0, m.cursor.Location())
		cell := fmt.Sprintf("%2d", day)

		switch {
		case sameDay(d, m.cursor):
			cell = cursorStyle.Render(cell)
		case !m.inRange(d):
			cell = dimStyle.Render(cell)
		}

		b.WriteString(cell)

		wd := int(d.Weekday())
		if wd == 6 || day == lastDay {
			b.WriteByte('\n')
		} else {
			b.WriteByte(' ')
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}
