// Package menu is a nested-navigation list widget composed on top of
// picker.Model. Where picker.Model is a flat single-choice list, menu.Model
// adds drill-down into an Item's Children as a stack of picker.Model
// levels — one per depth currently navigated into — so Up/Down/Home/End
// and rendering at any given level are exactly picker.Model's behavior,
// and only Enter (drill in or select) and Esc (pop back out) are
// menu-specific.
package menu

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/theme"
)

// Item is one entry in a Menu, optionally with nested Children. An Item
// with no Children is a leaf: confirming it with Enter emits SelectedMsg.
// An Item with Children drills into a submenu built from them instead.
type Item struct {
	Label    string
	Value    string
	Children []Item
}

// Model is a keyboard-navigable nested menu. It holds a stack of
// picker.Model levels (one per depth drilled into) plus the parallel Menu
// Items backing each level, so the Children of the item under any level's
// cursor can be looked up on Enter. The top of the stack is the level
// currently shown by View and driven by Update.
type Model struct {
	// Raw, when true, draws item labels unchanged. By default each label is
	// sanitised (ansi.Sanitize) so untrusted text cannot carry terminal
	// escape sequences. Item.Value is never altered.
	Raw bool

	// KeyMap holds the key bindings Update obeys, at every depth. New fills
	// it with DefaultKeyMap; a Model built with no bindings set behaves as
	// if it held DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent on the current
	// level: the wheel moves the cursor by WheelStep rows and a left click
	// on a row moves the cursor to it (it does not drill in). Off (the
	// default) ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the widget; mouse
	// events outside it are ignored. The app sets it each frame it moves.
	Bounds hittest.Rect
	// WheelStep is the rows the cursor moves per wheel notch; 0 means 3.
	WheelStep int

	stack      []picker.Model
	stackItems [][]Item

	// styling is what SetTheme applied; levels opened later get it too.
	// hasStyling is false until SetTheme runs, so levels keep picker.New's
	// default theme.
	styling    theme.Theme
	tokens     theme.Tokens
	hasStyling bool
}

// New builds a Model from items, with the root level's picker.Model built
// from items (mapped to picker.Item{Label, Value}) pushed as the initial
// and only stack entry.
func New(items []Item) Model {
	return Model{
		KeyMap:     DefaultKeyMap(),
		stack:      []picker.Model{pickerFor(items)},
		stackItems: [][]Item{items},
	}
}

// KeyMap is the set of keys Update reacts to.
type KeyMap struct {
	Up     keymap.Binding
	Down   keymap.Binding
	Top    keymap.Binding
	Bottom keymap.Binding
	// Select drills into an item's Children or confirms a leaf.
	Select keymap.Binding
	// Back pops to the parent level.
	Back keymap.Binding
}

// DefaultKeyMap returns the default bindings: up/down, home/end, enter to
// select, esc to go back.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:     keymap.NewBinding("up", "up"),
		Down:   keymap.NewBinding("down", "down"),
		Top:    keymap.NewBinding("first item", "home"),
		Bottom: keymap.NewBinding("last item", "end"),
		Select: keymap.NewBinding("select", "enter"),
		Back:   keymap.NewBinding("back", "esc"),
	}
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	k := m.KeyMap
	if len(k.Up.Keys)+len(k.Down.Keys)+len(k.Top.Keys)+len(k.Bottom.Keys)+len(k.Select.Keys)+len(k.Back.Keys) == 0 {
		return DefaultKeyMap()
	}
	return k
}

// Bindings returns the active bindings, for help text.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	return []keymap.Binding{k.Up, k.Down, k.Top, k.Bottom, k.Select, k.Back}
}

// pickerKeys is the picker.KeyMap equivalent of k for a level's navigation.
func (k KeyMap) pickerKeys() picker.KeyMap {
	return picker.KeyMap{Up: k.Up, Down: k.Down, Top: k.Top, Bottom: k.Bottom, Select: k.Select}
}

// pickerFor builds a picker.Model whose Items mirror items' Label/Value.
func pickerFor(items []Item) picker.Model {
	pItems := make([]picker.Item, len(items))
	for i, it := range items {
		pItems[i] = picker.Item{Label: it.Label, Value: it.Value}
	}
	return picker.New(pItems...)
}

