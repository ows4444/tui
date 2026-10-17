// Package timepicker picks a time of day: hours, minutes and optionally
// seconds and AM or PM, each a field. Left and Right move between the fields,
// Up and Down change the one that has the cursor by one (the minutes by Step),
// and typing digits sets it. Enter confirms. It works on time.Time with the
// standard library only, and keeps the date of the time it was given.
//
// Stability: experimental. Its API may change in any minor release.
package timepicker

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

// The fields of the picker, in the order Left and Right visit them.
const (
	fieldHour = iota
	fieldMinute
	fieldSecond
	fieldPeriod
)

// Model is a time-of-day picker. Build one with New.
type Model struct {
	// ShowSeconds adds a seconds field.
	ShowSeconds bool
	// Hour12 shows hours 1 to 12 and an AM or PM field instead of hours 0 to 23.
	Hour12 bool
	// Step is how far Up and Down move the minutes (and the seconds); 0 or less
	// is 1. Typing a number is not rounded to it.
	Step int
	// Labels are the texts of the AM and PM fields; empty ones are "AM" and "PM".
	AM, PM string
	Theme  theme.Theme

	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left click on a
	// field puts the cursor there and the wheel changes it. Off (the default)
	// ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the picker.
	Bounds hittest.Rect

	t       time.Time // the time being edited; its date is kept
	field   int
	typed   string // digits typed into the current field
	focused bool
}

// New returns a picker showing hours and minutes of t, in 24-hour form.
func New(t time.Time) Model {
	return Model{Step: 1, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), t: t}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	PrevField keymap.Binding // move the cursor to the field before
	NextField keymap.Binding // move the cursor to the field after
	Up        keymap.Binding // raise the field by one (a Step for minutes)
	Down      keymap.Binding // lower the field by one (a Step for minutes)
	Select    keymap.Binding // confirm the time
}

// DefaultKeyMap returns Left and Right, Up and Down, and Enter.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		PrevField: keymap.NewBinding("previous field", "left"),
		NextField: keymap.NewBinding("next field", "right"),
		Up:        keymap.NewBinding("raise", "up"),
		Down:      keymap.NewBinding("lower", "down"),
		Select:    keymap.NewBinding("select time", "enter"),
	}
}

func (km KeyMap) list() []keymap.Binding {
	return []keymap.Binding{km.PrevField, km.NextField, km.Up, km.Down, km.Select}
}

func (m Model) keys() KeyMap {
	for _, b := range m.KeyMap.list() {
		if len(b.Keys) > 0 {
			return m.KeyMap
		}
	}
	return DefaultKeyMap()
}

// Bindings returns the actions the picker honours, for help text.
func (m Model) Bindings() []keymap.Binding { return m.keys().list() }

// SelectedMsg is delivered (via the Cmd Update returns) when Enter confirms the
// time.
type SelectedMsg struct{ Value time.Time }

// Value is the time being edited: the date it was given, with the hour,
// minute and second chosen.
func (m Model) Value() time.Time { return m.t }

// SetValue replaces the time, and puts the cursor on the hours.
func (m *Model) SetValue(t time.Time) { m.t, m.field, m.typed = t, fieldHour, "" }

// Focus gives the picker keyboard focus. It returns no Cmd.
func (m *Model) Focus() tui.Cmd { m.focused = true; return nil }

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused, m.typed = false, "" }

// Focused reports whether the picker has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// fields lists the fields shown, in order.
func (m Model) fields() []int {
	f := []int{fieldHour, fieldMinute}
	if m.ShowSeconds {
		f = append(f, fieldSecond)
	}
	if m.Hour12 {
		f = append(f, fieldPeriod)
	}
	return f
}

func (m Model) step() int { return max(m.Step, 1) }

// Update handles the keys when the picker has focus.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev), nil
	}
	if !m.focused {
		return m, nil
	}
	km := m.keys()
	fs := m.fields()
	pos := 0
	for i, f := range fs {
		if f == m.field {
			pos = i
		}
	}
	switch {
	case keymap.Matches(msg, km.PrevField):
		m.typed = ""
		m.field = fs[max(pos-1, 0)]
	case keymap.Matches(msg, km.NextField):
		m.typed = ""
		m.field = fs[min(pos+1, len(fs)-1)]
	case keymap.Matches(msg, km.Up):
		m.typed = ""
		m.bump(+1)
	case keymap.Matches(msg, km.Down):
		m.typed = ""
		m.bump(-1)
	case keymap.Matches(msg, km.Select):
		v := m.t
		return m, func() tui.Msg { return SelectedMsg{Value: v} }
	default:
		if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes && len(k.Text) == 1 && k.Text[0] >= '0' && k.Text[0] <= '9' && !k.Mod.Ctrl() && !k.Mod.Alt() {
			m.digit(k.Text)
		} else if ok && k.Type == tui.KeyRunes && m.field == fieldPeriod {
			switch strings.ToLower(k.Text) {
			case "a":
				m.setPeriod(false)
			case "p":
				m.setPeriod(true)
			}
		}
	}
	return m, nil
}

