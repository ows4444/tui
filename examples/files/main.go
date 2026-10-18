// Command files is a two-pane file browser: folders on the left, the current
// folder's files on the right with a preview underneath. It only reads.
//
// Mouse: click a folder in the tree or a row in the list; double-click a row to
// open it; right-click a row for a menu; click a place in the path bar to go
// back up; drag the dividers to resize the panes; scroll any pane.
// Keyboard: Tab changes pane, arrows move, Enter opens, Backspace goes up,
// . shows hidden files, s sorts the list, / filters it, ? help, q quits.
//
//	go run ./examples/files [dir]
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/breadcrumb"
	"github.com/ows4444/tui/contextmenu"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/splitpane"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/viewport"
	"github.com/ows4444/tui/widgets"
)

type pane int

const (
	paneTree pane = iota
	paneList
	panePreview
	numPanes
)

const (
	treeDepth  = 6   // levels of folders the tree reads
	treeBudget = 400 // folders the tree reads in all
	minWidth   = 70  // columns below which the tree goes
	minHeight  = 14  // rows below which the preview goes
)

type model struct {
	th   ui.Theming // the theme the Program chose; see SetTheme
	fsys fs.FS
	root string // name shown for the top folder

	dir        string // folder shown in the list, "." for the top
	items      []item
	showHidden bool

	tree  treeview.Model
	paths map[string]string // tree path -> folder path; built once, never changed

	table datatable.Model
	crumb breadcrumb.Model
	prev  viewport.Model
	// prevOf is the entry the preview shows, or -1.
	prevOf int

	split1, split2 splitpane.Model // tree | (list / preview)
	menu           contextmenu.Model
	menuItem       int // the item the menu was opened on

	focus pane
	help  bool

	width, height int
}

func newModel(fsys fs.FS, root string) model {
	m := model{fsys: fsys, root: root, prevOf: -1, focus: paneList}
	m.buildTree()
	m.table = datatable.New([]string{"Name", "Size", "Modified"}, nil)
	m.table.Mouse, m.table.Filter, m.table.TypeAhead = true, true, true
	m.crumb = breadcrumb.New(root)
	m.crumb.Mouse, m.crumb.Separator = true, "▸"
	m.prev = viewport.New(40, 10)
	m.prev.Mouse = true
	m.tree.Mouse, m.tree.TypeAhead = true, true

	m.split1 = splitpane.New(nil, nil)
	m.split1.Min1, m.split1.Min2, m.split1.Mouse = 16, 30, true
	m.split2 = splitpane.New(nil, nil)
	m.split2.Direction = splitpane.Rows
	m.split2.Min1, m.split2.Min2, m.split2.Mouse = 4, 4, true
	// Totals stand in until the first frame is laid out; SetPos clamps to them.
	m.split1.SetTotal(80)
	m.split2.SetTotal(21)
	m.split1.SetPos(28) // the tree is a narrow column by default
	m.split2.SetPos(12)

	m.menu = contextmenu.New(
		contextmenu.Item{Label: "Open", Value: "open", Accel: 'o'},
		contextmenu.Item{Label: "Copy path", Value: "copy", Accel: 'c'},
		contextmenu.Item{Separator: true},
		contextmenu.Item{Label: "Go up", Value: "up", Accel: 'u', Shortcut: "bksp"},
	)
	m.menu.Mouse = true
	return m.cd(".")
}

// buildTree reads the folders below the root into the tree.
func (m *model) buildTree() {
	budget := treeBudget
	t := buildTree(m.fsys, ".", m.root, treeDepth, m.showHidden, &budget)
	m.paths = map[string]string{}
	m.tree = treeview.New(t.node("0", m.paths))
	m.tree = treeKey(m.tree, tui.Key{Type: tui.KeyRight}) // open the top folder
}

func (m model) Init() tui.Cmd { return nil }

// treeKey sends one key to the tree. A tree built from fixed nodes answers at
// once, so the Cmd it returns (a notification nobody asked for) is dropped.
func treeKey(t treeview.Model, k tui.Key) treeview.Model {
	t, _ = t.Update(k)
	return t
}

// ---- navigation ------------------------------------------------------------

// cd shows folder dir: its list, the path bar, and the tree row, which is
// opened and selected when the tree has it.
func (m model) cd(dir string) model {
	items, err := list(m.fsys, dir, m.showHidden)
	if err != nil {
		return m
	}
	m.dir, m.items, m.prevOf = dir, items, -1
	rows := make([][]string, len(items))
	for i, it := range items {
		rows[i] = row(it)
	}
	m.table.SetRows(rows)
	m.table.SetCursor(0)
	m.crumb.Items = crumbs(m.root, dir)
	m.crumb.SetCursor(len(m.crumb.Items) - 1)
	m.revealInTree(dir)
	return m.follow()
}