// SelectedMsg is delivered (via the Cmd Update returns) when a leaf Item
// (no Children) is confirmed with Enter, mirroring picker.Model.
// SelectedMsg's confirm-on-Enter convention.
type SelectedMsg struct {
	Item Item
}

// current returns the index of the top-of-stack level.
func (m Model) current() int { return len(m.stack) - 1 }

// Update forwards the Up/Down/Top/Bottom bindings (and, with Mouse on, mouse
// events) to the current level's picker.Model (#403). Select drills into an item's Children by pushing a new
// picker.Model level (#404), or — for a leaf — emits menu.SelectedMsg for
// it (#405). Esc pops the current level back to its parent, preserving
// the parent's cursor position, when more than the root level is present
// (#406); at the root it is a no-op (#407). Any other Msg is forwarded to
// the current level's picker.Model, matching picker.Model's own no-op
// behavior for a non-Key Msg.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	_, isKey := msg.(tui.Key)
	_, isMouse := msg.(tui.MouseEvent)
	if !isKey && !(isMouse && m.Mouse) {
		return m, nil
	}

	// Copy-on-write: stack and stackItems may be shared with copies of m, so
	// never write through or append into their backing arrays.
	m.stack = append([]picker.Model(nil), m.stack...)
	m.stackItems = append([][]Item(nil), m.stackItems...)

	i := m.current()
	k := m.keys()

	if keymap.Matches(msg, k.Back) {
		if i > 0 {
			m.stack = m.stack[:i]
			m.stackItems = m.stackItems[:i]
		}
		return m, nil
	}

	if keymap.Matches(msg, k.Select) {
		items := m.stackItems[i]
		if len(items) == 0 {
			return m, nil
		}
		idx := m.stack[i].Cursor()
		item := items[idx]
		if len(item.Children) > 0 {
			child := pickerFor(item.Children)
			if m.hasStyling {
				child.Theme = m.styling
			}
			child = child.WithTokens(m.tokens)
			m.stack = append(m.stack, child)
			m.stackItems = append(m.stackItems, item.Children)
			return m, nil
		}
		return m, func() tui.Msg { return SelectedMsg{Item: item} }
	}

	lvl := m.stack[i]
	lvl.KeyMap = k.pickerKeys()
	lvl.Mouse, lvl.Bounds, lvl.WheelStep = m.Mouse, m.Bounds, m.WheelStep
	lvl, cmd := lvl.Update(msg)
	m.stack[i] = lvl
	return m, cmd
}

// View delegates to the current (top-of-stack) level's picker.Model.View.
func (m Model) View() string { return m.level().View() }

// level is the current picker.Model with its labels made safe to draw
// (unless Raw). Value, cursor and everything else are unchanged.
func (m Model) level() picker.Model {
	pk := m.stack[m.current()]
	if m.Raw {
		return pk
	}
	items := make([]picker.Item, len(pk.Items))
	for i, it := range pk.Items {
		it.Label = ansi.Clean(false, it.Label)
		items[i] = it
	}
	pk.Items = items
	return pk
}

// Compile-time proof that Model satisfies tui.Component[Model] — see
// tui.Component's doc comment for what this contract means and why.
var _ tui.Component[Model] = Model{}

// Linearize renders the current menu level as plain text for accessible
// output (see tui.Linearizer): one line per item with its label and
// position, ", submenu" on items that have children, and ", selected" on
// the cursor row. When the menu is drilled into a submenu, a first line
// "Submenu: A, B" names the path of parent items entered. An empty level
// reads "No items".
func (m Model) Linearize() string {
	var lines []string
	if depth := m.current(); depth > 0 {
		path := make([]string, depth)
		for i := 0; i < depth; i++ {
			path[i] = ansi.Clean(m.Raw, m.stackItems[i][m.stack[i].Cursor()].Label)
		}
		lines = append(lines, "Submenu: "+strings.Join(path, ", "))
	}
	items := m.stackItems[m.current()]
	if len(items) == 0 {
		return strings.Join(append(lines, "No items"), "\n")
	}
	cursor := m.stack[m.current()].Cursor()
	for i, it := range items {
		line := ansi.Clean(m.Raw, it.Label) + ", item " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(items))
		if len(it.Children) > 0 {
			line += ", submenu"
		}
		if i == cursor {
			line += ", selected"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
