// Package menubar is a horizontal menu bar of titled menus, each opening a
// dropdown of items below its title. Left and Right move between menus, Up and
// Down move within the dropdown, Enter activates, Esc closes, Alt plus a menu's
// accelerator letter opens it, and (with Mouse on) a click opens a menu or
// picks an item. The dropdown is a contextmenu.Model, so items share its
// separators, disabled entries, accelerators and shortcut hints.
//
// The bar draws one row with View; the open dropdown is an overlay drawn by
// Render over the rest of the screen.
package menubar

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/contextmenu"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Item is one row of a dropdown; it is contextmenu.Item.
type Item = contextmenu.Item

// Menu is one titled menu of the bar.
type Menu struct {
	// Title is the text on the bar. Unless Model.Raw is set it is sanitised.
	Title string
	// Accel is the accelerator letter: Alt plus it opens the menu. Zero means
	// none. The first occurrence of the letter in Title is underlined.
	Accel rune
	// Items are the dropdown rows.
	Items []Item
}

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	// Focus opens the first menu while the bar is idle.
	Focus keymap.Binding
	// Left and Right move to the previous and next menu, wrapping around.
	Left  keymap.Binding
	Right keymap.Binding
	// Up, Down, Top and Bottom move within the open dropdown.
	Up     keymap.Binding
	Down   keymap.Binding
	Top    keymap.Binding
	Bottom keymap.Binding
	// Select activates the highlighted item.
	Select keymap.Binding
	// Close closes the open dropdown.
	Close keymap.Binding
}

// DefaultKeyMap returns the default bindings: f10 focuses the bar, left/right
// switch menus, up/down and home/end move, enter selects, esc closes.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Focus:  keymap.NewBinding("open menu bar", "f10"),
		Left:   keymap.NewBinding("previous menu", "left"),
		Right:  keymap.NewBinding("next menu", "right"),
		Up:     keymap.NewBinding("up", "up"),
		Down:   keymap.NewBinding("down", "down"),
		Top:    keymap.NewBinding("first item", "home"),
		Bottom: keymap.NewBinding("last item", "end"),
		Select: keymap.NewBinding("select", "enter"),
		Close:  keymap.NewBinding("close", "esc"),
	}
}

func (k KeyMap) isZero() bool {
	return len(k.Focus.Keys)+len(k.Left.Keys)+len(k.Right.Keys)+len(k.Up.Keys)+len(k.Down.Keys)+len(k.Top.Keys)+len(k.Bottom.Keys)+len(k.Select.Keys)+len(k.Close.Keys) == 0
}

// Model is a menu bar. Build it with New.
type Model struct {
	// Menus are the bar's menus, left to right.
	Menus []Menu
	// Theme styles the bar and its dropdown.
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Raw, when true, draws titles and labels unchanged instead of
	// sanitising them.
	Raw bool

	// KeyMap holds the key bindings Update obeys. New fills it with
	// DefaultKeyMap; a Model with no bindings set behaves as if it held
	// DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a left press on a
	// title opens that menu (or closes it if it is open), hovering another
	// title while a menu is open switches to it, and a left press on a
	// dropdown item selects it. A press anywhere else closes the dropdown.
	// Off (the default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the bar (its first
	// row holds the titles); the dropdown opens under the title, on the row
	// below Bounds.Y. The zero value is the top-left corner.
	Bounds hittest.Rect
	// ScreenW and ScreenH are the screen size the dropdown is kept inside.
	// Zero means unbounded. Render uses the size of its base instead.
	ScreenW, ScreenH int

	open   bool
	active int
	drop   contextmenu.Model
}

// New returns an idle Model of menus with the default theme and keys.
func New(menus ...Menu) Model {
	return Model{Menus: menus, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

func (m Model) keys() KeyMap {
	if m.KeyMap.isZero() {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the actions the bar currently honours, for help text. An
// idle bar honours only Focus.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	if !m.open {
		return []keymap.Binding{k.Focus}
	}
	return []keymap.Binding{k.Left, k.Right, k.Up, k.Down, k.Top, k.Bottom, k.Select, k.Close}
}

// SelectedMsg is delivered (via the Cmd Update returns) when a dropdown item
// is activated. The bar closes first.
type SelectedMsg struct {
	// Menu is the index in Menus of the menu the item belongs to; Index is
	// the item's index in that menu's Items.
	Menu, Index int
	Item        Item
}

// ClosedMsg is delivered (via the Cmd Update returns) when the open dropdown
// is closed without a selection.
type ClosedMsg struct{}

// Open reports whether a dropdown is open.
func (m Model) Open() bool { return m.open }

// Active returns the index of the open (or last opened) menu.
func (m Model) Active() int { return m.active }

// Show opens menu i, clamped to a valid index. It does nothing without menus.
func (m *Model) Show(i int) {
	if len(m.Menus) == 0 {
		return
	}
	i = max(0, min(i, len(m.Menus)-1))
	x := m.Bounds.X
	for j := 0; j < i; j++ {
		x += m.titleWidth(j)
	}
	menu := m.Menus[i]
	d := contextmenu.New(menu.Items...)
	d.Theme, d.Raw, d.Mouse = m.themed(), m.Raw, true
	d.ScreenW, d.ScreenH = m.ScreenW, m.ScreenH
	d.Label = m.text(menu.Title) + " menu"
	k := m.keys()
	d.KeyMap = contextmenu.KeyMap{Open: k.Focus, Up: k.Up, Down: k.Down, Top: k.Top, Bottom: k.Bottom, Select: k.Select, Close: k.Close}
	d.Show(x, m.Bounds.Y+1)
	m.open, m.active, m.drop = true, i, d
}

// Hide closes the dropdown without sending any message.
func (m *Model) Hide() { m.open = false }

func (m Model) text(s string) string { return ansi.Clean(m.Raw, s) }

// titleWidth is the cells one title takes on the bar: its text and a space or
// bracket each side.
func (m Model) titleWidth(i int) int { return ansi.Width(m.text(m.Menus[i].Title)) + 2 }

// titleAt returns the index of the menu whose title is under screen cell
// (x, y), or -1. Only the first row of Bounds holds titles.
func (m Model) titleAt(x, y int) int {
	if y != m.Bounds.Y || x < m.Bounds.X || (m.Bounds.W > 0 && x >= m.Bounds.X+m.Bounds.W) {
		return -1
	}
	pos := m.Bounds.X
	for i := range m.Menus {
		w := m.titleWidth(i)
		if x < pos+w {
			return i
		}
		pos += w
	}
	return -1
}

// Update drives the bar. While idle it reacts to the Focus key and to Alt plus
// an accelerator; while a dropdown is open it switches menus on Left/Right and
// passes the other keys to the dropdown. With Mouse on it also handles clicks
// (see Model.Mouse). The returned Cmd delivers SelectedMsg or ClosedMsg. Any
// other Msg is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	if len(m.Menus) == 0 {
		return m, nil
	}
	switch ev := msg.(type) {
	case tui.Key:
		return m.updateKey(ev)
	case tui.MouseEvent:
		if m.Mouse {
			return m.updateMouse(ev)
		}
	}
	return m, nil
}

