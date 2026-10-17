// Package dropdown is a closed one-line control that shows the chosen option
// and opens a list of the others, the Select of web component libraries.
// Enter, Space or Down opens it, Up and Down move through the list (typing
// letters jumps to the next option that starts with them), Enter or Space
// chooses and Esc closes without changing anything. Closed, Left and Right
// step through the options. With the mouse on, a click opens it and a click
// on an option chooses it.
//
// The open list is drawn under the control and pushes what is below it down,
// so it works in any layout. For a list that floats over the page, draw
// Popup with the popover package.
//
// Stability: experimental. Its API may change in any minor release.
package dropdown

import (
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// defaultMaxRows is how many options the open list shows at once.
const defaultMaxRows = 8

// typeAheadReset is how long a pause ends the letters typed to jump.
const typeAheadReset = time.Second

// now is the clock the type-ahead uses; a test replaces it.
var now = time.Now

// Model is one dropdown. The zero value is not ready to use; build one with
// New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one dropdown's change from another's.
	ID string
	// Label is the text before the control. An empty Label draws the control
	// alone.
	Label string
	// Options are the choices, in order.
	Options []string
	// Placeholder is shown while nothing is chosen. Empty means "Select".
	Placeholder string
	// MinWidth is the least width of the value area; the widest option or the
	// placeholder is used if it is larger. 0 means that width.
	MinWidth int
	// MaxRows is how many options the open list shows before it scrolls.
	// 0 means 8.
	MaxRows int
	// Disabled stops the dropdown opening or changing; it is drawn faint.
	Disabled bool
	Theme    theme.Theme

	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// the closed control opens it, and when open a press on an option chooses
	// it and a press anywhere else closes it. Off (the default) ignores the
	// mouse.
	Mouse bool
	// Bounds is the screen rectangle of the closed control, one row. The open
	// list's options are the rows right below it.
	Bounds hittest.Rect

	selected int // -1 when nothing is chosen
	cursor   int // the highlighted option while open
	offset   int // the first option shown while open
	open     bool
	focused  bool
	typed    string
	typedAt  time.Time
}

// New returns a dropdown labelled label with the given options, nothing
// chosen.
func New(label string, options ...string) Model {
	return Model{Label: label, Options: options, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap(), selected: -1}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Open    keymap.Binding // open the list (closed)
	Prev    keymap.Binding // choose the previous option (closed) or move up (open)
	Next    keymap.Binding // choose the next option (closed) or move down (open)
	Confirm keymap.Binding // choose the highlighted option and close (open)
	Cancel  keymap.Binding // close without choosing (open)
	First   keymap.Binding // highlight the first option (open)
	Last    keymap.Binding // highlight the last option (open)
	PageUp  keymap.Binding // move up a page (open)
	PageDn  keymap.Binding // move down a page (open)
}

