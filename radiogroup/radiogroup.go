// Package radiogroup is a set of options of which exactly one can be chosen.
// The arrow keys move between the options and choose as they go, so the
// option the cursor is on is the chosen one; Space or Enter chooses it when
// nothing is chosen yet.
//
// Stability: experimental. Its API may change in any minor release.
package radiogroup

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Option is one choice of a group.
type Option struct {
	// Label is the text shown.
	Label string
	// Value is what Model.Value returns when this option is chosen; an empty
	// Value stands for the Label.
	Value string
	// Disabled stops the option being chosen; the cursor skips it.
	Disabled bool
}

func (o Option) value() string {
	if o.Value != "" {
		return o.Value
	}
	return o.Label
}

// Model is a group of radio options. The zero value is an empty group; build
// one with New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one group's change from another's.
	ID      string
	Options []Option
	// Horizontal draws the options on one row, Gap cells apart, and not one
	// under another.
	Horizontal bool
	// Gap is the number of blank cells between two options of a Horizontal
	// group.
	Gap   int
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// an option chooses it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the group.
	Bounds hittest.Rect

	focused bool
	cursor  int
	chosen  int // index + 1 of the chosen option; 0 for none
}

// New returns a vertical group with one option per label and nothing chosen.
func New(labels ...string) Model {
	m := Model{Gap: 2, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
	for _, l := range labels {
		m.Options = append(m.Options, Option{Label: l})
	}
	return m
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Prev   keymap.Binding // move to, and choose, the option before
	Next   keymap.Binding // move to, and choose, the option after
	First  keymap.Binding // move to, and choose, the first option
	Last   keymap.Binding // move to, and choose, the last option
	Choose keymap.Binding // choose the option under the cursor
}

// DefaultKeyMap returns the arrow keys, Home, End, and Space or Enter.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Prev:   keymap.NewBinding("previous", "up", "left"),
		Next:   keymap.NewBinding("next", "down", "right"),
		First:  keymap.NewBinding("first", "home"),
		Last:   keymap.NewBinding("last", "end"),
		Choose: keymap.NewBinding("choose", "space", "enter"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Prev.Keys)+len(km.Next.Keys)+len(km.First.Keys)+len(km.Last.Keys)+len(km.Choose.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the group honours, with descriptions, for
// help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return []keymap.Binding{km.Prev, km.Next, km.First, km.Last, km.Choose}
}

// ChangedMsg is delivered (via the Cmd Update returns) when a different
// option is chosen.
type ChangedMsg struct {
	// ID is the group's ID.
	ID string
	// Index is the chosen option's place in Options.
	Index int
	// Value is the chosen option's Value, or its Label when that is empty.
	Value string
}

func (m Model) usable(i int) bool {
	return i >= 0 && i < len(m.Options) && !m.Options[i].Disabled
}

// step returns the first usable option from, and not counting, index from in
// direction dir, wrapping round; from itself when there is no other.
func (m Model) step(from, dir int) int {
	n := len(m.Options)
	for k := 1; k <= n; k++ {
		if i := ((from+dir*k)%n + n) % n; m.usable(i) {
			return i
		}
	}
	return from
}

// Focus gives the group keyboard focus, with the cursor on the chosen option
// if there is one. It returns no Cmd; the result is there so a group can
// stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	if m.chosen > 0 {
		m.cursor = m.chosen - 1
	} else if !m.usable(m.cursor) && len(m.Options) > 0 {
		m.cursor = m.step(-1, 1)
	}
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the group has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// Cursor returns the index of the option the cursor is on.
func (m Model) Cursor() int { return m.cursor }

// Index returns the chosen option's place in Options, and false when nothing
// is chosen.
func (m Model) Index() (int, bool) { return m.chosen - 1, m.chosen > 0 }

// Value returns the chosen option's Value, or its Label when that is empty;
// "" when nothing is chosen.
func (m Model) Value() string {
	if m.chosen == 0 || m.chosen > len(m.Options) {
		return ""
	}
	return m.Options[m.chosen-1].value()
}

// Select chooses option i without a key press: no ChangedMsg is delivered.
// An index that is out of range, or names a disabled option, is ignored.
func (m *Model) Select(i int) {
	if m.usable(i) {
		m.cursor, m.chosen = i, i+1
	}
}

// SelectValue chooses the first option whose Value (or Label, when its Value
// is empty) is v, and reports whether there was one.
func (m *Model) SelectValue(v string) bool {
	for i, o := range m.Options {
		if o.value() == v && m.usable(i) {
			m.Select(i)
			return true
		}
	}
	return false
}

