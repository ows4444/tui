// Package contextmenu is a popup menu opened at an anchor point, typically on
// a right-click or a key press. It lists Items (with separators, disabled
// entries, accelerator letters and optional shortcut hints), moves a cursor
// over the enabled ones, reports the confirmed item as SelectedMsg, and closes
// on Esc or a click outside. Render composites it over a base view, shifted or
// flipped so that it stays on screen.
//
// Like popover.Model it is an overlay: a parent calls Render every frame and
// the closed menu leaves the base unchanged.
package contextmenu

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

// Item is one row of a menu.
type Item struct {
	// Label is the text shown. Unless Model.Raw is set it is sanitised
	// (ansi.Clean) so untrusted text cannot carry terminal escapes.
	Label string
	// Value is returned in SelectedMsg and never altered.
	Value string
	// Shortcut is an optional hint drawn right-aligned (for example
	// "ctrl+c"). It is display only.
	Shortcut string
	// Accel is the accelerator letter: while the menu is open, typing it
	// (case-insensitively) activates the item. Zero means none. When Label
	// contains the letter, its first occurrence is underlined.
	Accel rune
	// Disabled makes the item unselectable: the cursor skips it and it
	// cannot be activated.
	Disabled bool
	// Separator makes the row a horizontal rule. Every other field is ignored.
	Separator bool
}

// Selectable reports whether the cursor may rest on the item.
func (it Item) Selectable() bool { return !it.Separator && !it.Disabled }

// KeyMap names the keys of each action of a Model.
type KeyMap struct {
	// Open opens the menu at its current anchor while it is closed.
	Open   keymap.Binding
	Up     keymap.Binding
	Down   keymap.Binding
	Top    keymap.Binding
	Bottom keymap.Binding
	// Select confirms the item under the cursor.
	Select keymap.Binding
	// Close closes the menu without a selection.
	Close keymap.Binding
}

// DefaultKeyMap returns the default bindings: shift+f10 or the menu key opens
// it, up/down and home/end move, enter selects, esc closes.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Open:   keymap.NewBinding("open menu", "shift+f10"),
		Up:     keymap.NewBinding("up", "up"),
		Down:   keymap.NewBinding("down", "down"),
		Top:    keymap.NewBinding("first item", "home"),
		Bottom: keymap.NewBinding("last item", "end"),
		Select: keymap.NewBinding("select", "enter"),
		Close:  keymap.NewBinding("close", "esc"),
	}
}

func (k KeyMap) isZero() bool {
	return len(k.Open.Keys)+len(k.Up.Keys)+len(k.Down.Keys)+len(k.Top.Keys)+len(k.Bottom.Keys)+len(k.Select.Keys)+len(k.Close.Keys) == 0
}

// Model is a popup menu. The zero value is a closed, empty menu; use New.
type Model struct {
	// Items are the rows, top to bottom.
	Items []Item
	// Theme styles the box, the cursor row and disabled items.
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Label is the accessible name of the menu; "" reads as "Context menu".
	Label string
	// Raw, when true, draws labels unchanged instead of sanitising them.
	Raw bool

	// KeyMap holds the key bindings Update obeys. New fills it with
	// DefaultKeyMap; a Model with no bindings set behaves as if it held
	// DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent: a right press
	// opens the menu at the pointer; while open, hovering moves the cursor, a
	// left press on an item selects it, a press elsewhere closes the menu and
	// the wheel moves the cursor. Off (the default) ignores the mouse.
	Mouse bool
	// ScreenW and ScreenH are the screen size the menu is kept inside, for
	// placement and mouse hit-testing. Zero means unbounded. Render uses the
	// size of its base instead.
	ScreenW, ScreenH int
	// AnchorX and AnchorY are the screen cell the menu's top-left corner is
	// placed at. Show sets them.
	AnchorX, AnchorY int

	open   bool
	cursor int
}

// New returns a closed Model of items with the default theme and keys.
func New(items ...Item) Model {
	return Model{Items: items, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

func (m Model) keys() KeyMap {
	if m.KeyMap.isZero() {
		return DefaultKeyMap()
	}
	return m.KeyMap
}

// Bindings returns the actions the menu currently honours, for help text. A
// closed menu honours only Open.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	if !m.open {
		return []keymap.Binding{k.Open}
	}
	return []keymap.Binding{k.Up, k.Down, k.Top, k.Bottom, k.Select, k.Close}
}

