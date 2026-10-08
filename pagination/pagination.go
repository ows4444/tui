// Package pagination is a row of page numbers with the current one marked:
// Left and Right go to the page before and after, Home and End to the first
// and last, and with the mouse on a click on a number goes to that page.
// Pages far from the current one are folded into an ellipsis, so the row
// stays short however many pages there are.
//
// Stability: experimental. Its API may change in any minor release.
package pagination

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Model is one pager. The zero value is not ready to use; build one with
// New.
type Model struct {
	// ID is carried by ChangedMsg, to tell one pager's change from
	// another's.
	ID string
	// Total is the number of pages. With a Total under 1 there is nothing
	// to draw.
	Total int
	// Siblings is how many pages are shown on each side of the current one,
	// besides the first and the last, which are always shown. A negative
	// Siblings is read as 0.
	Siblings int
	// Disabled stops the page being changed; the pager is drawn faint.
	Disabled bool
	Theme    theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// KeyMap holds the keys for each action. New fills it with DefaultKeyMap;
	// a Model built as a struct literal with a zero KeyMap behaves as if it
	// held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on
	// a page number goes to that page. Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the pager.
	Bounds hittest.Rect

	page    int // 1-based; 0 is read as 1
	focused bool
}

// New returns a pager over total pages, on page 1, showing one page either
// side of the current one.
func New(total int) Model {
	return Model{Total: total, Siblings: 1, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	Prev  keymap.Binding // go to the page before
	Next  keymap.Binding // go to the page after
	First keymap.Binding // go to the first page
	Last  keymap.Binding // go to the last page
}

// DefaultKeyMap returns Left and Right, Home and End.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Prev:  keymap.NewBinding("previous page", "left"),
		Next:  keymap.NewBinding("next page", "right"),
		First: keymap.NewBinding("first page", "home"),
		Last:  keymap.NewBinding("last page", "end"),
	}
}

func (m Model) keys() KeyMap {
	km := m.KeyMap
	if len(km.Prev.Keys)+len(km.Next.Keys)+len(km.First.Keys)+len(km.Last.Keys) == 0 {
		return DefaultKeyMap()
	}
	return km
}

// Bindings returns the actions the pager honours, with descriptions, for
// help text. A disabled pager has none.
func (m Model) Bindings() []keymap.Binding {
	if m.Disabled {
		return nil
	}
	km := m.keys()
	return []keymap.Binding{km.Prev, km.Next, km.First, km.Last}
}

// ChangedMsg is delivered (via the Cmd Update returns) when the page
// changes.
type ChangedMsg struct {
	// ID is the pager's ID.
	ID string
	// Page is the new page, counted from 1.
	Page int
}

// Page returns the current page, counted from 1 and never past Total. It is
// 0 when there are no pages.
func (m Model) Page() int {
	if m.Total < 1 {
		return 0
	}
	return min(max(m.page, 1), m.Total)
}

// SetPage goes to page p, held between 1 and Total, without a key press: no
// ChangedMsg is delivered.
func (m *Model) SetPage(p int) { m.page = min(max(p, 1), max(m.Total, 1)) }

// Focus gives the pager keyboard focus. It returns no Cmd; the result is
// there so a pager can stand where any focusable widget does.
func (m *Model) Focus() tui.Cmd {
	m.focused = true
	return nil
}

// Blur takes keyboard focus away.
func (m *Model) Blur() { m.focused = false }

// Focused reports whether the pager has keyboard focus.
func (m Model) Focused() bool { return m.focused }

// goTo moves to page p and returns the Cmd that reports it, or nil when the
// page did not change.
func (m Model) goTo(p int) (Model, tui.Cmd) {
	if m.Total < 1 {
		return m, nil
	}
	p = min(max(p, 1), m.Total)
	if p == m.Page() {
		return m, nil
	}
	m.page = p
	msg := ChangedMsg{ID: m.ID, Page: p}
	return m, func() tui.Msg { return msg }
}