// bump moves the current field by dir, wrapping, the minutes and seconds by
// Step. The date does not change.
func (m *Model) bump(dir int) {
	h, mi, s := m.t.Hour(), m.t.Minute(), m.t.Second()
	switch m.field {
	case fieldHour:
		h = (h + dir + 24) % 24
	case fieldMinute:
		mi = ((mi+dir*m.step())%60 + 60) % 60
	case fieldSecond:
		s = ((s+dir*m.step())%60 + 60) % 60
	case fieldPeriod:
		h = (h + 12) % 24
	}
	m.set(h, mi, s)
}

func (m *Model) set(h, mi, s int) {
	y, mo, d := m.t.Date()
	m.t = time.Date(y, mo, d, h, mi, s, 0, m.t.Location())
}

func (m *Model) setPeriod(pm bool) {
	h := m.t.Hour() % 12
	if pm {
		h += 12
	}
	m.set(h, m.t.Minute(), m.t.Second())
}

// digit adds a typed digit to the current field. Two digits fill it; a first
// digit that cannot start a valid two-digit number fills it at once.
func (m *Model) digit(d string) {
	if m.field == fieldPeriod {
		return
	}
	limit := 59
	if m.field == fieldHour {
		limit = 23
		if m.Hour12 {
			limit = 12
		}
	}
	m.typed += d
	n := int(m.typed[0] - '0')
	if len(m.typed) == 2 {
		n = n*10 + int(m.typed[1]-'0')
	}
	if n > limit {
		n = limit
	}
	if m.field == fieldHour && m.Hour12 {
		n = max(n, 1)
		pm := m.t.Hour() >= 12
		n %= 12
		if pm {
			n += 12
		}
	}
	h, mi, s := m.t.Hour(), m.t.Minute(), m.t.Second()
	switch m.field {
	case fieldHour:
		h = n
	case fieldMinute:
		mi = n
	case fieldSecond:
		s = n
	}
	m.set(h, mi, s)
	if len(m.typed) == 2 || n*10 > limit {
		m.typed = ""
		fs := m.fields()
		for i, f := range fs {
			if f == m.field && i+1 < len(fs) {
				m.field = fs[i+1]
				break
			}
		}
	}
}

func (m Model) period() string {
	am, pm := ansi.Sanitize(m.AM), ansi.Sanitize(m.PM) // caller text
	if am == "" {
		am = "AM"
	}
	if pm == "" {
		pm = "PM"
	}
	if m.t.Hour() >= 12 {
		return pm
	}
	return am
}

func (m Model) hourText() int {
	h := m.t.Hour()
	if m.Hour12 {
		if h = h % 12; h == 0 {
			h = 12
		}
	}
	return h
}

// parts are the texts of the fields, in order.
func (m Model) parts() []string {
	var out []string
	for _, f := range m.fields() {
		switch f {
		case fieldHour:
			out = append(out, fmt.Sprintf("%02d", m.hourText()))
		case fieldMinute:
			out = append(out, fmt.Sprintf("%02d", m.t.Minute()))
		case fieldSecond:
			out = append(out, fmt.Sprintf("%02d", m.t.Second()))
		case fieldPeriod:
			out = append(out, m.period())
		}
	}
	return out
}

// View renders "14:30" (or "02:30:15 PM"), the field with the cursor in the
// focus colour while the picker has focus.
func (m Model) View() string {
	st := m.themed().ResolvedStates()
	var b strings.Builder
	for i, f := range m.fields() {
		text := m.parts()[i]
		if i > 0 {
			if f == fieldPeriod {
				b.WriteByte(' ')
			} else {
				b.WriteByte(':')
			}
		}
		if m.focused && f == m.field {
			text = st.Focus.Bold().Render(text)
		}
		b.WriteString(text)
	}
	return b.String()
}

// fieldAt is the field drawn at column x of the picker, or -1.
func (m Model) fieldAt(x int) int {
	col := 0
	for i, f := range m.fields() {
		w := ansi.Width(m.parts()[i])
		if x >= col && x < col+w {
			return f
		}
		col += w + 1
	}
	return -1
}

func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m
	}
	f := m.fieldAt(ev.X - m.Bounds.X)
	switch ev.Button {
	case tui.MouseButtonLeft:
		if f >= 0 {
			m.field, m.typed = f, ""
		}
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		if f >= 0 {
			m.field, m.typed = f, ""
			if ev.Button == tui.MouseButtonWheelUp {
				m.bump(+1)
			} else {
				m.bump(-1)
			}
		}
	}
	return m
}

// Compile-time proof that Model satisfies tui.Component[Model].
var _ tui.Component[Model] = Model{}

// Linearize renders the picker as plain text for accessible output: "time
// picker, 14:30, hours" naming the field the cursor is on.
func (m Model) Linearize() string {
	names := map[int]string{fieldHour: "hours", fieldMinute: "minutes", fieldSecond: "seconds", fieldPeriod: "AM or PM"}
	text := strings.Join(m.parts(), ":")
	if m.Hour12 {
		p := m.parts()
		text = strings.Join(p[:len(p)-1], ":") + " " + p[len(p)-1]
	}
	return "time picker, " + text + ", " + names[m.field]
}