// choose moves the cursor to option i and chooses it, returning the Cmd that
// reports the change, or nil when i was already chosen.
func (m Model) choose(i int) (Model, tui.Cmd) {
	if !m.usable(i) {
		return m, nil
	}
	m.cursor = i
	if m.chosen == i+1 {
		return m, nil
	}
	m.chosen = i + 1
	msg := ChangedMsg{ID: m.ID, Index: i, Value: m.Options[i].value()}
	return m, func() tui.Msg { return msg }
}

// Update moves the cursor on the arrow keys, Home and End, skipping disabled
// options and wrapping at the ends, and chooses the option it lands on.
// Space or Enter chooses the option under the cursor. With Mouse on, a left
// press on an option chooses it. Only a group that has focus acts on keys.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev)
	}
	if !m.focused || len(m.Options) == 0 {
		return m, nil
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Prev):
		return m.choose(m.step(m.cursor, -1))
	case keymap.Matches(msg, km.Next):
		return m.choose(m.step(m.cursor, 1))
	case keymap.Matches(msg, km.First):
		return m.choose(m.step(-1, 1))
	case keymap.Matches(msg, km.Last):
		return m.choose(m.step(len(m.Options), -1))
	case keymap.Matches(msg, km.Choose):
		return m.choose(m.cursor)
	}
	return m, nil
}

// optionAt returns the option drawn at local cell (x, y), or -1.
func (m Model) optionAt(x, y int) int {
	if !m.Horizontal {
		if y >= 0 && y < len(m.Options) && x < m.optionWidth(y) {
			return y
		}
		return -1
	}
	if y != 0 {
		return -1
	}
	left := 0
	for i := range m.Options {
		if w := m.optionWidth(i); x >= left && x < left+w {
			return i
		} else {
			left += w + m.gap()
		}
	}
	return -1
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.Mouse || ev.Action != tui.MouseActionPress || ev.Button != tui.MouseButtonLeft ||
		!m.Bounds.Contains(ev.X, ev.Y) {
		return m, nil
	}
	lx, ly := m.Bounds.Local(ev.X, ev.Y)
	return m.choose(m.optionAt(lx, ly))
}

func (m Model) gap() int {
	if m.Gap < 0 {
		return 0
	}
	return m.Gap
}

func (m Model) label(i int) string { return ansi.Sanitize(m.Options[i].Label) }

// optionWidth is the number of cells option i takes: its mark, a space and
// its label.
func (m Model) optionWidth(i int) int { return 4 + ansi.Width(m.label(i)) }

// option renders option i as "(●) Label" when chosen and "( ) Label" when
// not, so the choice reads without colour. The option under the cursor of a
// focused group is drawn with angle brackets, "<●> Label", in the focus
// style; a disabled one is dimmed and marked "(-)".
func (m Model) option(i int) string {
	t := m.themed()
	st := t.ResolvedStates()
	g := t.GlyphSet()
	mark := " "
	if m.chosen == i+1 {
		mark = g.Dot
	}
	open, shut := "(", ")"
	style := ansi.NewStyle().Foreground(t.Text)
	switch {
	case m.Options[i].Disabled:
		if mark == " " {
			mark = g.Dash
		}
		style = st.Disabled
	case m.focused && m.cursor == i:
		open, shut = "<", ">"
		style = st.Focus.Bold()
	case m.chosen == i+1:
		style = ansi.NewStyle().Foreground(t.Primary)
	}
	return style.Render(open + mark + shut + " " + m.label(i))
}

// View renders the options one under another, or on one row when Horizontal.
func (m Model) View() string {
	parts := make([]string, len(m.Options))
	for i := range m.Options {
		parts[i] = m.option(i)
	}
	if m.Horizontal {
		return strings.Join(parts, strings.Repeat(" ", m.gap()))
	}
	return strings.Join(parts, "\n")
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the group as plain text for accessible output (see
// tui.Linearizer): one line per option with its position, ", chosen" on the
// chosen one and ", unavailable" on a disabled one, e.g. "Medium, radio
// button 2 of 3, chosen". An empty group reads "No options".
func (m Model) Linearize() string {
	if len(m.Options) == 0 {
		return "No options"
	}
	lines := make([]string, len(m.Options))
	for i, o := range m.Options {
		l := m.label(i) + ", radio button " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Options))
		if m.chosen == i+1 {
			l += ", chosen"
		}
		if o.Disabled {
			l += ", unavailable"
		}
		lines[i] = l
	}
	return strings.Join(lines, "\n")
}
