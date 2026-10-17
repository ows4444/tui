// Command jsonx explores a JSON document as a tree: open and close branches,
// search keys and values, see the path of the selected value and a preview of
// it, and copy the path or the value. Object members keep their order and
// numbers keep their text. With no file and no pipe it opens a sample.
//
// Mouse: click a row to select it, click the arrow (or double-click) to open
// or close it, scroll the wheel, click Path or Value in the top line to copy.
// Keyboard: arrows or hjkl move (Right opens or steps in, Left closes or steps
// out), Enter or Space toggles, * opens everything below, E and C open and
// close everything, / searches (Enter keeps the search, Esc drops it), n and N
// go to the next and previous match, p copies the path, y the value, F1 shows
// help, q or Ctrl+C quits. Copying uses the terminal's clipboard escape
// (OSC 52), which not every terminal allows.
//
//	go run ./examples/jsonx [file.json]
//	curl -s https://example.com/api | go run ./examples/jsonx
package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	chrome   = 4  // title, path, status, hints
	wideFrom = 70 // narrowest terminal that shows the preview beside the tree
)

type row struct {
	n     *node
	depth int
}

type model struct {
	root   *node
	source string
	open   map[*node]bool // containers that are open
	cur    int            // index into rows()
	top    int

	query  string
	hits   []*node
	hitSet map[*node]bool
	typing bool
	in     textinput.Model
	origin int // the cursor when a search began, so Esc can go back
	msg    string
	help   bool
	hover  string

	width, height int
}

func newModel(root *node, source string) model {
	in := textinput.New()
	in.Prompt = "/ "
	in.Width = 40
	m := model{root: root, source: source, open: map[*node]bool{root: true}, in: in}
	return m
}

func (m model) Init() tui.Cmd { return nil }

// ---- rows ------------------------------------------------------------------

// rows are the visible lines: the root, and below each open container its
// children.
func (m model) rows() []row {
	var out []row
	var walk func(*node, int)
	walk = func(n *node, d int) {
		out = append(out, row{n, d})
		if n.container() && m.open[n] {
			for _, c := range n.children {
				walk(c, d+1)
			}
		}
	}
	walk(m.root, 0)
	return out
}

func (m model) selected() *node {
	rs := m.rows()
	return rs[max(0, min(m.cur, len(rs)-1))].n
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) visRows() int { return max(1, m.size().H-chrome) }

func (m model) wide() bool { return m.size().W >= wideFrom }

func (m model) treeW() int {
	if m.wide() {
		return m.size().W * 55 / 100
	}
	return m.size().W
}

// keep clamps the cursor and scrolls to show it.
func (m model) keep() model {
	n := len(m.rows())
	m.cur = max(0, min(m.cur, n-1))
	if m.cur < m.top {
		m.top = m.cur
	}
	if m.cur >= m.top+m.visRows() {
		m.top = m.cur - m.visRows() + 1
	}
	m.top = max(0, min(m.top, max(0, n-m.visRows())))
	return m
}

func (m model) indexOf(n *node) int {
	for i, r := range m.rows() {
		if r.n == n {
			return i
		}
	}
	return -1
}

// reveal opens every ancestor of n and puts the cursor on it.
func (m model) reveal(n *node) model {
	open := make(map[*node]bool, len(m.open))
	for k, v := range m.open {
		open[k] = v
	}
	for p := n.parent; p != nil; p = p.parent {
		open[p] = true
	}
	m.open = open
	if i := m.indexOf(n); i >= 0 {
		m.cur = i
	}
	return m.keep()
}

func (m model) setOpen(n *node, on bool) model {
	open := make(map[*node]bool, len(m.open))
	for k, v := range m.open {
		open[k] = v
	}
	open[n] = on
	m.open = open
	return m.keep()
}

// setAll opens or closes n and everything below it.
func (m model) setAll(n *node, on bool) model {
	open := make(map[*node]bool, len(m.open))
	for k, v := range m.open {
		open[k] = v
	}
	var walk func(*node)
	walk = func(x *node) {
		if x.container() {
			open[x] = on
		}
		for _, c := range x.children {
			walk(c)
		}
	}
	walk(n)
	open[m.root] = true // the root stays open so there is always something to see
	m.open = open
	return m.keep()
}

