// Command pkgs is a package manager's browser: a list you filter as you type,
// a detail pane for the one selected, a pager, and an install queue that
// installs, updates and removes with a progress bar. The registry is made up
// and nothing is downloaded or written: installing only changes what the
// screen says is installed.
//
// Mouse: click a row, click a tab to show all, installed or updatable
// packages, click a page number, click the Install, Update or Remove button in
// the detail pane, click Cancel on the queue line, scroll the wheel to move.
// Keyboard: arrows or jk move, PgUp and PgDn change page, / filters (Enter
// keeps the filter, Esc drops it), Tab changes tab, Enter or i installs or
// updates (with what it needs), x removes, U updates everything that can be,
// c cancels the queue, F1 shows help, q or Ctrl+C quits.
//
//	go run ./examples/pkgs
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/fuzzy"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/pagination"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	chrome    = 5 // title, filter, queue, pager, hints
	wideFrom  = 70
	tickEvery = 60 * time.Millisecond
)

type tab int

const (
	tabAll tab = iota
	tabInstalled
	tabUpdates
)

var tabNames = [...]string{"All", "Installed", "Updates"}

type jobKind int

const (
	jobInstall jobKind = iota
	jobUpdate
	jobRemove
)

type job struct {
	idx  int
	kind jobKind
}

type tickMsg struct{ seq int }

type model struct {
	pkgs []pkg
	tab  tab
	cur  int // index into visible()

	query  string
	typing bool
	in     textinput.Model

	queue   []job // queue[0] is running
	ticks   int   // ticks the running job has had
	seq     int   // tells a stale timer from the current one
	running bool

	msg     string
	help    bool
	hover   string
	reduced bool

	width, height int
}

func newModel(reduced bool) model {
	in := textinput.New()
	in.Prompt = "/ "
	in.Width = 30
	return model{pkgs: registry(), in: in, reduced: reduced}
}

func (m model) Init() tui.Cmd { return nil }

// ---- the list --------------------------------------------------------------

type entry struct {
	idx int
	pos []int // matched rune offsets in the name
}

func (m model) inTab(p pkg) bool {
	switch m.tab {
	case tabInstalled:
		return p.installed != ""
	case tabUpdates:
		return p.status() == updatable
	}
	return true
}

// visible are the packages in the tab that match the filter, best match first.
func (m model) visible() []entry {
	var base []int
	for i, p := range m.pkgs {
		if m.inTab(p) {
			base = append(base, i)
		}
	}
	if m.query == "" {
		out := make([]entry, len(base))
		for i, b := range base {
			out[i] = entry{idx: b}
		}
		return out
	}
	res := fuzzy.Find(m.query, len(base), func(i int) string { return m.pkgs[base[i]].name + " " + m.pkgs[base[i]].desc })
	out := make([]entry, 0, len(res))
	for _, r := range res {
		e := entry{idx: base[r.Index]}
		n := len([]rune(m.pkgs[e.idx].name))
		for _, p := range r.Positions {
			if p < n {
				e.pos = append(e.pos, p)
			}
		}
		out = append(out, e)
	}
	return out
}

func (m model) counts() [3]int {
	var c [3]int
	for _, p := range m.pkgs {
		c[0]++
		if p.installed != "" {
			c[1]++
		}
		if p.status() == updatable {
			c[2]++
		}
	}
	return c
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) wide() bool { return m.size().W >= wideFrom }

func (m model) perPage() int { return max(1, m.size().H-chrome) }

func (m model) listW() int {
	if m.wide() {
		return m.size().W * 45 / 100
	}
	return m.size().W
}

func (m model) pages() int {
	return max(1, (len(m.visible())+m.perPage()-1)/m.perPage())
}

func (m model) page() int { return m.cur/m.perPage() + 1 }

func (m model) clamp() model {
	m.cur = max(0, min(m.cur, len(m.visible())-1))
	return m
}

func (m model) selected() (int, bool) {
	v := m.visible()
	if len(v) == 0 {
		return 0, false
	}
	return v[min(m.cur, len(v)-1)].idx, true
}

// ---- the queue -------------------------------------------------------------

func (m model) queued(i int) bool {
	for _, j := range m.queue {
		if j.idx == i {
			return true
		}
	}
	return false
}

func jobTicks(p pkg, k jobKind) int {
	if k == jobRemove {
		return 5
	}
	return max(8, p.sizeKB/40)
}