func (m Model) accel(key tui.Key) int {
	if key.Type != tui.KeyRunes || !key.Mod.Alt() || utf8.RuneCountInString(key.Text) != 1 {
		return -1
	}
	r, _ := utf8.DecodeRuneInString(key.Text)
	r = unicode.ToLower(r)
	for i, mn := range m.Menus {
		if mn.Accel != 0 && unicode.ToLower(mn.Accel) == r {
			return i
		}
	}
	return -1
}

func (m Model) updateKey(key tui.Key) (Model, tui.Cmd) {
	k := m.keys()
	if i := m.accel(key); i >= 0 {
		m.Show(i)
		return m, nil
	}
	if !m.open {
		if keymap.Matches(key, k.Focus) {
			m.Show(0)
		}
		return m, nil
	}
	n := len(m.Menus)
	switch {
	case keymap.Matches(key, k.Left):
		m.Show((m.active + n - 1) % n)
		return m, nil
	case keymap.Matches(key, k.Right):
		m.Show((m.active + 1) % n)
		return m, nil
	}
	return m.forward(key)
}

// forward hands msg to the dropdown and turns what it reports into the bar's
// own message. Cmds of contextmenu only build a message, so running one here
// is immediate and has no effect on the world.
func (m Model) forward(msg tui.Msg) (Model, tui.Cmd) {
	var cmd tui.Cmd
	m.drop, cmd = m.drop.Update(msg)
	if cmd == nil {
		return m, nil
	}
	switch r := cmd().(type) {
	case contextmenu.SelectedMsg:
		m.open = false
		sel := SelectedMsg{Menu: m.active, Index: r.Index, Item: r.Item}
		return m, func() tui.Msg { return sel }
	case contextmenu.ClosedMsg:
		m.open = false
		return m, func() tui.Msg { return ClosedMsg{} }
	}
	return m, nil
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if i := m.titleAt(ev.X, ev.Y); i >= 0 {
		switch {
		case ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft:
			if m.open && m.active == i {
				m.open = false
				return m, func() tui.Msg { return ClosedMsg{} }
			}
			m.Show(i)
		case ev.Action == tui.MouseActionMotion && m.open && m.active != i:
			m.Show(i)
		}
		return m, nil
	}
	if !m.open {
		return m, nil
	}
	return m.forward(ev)
}

// Render composites the open dropdown over base, kept inside base's
// size. An idle bar returns base unchanged.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}
	return m.drop.Render(base)
}

// View renders the bar's single row. The open menu's title is bracketed and
// bold, so it shows without colour; the others are padded with spaces.
func (m Model) View() string {
	st := m.themed().ResolvedStates()
	var b strings.Builder
	for i, mn := range m.Menus {
		title := m.text(mn.Title)
		if m.open && i == m.active {
			b.WriteString(st.Selected.Bold().Render("[" + title + "]"))
			continue
		}
		idx := -1
		if mn.Accel != 0 {
			idx = strings.IndexFunc(title, func(r rune) bool { return unicode.ToLower(r) == unicode.ToLower(mn.Accel) })
		}
		if idx < 0 {
			b.WriteString(" " + title + " ")
			continue
		}
		n := len(string([]rune(title[idx:])[:1]))
		u := ansi.NewStyle().Underline()
		b.WriteString(" " + title[:idx] + u.Render(title[idx:idx+n]) + title[idx+n:] + " ")
	}
	return b.String()
}

// Compile-time proof that Model satisfies tui.Overlay[Model].
var _ tui.Overlay[Model] = Model{}

// Linearize renders the bar as plain text for accessible output (see
// tui.Linearizer): "Menu bar", then one line per menu such as
// "File, menu 1 of 3, expanded", and, when a dropdown is open, its items
// under the name "File menu". With no menus it reads "No menus".
func (m Model) Linearize() string {
	if len(m.Menus) == 0 {
		return "No menus"
	}
	lines := []string{"Menu bar"}
	for i, mn := range m.Menus {
		line := m.text(mn.Title) + ", menu " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(m.Menus))
		if m.open && i == m.active {
			line += ", expanded"
		}
		lines = append(lines, line)
	}
	if m.open {
		lines = append(lines, m.drop.Linearize())
	}
	return strings.Join(lines, "\n")
}