// Show opens the menu with its top-left corner at (x, y) and the cursor on
// the first selectable item.
func (m *Model) Show(x, y int) {
	m.AnchorX, m.AnchorY, m.open = x, y, true
	m.cursor = m.step(-1, 1)
}

// Hide closes the menu without sending any message.
func (m *Model) Hide() { m.open = false }

// Open reports whether the menu is shown.
func (m Model) Open() bool { return m.open }

// Cursor returns the index in Items of the highlighted item, or -1 when no
// item is selectable.
func (m Model) Cursor() int {
	if m.cursor < 0 || m.cursor >= len(m.Items) || !m.Items[m.cursor].Selectable() {
		return -1
	}
	return m.cursor
}

// SetCursor moves the cursor to item i. It does nothing unless i is a
// selectable item.
func (m *Model) SetCursor(i int) {
	if i >= 0 && i < len(m.Items) && m.Items[i].Selectable() {
		m.cursor = i
	}
}

// SelectedMsg is delivered (via the Cmd Update returns) when an item is
// confirmed. The menu closes first.
type SelectedMsg struct {
	Index int
	Item  Item
}

// ClosedMsg is delivered (via the Cmd Update returns) when the menu is closed
// without a selection: by the Close key or by a click outside it.
type ClosedMsg struct{}

// step returns the nearest selectable index from from in direction dir
// (+1 or -1), exclusive of from, or -1 if there is none.
func (m Model) step(from, dir int) int {
	for i := from + dir; i >= 0 && i < len(m.Items); i += dir {
		if m.Items[i].Selectable() {
			return i
		}
	}
	return -1
}

// move returns the cursor after moving n selectable items (n<0 is up), clamped
// at the ends.
func (m Model) move(n int) int {
	c := m.cursor
	dir := 1
	if n < 0 {
		dir, n = -1, -n
	}
	for ; n > 0; n-- {
		next := m.step(c, dir)
		if next < 0 {
			break
		}
		c = next
	}
	return c
}

func (m Model) confirm(i int) (Model, tui.Cmd) {
	m.open = false
	item := m.Items[i]
	return m, func() tui.Msg { return SelectedMsg{Index: i, Item: item} }
}

func (m Model) dismiss() (Model, tui.Cmd) {
	m.open = false
	return m, func() tui.Msg { return ClosedMsg{} }
}

// Update handles keys and, with Mouse on, mouse events. While closed it only
// reacts to the Open key (at the current anchor) and a right press; any other
// Msg is a no-op. The returned Cmd delivers SelectedMsg or ClosedMsg.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	switch ev := msg.(type) {
	case tui.MouseEvent:
		if m.Mouse {
			return m.updateMouse(ev)
		}
	case tui.Key:
		return m.updateKey(ev)
	}
	return m, nil
}

func (m Model) updateKey(key tui.Key) (Model, tui.Cmd) {
	k := m.keys()
	if !m.open {
		if keymap.Matches(key, k.Open) {
			m.Show(m.AnchorX, m.AnchorY)
		}
		return m, nil
	}
	switch {
	case keymap.Matches(key, k.Close):
		return m.dismiss()
	case keymap.Matches(key, k.Up):
		m.cursor = m.move(-1)
	case keymap.Matches(key, k.Down):
		m.cursor = m.move(1)
	case keymap.Matches(key, k.Top):
		m.cursor = m.step(-1, 1)
	case keymap.Matches(key, k.Bottom):
		m.cursor = m.step(len(m.Items), -1)
	case keymap.Matches(key, k.Select):
		if c := m.Cursor(); c >= 0 {
			return m.confirm(c)
		}
	default:
		if key.Type == tui.KeyRunes && key.Mod == 0 && utf8.RuneCountInString(key.Text) == 1 {
			r, _ := utf8.DecodeRuneInString(key.Text)
			r = unicode.ToLower(r)
			for i, it := range m.Items {
				if it.Selectable() && it.Accel != 0 && unicode.ToLower(it.Accel) == r {
					return m.confirm(i)
				}
			}
		}
	}
	return m, nil
}