// ---- search ----------------------------------------------------------------

func (m model) search(q string) model {
	m.query = q
	m.hits = m.root.matches(q)
	m.hitSet = make(map[*node]bool, len(m.hits))
	for _, h := range m.hits {
		m.hitSet[h] = true
	}
	return m
}

// jump goes to the first match at or after the cursor's place in the document
// (dir 1), or the last one before it (dir -1), wrapping round.
func (m model) jump(dir int, from *node) model {
	if len(m.hits) == 0 {
		return m
	}
	order := m.docOrder()
	pos := order[from]
	best := -1
	for i, h := range m.hits {
		p := order[h]
		if (dir > 0 && p > pos) || (dir < 0 && p < pos) {
			if dir > 0 {
				best = i
				break
			}
			best = i
		}
	}
	if best < 0 {
		best = 0
		if dir < 0 {
			best = len(m.hits) - 1
		}
	}
	return m.reveal(m.hits[best])
}

// docOrder numbers every node by its place in the document.
func (m model) docOrder() map[*node]int {
	order := map[*node]int{}
	var walk func(*node)
	walk = func(n *node) {
		order[n] = len(order)
		for _, c := range n.children {
			walk(c)
		}
	}
	walk(m.root)
	return order
}

// firstHitFrom is like jump forward but a match on the node itself counts.
func (m model) firstHitFrom(n *node) model {
	if len(m.hits) == 0 {
		return m
	}
	if m.hitSet[n] {
		return m
	}
	return m.jump(1, n)
}

func (m model) matchNumber() int {
	sel := m.selected()
	for i, h := range m.hits {
		if h == sel {
			return i + 1
		}
	}
	return 0
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.keep(), nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if m.typing {
		return m.typeKey(k)
	}
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	m.msg = ""
	if next, ok := m.moveKey(k); ok {
		return next.keep(), nil
	}
	return m.actionKey(k)
}

func (m model) moveKey(k tui.Key) (model, bool) {
	page := max(1, m.visRows()-1)
	switch k.Type {
	case tui.KeyUp:
		m.cur--
	case tui.KeyDown:
		m.cur++
	case tui.KeyPgUp:
		m.cur -= page
	case tui.KeyPgDown:
		m.cur += page
	case tui.KeyHome:
		m.cur = 0
	case tui.KeyEnd:
		m.cur = len(m.rows()) - 1
	case tui.KeyRight:
		return m.stepIn(), true
	case tui.KeyLeft:
		return m.stepOut(), true
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.cur--
		case "j":
			m.cur++
		case "g":
			m.cur = 0
		case "G":
			m.cur = len(m.rows()) - 1
		case "l":
			return m.stepIn(), true
		case "h":
			return m.stepOut(), true
		default:
			return m, false
		}
	default:
		return m, false
	}
	return m, true
}

// stepIn opens a closed container, or moves to the first child of an open one.
func (m model) stepIn() model {
	n := m.selected()
	switch {
	case n.container() && !m.open[n]:
		return m.setOpen(n, true)
	case n.container() && len(n.children) > 0:
		m.cur++
	}
	return m
}

// stepOut closes an open container, or moves to the parent.
func (m model) stepOut() model {
	n := m.selected()
	switch {
	case n.container() && m.open[n] && n != m.root:
		return m.setOpen(n, false)
	case n.parent != nil:
		m.cur = m.indexOf(n.parent)
	}
	return m
}

func (m model) toggle() model {
	n := m.selected()
	if !n.container() || n == m.root {
		return m
	}
	return m.setOpen(n, !m.open[n])
}

func (m model) actionKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEnter:
		return m.toggle(), nil
	case tui.KeyEsc:
		return m.search("").clearHits(), nil
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) clearHits() model {
	m.msg = ""
	return m
}

