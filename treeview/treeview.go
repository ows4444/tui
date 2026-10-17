// Package treeview is a hierarchical expandable tree — InkUI's "TreeView"
// (file-browser-style navigation).
package treeview

import (
	"github.com/ows4444/tui/internal/typeahead"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/fuzzy"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/theme"
)

// Node is one tree node. A Node with no Children and no Load is a leaf.
type Node struct {
	Label    string
	Children []Node
	// Load, when set on a node with no Children, makes it a branch whose
	// children are fetched the first time it is expanded: Update returns a Cmd
	// that calls Load and delivers a LoadedMsg, and the node opens when that
	// message comes back through Update. A Load that returns no nodes leaves an
	// empty branch. Load runs off the program's loop, so it may block on disk
	// or network; it must not touch the Model. Children, when set, win over Load.
	Load func() []Node
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

	// Filter, when true, lets the person type to narrow the tree: printable
	// keys that no binding in KeyMap uses are added to a query, Backspace
	// removes the last rune and Esc clears the query. Rows whose label
	// fuzzy-matches the query (see package fuzzy) show with the matched runes
	// underlined, together with their ancestors, which open so the match is
	// visible; the tree keeps its order. The cursor moves to the best match as
	// the query changes. The default Select and Expand keys include Space and
	// Right, so a query cannot hold a space. Off (the default), the tree is
	// unchanged. A first line shows the query.
	//
	// A node whose children are not loaded yet (see Node.Load) is not searched.
	Filter bool
	// TypeAhead, when true, makes typing letters jump to the next visible node whose
	// label starts with them, ignoring case: "b" finds the next one that starts with
	// b, "bl" the next that starts with bl, the same letter again goes on to the
	// one after, and a pause of a second starts a new word. Keys that a binding in
	// KeyMap uses keep their meaning, so a letter some binding uses cannot start a word. Off (the default), letters do
	// nothing.
	TypeAhead bool
	// With Filter on, typing narrows the tree instead and TypeAhead does nothing.

	// Name tells LoadedMsg apart when two lazy trees are on screen: a message
	// is taken only by the Model whose Name equals its own. Empty is a name.
	Name string
	// Notify, when true, makes Update return a Cmd that delivers ExpandedMsg
	// or CollapsedMsg when a branch opens or closes. Off (the default), toggling
	// returns no Cmd, as before. Loading a lazy node returns its Cmd either way.
	Notify bool

	cursor   int
	expanded map[string]bool
	query    string
	ta       typeahead.State
	// loaded holds the children fetched for lazy nodes, by path; loading the
	// paths whose Load is running. Both are replaced, not edited, like expanded.
	loaded  map[string][]Node
	loading map[string]bool
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
		_, ly := m.Bounds.Local(ev.X, ev.Y)
		if m.Filter {
			ly-- // the query line
		}
		if ly >= 0 && ly < n {
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
	// branch reports that the row has children, loaded or still to load.
	branch bool
	// label is the text drawn: the node's label, sanitised unless Raw.
	label string
	// pos are the rune offsets in label the query matched; nil when the row
	// is shown only because a descendant matched, or there is no query.
	pos []int
	// own reports that label itself matched the query.
	own   bool
	score int
	// open reports that the row's children are listed, whether the person
	// expanded it or the query forced it open.
	open bool
	// forced reports that open comes from the query alone.
	forced bool
	// sib and sibs are the row's place among its visible siblings, from 0.
	sib, sibs int
}

// Query returns the text the tree is filtered by; always empty when Filter
// is off.
func (m Model) Query() string { return m.query }

// SetQuery sets the filter text and puts the cursor on the best matching row.
// It has no effect when Filter is off.
func (m *Model) SetQuery(q string) {
	if !m.Filter {
		return
	}
	m.query = q
	best, row := 0, -1
	for i, it := range m.flatten() {
		if it.own && (row < 0 || it.score > best) {
			best, row = it.score, i
		}
	}
	if row >= 0 {
		m.cursor = row
	} else {
		m.cursor = 0
	}
}

// filtering reports whether a query is narrowing the tree.
func (m Model) filtering() bool { return m.Filter && m.query != "" }

// matchesBelow reports whether any descendant of n matches the query.
func (m Model) matchesBelow(n *Node, path string) bool {
	kids := m.kids(n, path)
	for i := range kids {
		c := &kids[i]
		if _, _, ok := fuzzy.Score(m.query, ansi.Clean(m.Raw, c.Label)); ok || m.matchesBelow(c, path+"."+strconv.Itoa(i)) {
			return true
		}
	}
	return false
}

func (m Model) flatten() []visible {
	var items []visible
	filtering := m.filtering()
	var walk func(nodes []Node, depth int, prefix string)
	walk = func(nodes []Node, depth int, prefix string) {
		start := len(items)
		for i := range nodes {
			path := prefix + strconv.Itoa(i)
			n := &nodes[i]
			v := visible{node: n, depth: depth, path: path, label: ansi.Clean(m.Raw, n.Label)}
			sub := false
			if filtering {
				score, pos, ok := fuzzy.Score(m.query, v.label)
				sub = len(m.kids(n, path)) > 0 && m.matchesBelow(n, path)
				if !ok && !sub {
					continue
				}
				v.own, v.pos, v.score = ok, pos, score
			}
			kids := m.kids(n, path)
			branch := len(kids) > 0 || n.Load != nil
			v.branch = branch
			v.open = branch && m.expanded[path]
			if branch && filtering && sub && !v.open {
				v.open, v.forced = true, true
			}
			items = append(items, v)
			if v.open {
				walk(kids, depth+1, path+".")
			}
		}
		// Sibling places count only the rows that survived the filter.
		n := 0
		for j := start; j < len(items); j++ {
			if items[j].depth == depth {
				n++
			}
		}
		k := 0
		for j := start; j < len(items); j++ {
			if items[j].depth == depth {
				items[j].sib, items[j].sibs = k, n
				k++
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
			HasChildren: item.branch,
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
	if lm, ok := msg.(LoadedMsg); ok {
		return m.loadedMsg(lm)
	}
	ev, isMouse := msg.(tui.MouseEvent)
	if _, isKey := msg.(tui.Key); !isKey && !isMouse {
		return m, nil
	}
	items := m.flatten()
	if len(items) == 0 {
		if k, ok := msg.(tui.Key); ok && m.Filter {
			return m.editQuery(k), nil
		}
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
		if item.branch {
			if !item.forced {
				return m.flip(item)
			}
			return m, nil
		}
		node, path := item.node, item.path
		return m, func() tui.Msg { return SelectedMsg{Node: node, Path: path} }
	case keymap.Matches(msg, k.Expand):
		if item := items[m.cursor]; item.branch && !m.expanded[item.path] {
			return m.flip(item)
		}
	case keymap.Matches(msg, k.Collapse):
		if item := items[m.cursor]; item.branch && m.expanded[item.path] && !item.forced {
			return m.flip(item)
		}
	case m.Filter:
		return m.editQuery(msg.(tui.Key)), nil
	case m.TypeAhead:
		if text, ok := typeahead.Printable(msg); ok {
			prefix, from := m.ta.Feed(text, m.cursor)
			if i, ok := typeahead.Find(prefix, from, len(items), func(i int) string { return items[i].label }); ok {
				m.cursor = i
			}
		}
	}
	return m, nil
}

// kids returns the children of n, which sits at path: its own, or those a
// Load fetched.
func (m Model) kids(n *Node, path string) []Node {
	if len(n.Children) > 0 {
		return n.Children
	}
	return m.loaded[path]
}

// flip opens or closes the branch item. Opening a lazy node that has not been
// loaded starts the load instead, and the node opens when LoadedMsg arrives.
func (m Model) flip(item visible) (Model, tui.Cmd) {
	path := item.path
	if !m.expanded[path] && item.node.Load != nil && len(item.node.Children) == 0 {
		if _, done := m.loaded[path]; !done {
			if m.loading[path] {
				return m, nil
			}
			m.loading = withKey(m.loading, path, true)
			load, name := item.node.Load, m.Name
			return m, func() tui.Msg { return LoadedMsg{Name: name, Path: path, Children: load()} }
		}
	}
	opened := !m.expanded[path]
	m.toggle(path)
	return m, m.notify(item.node, path, opened)
}

// notify is the Cmd for ExpandedMsg or CollapsedMsg, nil unless Notify is on.
func (m Model) notify(n *Node, path string, opened bool) tui.Cmd {
	if !m.Notify {
		return nil
	}
	if opened {
		return func() tui.Msg { return ExpandedMsg{Node: n, Path: path} }
	}
	return func() tui.Msg { return CollapsedMsg{Node: n, Path: path} }
}

// loadedMsg stores the children a Load returned and opens the node.
func (m Model) loadedMsg(lm LoadedMsg) (Model, tui.Cmd) {
	if lm.Name != m.Name || !m.loading[lm.Path] {
		return m, nil
	}
	children := lm.Children
	if children == nil {
		children = []Node{}
	}
	loaded := make(map[string][]Node, len(m.loaded)+1)
	for k, v := range m.loaded {
		loaded[k] = v
	}
	loaded[lm.Path] = children
	m.loaded = loaded
	m.loading = withKey(m.loading, lm.Path, false)
	m.expanded = withKey(m.expanded, lm.Path, true)
	var node *Node
	for _, it := range m.flatten() {
		if it.path == lm.Path {
			node = it.node
			break
		}
	}
	return m, m.notify(node, lm.Path, true)
}

// withKey returns a copy of set with key set to v, dropping it when false.
func withKey(set map[string]bool, key string, v bool) map[string]bool {
	next := make(map[string]bool, len(set)+1)
	for k, on := range set {
		next[k] = on
	}
	if v {
		next[key] = true
	} else {
		delete(next, key)
	}
	return next
}

// LoadedMsg carries the children a Node.Load returned. Update returns it from a
// Cmd; pass it back to the Model's Update like any other Msg. Name is the
// Model's Name, so a second tree ignores it.
type LoadedMsg struct {
	Name     string
	Path     string
	Children []Node
}

// ExpandedMsg is delivered when a branch opens, when Notify is on or after a
// lazy node loads. Node is nil if the node is not visible (a filter hides it).
type ExpandedMsg struct {
	Node *Node
	Path string
}

// CollapsedMsg is delivered when a branch closes and Notify is on.
type CollapsedMsg struct {
	Node *Node
	Path string
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

// View renders the currently visible rows (expanded subtrees included),
// indented by depth, with the row under the cursor highlighted. With Filter
// on, a first line shows the query and only matching rows and their
// ancestors follow, matched runes underlined.
func (m Model) View() string {
	items := m.flatten()
	var b strings.Builder
	plain := ansi.NewStyle()
	if m.Filter {
		b.WriteString("/ " + m.query)
		if m.query != "" {
			b.WriteString(plain.Faint().Render(" (" + strconv.Itoa(len(items)) + ")"))
		}
		if len(items) == 0 {
			if m.query != "" {
				b.WriteString("\n  " + plain.Faint().Render("no matches"))
			}
			return b.String()
		}
		b.WriteByte('\n')
	}
	if len(items) == 0 {
		return ""
	}
	cursor := clamp(m.cursor, 0, len(items)-1)

	cursorStyle := m.themed().ResolvedStates().Selected.Bold()

	for i, item := range items {
		prefix, base := "  ", plain
		if i == cursor {
			prefix, base = "> ", cursorStyle
		}
		var label strings.Builder
		for _, run := range fuzzy.Split(item.label, item.pos) {
			st := base
			if run.Match {
				st = base.Underline()
			}
			if i == cursor || run.Match {
				label.WriteString(st.Render(run.Text))
			} else {
				label.WriteString(run.Text)
			}
		}
		marker := " "
		if item.branch {
			if item.open {
				marker = m.themed().GlyphSet().Expanded
			} else {
				marker = m.themed().GlyphSet().Collapsed
			}
		}
		b.WriteString(prefix + strings.Repeat("  ", item.depth) + marker + " " + label.String())
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
	items := m.flatten()
	var lines []string
	if m.filtering() {
		if len(items) == 0 {
			return "Filter " + strconv.Quote(m.query) + ", no matches"
		}
		lines = append(lines, "Filter "+strconv.Quote(m.query)+", "+strconv.Itoa(len(items))+" rows")
	}
	for i, it := range items {
		line := it.label + ", level " + strconv.Itoa(it.depth+1)
		if it.branch {
			if it.open {
				line += ", expanded"
			} else {
				line += ", collapsed"
			}
		}
		line += ", item " + strconv.Itoa(it.sib+1) + " of " + strconv.Itoa(it.sibs)
		if i == m.cursor {
			line += ", selected"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
