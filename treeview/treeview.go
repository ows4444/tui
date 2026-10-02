// Package treeview is a hierarchical expandable tree — InkUI's "TreeView"
// (file-browser-style navigation).
package treeview

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Node is one tree node. A Node with no Children is a leaf.
type Node struct {
	Label    string
	Children []Node
}

// Model is a keyboard-navigable tree. Expand state is tracked by a path
// (child indices joined with '.', e.g. "0.2.1"), which stays stable across
// renders as long as the tree's shape itself doesn't change out from under
// it — building a new Roots slice with different structure invalidates any
// previously-set expand state.
//
// Model also serves as TreeSelect, a single-selection hierarchical picker:
// no separate type or package was created for it, since Model's flatten,
// expand-state and cursor-navigation logic (plus Enter/Space's
// toggle-if-branch/confirm-if-leaf behavior and the SelectedMsg it emits)
// already implements what TreeSelect needs.
type Model struct {
	Roots []Node
	Theme theme.Theme

	// tokens is the per-instance colour override set by WithTokens.
	tokens theme.Tokens
	// Raw, when true, draws node labels unchanged. By default each label is
	// sanitised (ansi.Sanitize) so untrusted text, such as file names, cannot
	// carry terminal escape sequences.
	Raw bool

	// KeyMap holds the keys Update reacts to; New fills it with
	// DefaultKeyMap. A Model built as a struct literal with no KeyMap set
	// uses DefaultKeyMap.
	KeyMap KeyMap

	// Mouse, when true, makes Update handle tui.MouseEvent inside Bounds:
	// the wheel moves the cursor by WheelStep rows and a left click puts the
	// cursor on the row under the pointer. The zero value ignores the mouse.
	Mouse bool
	// Bounds is the screen rectangle where the app draws the tree (its
	// top-left cell is the first row). The app sets it.
	Bounds hittest.Rect
	// WheelStep is the rows moved per wheel notch; zero means 3.
	WheelStep int

	cursor   int
	expanded map[string]bool
}

// KeyMap names the keys Update reacts to.
type KeyMap struct {
	Up, Down keymap.Binding
	// Select toggles a branch or confirms a leaf (SelectedMsg).
	Select keymap.Binding
	// Expand opens the branch under the cursor; Collapse closes it.
	Expand, Collapse keymap.Binding
}

// DefaultKeyMap returns the default keys: Up/Down, Enter or Space to select,
// Right to expand and Left to collapse.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:       keymap.NewBinding("up", "up"),
		Down:     keymap.NewBinding("down", "down"),
		Select:   keymap.NewBinding("select", "enter", "space"),
		Expand:   keymap.NewBinding("expand", "right"),
		Collapse: keymap.NewBinding("collapse", "left"),
	}
}

// keys returns m.KeyMap, or DefaultKeyMap when it is unset.
func (m Model) keys() KeyMap {
	k := m.KeyMap
	if len(k.Up.Keys)+len(k.Down.Keys)+len(k.Select.Keys)+len(k.Expand.Keys)+len(k.Collapse.Keys) == 0 {
		return DefaultKeyMap()
	}
	return k
}

// Bindings returns the active bindings, for help widgets.
func (m Model) Bindings() []keymap.Binding {
	k := m.keys()
	return []keymap.Binding{k.Up, k.Down, k.Select, k.Expand, k.Collapse}
}

// mouse handles a wheel or left-click inside Bounds.
func (m Model) mouse(ev tui.MouseEvent, n int) Model {
	if !m.Mouse || ev.Action != tui.MouseActionPress || !m.Bounds.Contains(ev.X, ev.Y) {
		return m
	}
	step := m.WheelStep
	if step <= 0 {
		step = 3
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.cursor = clamp(m.cursor-step, 0, n-1)
	case tui.MouseButtonWheelDown:
		m.cursor = clamp(m.cursor+step, 0, n-1)
	case tui.MouseButtonLeft:
		if _, ly := m.Bounds.Local(ev.X, ev.Y); ly < n {
			m.cursor = ly
		}
	}
	return m
}

// New builds a Model from roots, all initially collapsed.
func New(roots ...Node) Model {
	return Model{Roots: roots, Theme: theme.DarkTheme(), KeyMap: DefaultKeyMap()}
}

// visible is one flattened, currently-visible row.
type visible struct {
	node  *Node
	depth int
	path  string
}

func (m Model) flatten() []visible {
	var items []visible
	var walk func(nodes []Node, depth int, prefix string)
	walk = func(nodes []Node, depth int, prefix string) {
		for i := range nodes {
			path := prefix + strconv.Itoa(i)
			items = append(items, visible{node: &nodes[i], depth: depth, path: path})
			if len(nodes[i].Children) > 0 && m.expanded[path] {
				walk(nodes[i].Children, depth+1, path+".")
			}
		}
	}
	walk(m.Roots, 0, "")
	return items
}

// Cursor returns the index of the visible row currently under the cursor.
func (m Model) Cursor() int { return m.cursor }

// SetCursor moves the cursor to i among the currently visible rows,
// clamped to a valid index (or 0 with none visible).
func (m *Model) SetCursor(i int) {
	items := m.flatten()
	if len(items) == 0 {
		m.cursor = 0
		return
	}
	m.cursor = clamp(i, 0, len(items)-1)
}

// IsExpanded reports whether the node at path is expanded.
func (m Model) IsExpanded(path string) bool { return m.expanded[path] }

