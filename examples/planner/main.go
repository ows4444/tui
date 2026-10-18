// Command planner books a room: pick a check-in and a check-out day on a
// calendar, set the two times, and book. Days that are past or already booked
// cannot be picked, and a stay cannot run across a booked day. The calendar
// and the prices are made up and nothing is sent anywhere.
//
// Mouse: click a day (the first click sets the check-in, the second the
// check-out), click ‹ or › or use the wheel to change month, click a time
// field and scroll to change it, click Book or Clear. Keyboard: arrows move
// the day, PgUp and PgDn change month, Enter or Space picks, T goes to today,
// Esc clears the choice, clicking the month name goes to today, Tab moves between the calendar, the two times and
// Book, Up and Down change a time field, F1 shows help, q quits.
//
//	go run ./examples/planner
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/timepicker"
	"github.com/ows4444/tui/widgets"
)

const (
	pricePerNight = 80
	cellW         = 3
	sideFrom      = 40 // narrowest terminal that puts the times beside the calendar
)

type focus int

const (
	onCal focus = iota
	onIn
	onOut
	onBook
	numFocus
)

type model struct {
	th     ui.Theming // the theme the Program chose; see SetTheme
	today  day
	booked map[string]bool
	month  day // first of the month shown
	cursor day

	start, end day // zero while not chosen
	in, out    timepicker.Model

	focus focus
	hover day
	msg   string
	done  bool // the stay has been booked
	help  bool

	width, height int
}

func initialModel(now time.Time) model {
	today := startOfDay(now)
	m := model{today: today, booked: map[string]bool{}, month: firstOfMonth(today), cursor: today}
	for _, off := range []int{5, 6, 7, 13, 20, 21, 34} { // a few made-up bookings
		m.booked[dayKey(addDays(today, off))] = true
	}
	m.in = timepicker.New(time.Date(2000, 1, 1, 15, 0, 0, 0, time.UTC))
	m.out = timepicker.New(time.Date(2000, 1, 1, 11, 0, 0, 0, time.UTC))
	for _, p := range []*timepicker.Model{&m.in, &m.out} {
		p.Step, p.Mouse = 15, true
	}
	return m
}

func (m model) Init() tui.Cmd { return nil }

// ---- rules -----------------------------------------------------------------

func (m model) unavailable(d day) bool { return d.Before(m.today) || m.booked[dayKey(d)] }

func (m model) hasStart() bool { return !m.start.IsZero() }
func (m model) hasEnd() bool   { return !m.end.IsZero() }

// crossesBooked reports a booked night inside the stay from a to b (the day
// of check-out itself may be booked: the guest leaves in the morning).
func (m model) crossesBooked(a, b day) bool {
	for d := a; d.Before(b); d = addDays(d, 1) {
		if m.booked[dayKey(d)] {
			return true
		}
	}
	return false
}

// pick handles a chosen day: the first sets the check-in, the next the
// check-out, and one after that starts again.
func (m model) pick(d day) model {
	m.msg, m.done = "", false
	leaving := m.hasStart() && !m.hasEnd() && !d.Before(m.start) // a booked day is a booked night: you may leave on it
	switch {
	case d.Before(m.today):
		m.msg = "That day has passed"
		return m
	case m.booked[dayKey(d)] && !leaving:
		m.msg = "That day is already booked"
		return m
	}
	m.cursor = d
	switch {
	case !m.hasStart() || m.hasEnd() || d.Before(m.start):
		m.start, m.end = d, day{}
	case m.crossesBooked(m.start, d):
		m.msg = "A day in between is already booked"
	default:
		m.end = d
	}
	return m
}

// problem is why the stay cannot be booked yet, or "".
func (m model) problem() string {
	switch {
	case !m.hasStart():
		return "Pick a check-in day"
	case !m.hasEnd():
		return "Pick a check-out day"
	case m.start.Equal(m.end) && !m.timeOf(m.out).After(m.timeOf(m.in)):
		return "Check-out must be after check-in"
	}
	return ""
}

func (m model) timeOf(p timepicker.Model) time.Time { return p.Value() }