// Update changes the page on Left, Right, Home and End when the pager has
// focus, and, with Mouse on, on a left press on a page number. A disabled
// pager ignores both.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if m.Disabled {
		return m, nil
	}
	if ev, ok := msg.(tui.MouseEvent); ok {
		if m.Mouse && ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft && m.Bounds.Contains(ev.X, ev.Y) {
			if p := m.pageAt(ev.X - m.Bounds.X); p > 0 {
				return m.goTo(p)
			}
		}
		return m, nil
	}
	if !m.focused {
		return m, nil
	}
	km := m.keys()
	switch {
	case keymap.Matches(msg, km.Prev):
		return m.goTo(m.Page() - 1)
	case keymap.Matches(msg, km.Next):
		return m.goTo(m.Page() + 1)
	case keymap.Matches(msg, km.First):
		return m.goTo(1)
	case keymap.Matches(msg, km.Last):
		return m.goTo(m.Total)
	}
	return m, nil
}

// shown returns the pages drawn, in order, with 0 standing for an ellipsis:
// the first and last, and the current one with Siblings on each side. A gap
// of exactly one page shows that page, since an ellipsis would take as much
// room and say less.
func (m Model) shown() []int {
	if m.Total < 1 {
		return nil
	}
	sib := max(m.Siblings, 0)
	lo, hi := max(m.Page()-sib, 1), min(m.Page()+sib, m.Total)
	var out []int
	add := func(from, to int) {
		switch gap := to - from - 1; {
		case gap == 1:
			out = append(out, from+1)
		case gap > 1:
			out = append(out, 0)
		}
	}
	if lo > 1 {
		out = append(out, 1)
		add(1, lo)
	}
	for p := lo; p <= hi; p++ {
		out = append(out, p)
	}
	if hi < m.Total {
		add(hi, m.Total)
		out = append(out, m.Total)
	}
	return out
}

// cell is how one entry of shown is drawn, without styling: a page as its
// number between two cells, an ellipsis as the theme's glyph between two.
func (m Model) cell(p int) string {
	if p == 0 {
		return " " + m.themed().GlyphSet().Ellipsis + " "
	}
	return " " + strconv.Itoa(p) + " "
}

// pageAt returns the page drawn at local column x, or 0 for an ellipsis or
// a column outside the row.
func (m Model) pageAt(x int) int {
	left := 0
	for _, p := range m.shown() {
		w := ansi.Width(m.cell(p))
		if x >= left && x < left+w {
			return p
		}
		left += w
	}
	return 0
}

// Width is the number of cells View takes. It changes with the page, as
// numbers gain digits and ellipses come and go.
func (m Model) Width() int {
	w := 0
	for _, p := range m.shown() {
		w += ansi.Width(m.cell(p))
	}
	return w
}

// View renders the page numbers on one row, each between two cells. The
// current page is in square brackets, "[5]", so it reads without colour, and
// in angle brackets, "<5>", when the pager has focus. A disabled pager is
// faint. With no pages it renders "".
func (m Model) View() string {
	t := m.themed()
	st := t.ResolvedStates()
	other := ansi.NewStyle().Foreground(t.Text)
	fold := ansi.NewStyle().Foreground(t.Muted)
	current := ansi.NewStyle().Bold().Foreground(t.Primary)
	if m.Disabled {
		other, fold, current = st.Disabled.Faint(), st.Disabled.Faint(), st.Disabled.Faint()
	}
	var b strings.Builder
	for _, p := range m.shown() {
		switch {
		case p == 0:
			b.WriteString(fold.Render(m.cell(p)))
		case p == m.Page():
			open, shut, style := "[", "]", current
			if m.focused && !m.Disabled {
				open, shut, style = "<", ">", st.Focus.Bold()
			}
			b.WriteString(style.Render(open + strconv.Itoa(p) + shut))
		default:
			b.WriteString(other.Render(m.cell(p)))
		}
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the pager as plain text for accessible output (see
// tui.Linearizer): "page 5 of 20", with ", unavailable" when disabled, and
// "no pages" when there are none.
func (m Model) Linearize() string {
	if m.Total < 1 {
		return "no pages"
	}
	out := "page " + strconv.Itoa(m.Page()) + " of " + strconv.Itoa(m.Total)
	if m.Disabled {
		out += ", unavailable"
	}
	return out
}
