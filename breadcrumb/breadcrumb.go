// Package breadcrumb is a trail of the places that lead to the current one,
// "Home / Docs / Guide", that can be walked back along: Left and Right move
// a cursor between the places, and Enter or Space chooses the one it is on.
// With the mouse on, so does a click.
//
// Choosing reports the place and changes nothing: the caller owns the trail
// and shortens it, or goes elsewhere, as it sees fit.
//
// Stability: experimental. Its API may change in any minor release.
package breadcrumb

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one trail. The zero value is an empty trail; build one with New.
type Model struct {
	// ID is carried by ChosenMsg, to tell one trail's choice from another's.
	ID string
	// Items are the places, from the root to the current one, which is the
	// last.
	Items []string
	// Separator stands between two places. Empty is read as "/".
	Separator string
	// Max, when above 1, is the most places drawn: a longer trail keeps its
	// first place and its last Max-1, with an ellipsis for the ones between.
	// The folded places cannot be chosen.
	Max   int
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// a place chooses it. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the trail.
	Bounds hittest.Rect

	focused bool
	cursor  int  // index into Items
	moved   bool // the cursor was put somewhere; until then it is on the last place
}

// New returns a trail of the given places, the last being the current one.
func New(items ...string) Model {
	return Model{Items: items, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Prev   keymap.Binding // move to the place before
	Next   keymap.Binding // move to the place after
	First  keymap.Binding // move to the first place
	Last   keymap.Binding // move to the last place
	Choose keymap.Binding // choose the place under the cursor
}

// DefaultKeyMap returns Left and Right, Home and End, and Enter or Space.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Prev:   keymap.NewBinding("previous", "left"),
		Next:   keymap.NewBinding("next", "right"),
		First:  keymap.NewBinding("first", "home"),
		Last:   keymap.NewBinding("last", "end"),
		Choose: keymap.NewBinding("go", "enter", "space"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Prev.Keys)+len(km.Next.Keys)+len(km.First.Keys)+len(km.Last.Keys)+len(km.Choose.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the trail honours, with descriptions, for
// help text.
func (m Model) Bindings() []keymap.Binding {
	km := m.keys()
	return []keymap.Binding{km.Prev, km.Next, km.First, km.Last, km.Choose}
}

// ChosenMsg is delivered (via the Cmd Update returns) when a place is
// chosen, by key or by pointer. Choosing the last place, the current one, is
// reported like any other.
type ChosenMsg struct {
	// ID is the trail's ID.
	ID string
	// Index is the chosen place's position in Items.
	Index int
	// Label is the chosen place's text.
	Label string
}

// visible returns the indexes of Items that are drawn, in order, with -1
// standing for the ellipsis that folds the ones between.
func (m Model) visible() []int {
	n := len(m.Items)
	if m.Max <= 1 || n <= m.Max {
		out := make([]int, n)
		for i := range out {
			out[i] = i
		}
		return out
	}
	out := []int{0, -1}
	for i := n - (m.Max - 1); i < n; i++ {
		out = append(out, i)
	}
	return out
}

// Cursor returns the index in Items of the place the cursor is on: the last
// place until a key or a click has moved it. It is -1 for an empty trail.
func (m Model) Cursor() int {
	if len(m.Items) == 0 {
		return -1
	}
	if !m.moved {
		return len(m.Items) - 1
	}
	return min(max(m.cursor, 0), len(m.Items)-1)
}

// SetCursor moves the cursor to place i. An index that is out of range, or
// names a place folded into the ellipsis, is ignored.
func (m *Model) SetCursor(i int) {
	for _, v := range m.visible() {
		if v == i && i >= 0 {
			m.cursor, m.moved = i, true
		}
	}
}

// Focus gives the trail keyboard focus. It returns no Cmd; the result is
// there so a trail can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the trail has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// step moves the cursor by dir places among the ones drawn, stopping at
// either end.
func (m Model) step(dir int) Model {
	var places []int
	for _, v := range m.visible() {
		if v >= 0 {
			places = append(places, v)
		}
	}
	at := 0
	for k, v := range places {
		if v == m.Cursor() {
			at = k
		}
	}
	at = min(max(at+dir, 0), len(places)-1)
	m.cursor, m.moved = places[at], true
	return m
}

func (m Model) choose(i int) (Model, tui.Cmd) {
	m.cursor, m.moved = i, true
	msg := ChosenMsg{ID: m.ID, Index: i, Label: m.Items[i]}
	return m, func() tui.Msg { return msg }
}

// Update moves the cursor on Left, Right, Home and End, stopping at the
// ends, and chooses the place under it on Enter or Space, when the trail has
// focus. With Mouse on, a left press on a place moves the cursor to it and
// chooses it.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if len(m.Items) == 0 {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft && m.Bounds.Contains(ev.X, ev.Y) {
			if i := m.itemAt(ev.X - m.Bounds.X); i >= 0 {
				return m.choose(i)
			}
		}
		return m, nil
	}
	if !m.focused {
		return m, nil
	}
	km := m.keys()
	n := len(m.Items)
	switch {
	case keymap.Matches(msg, km.Prev):
		return m.step(-1), nil
	case keymap.Matches(msg, km.Next):
		return m.step(1), nil
	case keymap.Matches(msg, km.First):
		return m.step(-n), nil
	case keymap.Matches(msg, km.Last):
		return m.step(n), nil
	case keymap.Matches(msg, km.Choose):
		return m.choose(m.Cursor())
	}
	return m, nil
}