func row(it item) []string {
	when := "—"
	if !it.mod.IsZero() {
		when = it.mod.Format("2006-01-02 15:04")
	}
	if it.dir {
		return []string{"▸ " + it.name + "/", "—", when}
	}
	return []string{"  " + it.name, humanSize(it.size), when}
}

// revealInTree opens every folder above dir and puts the tree's cursor on it.
func (m *model) revealInTree(dir string) {
	var target string
	for tp, fp := range m.paths {
		if fp == dir {
			target = tp
		}
	}
	if target == "" {
		return
	}
	parts := strings.Split(target, ".")
	for n := 1; n < len(parts); n++ {
		anc := strings.Join(parts[:n], ".")
		if !m.tree.IsExpanded(anc) {
			m.selectTreeRow(anc)
			m.tree = treeKey(m.tree, tui.Key{Type: tui.KeyRight})
		}
	}
	m.selectTreeRow(target)
}

func (m *model) selectTreeRow(treePath string) {
	for i, r := range m.tree.VisibleRows() {
		if r.Path == treePath {
			m.tree.SetCursor(i)
			return
		}
	}
}

// treeDir is the folder under the tree's cursor.
func (m model) treeDir() (string, bool) {
	rows := m.tree.VisibleRows()
	if c := m.tree.Cursor(); c >= 0 && c < len(rows) {
		p, ok := m.paths[rows[c].Path]
		return p, ok
	}
	return "", false
}

// current is the item under the list's cursor.
func (m model) current() (item, bool) {
	i := m.table.SourceRow(m.table.Cursor())
	if i < 0 || i >= len(m.items) {
		return item{}, false
	}
	return m.items[i], true
}

func (m model) fullPath(it item) string { return path.Join(m.dir, it.name) }

// summary describes a folder for the preview: how many folders and files it
// holds.
func (m model) summary(dir string) string {
	items, err := list(m.fsys, dir, m.showHidden)
	if err != nil {
		return "Cannot read: " + err.Error()
	}
	dirs := 0
	for _, it := range items {
		if it.dir {
			dirs++
		}
	}
	return fmt.Sprintf("%d folders, %d files", dirs, len(items)-dirs)
}

// follow makes the preview match the list's cursor: the file's start, or the
// folder's size.
func (m model) follow() model {
	it, ok := m.current()
	if !ok {
		return m
	}
	i := m.table.SourceRow(m.table.Cursor())
	if i == m.prevOf {
		return m
	}
	m.prevOf = i
	if it.dir {
		m.prev.SetContent(fmt.Sprintf("%s/\n\n%s\nEnter opens it.", it.name, m.summary(m.fullPath(it))))
	} else {
		m.prev.SetContent(preview(m.fsys, m.fullPath(it), it.size))
	}
	m.prev.GotoTop()
	return m
}

func (m model) open(i int) model {
	if i < 0 || i >= len(m.items) {
		return m
	}
	it := m.items[i]
	if it.dir {
		return m.cd(m.fullPath(it))
	}
	m.focus = panePreview
	return m
}

func (m model) up() model {
	if m.dir == "." {
		return m
	}
	return m.cd(path.Dir(m.dir))
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width, m.height = rs.Width, rs.Height
	}
	m = m.relayout()
	switch msg := msg.(type) {
	case tui.MouseEvent:
		return m.mouse(msg)
	case tui.Key:
		return m.key(msg)
	case datatable.SelectedMsg:
		return m.open(msg.SourceRow), nil
	case breadcrumb.ChosenMsg:
		return m.cd(dirAt(m.crumb.Items, msg.Index+1)), nil
	case contextmenu.SelectedMsg:
		return m.menuChoice(msg.Item.Value)
	}
	return m, nil
}