// enqueue adds jobs and starts the runner if it is idle.
func (m model) enqueue(js ...job) (model, tui.Cmd) {
	m.queue = append(append([]job(nil), m.queue...), js...)
	if m.reduced {
		for len(m.queue) > 0 {
			m = m.finish()
		}
		return m, nil
	}
	if m.running {
		return m, nil
	}
	m.running = true
	return m, m.tick()
}

func (m model) tick() tui.Cmd {
	seq := m.seq
	return tui.FromCtx(motion.After(tickEvery, func(time.Time) tui.Msg { return tickMsg{seq} }))
}

func (m model) onTick(t tickMsg) (tui.Model, tui.Cmd) {
	if t.seq != m.seq || len(m.queue) == 0 {
		return m, nil
	}
	m.ticks++
	j := m.queue[0]
	if m.ticks >= jobTicks(m.pkgs[j.idx], j.kind) {
		m = m.finish()
	}
	if len(m.queue) == 0 {
		m.running = false
		return m, nil
	}
	return m, m.tick()
}

// finish applies the running job.
func (m model) finish() model {
	j := m.queue[0]
	ps := append([]pkg(nil), m.pkgs...)
	switch j.kind {
	case jobRemove:
		ps[j.idx].installed = ""
		m.msg = "Removed " + ps[j.idx].name
	case jobUpdate:
		ps[j.idx].installed = ps[j.idx].latest
		m.msg = "Updated " + ps[j.idx].name + " to " + ps[j.idx].latest
	default:
		ps[j.idx].installed = ps[j.idx].latest
		m.msg = "Installed " + ps[j.idx].name + " " + ps[j.idx].latest
	}
	m.pkgs, m.queue, m.ticks = ps, m.queue[1:], 0
	return m.clamp()
}

func (m model) cancel() model {
	if len(m.queue) == 0 {
		m.msg = "The queue is empty"
		return m
	}
	m.queue, m.ticks, m.running = nil, 0, false
	m.seq++
	m.msg = "Queue cancelled"
	return m
}

func (m model) install() (tui.Model, tui.Cmd) {
	i, ok := m.selected()
	if !ok {
		return m, nil
	}
	p := m.pkgs[i]
	switch {
	case m.queued(i):
		m.msg = p.name + " is already in the queue"
		return m, nil
	case p.status() == installedOK:
		m.msg = p.name + " is up to date"
		return m, nil
	case p.status() == updatable:
		m.msg = "Updating " + p.name
		return m.enqueue(job{i, jobUpdate})
	}
	skip := map[int]bool{}
	for _, j := range m.queue {
		skip[j.idx] = true
	}
	order := installOrder(m.pkgs, i, skip)
	var js []job
	for _, k := range order {
		js = append(js, job{k, jobInstall})
	}
	m.msg = "Installing " + p.name
	if len(order) > 1 {
		m.msg += " with " + joinNames(m.pkgs, order[:len(order)-1])
	}
	return m.enqueue(js...)
}

func (m model) remove() (tui.Model, tui.Cmd) {
	i, ok := m.selected()
	if !ok {
		return m, nil
	}
	p := m.pkgs[i]
	switch {
	case p.installed == "":
		m.msg = p.name + " is not installed"
	case m.queued(i):
		m.msg = p.name + " is in the queue"
	case len(dependents(m.pkgs, i)) > 0:
		m.msg = "Cannot remove " + p.name + ": " + strings.Join(dependents(m.pkgs, i), ", ") + " needs it"
	default:
		m.msg = "Removing " + p.name
		return m.enqueue(job{i, jobRemove})
	}
	return m, nil
}

func (m model) updateAll() (tui.Model, tui.Cmd) {
	var js []job
	for i, p := range m.pkgs {
		if p.status() == updatable && !m.queued(i) {
			js = append(js, job{i, jobUpdate})
		}
	}
	if len(js) == 0 {
		m.msg = "Everything is up to date"
		return m, nil
	}
	m.msg = fmt.Sprintf("Updating %d packages", len(js))
	return m.enqueue(js...)
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.clamp(), nil
	case tickMsg:
		return m.onTick(msg)
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
	if next, ok := m.moveKey(k); ok {
		return next.clamp(), nil
	}
	return m.actionKey(k)
}

