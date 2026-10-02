// Package picker is a single-choice list widget — InkUI's "Select",
// renamed because 'select' is a Go keyword and can't be a package name.
package picker

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Item is one entry in a Model's list. Value is kept separate from Label
// in case a caller wants a user-facing label distinct from an underlying
// id/value; when they're the same, use NewStrings instead of building
// Items by hand.
type Item struct {
	Label string
	Value string
}

// Model is a keyboard-navigable single-choice list. It renders all of its
// Items with no internal scrolling — for a list too long to fit on
// screen, compose it inside a viewport.Model instead of expecting Model
// to scroll itself.
type Model struct {
	Items []Item
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the key bindings Update obeys. New and NewStrings fill it
	// with DefaultKeyMap; a Model built as a literal with no bindings set
	// behaves as if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: the wheel moves
	// the cursor by WheelStep rows and a left click on a row moves the
	// cursor to it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the widget; mouse
	// events outside it are ignored. The app sets it each frame it moves.
	Bounds hittest.Rect
	// WheelStep is the rows the cursor moves per wheel notch; 0 means 3.
	WheelStep int

	cursor int
}

// KeyMap is the set of keys Update reacts to.
type KeyMap struct {
	Up     keymap.Binding
	Down   keymap.Binding
	Top    keymap.Binding
	Bottom keymap.Binding
	Select keymap.Binding
}

// DefaultKeyMap returns the default bindings: up/down, home/end, enter.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     keymap.NewBinding("up", "up"),
		Down:   keymap.NewBinding("down", "down"),
		Top:    keymap.NewBinding("first item", "home"),
		Bottom: keymap.NewBinding("last item", "end"),
		Select: keymap.NewBinding("select", "enter"),
	}
}

// isZero reports whether no binding in k has any key.
func (k KeyMap) isZero() bool {
	return len(k.Up.Keys)+len(k.Down.Keys)+len(k.Top.Keys)+len(k.Bottom.Keys)+len(k.Select.Keys) == 0
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	if m.KeyMap.isZero() {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the active bindings, for help text.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	return []keymap.Binding{k.Up, k.Down, k.Top, k.Bottom, k.Select}
}

// New builds a Model from items, with the cursor on the first one.
func New(items ...Item) Model {
	return Model{Items: items, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// NewStrings builds a Model whose Items each have an equal Label and Value.
func NewStrings(items ...string) Model {
	its := make([]Item, len(items))
	for i, s := range items {
		its[i] = Item{Label: s, Value: s}
	}
	return Model{Items: its, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// Cursor returns the index of the item currently under the cursor.
func (m Model) Cursor() int { return m.cursor }

// SetCursor moves the cursor to i, clamped to a valid item index (or 0
// with no items).
func (m *Model) SetCursor(i int) {
	if len(m.Items) == 0 {
		m.cursor = 0
		return
	}
	m.cursor = clamp(i, 0, len(m.Items)-1)
}

// Highlighted returns the item currently under the cursor, or the zero
// Item if the list is empty.
func (m Model) Highlighted() Item {
	if len(m.Items) == 0 {
		return Item{}
	}
	return m.Items[m.cursor]
}

// SelectedMsg is delivered (via the Cmd Update returns) when the
// highlighted item is confirmed with Enter.
type SelectedMsg struct {
	Index int
	Item  Item
}

// Update moves the cursor on the Up/Down/Top/Bottom bindings and confirms
// the highlighted item on Select, returning a Cmd that delivers SelectedMsg.
// With Mouse on it also handles tui.MouseEvent (see Model.Mouse). Any other
// Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if ev, ok := msg.(tui.MouseEvent); ok {
		return m.updateMouse(ev), nil
	}
	if _, ok := msg.(tui.Key); !ok {
		return m, nil
	}

	k := m.keys()
	switch {
	case keymap.Matches(msg, k.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keymap.Matches(msg, k.Down):
		if m.cursor < len(m.Items)-1 {
			m.cursor++
		}
	case keymap.Matches(msg, k.Top):
		m.cursor = 0
	case keymap.Matches(msg, k.Bottom):
		if len(m.Items) > 0 {
			m.cursor = len(m.Items) - 1
		}
	case keymap.Matches(msg, k.Select):
		if len(m.Items) == 0 {
			return m, nil
		}
		idx, item := m.cursor, m.Items[m.cursor]
		return m, func() tui.Msg { return SelectedMsg{Index: idx, Item: item} }
	}
	return m, nil
}

// updateMouse applies a mouse event when Mouse is on and it is in Bounds.
func (m Model) updateMouse(ev tui.MouseEvent) Model {
	if !m.Mouse || !m.Bounds.Contains(ev.X, ev.Y) || ev.Action != tui.MouseActionPress {
		return m
	}
	step := m.WheelStep
	if step <= 0 {
		step = 3
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.SetCursor(m.cursor - step)
	case tui.MouseButtonWheelDown:
		m.SetCursor(m.cursor + step)
	case tui.MouseButtonLeft:
		_, row := m.Bounds.Local(ev.X, ev.Y)
		if row < len(m.Items) {
			m.SetCursor(row)
		}
	}
	return m
}

// View renders the item list with the highlighted item marked by a cursor.
func (m Model) View() string {
	var b strings.Builder
	cursorStyle := m.themed().ResolvedStates().Selected.Bold()
	for i, item := range m.Items {
		prefix := "  "
		label := item.Label
		if i == m.cursor {
			prefix = "> "
			label = cursorStyle.Render(label)
		}
		b.WriteString(prefix + label)
		if i < len(m.Items)-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the list as plain text for accessible output (see
// tui.Linearizer): one line per item with its label and position, and
// ", selected" on the cursor row, e.g. "Blue, item 2 of 3, selected". No
// cursor marker or styling. An empty list reads "No items".
func (m Model) Linearize() string {
	if len(m.Items) == 0 {
		return "No items"
	}
	lines := make([]string, len(m.Items))
	for i, item := range m.Items {
		lines[i] = item.Label + ", item " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Items))
		if i == m.cursor {
			lines[i] += ", selected"
		}
	}
	return strings.Join(lines, "\n")
}
