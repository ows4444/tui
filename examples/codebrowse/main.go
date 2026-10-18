// Command codebrowse browses a source folder: a tree of the files on the left,
// the selected file on the right with line numbers and syntax colours. Jump to
// a line, search the file, and scroll sideways through long lines. It only
// reads: nothing is written.
//
// Mouse: click a folder to open or close it, click a file to open it, click a
// line to put the cursor there, scroll the wheel in either pane. Keyboard: Tab
// swaps between the tree and the file; in the tree the arrows move, Right or
// Enter opens, Left closes or steps out; in the file the arrows, PgUp, PgDn,
// Home and End move, Left and Right scroll sideways, g and G go to the top and
// bottom, : jumps to a line (a number, or "end"), / searches and n and N go to
// the next and previous match, . shows or hides dot files, F1 shows help, q or
// Ctrl+C quits.
//
//	go run ./examples/codebrowse [folder]
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdown"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	treeW  = 30
	wideAt = 70
	chrome = 4 // title, path bar, status, hints
)

type pane int

const (
	inTree pane = iota
	inFile
)

type promptKind int

const (
	noPrompt promptKind = iota
	promptLine
	promptSearch
)

// row is one line of the tree.
type row struct {
	e     entry
	depth int
}

type model struct {
	th     ui.Theming // the theme the Program chose; see SetTheme
	fsys   fs.FS
	root   string // what the tree is called
	hidden bool

	open    map[string]bool    // folders that are open
	listing map[string][]entry // folders already read
	cut     map[string]bool    // folders listed only in part
	errs    map[string]string  // folders that could not be read

	tCur, tTop int // tree cursor and first row shown

	f        file
	cur, top int // file cursor line and first line shown
	left     int // columns scrolled sideways
	query    string
	hits     []int
	prompt   promptKind
	in       textinput.Model
	pane     pane
	msg      string
	help     bool

	width, height int
}

func newModel(fsys fs.FS, root string) model {
	in := textinput.New()
	in.Width = 30
	m := model{fsys: fsys, root: root, open: map[string]bool{".": true}, listing: map[string][]entry{}, cut: map[string]bool{}, errs: map[string]string{}, in: in}
	return m.ensure(".").firstFile()
}

func (m model) Init() tui.Cmd { return nil }

// ---- the tree --------------------------------------------------------------

// ensure reads folder dir if it has not been read.
func (m model) ensure(dir string) model {
	if _, ok := m.listing[dir]; ok || m.errs[dir] != "" {
		return m
	}
	es, cut, err := readDir(m.fsys, dir, m.hidden)
	l, c, e := copyMap(m.listing), copyBool(m.cut), copyStr(m.errs)
	if err != nil {
		e[dir] = err.Error()
	} else {
		l[dir], c[dir] = es, cut
	}
	m.listing, m.cut, m.errs = l, c, e
	return m
}