func (m model) total() int { return max(1, nights(m.start, m.end)) * pricePerNight }

func (m model) clear() model {
	m.start, m.end, m.msg, m.done = day{}, day{}, "", false
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyCtrlC:
		return m, tui.Quit()
	case k.Type == tui.KeyF1:
		m.help = !m.help
		return m, nil
	case m.help:
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	case k.Type == tui.KeyTab:
		step := focus(1)
		if k.Mod.Shift() {
			step = numFocus - 1
		}
		return m.focusOn((m.focus + step) % numFocus)
	}
	switch m.focus {
	case onCal:
		return m.calKey(k)
	case onBook:
		return m.bookKey(k)
	}
	if k.Type == tui.KeyEsc {
		return m.focusOn(onCal)
	}
	return m.timeKey(k)
}

func (m model) focusOn(f focus) (tui.Model, tui.Cmd) {
	m.in.Blur()
	m.out.Blur()
	m.focus = f
	switch f {
	case onIn:
		return m, m.in.Focus()
	case onOut:
		return m, m.out.Focus()
	}
	return m, nil
}

func (m model) timeKey(k tui.Key) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	if m.focus == onIn {
		m.in, cmd = m.in.Update(k)
	} else {
		m.out, cmd = m.out.Update(k)
	}
	return m, cmd
}

// move puts the cursor on d, and shows its month.
func (m model) move(d day) model {
	m.cursor = d
	m.month = firstOfMonth(d)
	return m
}

func (m model) calKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyLeft:
		return m.move(addDays(m.cursor, -1)), nil
	case tui.KeyRight:
		return m.move(addDays(m.cursor, 1)), nil
	case tui.KeyUp:
		return m.move(addDays(m.cursor, -7)), nil
	case tui.KeyDown:
		return m.move(addDays(m.cursor, 7)), nil
	case tui.KeyPgUp:
		return m.shift(-1), nil
	case tui.KeyPgDown:
		return m.shift(1), nil
	case tui.KeyEnter:
		return m.pick(m.cursor), nil
	case tui.KeyEsc:
		return m.clear(), nil
	case tui.KeyRunes:
		switch k.Text {
		case " ":
			return m.pick(m.cursor), nil
		case "t":
			return m.move(m.today), nil
		case "q":
			return m, tui.Quit()
		case "h":
			return m.move(addDays(m.cursor, -1)), nil
		case "l":
			return m.move(addDays(m.cursor, 1)), nil
		case "k":
			return m.move(addDays(m.cursor, -7)), nil
		case "j":
			return m.move(addDays(m.cursor, 7)), nil
		}
	}
	return m, nil
}

// shift changes the shown month, keeping the cursor on the same day number
// (or the last day of a shorter month).
func (m model) shift(n int) model {
	first := addMonths(m.month, n)
	d := min(m.cursor.Day(), addDays(addMonths(first, 1), -1).Day())
	m.month = first
	m.cursor = date(first.Year(), first.Month(), d)
	return m
}

func (m model) bookKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyEnter, k.Type == tui.KeyRunes && k.Text == " ":
		return m.book(), nil
	case k.Type == tui.KeyEsc:
		return m.focusOn(onCal)
	case k.Type == tui.KeyRunes && k.Text == "q":
		return m, tui.Quit()
	}
	return m, nil
}

func (m model) book() model {
	if p := m.problem(); p != "" {
		m.msg = p
		return m
	}
	m.done, m.msg = true, ""
	return m
}

// ---- mouse -----------------------------------------------------------------

