// Command docsreader reads a folder of Markdown files as a small documentation
// site in the terminal: a sidebar of the pages with the headings of the open
// one, links you can follow from page to page, back and forward through where
// you have been, and search across every page. With no folder it opens a
// made-up documentation set that links to itself.
//
// Mouse: click a page or a heading in the sidebar, click ‹ or › for back and
// forward, scroll the wheel, click a search result. Keyboard: Tab and Shift+Tab
// pick a link in the page and Enter follows it; [ and ] jump between headings;
// j, k, space, b, g and G scroll; Backspace or Alt+Left goes back, Alt+Right
// forward; s moves into the sidebar (Up, Down, Enter, then Esc to leave);
// / searches every page; ? or F1 shows help; q or Ctrl+C quits. A link to a web
// address is copied to the clipboard, not opened.
//
//	go run ./examples/docsreader [docs-folder]
package main

import (
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdownview"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	sideW  = 30
	wideAt = 70
	chrome = 3 // title, status, hints
)

// loc is a place in the docs: a page and the heading in view.
type loc struct {
	page   int
	anchor string
}

type model struct {
	th     ui.Theming // the theme the Program chose; see SetTheme
	pages  []page
	cur    int
	reader markdownview.Model
	back   []loc
	fwd    []loc

	side    bool // the sidebar has the keys
	sideCur int
	sideTop int

	searching bool
	in        textinput.Model
	hits      []hit
	hitCur    int

	msg   string
	help  bool
	hover string

	width, height int
}

func newModel(fsys fs.FS) model {
	in := textinput.New()
	in.Prompt = "Search ▸ "
	in.Width = 30
	m := model{pages: loadPages(fsys), in: in}
	if len(m.pages) == 0 {
		m.reader = markdownview.New("# No pages\n\nThis folder has no Markdown files.", 40, 10)
		return m
	}
	m.reader = markdownview.New(m.pages[0].src, 40, 10)
	m.reader.Mouse = true
	return m.fit()
}

func (m model) Init() tui.Cmd { return nil }

// ---- geometry --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 28}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) wide() bool { return m.size().W >= wideAt }

func (m model) bodyH() int { return max(1, m.size().H-chrome) }

func (m model) readerW() int {
	if m.wide() {
		return m.size().W - sideW - 1
	}
	return m.size().W
}

func (m model) fit() model {
	m.reader.SetSize(max(8, m.readerW()), m.bodyH())
	m.reader.Bounds = layout.Rect{X: m.size().W - m.readerW(), Y: 1, W: m.readerW(), H: m.bodyH()}
	return m
}

// ---- the sidebar -----------------------------------------------------------

type sideRow struct {
	page   int    // the page this row is, or -1
	anchor string // the heading it is, or ""
	text   string
	level  int
}

func (m model) sideRows() []sideRow {
	var rows []sideRow
	for i, p := range m.pages {
		rows = append(rows, sideRow{page: i, text: p.title})
		if i != m.cur {
			continue
		}
		for _, h := range m.reader.Headings() {
			if h.Level == 1 {
				continue // the page's own title is the row above
			}
			rows = append(rows, sideRow{page: -1, anchor: h.Anchor, text: h.Text, level: h.Level - 1})
		}
	}
	return rows
}

func (m model) keepSide() model {
	n := len(m.sideRows())
	m.sideCur = max(0, min(m.sideCur, n-1))
	if m.sideCur < m.sideTop {
		m.sideTop = m.sideCur
	}
	if m.sideCur >= m.sideTop+m.bodyH() {
		m.sideTop = m.sideCur - m.bodyH() + 1
	}
	m.sideTop = max(0, min(m.sideTop, max(0, n-m.bodyH())))
	return m
}

// ---- moving between pages --------------------------------------------------

// currentAnchor is the heading the top of the reader is under.
func (m model) currentAnchor() string {
	a := ""
	for _, h := range m.reader.Headings() {
		if h.Line <= m.reader.YOffset() {
			a = h.Anchor
		}
	}
	return a
}

func (m model) here() loc { return loc{m.cur, m.currentAnchor()} }