// VisibleRow is one currently-visible row, exported for callers (e.g.
// Sidebar) that need to render a Model's rows themselves rather
// than through View — the path uniquely and stably identifies a row (see
// Model's doc comment) so a caller can key its own per-row decoration
// (icons, badges, an externally-driven active selection) off it.
type VisibleRow struct {
	Label       string
	Path        string
	Depth       int
	HasChildren bool
}

// VisibleRows returns the same flattened, currently-visible rows View
// renders, in the same order, without rendering them.
func (m Model) VisibleRows() []VisibleRow {
	items := m.flatten()
	rows := make([]VisibleRow, len(items))
	for i, item := range items {
		rows[i] = VisibleRow{
			Label:       item.node.Label,
			Path:        item.path,
			Depth:       item.depth,
			HasChildren: len(item.node.Children) > 0,
		}
	}
	return rows
}

// toggle rebuilds the expanded map rather than mutating it in place — the
// same copy-on-write discipline multiselect.Toggle and accordion.Toggle
// use, for the same reason: Model is used with value semantics, and a
// shared map field mutated in place would corrupt any other Model value
// still holding a copy of it.
func (m *Model) toggle(path string) {
	next := make(map[string]bool, len(m.expanded)+1)
	for k, v := range m.expanded {
		next[k] = v
	}
	next[path] = !next[path]
	m.expanded = next
}

// SelectedMsg is delivered (via the Cmd Update returns) when a leaf node
// is confirmed with Enter or Space. Confirming a non-leaf node instead
// toggles its expand state and doesn't emit this.
type SelectedMsg struct {
	Node *Node
	Path string
}

// Update moves the cursor on Up/Down and, on Enter or Space, either
// toggles the row under it (if it has children) or confirms it as a leaf,
// returning a Cmd that delivers SelectedMsg. Right expands the row under
// the cursor if it has children and is collapsed; Left collapses it if it
// has children and is expanded. Left on an already-collapsed node, or
// Left/Right on a leaf, is a no-op. A Msg that isn't a Key is a no-op.
func (m Model) Update(msg tui.Msg) (Model, tui.Cmd) {
	ev, isMouse := msg.(tui.MouseEvent)
	if _, isKey := msg.(tui.Key); !isKey && !isMouse {
		return m, nil
	}
	items := m.flatten()
	if len(items) == 0 {
		return m, nil
	}
	m.cursor = clamp(m.cursor, 0, len(items)-1)
	if isMouse {
		return m.mouse(ev, len(items)), nil
	}

	k := m.keys()
	switch {
	case keymap.Matches(msg, k.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keymap.Matches(msg, k.Down):
		if m.cursor < len(items)-1 {
			m.cursor++
		}
	case keymap.Matches(msg, k.Select):
		item := items[m.cursor]
		if len(item.node.Children) > 0 {
			m.toggle(item.path)
			return m, nil
		}
		node, path := item.node, item.path
		return m, func() tui.Msg { return SelectedMsg{Node: node, Path: path} }
	case keymap.Matches(msg, k.Expand):
		item := items[m.cursor]
		if len(item.node.Children) > 0 && !m.expanded[item.path] {
			m.toggle(item.path)
		}
	case keymap.Matches(msg, k.Collapse):
		item := items[m.cursor]
		if len(item.node.Children) > 0 && m.expanded[item.path] {
			m.toggle(item.path)
		}
	}
	return m, nil
}

// View renders the currently visible rows (expanded subtrees included),
// indented by depth, with the row under the cursor highlighted.
func (m Model) View() string {
	items := m.flatten()
	if len(items) == 0 {
		return ""
	}
	cursor := clamp(m.cursor, 0, len(items)-1)

	cursorStyle := m.themed().ResolvedStates().Selected.Bold()

	var b strings.Builder
	for i, item := range items {
		prefix := "  "
		label := ansi.Clean(m.Raw, item.node.Label)
		if i == cursor {
			prefix = "> "
			label = cursorStyle.Render(label)
		}
		marker := " "
		if len(item.node.Children) > 0 {
			if m.expanded[item.path] {
				marker = m.themed().GlyphSet().Expanded
			} else {
				marker = m.themed().GlyphSet().Collapsed
			}
		}
		b.WriteString(prefix + strings.Repeat("  ", item.depth) + marker + " " + label)
		if i < len(items)-1 {
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

// Linearize renders the visible rows as plain text for accessible output
// (see tui.Linearizer): one line per node with its label, level, expanded
// or collapsed state (branches only), position among its siblings, and
// ", selected" on the cursor row, e.g. "src, level 1, expanded, item 2 of
// 3". No indentation, markers or styling.
func (m Model) Linearize() string {
	var lines []string
	var walk func(nodes []Node, depth int, prefix string)
	walk = func(nodes []Node, depth int, prefix string) {
		for i := range nodes {
			path := prefix + strconv.Itoa(i)
			n := &nodes[i]
			line := ansi.Clean(m.Raw, n.Label) + ", level " + strconv.Itoa(depth+1)
			if len(n.Children) > 0 {
				if m.expanded[path] {
					line += ", expanded"
				} else {
					line += ", collapsed"
				}
			}
			line += ", item " + strconv.Itoa(i+1) + " of " + strconv.Itoa(len(nodes))
			if len(lines) == m.cursor {
				line += ", selected"
			}
			lines = append(lines, line)
			if len(n.Children) > 0 && m.expanded[path] {
				walk(n.Children, depth+1, path+".")
			}
		}
	}
	walk(m.Roots, 0, "")
	return strings.Join(lines, "\n")
}