func copyMap(in map[string][]entry) map[string][]entry {
	out := make(map[string][]entry, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyBool(in map[string]bool) map[string]bool {
	out := make(map[string]bool, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyStr(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

// rows flattens the open folders into the lines of the tree.
func (m model) rows() []row {
	var out []row
	var walk func(dir string, depth int)
	walk = func(dir string, depth int) {
		for _, e := range m.listing[dir] {
			out = append(out, row{e: e, depth: depth})
			if e.dir && m.open[e.path] {
				if msg := m.errs[e.path]; msg != "" {
					out = append(out, row{e: entry{name: "✗ " + msg, path: e.path + "/!"}, depth: depth + 1})
				}
				walk(e.path, depth+1)
			}
		}
		if m.cut[dir] {
			out = append(out, row{e: entry{name: fmt.Sprintf("… only the first %d shown", maxEntries), path: dir + "/…"}, depth: depth})
		}
	}
	walk(".", 0)
	return out
}

func (m model) treeRows() int { return max(1, m.size().H-chrome) }

func (m model) keepTree() model {
	n := len(m.rows())
	m.tCur = max(0, min(m.tCur, n-1))
	if m.tCur < m.tTop {
		m.tTop = m.tCur
	}
	if m.tCur >= m.tTop+m.treeRows() {
		m.tTop = m.tCur - m.treeRows() + 1
	}
	m.tTop = max(0, min(m.tTop, max(0, n-m.treeRows())))
	return m
}

func (m model) toggleDir(p string) model {
	o := copyBool(m.open)
	o[p] = !o[p]
	m.open = o
	if o[p] {
		m = m.ensure(p)
	}
	return m.keepTree()
}

// openFile loads the file e and shows it.
func (m model) openFile(e entry) model {
	m.f = load(m.fsys, e.path, e.size)
	m.cur, m.top, m.left, m.query, m.hits, m.msg = 0, 0, 0, "", nil, ""
	return m
}

// firstFile shows the first file of the top folder, so there is something to read.
func (m model) firstFile() model {
	for i, r := range m.rows() {
		if !r.e.dir {
			m.tCur = i
			return m.openFile(r.e).keepTree()
		}
	}
	return m
}

// ---- the file --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) wide() bool { return m.size().W >= wideAt }

func (m model) fileRows() int { return max(1, m.size().H-chrome) }

func (m model) gutter() int { return len(strconv.Itoa(max(1, len(m.f.lines)))) + 2 }

func (m model) fileW() int {
	if m.wide() {
		return m.size().W - treeW - 1
	}
	return m.size().W
}

func (m model) textW() int { return max(1, m.fileW()-m.gutter()) }

func (m model) keepFile() model {
	n := len(m.f.lines)
	m.cur = max(0, min(m.cur, n-1))
	if m.cur < m.top {
		m.top = m.cur
	}
	if m.cur >= m.top+m.fileRows() {
		m.top = m.cur - m.fileRows() + 1
	}
	m.top = max(0, min(m.top, max(0, n-m.fileRows())))
	m.left = max(0, min(m.left, m.longest()))
	return m
}

// longest is how far the file can be scrolled sideways.
func (m model) longest() int {
	w := 0
	for _, l := range m.f.lines {
		w = max(w, ansi.Width(l))
	}
	return max(0, w-m.textW()+1)
}

func (m model) gotoLine(n int) model {
	m.cur = n - 1
	m.msg = ""
	return m.keepFile()
}

func (m model) search(q string) model {
	m.query = q
	m.hits = m.f.find(q)
	return m
}

// step moves to the next (dir 1) or previous (dir -1) match, wrapping round.
func (m model) step(dir int) model {
	if len(m.hits) == 0 {
		m.msg = "No match"
		return m
	}
	at := -1
	for i, h := range m.hits {
		if (dir > 0 && h > m.cur) || (dir < 0 && h < m.cur) {
			at = i
			if dir > 0 {
				break
			}
		}
	}
	if at < 0 {
		at = 0
		if dir < 0 {
			at = len(m.hits) - 1
		}
	}
	m.cur = m.hits[at]
	m.msg = fmt.Sprintf("%d of %d matches", at+1, len(m.hits))
	return m.keepFile()
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.keepTree().keepFile(), nil
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
	if m.prompt != noPrompt {
		return m.promptKey(k)
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
	switch k.Type {
	case tui.KeyTab:
		m.pane = 1 - m.pane
		return m, nil
	case tui.KeyRunes:
		switch k.Text {
		case "q":
			return m, tui.Quit()
		case ".":
			return m.toggleHidden(), nil
		}
	}
	if m.pane == inTree {
		return m.treeKey(k), nil
	}
	return m.fileKey(k)
}

func (m model) toggleHidden() model {
	m.hidden = !m.hidden
	m.listing, m.cut, m.errs = map[string][]entry{}, map[string]bool{}, map[string]string{}
	for p, on := range m.open {
		if on {
			m = m.ensure(p)
		}
	}
	m.msg = "Dot files hidden"
	if m.hidden {
		m.msg = "Dot files shown"
	}
	return m.keepTree()
}

func (m model) treeKey(k tui.Key) model {
	rs := m.rows()
	if len(rs) == 0 {
		return m
	}
	cur := rs[min(m.tCur, len(rs)-1)]
	switch k.Type {
	case tui.KeyUp:
		m.tCur--
	case tui.KeyDown:
		m.tCur++
	case tui.KeyPgUp:
		m.tCur -= m.treeRows()
	case tui.KeyPgDown:
		m.tCur += m.treeRows()
	case tui.KeyHome:
		m.tCur = 0
	case tui.KeyEnd:
		m.tCur = len(rs) - 1
	case tui.KeyRight, tui.KeyEnter:
		return m.activate(cur)
	case tui.KeyLeft:
		return m.collapse(cur, rs)
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.tCur--
		case "j":
			m.tCur++
		case "l":
			return m.activate(cur)
		case "h":
			return m.collapse(cur, rs)
		}
	}
	return m.keepTree()
}

// activate opens a folder, or shows a file and moves to it.
func (m model) activate(r row) model {
	switch {
	case r.e.dir && !m.open[r.e.path]:
		return m.toggleDir(r.e.path)
	case r.e.dir:
		m.tCur++
	case strings.HasSuffix(r.e.path, "/…") || strings.HasSuffix(r.e.path, "/!"):
	default:
		m = m.openFile(r.e)
		if !m.wide() {
			m.pane = inFile
		}
	}
	return m.keepTree()
}

// collapse closes an open folder, or moves to the folder above.
func (m model) collapse(r row, rs []row) model {
	if r.e.dir && m.open[r.e.path] {
		return m.toggleDir(r.e.path)
	}
	for i := m.tCur - 1; i >= 0; i-- {
		if rs[i].depth < r.depth {
			m.tCur = i
			break
		}
	}
	return m.keepTree()
}

func (m model) fileKey(k tui.Key) (tui.Model, tui.Cmd) {
	page := max(1, m.fileRows()-1)
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
		m.left = 0
	case tui.KeyEnd:
		m.cur = len(m.f.lines) - 1
	case tui.KeyLeft:
		m.left -= 8
	case tui.KeyRight:
		m.left += 8
	case tui.KeyEsc:
		m.query, m.hits = "", nil
		if !m.wide() {
			m.pane = inTree
		}
	case tui.KeyRunes:
		return m.fileRune(k.Text)
	}
	return m.keepFile(), nil
}

func (m model) fileRune(r string) (tui.Model, tui.Cmd) {
	switch r {
	case "k":
		m.cur--
	case "j":
		m.cur++
	case "h":
		m.left -= 8
	case "l":
		m.left += 8
	case "g":
		m.cur = 0
	case "G":
		m.cur = len(m.f.lines) - 1
	case ":":
		return m.startPrompt(promptLine, "Go to line ▸ ", ""), nil
	case "/":
		return m.startPrompt(promptSearch, "Search ▸ ", m.query), nil
	case "n":
		return m.step(1), nil
	case "N":
		return m.step(-1), nil
	}
	return m.keepFile(), nil
}

func (m model) startPrompt(k promptKind, label, initial string) model {
	m.prompt = k
	m.in.Prompt = label
	m.in.SetValue(initial)
	m.in.CursorEnd()
	m.in.Focus()
	return m
}

func (m model) promptKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.prompt = noPrompt
		m.in.Blur()
		return m, nil
	case tui.KeyEnter:
		return m.submit(), nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

func (m model) submit() model {
	v := strings.TrimSpace(m.in.Value())
	kind := m.prompt
	m.prompt = noPrompt
	m.in.Blur()
	switch kind {
	case promptLine:
		switch n, err := strconv.Atoi(v); {
		case v == "end" || v == "$":
			return m.gotoLine(len(m.f.lines))
		case err != nil || n < 1:
			m.msg = "Type a line number, or end"
		default:
			return m.gotoLine(min(n, len(m.f.lines)))
		}
	case promptSearch:
		m = m.search(v)
		if v != "" {
			m.cur--
			return m.step(1)
		}
	}
	return m
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.prompt != noPrompt || m.help {
		return m, nil
	}
	tr, treeShown := layout.RectOf(m.screen(), m.size(), "tree")
	fr, fileShown := layout.RectOf(m.screen(), m.size(), "file")
	inT, inF := treeShown && inside(tr, ev.X, ev.Y), fileShown && inside(fr, ev.X, ev.Y)
	d := 0
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		d = -3
	case tui.MouseButtonWheelDown:
		d = 3
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	switch {
	case inT && d != 0:
		m.tTop += d
		m.tTop = max(0, min(m.tTop, max(0, len(m.rows())-m.treeRows())))
		return m, nil
	case inF && d != 0:
		m.top += d
		m.top = max(0, min(m.top, max(0, len(m.f.lines)-m.fileRows())))
		m.cur = max(m.top, min(m.cur, m.top+m.fileRows()-1))
		return m, nil
	case inT:
		return m.clickTree(ev, tr), nil
	case inF:
		m.pane = inFile
		if i := m.top + ev.Y - fr.Y; i < len(m.f.lines) {
			m.cur = i
		}
		top := m.top
		m = m.keepFile()
		m.top = top
	}
	return m, nil
}

func (m model) clickTree(ev tui.MouseEvent, r layout.Rect) model {
	rs := m.rows()
	i := m.tTop + ev.Y - r.Y
	if i < 0 || i >= len(rs) {
		return m
	}
	m.pane, m.tCur = inTree, i
	m = m.activate(rs[i])
	m.tCur = i
	return m.keepTree()
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func classColor(c markdown.Class, th theme.Theme) ansi.Color {
	switch c {
	case markdown.Keyword:
		return th.Primary
	case markdown.Type, markdown.Key, markdown.Variable:
		return th.Info
	case markdown.String:
		return th.Success
	case markdown.Number:
		return th.Warning
	case markdown.Comment:
		return th.Muted
	case markdown.Literal:
		return th.Secondary
	}
	return th.Text
}

// codeLine is line i coloured and scrolled sideways, exactly w cells wide.
func (m model) codeLine(i, w int, th theme.Theme) string {
	var bg func(ansi.Style) ansi.Style = func(s ansi.Style) ansi.Style { return s }
	if i == m.cur {
		bg = func(s ansi.Style) ansi.Style { return s.Background(th.Selection) }
	}
	var b strings.Builder
	if m.f.spans != nil {
		for _, sp := range m.f.spans[i] {
			b.WriteString(bg(ansi.NewStyle().Foreground(classColor(sp.Class, th))).Render(sp.Text))
		}
	} else {
		b.WriteString(bg(ansi.NewStyle()).Render(m.f.lines[i]))
	}
	s := ansi.Truncate(ansi.TrimLeftWidth(b.String(), m.left), w)
	if i == m.cur {
		s += bg(ansi.NewStyle()).Render(strings.Repeat(" ", max(0, w-ansi.Width(s))))
	}
	return pad(s, w)
}

func (m model) fileNode(th theme.Theme) layout.Node {
	w, h := m.fileW(), m.fileRows()
	var rows []layout.FlexChild
	hit := map[int]bool{}
	for _, l := range m.hits {
		hit[l] = true
	}
	switch {
	case m.f.path == "":
		rows = append(rows, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(" No file selected. Pick one in the tree."))})
	case m.f.problem != "":
		rows = append(rows, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(" " + m.f.problem))})
	default:
		g := m.gutter()
		for i := m.top; i < len(m.f.lines) && len(rows) < h; i++ {
			mark := " "
			switch {
			case i == m.cur:
				mark = "›"
			case hit[i]:
				mark = "•"
			}
			num := ansi.NewStyle().Faint().Render(fmt.Sprintf("%*d", g-2, i+1))
			rows = append(rows, layout.FlexChild{Node: layout.Block(mark + num + " " + m.codeLine(i, w-g, th))})
		}
	}
	for len(rows) < h {
		rows = append(rows, layout.FlexChild{Node: layout.Block("")})
	}
	return layout.Named("file", layout.Fixed(layout.Column(0, rows...), layout.Size{W: w, H: h}))
}

func (m model) treeNode(th theme.Theme) layout.Node {
	h := m.treeRows()
	w := treeW
	if !m.wide() {
		w = m.size().W
	}
	rs := m.rows()
	var lines []layout.FlexChild
	for i := m.tTop; i < len(rs) && len(lines) < h; i++ {
		r := rs[i]
		icon := "  "
		switch {
		case r.e.dir && m.open[r.e.path]:
			icon = "▾ "
		case r.e.dir:
			icon = "▸ "
		}
		line := strings.Repeat("  ", r.depth) + icon + r.e.name
		if r.e.dir {
			line = ansi.NewStyle().Foreground(th.Info).Render(ansi.Truncate(line, w))
		}
		line = ansi.Truncate(line, w)
		if m.f.path == r.e.path && !r.e.dir {
			line = ansi.NewStyle().Bold().Render(line)
		}
		if i == m.tCur && m.pane == inTree {
			line = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary).Render(pad(ansi.StripANSI(line), w))
		}
		lines = append(lines, layout.FlexChild{Node: layout.Block(pad(line, w))})
	}
	for len(lines) < h {
		lines = append(lines, layout.FlexChild{Node: layout.Block("")})
	}
	return layout.Named("tree", layout.Fixed(layout.Column(0, lines...), layout.Size{W: w, H: h}))
}