// show makes page i the one on screen, scrolled to anchor if there is one.
func (m model) show(i int, anchor string) model {
	if i != m.cur || m.reader.YOffset() != 0 || anchor != "" {
		m.reader.SetSource(m.pages[i].src)
	}
	m.cur = i
	m = m.fit()
	if anchor != "" {
		m.reader.GoTo(anchor)
	}
	for r, row := range m.sideRows() {
		if row.page == i {
			m.sideCur = r
		}
	}
	m.searching, m.msg = false, ""
	return m.keepSide()
}

// go_ follows a link or a click to page i, remembering where it came from.
func (m model) goTo(i int, anchor string) model {
	if i == m.cur && anchor == "" && m.reader.YOffset() == 0 {
		return m
	}
	m.back = append(append([]loc(nil), m.back...), m.here())
	m.fwd = nil
	return m.show(i, anchor)
}

func (m model) goBack() model {
	if len(m.back) == 0 {
		m.msg = "Nothing to go back to"
		return m
	}
	to := m.back[len(m.back)-1]
	m.fwd = append(append([]loc(nil), m.fwd...), m.here())
	m.back = m.back[:len(m.back)-1]
	return m.show(to.page, to.anchor)
}

func (m model) goForward() model {
	if len(m.fwd) == 0 {
		m.msg = "Nothing to go forward to"
		return m
	}
	to := m.fwd[len(m.fwd)-1]
	m.back = append(append([]loc(nil), m.back...), m.here())
	m.fwd = m.fwd[:len(m.fwd)-1]
	return m.show(to.page, to.anchor)
}

func (m model) pageIndex(p string) int {
	for i, pg := range m.pages {
		if pg.path == p {
			return i
		}
	}
	return -1
}

// follow acts on a link the reader did not handle itself.
func (m model) follow(url string) (model, tui.Cmd) {
	t := resolve(m.pages[m.cur].path, url)
	if t.external {
		m.msg = "Link copied: " + url
		return m, tui.WriteClipboard(url)
	}
	i := m.pageIndex(t.page)
	if i < 0 {
		m.msg = "No such page: " + t.page
		return m, nil
	}
	anchor := t.anchor
	if anchor != "" {
		m = m.goTo(i, "")
		if !m.reader.GoTo(anchor) {
			m.msg = "No heading #" + anchor + " on this page"
		}
		return m, nil
	}
	return m.goTo(i, ""), nil
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.fit().keepSide(), nil
	case markdownview.LinkMsg:
		return m.follow(msg.URL)
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
	if len(m.pages) == 0 {
		if k.Type == tui.KeyRunes && k.Text == "q" {
			return m, tui.Quit()
		}
		return m, nil
	}
	switch {
	case m.searching:
		return m.searchKey(k)
	case k.Type == tui.KeyF1:
		m.help = !m.help
		return m, nil
	case m.help:
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	case k.Type == tui.KeyBackspace, k.Type == tui.KeyLeft && k.Mod.Alt():
		return m.goBack(), nil
	case k.Type == tui.KeyRight && k.Mod.Alt():
		return m.goForward(), nil
	case m.side:
		return m.sideKey(k), nil
	}
	m.msg = ""
	return m.readerKey(k)
}

// scrollKeys lets the reader be scrolled with the keys a pager uses.
var scrollKeys = map[string]tui.KeyType{"j": tui.KeyDown, "k": tui.KeyUp, " ": tui.KeyPgDown, "b": tui.KeyPgUp, "g": tui.KeyHome, "G": tui.KeyEnd}

func (m model) readerKey(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyRunes {
		switch k.Text {
		case "q":
			return m, tui.Quit()
		case "?":
			m.help = true
			return m, nil
		case "/":
			return m.startSearch()
		case "s":
			if m.wide() {
				m.side = true
			}
			return m, nil
		}
		if t, ok := scrollKeys[k.Text]; ok {
			k = tui.Key{Type: t}
		}
	}
	var cmd tui.Cmd
	m.reader, cmd = m.reader.Update(k)
	return m, cmd
}

func (m model) sideKey(k tui.Key) model {
	rows := m.sideRows()
	switch k.Type {
	case tui.KeyUp:
		m.sideCur--
	case tui.KeyDown:
		m.sideCur++
	case tui.KeyHome:
		m.sideCur = 0
	case tui.KeyEnd:
		m.sideCur = len(rows) - 1
	case tui.KeyEsc, tui.KeyTab:
		m.side = false
	case tui.KeyEnter, tui.KeyRight:
		return m.openRow(rows[max(0, min(m.sideCur, len(rows)-1))])
	case tui.KeyRunes:
		switch k.Text {
		case "j":
			m.sideCur++
		case "k":
			m.sideCur--
		case "s", "q":
			m.side = false
		}
	}
	return m.keepSide()
}