func (m model) rect(id string) (layout.Rect, bool) {
	return layout.RectOf(m.screen(), m.size(), id)
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// dayAt is the day drawn at x, y, or the zero time.
func (m model) dayAt(x, y int) day {
	g, ok := m.rect("grid")
	if !ok || !inside(g, x, y) {
		return day{}
	}
	ws := weeks(m.month)
	row, col := y-g.Y, (x-g.X)/cellW
	if row >= len(ws) || col > 6 {
		return day{}
	}
	return ws[row][col]
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	if ev.Action == tui.MouseActionMotion {
		m.hover = m.dayAt(ev.X, ev.Y)
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		if g, ok := m.rect("cal"); ok && inside(g, ev.X, ev.Y) {
			if ev.Button == tui.MouseButtonWheelUp {
				return m.shift(-1), nil
			}
			return m.shift(1), nil
		}
		return m.timeMouse(ev)
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	return m.click(ev)
}

func (m model) click(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if d := m.dayAt(ev.X, ev.Y); !d.IsZero() {
		m.focus = onCal
		return m.pick(d), nil
	}
	for _, id := range []string{"prev", "next", "today", "clear", "book"} {
		if r, ok := m.rect(id); ok && inside(r, ev.X, ev.Y) {
			return m.button(id)
		}
	}
	return m.timeMouse(ev)
}

func (m model) button(id string) (tui.Model, tui.Cmd) {
	switch id {
	case "prev":
		return m.shift(-1), nil
	case "next":
		return m.shift(1), nil
	case "today":
		return m.move(m.today), nil
	case "clear":
		return m.clear(), nil
	}
	m.focus = onBook
	return m.book(), nil
}

// timeMouse passes a mouse event to the time picker it is over.
func (m model) timeMouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	for _, t := range []struct {
		id string
		f  focus
	}{{"tin", onIn}, {"tout", onOut}} {
		r, ok := m.rect(t.id)
		if !ok || !inside(r, ev.X, ev.Y) {
			continue
		}
		next, _ := m.focusOn(t.f)
		m = next.(model)
		var cmd tui.Cmd
		if t.f == onIn {
			m.in.Bounds = r
			m.in, cmd = m.in.Update(ev)
		} else {
			m.out.Bounds = r
			m.out, cmd = m.out.Update(ev)
		}
		return m, cmd
	}
	return m, nil
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) side() bool { return m.size().W >= sideFrom }

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) link(id, label string, th theme.Theme) layout.FlexChild {
	return layout.FlexChild{Node: layout.Named(id, layout.Block(ansi.NewStyle().Foreground(th.Primary).Render(label)))}
}