func (m model) pathBar(th theme.Theme) string {
	w := m.size().W
	if m.f.path == "" {
		return ansi.NewStyle().Faint().Render(" " + m.root)
	}
	info := m.f.lang
	if info == "" {
		info = "text"
	}
	if m.f.problem == "" {
		info += fmt.Sprintf(" · Ln %d/%d · %s", m.cur+1, len(m.f.lines), humanSize(m.f.size))
	}
	left := ansi.NewStyle().Bold().Render(path.Join(m.root, m.f.path))
	gap := max(1, w-ansi.Width(left)-ansi.Width(info)-1)
	return ansi.Truncate(left+strings.Repeat(" ", gap)+ansi.NewStyle().Faint().Render(info), w)
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	if m.prompt != noPrompt {
		return ansi.Truncate(m.in.View(), w)
	}
	switch {
	case m.msg != "":
		return ansi.Truncate(m.msg, w)
	case m.f.truncated:
		return ansi.NewStyle().Foreground(th.Warning).Render(ansi.Truncate(fmt.Sprintf("⚠ showing the first %s of %s", humanSize(maxBytes), humanSize(m.f.size)), w))
	case m.query != "":
		return ansi.NewStyle().Faint().Render(ansi.Truncate(fmt.Sprintf("/%s  %d matching lines", m.query, len(m.hits)), w))
	}
	return ""
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 12))+a, m.size().W))}
	}
	return layout.Fixed(layout.Column(0,
		key("Tab", "tree or file"), key("Right Enter", "open a folder or file"), key("Left", "close or step out"),
		key(":", "go to a line (a number or end)"), key("/", "search the file"), key("n N", "next and previous match"),
		key("g G", "top and bottom"), key("Left Right", "scroll sideways in the file"), key(".", "show or hide dot files"),
		key("F1", "close help"), key("q", "quit"),
	), layout.Size{W: m.size().W, H: m.fileRows()})
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	var body layout.Node
	switch {
	case m.help:
		body = m.helpNode(th)
	case m.wide():
		rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
		var rules []layout.FlexChild
		for i := 0; i < m.treeRows(); i++ {
			rules = append(rules, layout.FlexChild{Node: layout.Block(rule)})
		}
		body = layout.Row(0, layout.FlexChild{Node: m.treeNode(th)}, layout.FlexChild{Node: layout.Column(0, rules...)}, layout.FlexChild{Node: m.fileNode(th)})
	case m.pane == inTree:
		body = m.treeNode(th)
	default:
		body = m.fileNode(th)
	}
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("codebrowse") + "  " + ansi.NewStyle().Faint().Render(m.root)
	hints := widgets.Footer(w, widgets.Hint{Key: "tab", Action: "pane"}, widgets.Hint{Key: ":", Action: "line"},
		widgets.Hint{Key: "/", Action: "search"}, widgets.Hint{Key: "n", Action: "next"}, widgets.Hint{Key: ".", Action: "dot files"},
		widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, w))},
		layout.FlexChild{Node: layout.Block(m.pathBar(th))},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	dir := "."
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "codebrowse: %s is not a folder\n", dir)
		os.Exit(1)
	}
	if _, err := ui.NewProgram(newModel(os.DirFS(dir), dir), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