func (m model) openRow(r sideRow) model {
	if r.page >= 0 {
		return m.goTo(r.page, "")
	}
	m.back = append(append([]loc(nil), m.back...), m.here())
	m.fwd = nil
	m.reader.GoTo(r.anchor)
	return m
}

// ---- search ----------------------------------------------------------------

func (m model) startSearch() (tui.Model, tui.Cmd) {
	m.searching = true
	m.in.SetValue("")
	m.hits, m.hitCur = nil, 0
	return m, m.in.Focus()
}

func (m model) searchKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.searching = false
		m.in.Blur()
		return m, nil
	case tui.KeyEnter:
		return m.openHit(m.hitCur), nil
	case tui.KeyUp:
		m.hitCur = max(0, m.hitCur-1)
		return m, nil
	case tui.KeyDown:
		m.hitCur = min(len(m.hits)-1, m.hitCur+1)
		return m, nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	m.hits, m.hitCur = search(m.pages, m.in.Value()), 0
	return m, cmd
}

func (m model) openHit(i int) model {
	if i < 0 || i >= len(m.hits) {
		return m
	}
	h := m.hits[i]
	m.in.Blur()
	m = m.goTo(h.page, "")
	m.searching = false
	for _, hd := range m.reader.Headings() {
		if hd.Text == plain(h.heading) {
			m.reader.GoTo(hd.Anchor)
		}
	}
	return m
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if len(m.pages) == 0 || m.help {
		return m, nil
	}
	if ev.Action == tui.MouseActionMotion {
		m.hover = ""
		for _, id := range []string{"back", "fwd"} {
			if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, ev.X, ev.Y) {
				m.hover = id
			}
		}
		return m, nil
	}
	if m.searching {
		return m.searchMouse(ev), nil
	}
	if ev.Button == tui.MouseButtonLeft && ev.Action == tui.MouseActionPress {
		for id, f := range map[string]func(model) model{"back": model.goBack, "fwd": model.goForward} {
			if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, ev.X, ev.Y) {
				return f(m), nil
			}
		}
		if r, ok := layout.RectOf(m.screen(), m.size(), "side"); ok && inside(r, ev.X, ev.Y) {
			rows := m.sideRows()
			if i := m.sideTop + ev.Y - r.Y; i >= 0 && i < len(rows) {
				m.side, m.sideCur = true, i
				return m.openRow(rows[i]), nil
			}
			return m, nil
		}
	}
	var cmd tui.Cmd
	m.reader, cmd = m.reader.Update(ev)
	return m, cmd
}

func (m model) searchMouse(ev tui.MouseEvent) model {
	if ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress {
		return m
	}
	if r, ok := layout.RectOf(m.screen(), m.size(), "hits"); ok && inside(r, ev.X, ev.Y) {
		top := max(0, min(m.hitCur-m.bodyH()+2, max(0, len(m.hits)-(m.bodyH()-1))))
		return m.openHit(top + ev.Y - r.Y)
	}
	return m
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) titleNode(th theme.Theme) layout.Node {
	w := m.size().W
	btn := func(id, label string, on bool) layout.FlexChild {
		st := ansi.NewStyle().Foreground(th.Primary)
		switch {
		case !on:
			st = ansi.NewStyle().Faint()
		case m.hover == id:
			st = st.Bold().Underline()
		}
		return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))}
	}
	name := "docsreader"
	if len(m.pages) > 0 {
		name += "  " + m.pages[m.cur].title
	}
	return layout.Fixed(layout.Row(1,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(ansi.NewStyle().Bold().Foreground(th.Primary).Render(name), max(1, w-8))), Grow: 1},
		btn("back", "[‹]", len(m.back) > 0), btn("fwd", "[›]", len(m.fwd) > 0),
	), layout.Size{W: w, H: 1})
}