func (m model) runeKey(r string) (tui.Model, tui.Cmd) {
	switch r {
	case " ":
		return m.toggle(), nil
	case "q":
		return m, tui.Quit()
	case "*":
		return m.setAll(m.selected(), true), nil
	case "E":
		return m.setAll(m.root, true), nil
	case "C":
		m = m.setAll(m.root, false)
		m.cur = 0
		return m.keep(), nil
	case "/":
		m.typing, m.origin = true, m.cur
		m.in.SetValue(m.query)
		m.in.CursorEnd()
		return m, m.in.Focus()
	case "n":
		return m.jump(1, m.selected()), nil
	case "N":
		return m.jump(-1, m.selected()), nil
	case "p":
		return m.copyPath()
	case "y":
		return m.copyValue()
	}
	return m, nil
}

func (m model) typeKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.typing = false
		m.in.Blur()
		m = m.search("")
		m.cur = m.origin
		return m.keep(), nil
	case tui.KeyEnter:
		m.typing = false
		m.in.Blur()
		if len(m.hits) == 0 && m.query != "" {
			m.msg = "No match for " + strconv.Quote(m.query)
		}
		return m, nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	if m.in.Value() != m.query {
		m = m.search(m.in.Value())
		m.cur = m.origin
		m = m.firstHitFrom(m.selected())
	}
	return m, cmd
}

func (m model) copyPath() (tui.Model, tui.Cmd) {
	p := m.selected().path()
	m.msg = "Copied " + p + " (if your terminal allows it)"
	return m, tui.WriteClipboard(p)
}