func (m model) moveKey(k tui.Key) (model, bool) {
	switch k.Type {
	case tui.KeyUp:
		m.cur--
	case tui.KeyDown:
		m.cur++
	case tui.KeyPgUp:
		m.cur -= m.perPage()
	case tui.KeyPgDown:
		m.cur += m.perPage()
	case tui.KeyHome:
		m.cur = 0
	case tui.KeyEnd:
		m.cur = len(m.visible()) - 1
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.cur--
		case "j":
			m.cur++
		case "g":
			m.cur = 0
		case "G":
			m.cur = len(m.visible()) - 1
		default:
			return m, false
		}
	default:
		return m, false
	}
	m.msg = ""
	return m, true
}

func (m model) actionKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyTab:
		return m.setTab((m.tab + 1) % 3), nil
	case tui.KeyEnter:
		return m.install()
	case tui.KeyEsc:
		m.query = ""
		return m.clamp(), nil
	case tui.KeyRunes:
		switch k.Text {
		case "/":
			m.typing = true
			m.in.SetValue(m.query)
			m.in.CursorEnd()
			return m, m.in.Focus()
		case "i":
			return m.install()
		case "x":
			return m.remove()
		case "U":
			return m.updateAll()
		case "c":
			return m.cancel(), nil
		case "q":
			return m, tui.Quit()
		}
	}
	return m, nil
}

func (m model) setTab(t tab) model {
	m.tab, m.cur, m.msg = t, 0, ""
	return m
}

func (m model) typeKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.typing, m.query = false, ""
		m.in.Blur()
		return m.clamp(), nil
	case tui.KeyEnter:
		m.typing = false
		m.in.Blur()
		return m, nil
	case tui.KeyUp, tui.KeyDown:
		m.cur += map[tui.KeyType]int{tui.KeyUp: -1, tui.KeyDown: 1}[k.Type]
		return m.clamp(), nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	if m.in.Value() != m.query {
		m.query, m.cur = m.in.Value(), 0
	}
	return m, cmd
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"tab-0", "tab-1", "tab-2", "act-main", "act-remove", "cancel", "pager"}
	for i := 0; i < m.perPage(); i++ {
		ids = append(ids, "row-"+strconv.Itoa(i))
	}
	return ids
}

