// Package multiselect is a multi-choice list widget — InkUI's
// "MultiSelect": a cursor moves between items, Space toggles the item
// under it, and Enter confirms the whole set of checked items.
package multiselect

import (
	"sort"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Item is one entry in a Model's list.
type Item struct {
	Label string
	Value string
}

// Model is a keyboard-navigable multi-choice list. Like picker.Model, it
// renders all of its Items with no internal scrolling.
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
	// cursor to it (it does not toggle). Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the widget; mouse
	// events outside it are ignored. The app sets it each frame it moves.
	Bounds hittest.Rect
	// WheelStep is the rows the cursor moves per wheel notch; 0 means 3.
	WheelStep int

	cursor   int
	selected map[int]bool
}

// New builds a Model from items, all initially unselected.
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

// KeyMap is the set of keys Update reacts to.
type KeyMap struct {
	Up      keymap.Binding
	Down    keymap.Binding
	Top     keymap.Binding
	Bottom  keymap.Binding
	Toggle  keymap.Binding
	Confirm keymap.Binding
}

// DefaultKeyMap returns the default bindings: up/down, home/end, space to
// toggle, enter to confirm.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:      keymap.NewBinding("up", "up"),
		Down:    keymap.NewBinding("down", "down"),
		Top:     keymap.NewBinding("first item", "home"),
		Bottom:  keymap.NewBinding("last item", "end"),
		Toggle:  keymap.NewBinding("toggle", "space"),
		Confirm: keymap.NewBinding("confirm", "enter"),
	}
}

// isZero reports whether no binding in k has any key.
func (k KeyMap) isZero() bool {
	return len(k.Up.Keys)+len(k.Down.Keys)+len(k.Top.Keys)+len(k.Bottom.Keys)+len(k.Toggle.Keys)+len(k.Confirm.Keys) == 0
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
	return []keymap.Binding{k.Up, k.Down, k.Top, k.Bottom, k.Toggle, k.Confirm}
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

// IsSelected reports whether item i is checked.
func (m Model) IsSelected(i int) bool { return m.selected[i] }

// Toggle flips whether item i is selected. It always replaces m.selected
// with a fresh map rather than mutating the existing one in place: Model
// is used with value semantics (Update returns a new Model each call, the
// same convention textinput/viewport/picker follow), but a map field's
// header copies by value while its underlying table does not — mutating
// it in place would silently corrupt any other Model value that still
// shares that same map, e.g. a caller that kept the pre-Update value
// around. Copy-on-write here keeps every Model value's selection
// independent, the same guarantee the rest of this codebase already gets
// for free from immutable types.
func (m *Model) Toggle(i int) {
	next := make(map[int]bool, len(m.selected)+1)
	for k, v := range m.selected {
		next[k] = v
	}
	next[i] = !next[i]
	m.selected = next
}

// SelectedItems returns the checked items, in list order.
func (m Model) SelectedItems() []Item {
	var out []Item
	for _, i := range m.SelectedIndexes() {
		out = append(out, m.Items[i])
	}
	return out
}

// SelectedIndexes returns the checked indexes in ascending order — map
// iteration order is randomized in Go, so this sorts explicitly rather
// than leaving callers to discover that the hard way.
func (m Model) SelectedIndexes() []int {
	out := make([]int, 0, len(m.selected))
	for i, on := range m.selected {
		if on {
			out = append(out, i)
		}
	}
	sort.Ints(out)
	return out
}

// ConfirmedMsg is delivered (via the Cmd Update returns) when the current
// selection is confirmed with Enter.
type ConfirmedMsg struct {
	Items []Item
}

// Update moves the cursor on the Up/Down/Top/Bottom bindings, toggles the
// item under it on Toggle, and confirms the selection on Confirm, returning
// a Cmd that delivers ConfirmedMsg. With Mouse on it also handles
// tui.MouseEvent (see Model.Mouse). Any other Msg is a no-op.
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
	case keymap.Matches(msg, k.Toggle):
		if len(m.Items) > 0 {
			m.Toggle(m.cursor)
		}
	case keymap.Matches(msg, k.Confirm):
		items := m.SelectedItems()
		return m, func() tui.Msg { return ConfirmedMsg{Items: items} }
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

// View renders the item list with a cursor marker and a checkbox per item.
func (m Model) View() string {
	var b strings.Builder
	cursorStyle := m.themed().ResolvedStates().Selected.Bold()
	checkStyle := ansi.NewStyle().Foreground(m.themed().Success)
	for i, item := range m.Items {
		prefix := "  "
		if i == m.cursor {
			prefix = cursorStyle.Render("> ")
		}
		check := "[ ]"
		if m.selected[i] {
			check = checkStyle.Render("[x]")
		}
		b.WriteString(prefix + check + " " + item.Label)
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
// tui.Linearizer): one line per item with its label, "checked" or "not
// checked", its position, and ", current" on the cursor row, e.g. "Milk,
// checked, item 1 of 3, current". No checkbox glyphs or styling. An empty
// list reads "No items".
func (m Model) Linearize() string {
	if len(m.Items) == 0 {
		return "No items"
	}
	lines := make([]string, len(m.Items))
	for i, item := range m.Items {
		state := "not checked"
		if m.selected[i] {
			state = "checked"
		}
		lines[i] = item.Label + ", " + state + ", item " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Items))
		if i == m.cursor {
			lines[i] += ", current"
		}
	}
	return strings.Join(lines, "\n")
}