// DefaultKeyMap returns Enter, Space and Down to open; Left and Up, Right and
// Down to step or move; Enter and Space to choose; Esc to close; Home, End,
// PgUp and PgDn.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Open:    keymap.NewBinding("open", "enter", "space", "down"),
		Prev:    keymap.NewBinding("previous", "left", "up"),
		Next:    keymap.NewBinding("next", "right", "down"),
		Confirm: keymap.NewBinding("choose", "enter", "space"),
		Cancel:  keymap.NewBinding("close", "esc"),
		First:   keymap.NewBinding("first", "home"),
		Last:    keymap.NewBinding("last", "end"),
		PageUp:  keymap.NewBinding("page up", "pgup"),
		PageDn:  keymap.NewBinding("page down", "pgdown"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Open.Keys)+len(km.Confirm.Keys)+len(km.Next.Keys)+len(km.Cancel.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the dropdown honours now, with descriptions,
// for help text. A disabled one has none.
func (m Model) Bindings() []keymap.Binding {
	if m.Disabled {
		return nil
	}
	km := m.keys()
	if m.open {
		return []keymap.Binding{km.Prev, km.Next, km.Confirm, km.Cancel}
	}
	return []keymap.Binding{km.Open, km.Prev, km.Next}
}

// ChangedMsg is delivered (via the Cmd Update returns) when a key or a click
// chooses a different option.
type ChangedMsg struct {
	// ID is the dropdown's ID.
	ID string
	// Index is the chosen option's position in Options.
	Index int
	// Value is the chosen option.
	Value string
}

// Selected returns the index of the chosen option, or -1 when nothing is
// chosen.
func (m Model) Selected() int {
	if m.selected < 0 || m.selected >= len(m.Options) {
		return -1
	}
	return m.selected
}

// Value returns the chosen option, or "" when nothing is chosen.
func (m Model) Value() string {
	if i := m.Selected(); i >= 0 {
		return m.Options[i]
	}
	return ""
}

// SetSelected chooses option i without a key press: no ChangedMsg is
// delivered. An index outside Options chooses nothing.
func (m *Model) SetSelected(i int) {
	if i < 0 || i >= len(m.Options) {
		i = -1
	}
	m.selected = i
}

// Open reports whether the list is open.
func (m Model) Open() bool { return m.open }

// SetOpen opens or closes the list. Opening highlights the chosen option, or
// the first. A disabled dropdown, or one without options, stays closed.
func (m *Model) SetOpen(open bool) {
	if open && (m.Disabled || len(m.Options) == 0) {
		return
	}
	m.open = open
	m.typed = ""
	if open {
		m.cursor = max(m.selected, 0)
		m.scrollTo(m.cursor)
	}
}

// Focus gives the dropdown keyboard focus. It returns no Cmd; the result is
// there so a dropdown can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away and closes the list.
func (m *Model) Blur() {
	m.focused = false
	m.open = false
	m.typed = ""
}

// Focused reports whether the dropdown has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// start is the first option shown, kept inside Options in case they changed
// while the list was open.
func (m Model) start() int { return max(0, min(m.offset, len(m.Options)-m.rows())) }

func (m Model) rows() int {
	n := m.MaxRows
	if n <= 0 {
		n = defaultMaxRows
	}
	return min(n, len(m.Options))
}

// scrollTo moves the window so row i is inside it.
func (m *Model) scrollTo(i int) {
	rows := m.rows()
	if i < m.offset {
		m.offset = i
	}
	if i >= m.offset+rows {
		m.offset = i - rows + 1
	}
	m.offset = max(0, min(m.offset, len(m.Options)-rows))
}

func (m *Model) moveTo(i int) {
	if len(m.Options) == 0 {
		return
	}
	m.cursor = max(0, min(i, len(m.Options)-1))
	m.scrollTo(m.cursor)
}

// choose makes option i the chosen one and returns the Cmd that reports it, or
// nil when it already was.
func (m Model) choose(i int) (Model, tui.Cmd) {
	if i < 0 || i >= len(m.Options) || i == m.selected {
		return m, nil
	}
	m.selected = i
	msg := ChangedMsg{ID: m.ID, Index: i, Value: m.Options[i]}
	return m, func() tui.Msg { return msg }
}

// Update handles keys when the dropdown has focus, and the mouse when Mouse is
// on. A disabled dropdown ignores both. Open, it takes every key: the ones it
// does not use are dropped, so a form around it does not move on.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if m.Disabled {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if !m.focused {
		return m, nil
	}
	km := m.keys()
	if !m.open {
		switch {
		case keymap.Matches(msg, km.Open):
			m.SetOpen(true)
		case keymap.Matches(msg, km.Prev):
			return m.choose(max(m.Selected()-1, 0))
		case keymap.Matches(msg, km.Next):
			return m.choose(min(m.Selected()+1, len(m.Options)-1))
		}
		return m, nil
	}
	switch {
	case keymap.Matches(msg, km.Confirm):
		m.SetOpen(false)
		return m.choose(m.cursor)
	case keymap.Matches(msg, km.Cancel):
		m.SetOpen(false)
	case keymap.Matches(msg, km.Prev):
		m.moveTo(m.cursor - 1)
	case keymap.Matches(msg, km.Next):
		m.moveTo(m.cursor + 1)
	case keymap.Matches(msg, km.First):
		m.moveTo(0)
	case keymap.Matches(msg, km.Last):
		m.moveTo(len(m.Options) - 1)
	case keymap.Matches(msg, km.PageUp):
		m.moveTo(m.cursor - max(m.rows()-1, 1))
	case keymap.Matches(msg, km.PageDn):
		m.moveTo(m.cursor + max(m.rows()-1, 1))
	default:
		if k, ok := msg.(tui.Key); ok && k.Type == tui.KeyRunes && k.Mod == 0 && k.Text != "" {
			m.typeAhead(k.Text)
		}
	}
	return m, nil
}

// typeAhead moves the highlight to the next option that starts with the
// letters typed in the last second, ignoring case.
func (m *Model) typeAhead(text string) {
	t := now()
	if t.Sub(m.typedAt) > typeAheadReset {
		m.typed = ""
	}
	m.typedAt = t
	m.typed += text
	start := m.cursor
	if len([]rune(m.typed)) > 1 {
		start-- // a longer prefix may still match the highlighted option
	}
	for n := 1; n <= len(m.Options); n++ {
		i := ((start+n)%len(m.Options) + len(m.Options)) % len(m.Options)
		if strings.HasPrefix(strings.ToLower(m.Options[i]), strings.ToLower(m.typed)) {
			m.moveTo(i)
			return
		}
	}
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft {
		return m, nil
	}
	if m.Bounds.Contains(ev.X, ev.Y) {
		m.SetOpen(!m.open)
		return m, nil
	}
	if !m.open {
		return m, nil
	}
	row := ev.Y - (m.Bounds.Y + 1)
	if ev.X >= m.Bounds.X && ev.X < m.Bounds.X+m.Bounds.W && row >= 0 && row < m.rows() {
		m.SetOpen(false)
		return m.choose(m.start() + row)
	}
	m.SetOpen(false)
	return m, nil
}