func (m Model) separator() string {
	if s := ansi.Sanitize(m.Separator); s != "" {
		return s
	}
	return "/"
}

// text is what stands for entry v of visible: the place's label, or the
// ellipsis.
func (m Model) text(v int) string {
	if v < 0 {
		return m.themed().GlyphSet().Ellipsis
	}
	return ansi.Sanitize(m.Items[v])
}

// itemAt returns the index in Items of the place drawn at local column x,
// or -1 for a separator, the ellipsis, or a column outside the trail. Each
// entry owns its text and the cell either side of it.
func (m Model) itemAt(x int) int {
	left := 0
	sep := ansi.Width(m.separator())
	for k, v := range m.visible() {
		if k > 0 {
			left += sep
		}
		w := ansi.Width(m.text(v)) + 2
		if x >= left && x < left+w {
			return v
		}
		left += w
	}
	return -1
}

// Width is the number of cells View takes: each entry with a cell either
// side of it, and the separators between.
func (m Model) Width() int {
	vis := m.visible()
	if len(vis) == 0 {
		return 0
	}
	w := ansi.Width(m.separator()) * (len(vis) - 1)
	for _, v := range vis {
		w += ansi.Width(m.text(v)) + 2
	}
	return w
}

// View renders the places on one row with the separator between them, each
// with a blank cell either side: " Home / Docs / Guide ". The last place,
// the current one, is bold. The place under the cursor of a trail that has
// focus takes angle brackets in its two cells, " Home /<Docs>/ Guide ", so
// focus reads without colour and does not change the width. An empty trail
// renders "".
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	item := ansi.NewStyle().Foreground(t.Text)
	current := ansi.NewStyle().Bold().Foreground(t.Primary)
	muted := ansi.NewStyle().Foreground(t.Muted)
	vis := m.visible()
	var b strings.Builder
	for k, v := range vis {
		if k > 0 {
			b.WriteString(muted.Render(m.separator()))
		}
		open, shut, style := " ", " ", item
		switch {
		case v < 0:
			style = muted
		case m.focused && v == m.Cursor():
			open, shut, style = "<", ">", st.Focus.Bold()
		case v == len(m.Items)-1:
			style = current
		}
		b.WriteString(style.Render(open + m.text(v) + shut))
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the trail as plain text for accessible output (see
// tui.Linearizer): one line per place with its position, and ", current" on
// the last, e.g. "Docs, place 2 of 3". Every place is listed, the folded
// ones too. An empty trail reads "No places".
func (m Model) Linearize() string {
	n := len(m.Items)
	if n == 0 {
		return "No places"
	}
	lines := make([]string, n)
	for i := range m.Items {
		lines[i] = m.text(i) + ", place " + strconv.Itoa(i+1) + " of " + strconv.Itoa(n)
		if i == n-1 {
			lines[i] += ", current"
		}
	}
	return strings.Join(lines, "\n")
}