func (m model) hit(x, y int) string {
	for _, id := range m.ids() {
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
	id := m.hit(ev.X, ev.Y)
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		m.cur--
		return m.clamp(), nil
	case tui.MouseButtonWheelDown:
		m.cur++
		return m.clamp(), nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	return m.click(id, ev)
}

func (m model) click(id string, ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	switch {
	case strings.HasPrefix(id, "tab-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "tab-"))
		return m.setTab(tab(n)), nil
	case strings.HasPrefix(id, "row-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "row-"))
		m.cur = (m.page()-1)*m.perPage() + n
		m.msg = ""
		m = m.clamp()
		if ev.Clicks >= 2 {
			return m.install()
		}
		return m, nil
	case id == "act-main":
		return m.install()
	case id == "act-remove":
		return m.remove()
	case id == "cancel":
		return m.cancel(), nil
	case id == "pager":
		return m.clickPager(ev), nil
	}
	return m, nil
}

func (m model) clickPager(ev tui.MouseEvent) model {
	r, ok := layout.RectOf(m.screen(), m.size(), "pager")
	if !ok {
		return m
	}
	pg := m.pagerModel()
	pg.Bounds = r
	next, _ := pg.Update(ev)
	if next.Page() != pg.Page() {
		m.cur = (next.Page() - 1) * m.perPage()
	}
	return m.clamp()
}

func (m model) pagerModel() pagination.Model {
	pg := pagination.New(m.pages())
	pg.Mouse = true
	pg.SetPage(m.page())
	return pg
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) titleLine(th theme.Theme) layout.Node {
	c := m.counts()
	kids := []layout.FlexChild{block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("pkgs"))}
	for i, name := range tabNames {
		id := "tab-" + strconv.Itoa(i)
		label := fmt.Sprintf(" %s %d ", name, c[i])
		st := ansi.NewStyle().Foreground(th.Muted)
		switch {
		case tab(i) == m.tab:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) filterLine(th theme.Theme) string {
	w := m.size().W
	switch {
	case m.typing:
		return ansi.Truncate(m.in.View()+ansi.NewStyle().Faint().Render(fmt.Sprintf("  %d match", len(m.visible()))), w)
	case m.query != "":
		return ansi.Truncate(ansi.NewStyle().Foreground(th.Warning).Render("/"+m.query)+ansi.NewStyle().Faint().Render(fmt.Sprintf("  %d of %d shown", len(m.visible()), len(m.pkgs))), w)
	}
	return ansi.NewStyle().Faint().Render("Press / to filter")
}

func (m model) mark(s string, pos []int, st ansi.Style) string {
	return fuzzy.Highlight(s, pos, func(x string) string { return st.Bold().Underline().Render(x) })
}

func (m model) rowLine(i int, e entry, selected bool, w int, th theme.Theme) string {
	p := m.pkgs[e.idx]
	sym, symStyle := " ", ansi.NewStyle()
	switch {
	case m.queued(e.idx):
		sym, symStyle = "…", ansi.NewStyle().Foreground(th.Warning)
	case p.status() == installedOK:
		sym, symStyle = "✓", ansi.NewStyle().Foreground(th.Success)
	case p.status() == updatable:
		sym, symStyle = "↑", ansi.NewStyle().Foreground(th.Info)
	}
	ver := p.latest
	if p.installed != "" {
		ver = p.installed
	}
	nameW := max(4, w-4-1-len(ver)-1)
	name := ansi.Truncate(p.name, nameW)
	if !selected {
		name = ansi.Truncate(m.mark(p.name, e.pos, ansi.NewStyle()), nameW)
	}
	cursor := "  "
	if selected {
		cursor = "› "
	}
	line := cursor + symStyle.Render(sym) + " " + pad(name, nameW) + " " + ansi.NewStyle().Faint().Render(ver)
	if selected {
		return ansi.NewStyle().Bold().Background(th.Selection).Render(pad(ansi.StripANSI(line), w))
	}
	return ansi.Truncate(line, w)
}

func (m model) listNode(th theme.Theme) layout.Node {
	w, h := m.listW(), m.perPage()
	v := m.visible()
	start := (m.page() - 1) * h
	var rows []layout.FlexChild
	for i := 0; i < h; i++ {
		line := ""
		switch {
		case start+i < len(v):
			line = m.rowLine(i, v[start+i], start+i == m.cur, w, th)
		case i == 0 && len(v) == 0:
			line = ansi.NewStyle().Faint().Render("  No package matches.")
		}
		rows = append(rows, layout.FlexChild{Node: layout.Named("row-"+strconv.Itoa(i), layout.Block(pad(ansi.Truncate(line, w), w)))})
	}
	return layout.Fixed(layout.Column(0, rows...), layout.Size{W: w, H: h})
}

func (m model) action(id, label string, on bool, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Primary)
	switch {
	case !on:
		st = ansi.NewStyle().Faint()
	case id == "act-main":
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	}
	if m.hover == id && on {
		st = st.Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(" "+label+" ")))}
}

func (m model) detailNode(th theme.Theme) layout.Node {
	w, h := m.size().W-m.listW()-1, m.perPage()
	rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
	i, ok := m.selected()
	var lines []layout.Node
	add := func(s string) { lines = append(lines, layout.Block(rule+pad(ansi.Truncate(s, w), w))) }
	if !ok {
		add("")
		return m.column(lines, h)
	}
	p := m.pkgs[i]
	add(ansi.NewStyle().Bold().Render(p.name) + " " + ansi.NewStyle().Faint().Render(p.latest))
	add(m.statusText(p, th))
	add("")
	for _, l := range strings.Split(ansi.Wrap(p.desc, max(1, w-1)), "\n") {
		add(l)
	}
	add("")
	add(ansi.NewStyle().Faint().Render("Size  ") + size(p.sizeKB))
	if len(p.deps) > 0 {
		add(ansi.NewStyle().Faint().Render("Needs"))
		for _, d := range p.deps {
			mark, st := "○", ansi.NewStyle().Faint()
			if j := index(m.pkgs, d); j >= 0 && m.pkgs[j].installed != "" {
				mark, st = "✓", ansi.NewStyle().Foreground(th.Success)
			}
			add("  " + st.Render(mark) + " " + d)
		}
	}
	add("")
	lines = append(lines, layout.Row(0, layout.FlexChild{Node: layout.Block(rule + " ")}, m.action("act-main", m.mainLabel(p), p.status() != installedOK && !m.queued(i), th),
		m.action("act-remove", "Remove", p.installed != "" && !m.queued(i) && len(dependents(m.pkgs, i)) == 0, th)))
	return m.column(lines, h)
}

func (m model) column(lines []layout.Node, h int) layout.Node {
	var kids []layout.FlexChild
	for _, l := range lines {
		kids = append(kids, layout.FlexChild{Node: l})
	}
	return layout.Fixed(layout.Column(0, kids...), layout.Size{W: m.size().W - m.listW(), H: h})
}

func (m model) mainLabel(p pkg) string {
	switch p.status() {
	case updatable:
		return "Update to " + p.latest
	case installedOK:
		return "Up to date"
	}
	return "Install"
}

func (m model) statusText(p pkg, th theme.Theme) string {
	switch p.status() {
	case installedOK:
		return ansi.NewStyle().Foreground(th.Success).Render("✓ installed")
	case updatable:
		return ansi.NewStyle().Foreground(th.Info).Render("↑ " + p.installed + " installed, update available")
	}
	return ansi.NewStyle().Faint().Render("not installed")
}

func (m model) queueLine(th theme.Theme) layout.Node {
	w := m.size().W
	if len(m.queue) == 0 {
		return layout.Fixed(layout.Block(ansi.NewStyle().Faint().Render("Queue empty")), layout.Size{W: w, H: 1})
	}
	j := m.queue[0]
	verb := [...]string{"Installing", "Updating", "Removing"}[j.kind]
	total := jobTicks(m.pkgs[j.idx], j.kind)
	const barW = 12
	fill := barW * m.ticks / total
	bar := ansi.NewStyle().Foreground(th.Primary).Render(strings.Repeat("━", fill)) + ansi.NewStyle().Faint().Render(strings.Repeat("─", barW-fill))
	more := ""
	if n := len(m.queue) - 1; n > 0 {
		more = fmt.Sprintf(" +%d waiting", n)
	}
	label := fmt.Sprintf("%s %s %s %3d%%%s ", verb, m.pkgs[j.idx].name, bar, 100*m.ticks/total, more)
	st := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == "cancel" {
		st = ansi.NewStyle().Foreground(th.Error).Underline()
	}
	return layout.Fixed(layout.Row(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(label, max(1, w-9)))},
		layout.FlexChild{Node: layout.Named("cancel", layout.Block(st.Render("[Cancel]")))},
	), layout.Size{W: w, H: 1})
}