func (m Model) label() string { return ansi.Sanitize(m.Label) }

func (m Model) placeholder() string {
	if m.Placeholder != "" {
		return ansi.Sanitize(m.Placeholder)
	}
	return "Select"
}

// valueWidth is the width of the value area: the widest option, the
// placeholder or Width.
func (m Model) valueWidth() int {
	w := max(m.MinWidth, ansi.Width(m.placeholder()))
	for _, o := range m.Options {
		w = max(w, ansi.Width(ansi.Sanitize(o)))
	}
	return w
}

func pad(s string, w int) string {
	if n := w - ansi.Width(s); n > 0 {
		return s + strings.Repeat(" ", n)
	}
	return s
}

// View renders "Label: value v" on one row, the value padded to the widest
// option so the control does not change size, with the chevron pointing right
// when closed and down when open. Focused, the value is in the focus style.
// Open, the options follow under the value, one per row, the chosen one marked
// and the highlighted one in the selection style; a list longer than MaxRows
// scrolls and ends with a row that counts the options out of view.
func (m Model) View() string {
	t := m.Theme
	st := t.ResolvedStates()
	g := t.GlyphSet()
	text := ansi.NewStyle().Foreground(t.Text)
	value, valueStyle := m.placeholder(), ansi.NewStyle().Foreground(t.Muted)
	if i := m.Selected(); i >= 0 {
		value, valueStyle = ansi.Sanitize(m.Options[i]), text
	}
	chevron := g.Collapsed
	if m.open {
		chevron = g.Expanded
	}
	switch {
	case m.Disabled:
		valueStyle, text = st.Disabled.Faint(), st.Disabled.Faint()
	case m.focused:
		valueStyle = st.Focus.Bold()
	}
	var b strings.Builder
	indent := ""
	if l := m.label(); l != "" {
		b.WriteString(text.Render(l + ": "))
		indent = strings.Repeat(" ", ansi.Width(l)+2)
	}
	b.WriteString(valueStyle.Render(pad(value, m.valueWidth())))
	b.WriteString(" " + valueStyle.Render(chevron))
	if m.open {
		for _, l := range m.list(text) {
			b.WriteString("\n" + indent + l)
		}
	}
	return b.String()
}

// list returns the open list's rows: the visible options, then a count of the
// ones out of view if there are any.
func (m Model) list(text ansi.Style) []string {
	t := m.Theme
	g := t.GlyphSet()
	rows, lo := m.rows(), m.start()
	out := make([]string, 0, rows+1)
	for i := lo; i < lo+rows; i++ {
		mark := "  "
		if i == m.selected {
			mark = g.Check + " "
		}
		style := text
		if i == m.cursor {
			style = t.ResolvedStates().Selected
		}
		out = append(out, style.Render(pad(mark+ansi.Sanitize(m.Options[i]), m.valueWidth()+2)))
	}
	var parts []string
	if lo > 0 {
		parts = append(parts, strconv.Itoa(lo)+" above")
	}
	if below := len(m.Options) - lo - rows; below > 0 {
		parts = append(parts, strconv.Itoa(below)+" below")
	}
	if len(parts) > 0 {
		out = append(out, ansi.NewStyle().Foreground(t.Muted).Render(strings.Join(parts, ", ")))
	}
	return out
}

// Popup returns the open list as a block of its own, without the control row:
// what View draws under the control. It is "" when the list is closed. Use it
// with popover to float the list over the page instead of pushing the page
// down.
func (m Model) Popup() string {
	if !m.open {
		return ""
	}
	return strings.Join(m.list(ansi.NewStyle().Foreground(m.Theme.Text)), "\n")
}

// Width is the number of cells the closed control takes.
func (m Model) Width() int {
	w := m.valueWidth() + 2 // a space and the chevron
	if l := m.label(); l != "" {
		w += ansi.Width(l) + 2
	}
	return w
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the dropdown as plain text for accessible output (see
// tui.Linearizer): "Plan, dropdown, Pro, 2 of 3", with ", expanded" when open
// and ", unavailable" when disabled.
func (m Model) Linearize() string {
	out := "dropdown"
	if l := m.label(); l != "" {
		out = l + ", " + out
	}
	if i := m.Selected(); i >= 0 {
		out += ", " + ansi.Sanitize(m.Options[i]) + ", " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Options))
	} else {
		out += ", " + m.placeholder()
	}
	if m.open {
		out += ", expanded"
	}
	if m.Disabled {
		out += ", unavailable"
	}
	return out
}