func (m model) cell(d day, th theme.Theme) string {
	if d.IsZero() {
		return strings.Repeat(" ", cellW)
	}
	st := ansi.NewStyle()
	switch {
	case m.booked[dayKey(d)]:
		st = ansi.NewStyle().Faint().Strikethrough()
	case d.Before(m.today):
		st = ansi.NewStyle().Faint()
	case d.Equal(m.start) || d.Equal(m.end):
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	case m.hasEnd() && d.After(m.start) && d.Before(m.end):
		st = ansi.NewStyle().Background(th.Selection)
	case d.Equal(m.today):
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	if d.Equal(m.cursor) && m.focus == onCal {
		st = st.Underline().Bold()
	} else if d.Equal(m.hover) && !m.unavailable(d) {
		st = st.Underline()
	}
	return st.Render(fmt.Sprintf("%2d ", d.Day()))
}

func (m model) calendar(th theme.Theme) layout.Node {
	head := layout.Row(1,
		m.link("prev", "‹", th),
		layout.FlexChild{Node: layout.Named("today", layout.Block(ansi.NewStyle().Bold().Render(fmt.Sprintf("%-*s", 3*7-4, m.month.Format("January 2006")))))},
		m.link("next", "›", th),
	)
	rows := []layout.FlexChild{block(ansi.NewStyle().Faint().Render("Mo Tu We Th Fr Sa Su "))}
	var grid []layout.FlexChild
	for _, w := range weeks(m.month) {
		var line strings.Builder
		for _, d := range w {
			line.WriteString(m.cell(d, th))
		}
		grid = append(grid, block(line.String()))
	}
	rows = append(rows, layout.FlexChild{Node: layout.Named("grid", layout.Column(0, grid...))})
	col := layout.Column(0, append([]layout.FlexChild{{Node: head}}, rows...)...)
	return layout.Named("cal", layout.Fixed(col, layout.Size{W: 7 * cellW}))
}

func (m model) summary(th theme.Theme) []layout.FlexChild {
	label := func(s string) layout.FlexChild { return block(ansi.NewStyle().Faint().Render(s)) }
	f := "Mon 2 Jan"
	var out []layout.FlexChild
	switch {
	case !m.hasStart():
		out = append(out, block("Pick check-in"))
	case !m.hasEnd():
		out = append(out, block(m.start.Format(f)+" → ?"), label("Pick check-out"))
	default:
		n := nights(m.start, m.end)
		stay := fmt.Sprintf("%d nights", n)
		switch n {
		case 0:
			stay = "day use"
		case 1:
			stay = "1 night"
		}
		out = append(out, block(m.start.Format(f)+" → "+m.end.Format(f)),
			label(fmt.Sprintf("%s, €%d", stay, m.total())))
	}
	return out
}

func (m model) timeBox(id, name string, p timepicker.Model, f focus, th theme.Theme) layout.Node {
	st := ansi.NewStyle().Faint()
	if m.focus == f {
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	return layout.Column(0,
		block(st.Render(name)),
		layout.FlexChild{Node: layout.Named(id, layout.Fixed(p.LayoutNode(), layout.Size{W: 8, H: 1}))},
	)
}

func (m model) buttons(th theme.Theme) layout.Node {
	book := ansi.NewStyle().Faint()
	if m.problem() == "" {
		book = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	}
	if m.focus == onBook {
		book = book.Underline()
	}
	clr := ansi.NewStyle().Foreground(th.Muted)
	return layout.Row(2,
		layout.FlexChild{Node: layout.Named("book", layout.Block(book.Render(" Book ")))},
		layout.FlexChild{Node: layout.Named("clear", layout.Block(clr.Render("Clear")))},
	)
}

func (m model) notice(th theme.Theme) string {
	switch {
	case m.done:
		return ansi.NewStyle().Bold().Foreground(th.Success).Render("✓ Booked: " + m.start.Format("2 Jan") + " to " + m.end.Format("2 Jan"))
	case m.msg != "":
		return ansi.NewStyle().Foreground(th.Warning).Render("⚠ " + m.msg)
	}
	return ansi.NewStyle().Faint().Render("struck-through nights are booked")
}

func (m model) helpPanel(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.NewStyle().Bold().Foreground(th.Primary).Render(fmt.Sprintf("%-14s", k)) + a)
	}
	return layout.Column(0,
		key("Arrows hjkl", "move the day"), key("PgUp PgDn", "change month"),
		key("Enter Space", "pick check-in, then check-out"), key("T, click month", "go to today"),
		key("Esc", "clear the choice"), key("Tab", "calendar, times, Book"),
		key("Up Down", "change a time field"), key("F1", "close help"), key("q", "quit"))
}

func (m model) screen() layout.Node {
	th := m.theme()
	sz := m.size()
	if m.help {
		return layout.Column(0, block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Book a room: keys")),
			layout.FlexChild{Node: m.helpPanel(th)})
	}
	right := []layout.FlexChild{}
	right = append(right, m.summary(th)...)
	right = append(right,
		layout.FlexChild{Node: m.timeBox("tin", "Check-in time", m.in, onIn, th)},
		layout.FlexChild{Node: m.timeBox("tout", "Check-out time", m.out, onOut, th)},
		layout.FlexChild{Node: m.buttons(th)},
	)
	body := layout.Node(layout.Row(2, layout.FlexChild{Node: m.calendar(th)}, layout.FlexChild{Node: layout.Column(0, right...)}))
	if !m.side() {
		body = layout.Column(0, layout.FlexChild{Node: m.calendar(th)}, layout.FlexChild{Node: layout.Column(0, right...)})
	}
	hints := widgets.Footer(sz.W, widgets.Hint{Key: "←↑↓→", Action: "day"}, widgets.Hint{Key: "enter", Action: "pick"},
		widgets.Hint{Key: "pgup/pgdn", Action: "month"}, widgets.Hint{Key: "tab", Action: "times"}, widgets.Hint{Key: "f1", Action: "help"})
	return layout.Column(0,
		block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Book a room")),
		layout.FlexChild{Node: body},
		block(ansi.Truncate(m.notice(th), sz.W)),
		block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, sz.W))),
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	p := ui.NewProgram(initialModel(time.Now()), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