func (m Model) updateMouse(ev tui.MouseEvent) (Model, tui.Cmd) {
	if !m.open {
		if ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonRight {
			m.Show(ev.X, ev.Y)
		}
		return m, nil
	}
	b := m.Bounds()
	row := m.rowAt(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion:
		if row >= 0 && m.Items[row].Selectable() {
			m.cursor = row
		}
	case ev.Action != tui.MouseActionPress:
	case ev.Button == tui.MouseButtonWheelUp:
		m.cursor = m.move(-1)
	case ev.Button == tui.MouseButtonWheelDown:
		m.cursor = m.move(1)
	case ev.Button == tui.MouseButtonRight && !b.Contains(ev.X, ev.Y):
		m.Show(ev.X, ev.Y) // a right click elsewhere moves the menu
	case ev.Button == tui.MouseButtonLeft || ev.Button == tui.MouseButtonRight:
		switch {
		case row >= 0 && m.Items[row].Selectable():
			return m.confirm(row)
		case !b.Contains(ev.X, ev.Y):
			return m.dismiss()
		}
	}
	return m, nil
}

// border is the thickness of the box edge on each side.
func (m Model) border() int {
	if m.themed().Border == (theme.Border{}) {
		return 0
	}
	return 1
}

// innerWidth is the width of one row of content.
func (m Model) innerWidth() int {
	label, short := 0, 0
	for _, it := range m.Items {
		if it.Separator {
			continue
		}
		label = max(label, ansi.Width(m.text(it.Label)))
		short = max(short, ansi.Width(m.text(it.Shortcut)))
	}
	w := 2 + label + 1
	if short > 0 {
		w += 2 + short
	}
	if len(m.Items) == 0 {
		w = len(emptyText) + 3
	}
	return w
}

const emptyText = "(no items)"

func (m Model) text(s string) string { return ansi.Clean(m.Raw, s) }

// size is the outer size of the drawn box, before clipping to the screen.
func (m Model) size() layout.Size {
	rows := max(len(m.Items), 1)
	b := m.border()
	return layout.Size{W: m.innerWidth() + 2*b, H: rows + 2*b}
}

// origin is where the box's top-left corner lands: at the anchor, shifted left
// to fit the screen width and flipped above the anchor when it would run off
// the bottom. It never goes negative.
func (m Model) origin(screenW, screenH int) (x, y int) {
	s := m.size()
	x, y = m.AnchorX, m.AnchorY
	if screenW > 0 && x+s.W > screenW {
		x = screenW - s.W
	}
	if screenH > 0 && y+s.H > screenH {
		if y-s.H >= 0 && y-s.H < screenH {
			y -= s.H
		}
		y = min(y, screenH-s.H)
	}
	return max(x, 0), max(y, 0)
}

// Bounds is the screen rectangle the open menu covers, clipped to the screen
// (ScreenW, ScreenH). A closed menu has an empty rectangle.
func (m Model) Bounds() hittest.Rect {
	if !m.open {
		return hittest.Rect{}
	}
	s := m.size()
	x, y := m.origin(m.ScreenW, m.ScreenH)
	r := hittest.Rect{X: x, Y: y, W: s.W, H: s.H}
	if m.ScreenW > 0 {
		r.W = max(min(r.W, m.ScreenW-r.X), 0)
	}
	if m.ScreenH > 0 {
		r.H = max(min(r.H, m.ScreenH-r.Y), 0)
	}
	return r
}

// rowAt returns the index of the item under screen cell (x, y), or -1.
func (m Model) rowAt(x, y int) int {
	b := m.Bounds()
	bd := m.border()
	if !b.Contains(x, y) || len(m.Items) == 0 {
		return -1
	}
	lx, ly := b.Local(x, y)
	if lx < bd || lx >= b.W-bd || ly < bd {
		return -1
	}
	if i := ly - bd; i < len(m.Items) {
		return i
	}
	return -1
}