func (m model) sideNode(th theme.Theme) layout.Node {
	h := m.bodyH()
	rows := m.sideRows()
	rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
	var kids []layout.FlexChild
	for i := m.sideTop; i < len(rows) && len(kids) < h; i++ {
		r := rows[i]
		label := strings.Repeat("  ", r.level) + ansi.Truncate(r.text, sideW-2-2*r.level)
		st := ansi.NewStyle()
		switch {
		case r.page == m.cur:
			st = ansi.NewStyle().Bold().Foreground(th.Primary)
		case r.page < 0:
			st = ansi.NewStyle().Faint()
		}
		if i == m.sideCur && m.side {
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		}
		kids = append(kids, block(st.Render(pad(" "+label, sideW-1))+rule))
	}
	for len(kids) < h {
		kids = append(kids, block(strings.Repeat(" ", sideW-1)+rule))
	}
	return layout.Named("side", layout.Fixed(layout.Column(0, kids...), layout.Size{W: sideW, H: h}))
}

func (m model) hitsNode(th theme.Theme) layout.Node {
	w, h := m.size().W, m.bodyH()
	var kids []layout.FlexChild
	kids = append(kids, block(ansi.Truncate(m.in.View(), w)))
	var list []layout.FlexChild
	top := max(0, min(m.hitCur-h+2, max(0, len(m.hits)-(h-1))))
	for i := top; i < len(m.hits) && len(list) < h-1; i++ {
		ht := m.hits[i]
		where := m.pages[ht.page].title
		if ht.heading != "" {
			where += " › " + ht.heading
		}
		line := ansi.NewStyle().Foreground(th.Primary).Render(ansi.Truncate(where, w/2)) + "  " + ansi.NewStyle().Faint().Render(ht.text)
		line = ansi.Truncate(line, w)
		if i == m.hitCur {
			line = ansi.NewStyle().Bold().Background(th.Selection).Render(pad(ansi.StripANSI(line), w))
		}
		list = append(list, block(line))
	}
	switch {
	case m.in.Value() == "":
		list = append(list, block(ansi.NewStyle().Faint().Render(" Type to search every page.")))
	case len(m.hits) == 0:
		list = append(list, block(ansi.NewStyle().Faint().Render(" No page contains that.")))
	}
	for len(list) < h-1 {
		list = append(list, block(""))
	}
	kids = append(kids, layout.FlexChild{Node: layout.Named("hits", layout.Column(0, list...))})
	return layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: h})
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))
	}
	return layout.Fixed(layout.Column(0,
		key("Tab", "pick the next link; Enter follows it"), key("[ ]", "previous and next heading"),
		key("j k space b", "scroll"), key("g G", "top and bottom"), key("Backspace", "go back (Alt+Left too)"),
		key("Alt+Right", "go forward"), key("s", "the sidebar (Esc to leave it)"), key("/", "search every page"),
		key("F1 ?", "this help"), key("q", "quit"),
	), layout.Size{W: m.size().W, H: m.bodyH()})
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	if len(m.pages) == 0 {
		return layout.Column(0, layout.FlexChild{Node: m.titleNode(th)}, layout.FlexChild{Node: m.reader.LayoutNode()})
	}
	var body layout.Node = m.reader.LayoutNode()
	switch {
	case m.help:
		body = m.helpNode(th)
	case m.searching:
		body = m.hitsNode(th)
	case m.wide():
		body = layout.Row(0, layout.FlexChild{Node: m.sideNode(th)}, layout.FlexChild{Node: body})
	}
	status := m.msg
	if status == "" {
		status = strconv.Itoa(len(m.pages)) + " pages · " + m.pages[m.cur].path
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "tab", Action: "link"}, widgets.Hint{Key: "/", Action: "search"},
		widgets.Hint{Key: "s", Action: "sidebar"}, widgets.Hint{Key: "bksp", Action: "back"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	return layout.Column(0,
		layout.FlexChild{Node: m.titleNode(th)},
		layout.FlexChild{Node: layout.Fixed(body, layout.Size{W: w, H: m.bodyH()})},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(status, w)))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	var fsys fs.FS = sampleDocs()
	if len(os.Args) > 1 {
		info, err := os.Stat(os.Args[1]) // #nosec G703 -- the folder the user asked to read
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "docsreader: %s is not a folder\n", os.Args[1])
			os.Exit(1)
		}
		fsys = os.DirFS(os.Args[1])
	}
	if _, err := ui.NewProgram(newModel(fsys), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