func (m model) copyValue() (tui.Model, tui.Cmd) {
	n := m.selected()
	what := "value"
	if n.container() {
		what = n.summary()
	}
	m.msg = "Copied " + what + " (if your terminal allows it)"
	return m, tui.WriteClipboard(n.copyText())
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) button(x, y int) string {
	for _, id := range []string{"copy-path", "copy-value"} {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.typing || m.help {
		return m, nil
	}
	if ev.Action == tui.MouseActionMotion {
		m.hover = m.button(ev.X, ev.Y)
		return m, nil
	}
	n := len(m.rows())
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.top = max(0, m.top-3)
		m.cur = min(m.cur, m.top+m.visRows()-1)
		return m, nil
	case tui.MouseButtonWheelDown:
		m.top = min(max(0, n-m.visRows()), m.top+3)
		m.cur = max(m.cur, m.top)
		return m, nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	switch m.button(ev.X, ev.Y) {
	case "copy-path":
		return m.copyPath()
	case "copy-value":
		return m.copyValue()
	}
	return m.clickRow(ev)
}

func (m model) clickRow(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	r, ok := layout.RectOf(m.screen(), m.size(), "tree")
	if !ok || !inside(r, ev.X, ev.Y) {
		return m, nil
	}
	i := m.top + ev.Y - r.Y
	rs := m.rows()
	if i >= len(rs) {
		return m, nil
	}
	m.cur = i
	n := rs[i].n
	arrowX := r.X + 2*rs[i].depth
	if n.container() && (ev.Clicks >= 2 || (ev.X >= arrowX && ev.X < arrowX+2)) {
		m = m.toggle()
	}
	return m.keepClick(), nil
}

// keepClick clamps without scrolling the clicked row to a different place.
func (m model) keepClick() model {
	top := m.top
	m = m.keep()
	m.top = top
	return m
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

// mark underlines every occurrence of the query in s (plain text) with st.
func (m model) mark(s string, st ansi.Style) string {
	if m.query == "" {
		return st.Render(s)
	}
	low, q := strings.ToLower(s), strings.ToLower(m.query)
	var b strings.Builder
	for {
		i := strings.Index(low, q)
		if i < 0 {
			b.WriteString(st.Render(s))
			return b.String()
		}
		b.WriteString(st.Render(s[:i]) + st.Bold().Underline().Reverse().Render(s[i:i+len(q)]))
		s, low = s[i+len(q):], low[i+len(q):]
	}
}

func (m model) valueStyle(n *node, th theme.Theme) ansi.Style {
	switch n.kind {
	case str:
		return ansi.NewStyle().Foreground(th.Success)
	case num:
		return ansi.NewStyle().Foreground(th.Info)
	case boolean:
		return ansi.NewStyle().Foreground(th.Warning)
	case null:
		return ansi.NewStyle().Faint()
	}
	return ansi.NewStyle().Faint()
}

func (m model) rowLine(r row, w int, th theme.Theme) string {
	n := r.n
	arrow := "  "
	if n.container() {
		arrow = "▸ "
		if m.open[n] {
			arrow = "▾ "
		}
	}
	key := "$"
	if n.parent != nil {
		key = n.key
		if n.parent.kind == array {
			key = "[" + n.key + "]"
		}
	}
	var b strings.Builder
	b.WriteString(strings.Repeat("  ", r.depth) + arrow)
	switch {
	case n.parent == nil:
		b.WriteString(ansi.NewStyle().Foreground(th.Primary).Render(key) + " ")
	default:
		b.WriteString(m.mark(key, ansi.NewStyle().Foreground(th.Primary)) + ansi.NewStyle().Faint().Render(": "))
	}
	if n.container() {
		if !m.open[n] || len(n.children) == 0 {
			b.WriteString(ansi.NewStyle().Faint().Render(n.summary()))
		}
	} else {
		b.WriteString(m.mark(n.scalarText(), m.valueStyle(n, th)))
	}
	line := ansi.Truncate(b.String(), w)
	if m.hitSet[n] {
		line = pad(line, w-2) + ansi.NewStyle().Foreground(th.Warning).Render(" ●")
	}
	return line
}

func (m model) treeNode(th theme.Theme) layout.Node {
	w, h := m.treeW(), m.visRows()
	rs := m.rows()
	var lines []layout.FlexChild
	for i := 0; i < h; i++ {
		idx := m.top + i
		line := ""
		if idx < len(rs) {
			line = m.rowLine(rs[idx], w, th)
			if idx == m.cur {
				line = m.cursorLine(rs[idx], w, th)
			}
		}
		lines = append(lines, layout.FlexChild{Node: layout.Block(pad(line, w))})
	}
	return layout.Named("tree", layout.Fixed(layout.Column(0, lines...), layout.Size{W: w, H: h}))
}

// cursorLine is the selected row: the plain text on the selection colour, bold,
// so it reads without relying on the colours of the value kinds.
func (m model) cursorLine(r row, w int, th theme.Theme) string {
	plain := ansi.StripANSI(m.rowLine(r, w, th))
	return ansi.NewStyle().Bold().Background(th.Selection).Render(pad(plain, w))
}

func (m model) previewNode(th theme.Theme) layout.Node {
	w, h := m.size().W-m.treeW()-1, m.visRows()
	n := m.selected()
	head := ansi.NewStyle().Faint().Render(fmt.Sprintf("%s · %s", kindName(n), sizeInfo(n)))
	lines := []string{head}
	for _, l := range strings.Split(n.pretty(8192), "\n") {
		lines = append(lines, l)
	}
	var kids []layout.FlexChild
	rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
	for i := 0; i < h; i++ {
		l := ""
		if i < len(lines) {
			l = ansi.Truncate(strings.ReplaceAll(lines[i], "\t", " "), w)
		}
		kids = append(kids, layout.FlexChild{Node: layout.Block(rule + pad(l, w))})
	}
	return layout.Fixed(layout.Column(0, kids...), layout.Size{W: w + 1, H: h})
}

func kindName(n *node) string {
	return [...]string{"object", "array", "string", "number", "boolean", "null"}[n.kind]
}

func sizeInfo(n *node) string {
	if n.container() {
		return fmt.Sprintf("%d values inside", n.count()-1)
	}
	return fmt.Sprintf("%d characters", len([]rune(n.text)))
}

func (m model) toolButton(id, label string, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == id {
		st = ansi.NewStyle().Bold().Foreground(th.Primary).Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render("["+label+"]")))}
}