// row renders item i at the inner width w.
func (m Model) row(i, w int) string {
	it := m.Items[i]
	st := m.themed().ResolvedStates()
	if it.Separator {
		g := m.themed().Glyphs.Resolved()
		return ansi.NewStyle().Foreground(m.themed().BorderColor).Render(" " + strings.Repeat(g.RuleH, max(w-2, 0)) + " ")
	}
	base := ansi.NewStyle()
	prefix := "  "
	switch {
	case i == m.cursor:
		base, prefix = st.Selected.Bold(), "> "
	case it.Disabled:
		base = st.Disabled
	}
	label := m.text(it.Label)
	var body string
	idx := -1
	if it.Accel != 0 && !it.Disabled {
		idx = strings.IndexFunc(label, func(r rune) bool { return unicode.ToLower(r) == unicode.ToLower(it.Accel) })
	}
	if idx >= 0 {
		_, n := utf8.DecodeRuneInString(label[idx:])
		body = base.Render(prefix+label[:idx]) + base.Underline().Render(label[idx:idx+n]) + base.Render(label[idx+n:])
	} else {
		body = base.Render(prefix + label)
	}
	used := 2 + ansi.Width(label)
	tail := ""
	if it.Shortcut != "" {
		s := m.text(it.Shortcut)
		pad := max(w-1-used-ansi.Width(s), 2)
		tail = base.Render(strings.Repeat(" ", pad) + s + " ")
		return body + tail
	}
	return body + base.Render(strings.Repeat(" ", max(w-used, 0)))
}

// content is the styled item rows, one line each.
func (m Model) content() string {
	w := m.innerWidth()
	if len(m.Items) == 0 {
		return m.themed().ResolvedStates().Disabled.Render("  " + emptyText + " ")
	}
	lines := make([]string, len(m.Items))
	for i := range m.Items {
		lines[i] = m.row(i, w)
	}
	return strings.Join(lines, "\n")
}

func (m Model) box() layout.Node {
	b := layout.NewBox()
	if m.border() > 0 {
		b = b.Border(m.themed().Border).BorderColor(m.themed().BorderColor)
	}
	return layout.BoxNode(b, layout.Block(m.content()))
}

// View renders the menu's box on its own, whether or not it is open, at its
// natural size.
func (m Model) View() string {
	s := m.size()
	return m.box().Render(s)
}

// Render composites the open menu over base, placed at the anchor and kept
// inside base: shifted left when it would pass base's right edge, flipped
// above the anchor when it would pass the bottom, and cut off if it is still
// too big. A closed menu returns base unchanged.
func (m Model) Render(base string) string {
	if !m.open {
		return base
	}
	lines := strings.Split(base, "\n")
	baseW := 0
	for _, l := range lines {
		baseW = max(baseW, ansi.Width(l))
	}
	x, y := m.origin(baseW, len(lines))
	box := strings.Split(m.View(), "\n")
	if room := len(lines) - y; len(box) > room {
		box = box[:max(room, 0)]
	}
	for i, l := range box {
		box[i] = ansi.Truncate(l, max(baseW-x, 0))
	}
	return layout.Overlay(base, strings.Join(box, "\n"), x, y)
}

// Compile-time proof that Model satisfies tui.Overlay[Model].
var _ tui.Overlay[Model] = Model{}

// Linearize renders the menu as plain text for accessible output (see
// tui.Linearizer), or "" while it is closed: the menu's Label (default
// "Context menu"), then one line per non-separator item such as
// "Copy, item 1 of 3, shortcut ctrl+c, selected". Disabled items read
// ", disabled". An empty menu reads "No items".
func (m Model) Linearize() string {
	if !m.open {
		return ""
	}
	name := m.Label
	if name == "" {
		name = "Context menu"
	}
	lines := []string{m.text(name)}
	total := 0
	for _, it := range m.Items {
		if !it.Separator {
			total++
		}
	}
	if total == 0 {
		return strings.Join(append(lines, "No items"), "\n")
	}
	n := 0
	for i, it := range m.Items {
		if it.Separator {
			continue
		}
		n++
		line := m.text(it.Label) + ", item " + strconv.Itoa(n) + " of " + strconv.Itoa(total)
		if it.Shortcut != "" {
			line += ", shortcut " + m.text(it.Shortcut)
		}
		if it.Disabled {
			line += ", disabled"
		}
		if i == m.cursor && it.Selectable() {
			line += ", selected"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