func (m model) pagerLine(th theme.Theme) layout.Node {
	w := m.size().W
	pg := m.pagerModel()
	msg := ansi.NewStyle().Faint().Render(ansi.Truncate(m.msg, max(1, w-pg.Width()-3)))
	return layout.Fixed(layout.Row(2,
		layout.FlexChild{Node: layout.Named("pager", layout.Fixed(pg.LayoutNode(), layout.Size{W: pg.Width(), H: 1}))},
		layout.FlexChild{Node: layout.Block(msg)},
	), layout.Size{W: w, H: 1})
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.Truncate("  "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))
	}
	return layout.Fixed(layout.Column(0,
		key("arrows jk", "move"), key("PgUp PgDn", "previous and next page"), key("/", "filter (Enter keeps, Esc drops)"),
		key("Tab", "all, installed, updates"), key("Enter i", "install or update, with what it needs"),
		key("x", "remove"), key("U", "update everything"), key("c", "cancel the queue"), key("F1", "close help"), key("q", "quit"),
	), layout.Size{W: m.size().W, H: m.perPage()})
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	var body layout.Node = m.listNode(th)
	switch {
	case m.help:
		body = m.helpNode(th)
	case m.wide():
		body = layout.Row(0, layout.FlexChild{Node: m.listNode(th)}, layout.FlexChild{Node: m.detailNode(th)})
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "/", Action: "filter"}, widgets.Hint{Key: "enter", Action: "install"},
		widgets.Hint{Key: "x", Action: "remove"}, widgets.Hint{Key: "tab", Action: "tab"}, widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	return layout.Column(0,
		layout.FlexChild{Node: m.titleLine(th)},
		layout.FlexChild{Node: layout.Block(m.filterLine(th))},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: m.queueLine(th)},
		layout.FlexChild{Node: m.pagerLine(th)},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	opts := []tui.ProgramOption{tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)}
	// ReducedMotion is known as soon as the Program exists, before Run.
	p := tui.NewProgram(newModel(false), opts...)
	if p.ReducedMotion() {
		p = tui.NewProgram(newModel(true), opts...)
	}
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