func (m model) menuChoice(v string) (tui.Model, tui.Cmd) {
	switch v {
	case "open":
		return m.open(m.menuItem), nil
	case "copy":
		if m.menuItem >= 0 && m.menuItem < len(m.items) {
			return m, tui.WriteClipboard(m.fullPath(m.items[m.menuItem]))
		}
	case "up":
		return m.up(), nil
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if m.menu.Open() {
		next, cmd := m.menu.Update(k)
		m.menu = next
		return m, cmd
	}
	if m.table.Filtering() && m.focus == paneList {
		return m.toPane(k)
	}
	switch k.Type {
	case tui.KeyCtrlC:
		return m, tui.Quit()
	case tui.KeyEsc:
		if m.help {
			m.help = false
			return m, nil
		}
		return m, tui.Quit()
	case tui.KeyTab:
		m.focus = (m.focus + 1) % numPanes
		return m, nil
	case tui.KeyBackspace:
		return m.up(), nil
	case tui.KeyRunes:
		switch k.Text {
		case "q":
			return m, tui.Quit()
		case "?":
			m.help = !m.help
			return m, nil
		case ".":
			m.showHidden = !m.showHidden
			m.buildTree()
			return m.cd(m.dir), nil
		}
	}
	return m.toPane(k)
}

// toPane gives a message to the focused pane's widget.
func (m model) toPane(msg tui.Msg) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	switch m.focus {
	case paneTree:
		m.tree, cmd = m.tree.Update(msg)
		if d, ok := m.treeDir(); ok && d != m.dir {
			m = m.cd(d)
			m.focus = paneTree
		}
	case paneList:
		m.table, cmd = m.table.Update(msg)
		m = m.follow()
	case panePreview:
		m.prev, cmd = m.prev.Update(msg)
	}
	return m, cmd
}

// geometry is where each pane is drawn. A split pane draws its two panes
// itself and does not report them to the layout, so the rectangles come from
// the split sizes, the same numbers the split draws with.
type geometry struct {
	crumb, split1, split2 layout.Rect
	tree, list, preview   layout.Rect // the widgets, below their titles
	hasTree, hasPreview   bool
}

func (m model) geometry() geometry {
	sz := m.size()
	g := geometry{crumb: layout.Rect{X: 0, Y: 1, W: sz.W, H: 1}}
	body := layout.Rect{X: 0, Y: 2, W: sz.W, H: max(0, sz.H-3)}
	right := body
	if m.showTree() {
		s1 := m.split1
		s1.Bounds = body
		s1.SetTotal(body.W)
		first, second := s1.Sizes()
		g.hasTree, g.split1 = true, body
		g.tree = layout.Rect{X: body.X, Y: body.Y + 1, W: first, H: max(0, body.H-1)}
		right = layout.Rect{X: body.X + first + 1, Y: body.Y, W: second, H: body.H}
	}
	g.list = right
	if m.showPreview() {
		s2 := m.split2
		s2.Bounds = right
		s2.SetTotal(right.H)
		first, second := s2.Sizes()
		g.hasPreview, g.split2 = true, right
		g.list = layout.Rect{X: right.X, Y: right.Y, W: right.W, H: first}
		g.preview = layout.Rect{X: right.X, Y: right.Y + first + 2, W: right.W, H: max(0, second-1)}
	}
	return g
}

// paneAt is the pane under a screen cell.
func (m model) paneAt(x, y int) (pane, bool) {
	g := m.geometry()
	switch {
	case g.hasTree && g.tree.Contains(x, y):
		return paneTree, true
	case g.hasPreview && g.preview.Contains(x, y):
		return panePreview, true
	case g.list.Contains(x, y):
		return paneList, true
	}
	return 0, false
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	if m.menu.Open() {
		m.menu, cmd = m.menu.Update(ev)
		return m, cmd
	}
	// The dividers see every event: they act on a press on themselves and on
	// the drag that follows.
	was := m.split1.Dragging() || m.split2.Dragging()
	m.split1, _ = m.split1.Update(ev)
	m.split2, _ = m.split2.Update(ev)
	if was || m.split1.Dragging() || m.split2.Dragging() {
		return m.relayout(), nil
	}
	if m.geometry().crumb.Contains(ev.X, ev.Y) {
		m.crumb, cmd = m.crumb.Update(ev)
		return m, cmd
	}
	p, ok := m.paneAt(ev.X, ev.Y)
	if !ok {
		return m, nil
	}
	if ev.Action == tui.MouseActionPress && (ev.Button == tui.MouseButtonLeft || ev.Button == tui.MouseButtonRight) {
		m.focus = p
	}
	switch p {
	case paneTree:
		m.tree, cmd = m.tree.Update(ev)
		if d, ok := m.treeDir(); ok && d != m.dir && ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress {
			m = m.cd(d)
			m.focus = paneTree
		}
	case paneList:
		return m.listMouse(ev)
	case panePreview:
		m.prev, cmd = m.prev.Update(ev)
	}
	return m, cmd
}