func (m model) pathLine(th theme.Theme) layout.Node {
	w := m.size().W
	p := ansi.NewStyle().Bold().Foreground(th.Primary).Render(m.selected().path())
	return layout.Fixed(layout.Row(1,
		layout.FlexChild{Node: layout.Block(ansi.TruncateStart(ansi.StripANSI(p), max(4, w-18), "…")), Grow: 1},
		m.toolButton("copy-path", "Path", th), m.toolButton("copy-value", "Value", th),
	), layout.Size{W: w, H: 1})
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	if m.typing {
		return ansi.Truncate(m.in.View()+ansi.NewStyle().Faint().Render(m.matchInfo()), w)
	}
	switch {
	case m.msg != "":
		return ansi.Truncate(m.msg, w)
	case m.query != "":
		return ansi.Truncate(ansi.NewStyle().Foreground(th.Warning).Render("/"+m.query)+" "+ansi.NewStyle().Faint().Render(m.matchInfo()), w)
	}
	return ansi.NewStyle().Faint().Render(ansi.Truncate(fmt.Sprintf("%d values · %d rows shown", m.root.count(), len(m.rows())), w))
}

func (m model) matchInfo() string {
	switch {
	case m.query == "":
		return ""
	case len(m.hits) == 0:
		return "  no matches"
	}
	if n := m.matchNumber(); n > 0 {
		return fmt.Sprintf("  %d of %d", n, len(m.hits))
	}
	return fmt.Sprintf("  %d matches", len(m.hits))
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Block(ansi.Truncate("  "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))}
	}
	lines := []layout.FlexChild{
		key("arrows hjkl", "move; Right opens, Left closes"), key("Enter Space", "open or close"),
		key("* E C", "open below, open all, close all"), key("/", "search keys and values"),
		key("n N", "next and previous match"), key("p y", "copy the path, copy the value"),
		key("g G", "first and last row"), key("Esc", "drop the search"), key("q", "quit"), key("F1", "close help"),
	}
	return layout.Fixed(layout.Column(0, lines...), layout.Size{W: m.size().W, H: m.visRows()})
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("jsonx") + "  " + ansi.NewStyle().Bold().Render(m.source)
	var body layout.Node = m.treeNode(th)
	switch {
	case m.help:
		body = m.helpNode(th)
	case m.wide():
		body = layout.Row(0, layout.FlexChild{Node: m.treeNode(th)}, layout.FlexChild{Node: m.previewNode(th)})
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "/", Action: "search"}, widgets.Hint{Key: "n", Action: "next"},
		widgets.Hint{Key: "p", Action: "path"}, widgets.Hint{Key: "y", Action: "value"},
		widgets.Hint{Key: "E/C", Action: "all"}, widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, w))},
		layout.FlexChild{Node: m.pathLine(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

// read returns the document named by args: a file, "-" or nothing for standard
// input, or the sample when nothing is piped.
func read(args []string, stdin io.Reader, piped bool) (data, source string, err error) {
	switch {
	case len(args) > 0 && args[0] != "-":
		b, err := os.ReadFile(args[0]) // #nosec G304 G703 -- a viewer opens the file its user names
		return string(b), args[0], err
	case len(args) > 0 || piped:
		b, err := io.ReadAll(stdin)
		return string(b), "stdin", err
	}
	return sampleJSON, "sample.json", nil
}

func main() {
	piped := false
	if info, err := os.Stdin.Stat(); err == nil && info.Mode()&os.ModeCharDevice == 0 {
		piped = true
	}
	data, source, err := read(os.Args[1:], os.Stdin, piped)
	if err != nil {
		fmt.Fprintln(os.Stderr, "jsonx:", err)
		os.Exit(1)
	}
	root, err := parse(strings.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "jsonx: %s: %v\n", source, err)
		os.Exit(1)
	}
	// Input comes from the terminal even when the document was piped in.
	opts := []tui.ProgramOption{tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)}
	if tty, err := os.Open("/dev/tty"); err == nil && piped {
		defer tty.Close()
		opts = append(opts, tui.WithInput(tty))
	}
	if _, err := tui.NewProgram(newModel(root, source), opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
