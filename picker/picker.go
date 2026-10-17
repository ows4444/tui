// Package picker is a single-choice list widget — InkUI's "Select",
// renamed because 'select' is a Go keyword and can't be a package name.
package picker

import (
	"github.com/ows4444/tui/internal/typeahead"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/fuzzy"
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

	// Filter, when true, lets the person type to narrow the list: printable
	// keys that no binding in KeyMap uses are added to a query, Backspace
	// removes the last rune and Esc clears the query. Items that fuzzy-match
	// the query (see package fuzzy) stay, best first, with the matched runes
	// underlined; the cursor moves to the best match as the query changes.
	// Off (the default), the list is unchanged. A first line shows the query.
	Filter bool
	// TypeAhead, when true, makes typing letters jump to the next item whose
	// label starts with them, ignoring case: "b" finds the next one that starts with
	// b, "bl" the next that starts with bl, the same letter again goes on to the
	// one after, and a pause of a second starts a new word. Keys that a binding in
	// KeyMap uses keep their meaning, so a letter some binding uses cannot start a word. Off (the default), letters do
	// nothing.
	TypeAhead bool

	cursor int
	query  string
	ta     typeahead.State
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

// Query returns the text the list is filtered by; always empty when Filter
// is off.
func (m Model) Query() string { return m.query }

// SetQuery sets the filter text and puts the cursor on the best match. It has
// no effect when Filter is off.
func (m *Model) SetQuery(q string) {
	if !m.Filter {
		return
	}
	m.query = q
	if v := m.visible(); len(v) > 0 {
		m.cursor = v[0].Index
	}
}

// visible lists the items that show, in order: every item when there is no
// query, the fuzzy matches best first otherwise.
func (m Model) visible() []fuzzy.Result {
	if !m.Filter || m.query == "" {
		out := make([]fuzzy.Result, len(m.Items))
		for i := range out {
			out[i].Index = i
		}
		return out
	}
	return fuzzy.Find(m.query, len(m.Items), func(i int) string { return m.Items[i].Label })
}

// rowOf is the row within vis that holds the cursor item, or -1.
func (m Model) rowOf(vis []fuzzy.Result) int {
	for i, r := range vis {
		if r.Index == m.cursor {
			return i
		}
	}
	return -1
}

// moveTo puts the cursor on row r of vis, clamped; with nothing visible it
// does nothing.
func (m *Model) moveTo(vis []fuzzy.Result, r int) {
	if len(vis) == 0 {
		return
	}
	m.cursor = vis[clamp(r, 0, len(vis)-1)].Index
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
	vis := m.visible()
	row := m.rowOf(vis)
	switch {
	case keymap.Matches(msg, k.Up):
		if row > 0 {
			m.moveTo(vis, row-1)
		} else if row < 0 {
			m.moveTo(vis, 0)
		}
	case keymap.Matches(msg, k.Down):
		m.moveTo(vis, row+1)
	case keymap.Matches(msg, k.Top):
		m.moveTo(vis, 0)
	case keymap.Matches(msg, k.Bottom):
		m.moveTo(vis, len(vis)-1)
	case keymap.Matches(msg, k.Select):
		if len(vis) == 0 || row < 0 {
			return m, nil
		}
		idx, item := m.cursor, m.Items[m.cursor]
		return m, func() tui.Msg { return SelectedMsg{Index: idx, Item: item} }
	case m.Filter:
		return m.editQuery(msg.(tui.Key)), nil
	case m.TypeAhead:
		if text, ok := typeahead.Printable(msg); ok {
			prefix, from := m.ta.Feed(text, m.cursor)
			if i, ok := typeahead.Find(prefix, from, len(m.Items), func(i int) string { return m.Items[i].Label }); ok {
				m.cursor = i
			}
		}
	}
	return m, nil
}

// editQuery applies a key to the filter text.
func (m Model) editQuery(k tui.Key) Model {
	q := m.query
	switch {
	case k.Type == tui.KeyEsc:
		q = ""
	case k.Type == tui.KeyBackspace:
		if r := []rune(q); len(r) > 0 {
			q = string(r[:len(r)-1])
		}
	case k.Type == tui.KeySpace:
		q += " "
	case k.Type == tui.KeyRunes && !k.Mod.Alt() && k.Text != "":
		q += k.Text
	default:
		return m
	}
	if q != m.query {
		m.SetQuery(q)
	}
	return m
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
	vis := m.visible()
	row := m.rowOf(vis)
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.moveTo(vis, row-step)
	case tui.MouseButtonWheelDown:
		m.moveTo(vis, row+step)
	case tui.MouseButtonLeft:
		_, y := m.Bounds.Local(ev.X, ev.Y)
		if m.Filter {
			y-- // the query line
		}
		if y >= 0 && y < len(vis) {
			m.moveTo(vis, y)
		}
	}
	return m
}

// View renders the item list with the highlighted item marked by a cursor.
// With Filter on, a first line shows the query and only the matching items
// follow, matched runes underlined.
func (m Model) View() string {
	var b strings.Builder
	sel := m.themed().ResolvedStates().Selected.Bold()
	plain := ansi.NewStyle()
	if m.Filter {
		b.WriteString("/ " + m.query)
		if m.query != "" {
			b.WriteString(plain.Faint().Render(" (" + strconv.Itoa(len(m.visible())) + ")"))
		}
		b.WriteByte('\n')
	}
	vis := m.visible()
	if m.Filter && len(vis) == 0 {
		b.WriteString("  " + plain.Faint().Render("no matches"))
		return b.String()
	}
	for n, r := range vis {
		prefix, base := "  ", plain
		if r.Index == m.cursor {
			prefix, base = "> ", sel
		}
		b.WriteString(prefix)
		for _, run := range fuzzy.Split(m.Items[r.Index].Label, r.Positions) {
			st := base
			if run.Match {
				st = base.Underline()
			}
			if r.Index == m.cursor || run.Match {
				b.WriteString(st.Render(run.Text))
			} else {
				b.WriteString(run.Text)
			}
		}
		if n < len(vis)-1 {
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
	vis := m.visible()
	var head []string
	if m.Filter && m.query != "" {
		if len(vis) == 0 {
			return "Filter " + strconv.Quote(m.query) + ", no matches"
		}
		head = []string{"Filter " + strconv.Quote(m.query) + ", " + strconv.Itoa(len(vis)) + " of " + strconv.Itoa(len(m.Items)) + " items"}
	}
	lines := make([]string, len(vis))
	for i, r := range vis {
		lines[i] = m.Items[r.Index].Label + ", item " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(vis))
		if r.Index == m.cursor {
			lines[i] += ", selected"
		}
	}
	return strings.Join(append(head, lines...), "\n")
}