func (m model) listMouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	left := ev
	if ev.Button == tui.MouseButtonRight {
		left.Button = tui.MouseButtonLeft // a right press selects the row too
	}
	var cmd tui.Cmd
	m.table, cmd = m.table.Update(left)
	m = m.follow()
	switch {
	case ev.Button == tui.MouseButtonRight && ev.Action == tui.MouseActionPress:
		if i := m.table.SourceRow(m.table.Cursor()); i >= 0 {
			m.menuItem = i
			m.menu.ScreenW, m.menu.ScreenH = m.size().W, m.size().H
			m.menu.Show(ev.X, ev.Y)
		}
	case ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress && ev.Clicks == 2:
		if ev.Y > m.geometry().list.Y+1 { // below the header
			return m.open(m.table.SourceRow(m.table.Cursor())), cmd
		}
	}
	return m, cmd
}

// relayout hands every widget the rectangle it is drawn in.
func (m model) relayout() model {
	g := m.geometry()
	m.split1.Bounds, m.split2.Bounds = g.split1, g.split2
	m.split1.SetTotal(g.split1.W)
	m.split2.SetTotal(g.split2.H)
	m.tree.Bounds = g.tree
	m.table.Bounds, m.table.Height, m.table.Width = g.list, max(1, g.list.H-2), g.list.W
	m.crumb.Bounds = g.crumb
	m.prev.Bounds, m.prev.Width, m.prev.Height = g.preview, g.preview.W, g.preview.H
	return m
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) showTree() bool    { return m.size().W >= minWidth }
func (m model) showPreview() bool { return m.size().H >= minHeight }

func (m model) title(text string, p pane, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	if m.focus == p {
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	return layout.FlexChild{Node: layout.Block(st.Render(text))}
}

func (m model) screen() layout.Node {
	th := m.theme()
	hide := ""
	if m.showHidden {
		hide = "  (hidden shown)"
	}
	head := ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary).Render(" Files ") +
		ansi.NewStyle().Faint().Render("  "+m.root+hide)

	crumb := layout.Block(m.crumb.View())

	tree := layout.Column(0,
		m.title("Folders", paneTree, th),
		layout.Fill(m.tree.LayoutNode()),
	)
	list := layout.Column(0, layout.Fill(m.table.LayoutNode()))
	var right layout.Node = list
	if m.showPreview() {
		prev := layout.Column(0,
			m.title("Preview", panePreview, th),
			layout.Fill(m.prev.LayoutNode()),
		)
		s2 := m.split2
		s2.First, s2.Second = list, prev
		right = s2.LayoutNode()
	}
	var body layout.Node = right
	if m.showTree() {
		s1 := m.split1
		s1.First, s1.Second = tree, right
		body = s1.LayoutNode()
	}

	status := m.status(th)
	root := layout.Column(0,
		layout.FlexChild{Node: layout.Block(head)},
		layout.FlexChild{Node: crumb},
		layout.Fill(body),
		layout.FlexChild{Node: layout.Block(status)},
	)
	if m.help {
		return layout.Stack(root, layout.Layer(5, layout.Anchor(layout.Center, 0, 0, m.helpBox(th))))
	}
	return root
}

func (m model) status(th theme.Theme) string {
	left := fmt.Sprintf("%d items", len(m.items))
	if it, ok := m.current(); ok {
		if it.dir {
			left = it.name + "/"
		} else {
			left = fmt.Sprintf("%s  %s", it.name, humanSize(it.size))
		}
	}
	hints := widgets.Footer(max(10, m.size().W-ansi.Width(left)-4),
		widgets.Hint{Key: "tab", Action: "pane"}, widgets.Hint{Key: "enter", Action: "open"},
		widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	gap := max(1, m.size().W-ansi.Width(left)-ansi.Width(hints)-1)
	return ansi.Truncate(" "+ansi.NewStyle().Foreground(th.Muted).Render(left)+strings.Repeat(" ", gap)+hints, m.size().W)
}

func (m model) helpBox(th theme.Theme) layout.Node {
	lines := []string{
		"Mouse",
		"  click          select a folder or file",
		"  double-click   open",
		"  right-click    menu",
		"  drag divider   resize panes",
		"  wheel          scroll",
		"Keys",
		"  tab            next pane",
		"  enter          open",
		"  backspace      up one folder",
		"  .              hidden files",
		"  s  /           sort, filter the list",
		"  ?              close this help",
	}
	return layout.BoxNode(layout.NewBox().Border(th.Border).PaddingAll(1), layout.Block(strings.Join(lines, "\n")))
}

func (m model) View() string {
	out := layout.DrawTight(m.screen(), m.size())
	if m.menu.Open() {
		return m.menu.Render(out)
	}
	return out
}

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	abs, err := filepath.Abs(dir)
	if err == nil {
		_, err = os.Stat(abs) // #nosec G703 -- the folder the user asked to browse
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	m := newModel(os.DirFS(abs), filepath.Base(abs))
	p := ui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion), tui.WithMultiClick(400*time.Millisecond))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
