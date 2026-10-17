// Command settings is a preferences screen: a list of categories on the left,
// and on the right the switches, sliders and choices of the one you picked.
// A changed setting is marked with a dot; nothing takes effect until you
// Apply, and Reset throws the changes away.
//
// Mouse: click a category, click a row, click a switch to flip it, click the
// arrows of a choice, click or drag along a slider, click Apply or Reset.
// Keyboard: Tab swaps between the category list and the settings, Up and Down
// move, Left and Right change a value, Space flips a switch, PgUp and PgDn
// change category, Ctrl+S applies, Ctrl+R resets, F1 shows help, q or Ctrl+Q
// quits (asking first if changes would be lost), Ctrl+C quits at once.
//
//	go run ./examples/settings              # in memory
//	go run ./examples/settings -file s.txt  # Apply saves here, and it is read at start
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	sidebarW = 18
	wideFrom = 60
	ctrlW    = 22 // width of the control column
	trackW   = 10
)

type pane int

const (
	inCats pane = iota
	inList
)

type model struct {
	cats  []category
	saved values // what Apply last kept
	cur   values // what the screen shows
	cat   int
	row   int
	pane  pane

	hover   string
	drag    int // row of the slider being dragged, or -1
	confirm bool
	help    bool
	status  string
	path    string // file Apply writes; empty keeps settings in memory

	width, height int
}

func initialModel() model {
	cats := catalogue()
	d := defaults(cats)
	return model{cats: cats, saved: d, cur: d, pane: inList, drag: -1}
}

func (m model) Init() tui.Cmd { return nil }

// ---- state -----------------------------------------------------------------

func (m model) items() []setting { return m.cats[m.cat].items }

func (m model) selected() setting { return m.items()[m.row] }

func (m model) dirtyKeys() []string {
	var out []string
	for _, c := range m.cats {
		for _, s := range c.items {
			if m.cur[s.key] != m.saved[s.key] {
				out = append(out, s.key)
			}
		}
	}
	return out
}

func (m model) dirtyIn(c category) int {
	n := 0
	for _, s := range c.items {
		if m.cur[s.key] != m.saved[s.key] {
			n++
		}
	}
	return n
}

func (m model) set(i int, x float64) model {
	s := m.items()[i]
	if !s.enabled(m.cur) {
		m.status = s.label + " needs " + m.needName(s) + " on"
		return m
	}
	m.cur = m.cur.with(s.key, s.clamp(x))
	m.status = ""
	return m
}

func (m model) needName(s setting) string {
	for _, c := range m.cats {
		for _, o := range c.items {
			if o.key == s.needs {
				return strings.ToLower(o.label)
			}
		}
	}
	return s.needs
}

func (m model) setCat(i int) model {
	n := len(m.cats)
	m.cat = ((i % n) + n) % n
	m.row = min(m.row, len(m.items())-1)
	m.status = ""
	return m
}

func (m model) apply() (model, tui.Cmd) {
	n := len(m.dirtyKeys())
	if n == 0 {
		m.status = "Nothing to apply"
		return m, nil
	}
	m.saved = m.cur
	m.status = fmt.Sprintf("Applied %d changes", n)
	if n == 1 {
		m.status = "Applied 1 change"
	}
	if m.path == "" {
		return m, nil
	}
	return m, writeFile(m.path, encode(m.cats, m.cur))
}

func (m model) reset() model {
	n := len(m.dirtyKeys())
	m.cur = m.saved
	m.status = "Nothing to reset"
	if n > 0 {
		m.status = "Changes discarded"
	}
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.MouseEvent:
		return m.mouse(msg)
	case tui.Key:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyCtrlC:
		return m, tui.Quit()
	case m.confirm:
		return m.confirmKey(k)
	case k.Type == tui.KeyF1:
		m.help = !m.help
		return m, nil
	case m.help:
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	case k.Type == tui.KeyCtrl:
		return m.ctrlKey(k.Code)
	}
	switch k.Type {
	case tui.KeyTab:
		m.pane = 1 - m.pane
	case tui.KeyPgUp:
		return m.setCat(m.cat - 1), nil
	case tui.KeyPgDown:
		return m.setCat(m.cat + 1), nil
	case tui.KeyEsc:
		if m.pane == inList && m.wide() {
			m.pane = inCats
			return m, nil
		}
		return m.quit()
	case tui.KeyRunes:
		if k.Text == "q" {
			return m.quit()
		}
	}
	if m.pane == inCats && m.wide() {
		return m.catsKey(k), nil
	}
	return m.listKey(k), nil
}

func (m model) ctrlKey(code rune) (tui.Model, tui.Cmd) {
	switch code {
	case 's':
		return m.apply()
	case 'r':
		return m.reset(), nil
	case 'q':
		return m.quit()
	case 'n':
		return m.setCat(m.cat + 1), nil
	case 'p':
		return m.setCat(m.cat - 1), nil
	}
	return m, nil
}

func (m model) quit() (tui.Model, tui.Cmd) {
	if len(m.dirtyKeys()) > 0 {
		m.confirm = true
		return m, nil
	}
	return m, tui.Quit()
}

func (m model) confirmKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyRunes && (k.Text == "y" || k.Text == "Y"):
		return m, tui.Quit()
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && (k.Text == "n" || k.Text == "N"):
		m.confirm = false
	}
	return m, nil
}

func (m model) catsKey(k tui.Key) model {
	switch k.Type {
	case tui.KeyUp:
		return m.setCat(m.cat - 1)
	case tui.KeyDown:
		return m.setCat(m.cat + 1)
	case tui.KeyRight, tui.KeyEnter:
		m.pane = inList
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			return m.setCat(m.cat - 1)
		case "j":
			return m.setCat(m.cat + 1)
		case "l":
			m.pane = inList
		}
	}
	return m
}

func (m model) listKey(k tui.Key) model {
	s := m.selected()
	val := m.cur[s.key]
	switch k.Type {
	case tui.KeyUp:
		m.row = max(0, m.row-1)
	case tui.KeyDown:
		m.row = min(len(m.items())-1, m.row+1)
	case tui.KeyLeft:
		return m.set(m.row, s.step(val, -1))
	case tui.KeyRight, tui.KeyEnter:
		return m.set(m.row, s.step(val, 1))
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.row = max(0, m.row-1)
		case "j":
			m.row = min(len(m.items())-1, m.row+1)
		case "h":
			return m.set(m.row, s.step(val, -1))
		case "l", " ":
			return m.set(m.row, s.step(val, 1))
		}
	}
	return m
}

// ---- mouse -----------------------------------------------------------------

func (m model) ids() []string {
	ids := []string{"apply", "reset", "yes", "no", "cat-prev", "cat-next"}
	for i := range m.items() {
		n := strconv.Itoa(i)
		ids = append(ids, "dec-"+n, "inc-"+n, "track-"+n, "ctl-"+n, "row-"+n)
	}
	for i := range m.cats {
		ids = append(ids, "cat-"+strconv.Itoa(i))
	}
	return ids
}

func (m model) hit(x, y int) string {
	for _, id := range m.ids() {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H {
			return id
		}
	}
	return ""
}

func idNum(id, prefix string) (int, bool) {
	if !strings.HasPrefix(id, prefix) {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(id, prefix))
	return n, err == nil
}

// slideTo sets slider row i from a pointer at column x.
func (m model) slideTo(i, x int) model {
	r, ok := layout.RectOf(m.screen(), m.size(), "track-"+strconv.Itoa(i))
	if !ok || r.W < 2 {
		return m
	}
	s := m.items()[i]
	frac := float64(max(0, min(x-r.X, r.W-1))) / float64(r.W-1)
	return m.set(i, s.min+frac*(s.max-s.min))
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch ev.Action {
	case tui.MouseActionMotion:
		if m.drag >= 0 {
			return m.slideTo(m.drag, ev.X), nil
		}
		m.hover = id
		return m, nil
	case tui.MouseActionRelease:
		m.drag = -1
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		d := 1
		if ev.Button == tui.MouseButtonWheelUp {
			d = -1
		}
		m.row = max(0, min(len(m.items())-1, m.row+d))
		return m, nil
	case tui.MouseButtonLeft:
		return m.click(id, ev.X)
	}
	return m, nil
}

func (m model) click(id string, x int) (tui.Model, tui.Cmd) {
	if m.confirm {
		switch id {
		case "yes":
			return m, tui.Quit()
		case "no":
			m.confirm = false
		}
		return m, nil
	}
	if n, ok := idNum(id, "cat-"); ok && id != "cat-prev" && id != "cat-next" {
		m.pane = inCats
		return m.setCat(n), nil
	}
	if n, ok := idNum(id, "row-"); ok {
		m.row, m.pane = n, inList
		return m, nil
	}
	return m.clickControl(id, x)
}

func (m model) clickControl(id string, x int) (tui.Model, tui.Cmd) {
	for _, p := range []string{"dec-", "inc-", "ctl-", "track-"} {
		n, ok := idNum(id, p)
		if !ok {
			continue
		}
		m.row, m.pane = n, inList
		s := m.items()[n]
		switch p {
		case "dec-":
			return m.set(n, s.step(m.cur[s.key], -1)), nil
		case "inc-", "ctl-":
			return m.set(n, s.step(m.cur[s.key], 1)), nil
		}
		if s.enabled(m.cur) {
			m.drag = n
		}
		return m.slideTo(n, x), nil
	}
	switch id {
	case "apply":
		return m.apply()
	case "reset":
		return m.reset(), nil
	case "cat-prev":
		return m.setCat(m.cat - 1), nil
	case "cat-next":
		return m.setCat(m.cat + 1), nil
	}
	return m, nil
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) wide() bool { return m.size().W >= wideFrom }

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func fit(s string, w int) string { return pad(ansi.Truncate(s, w), w) }

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) button(id, label string, on bool, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	switch {
	case !on:
		st = ansi.NewStyle().Faint()
	case m.hover == id:
		st = ansi.NewStyle().Bold().Foreground(th.Primary).Underline()
	case id == "apply":
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(" "+label+" ")))}
}

func (m model) title(th theme.Theme) layout.Node {
	n := len(m.dirtyKeys())
	state := ansi.NewStyle().Faint().Render("All changes applied")
	if n > 0 {
		state = ansi.NewStyle().Foreground(th.Warning).Render(fmt.Sprintf("● %d unsaved", n))
	}
	w := m.size().W
	return layout.Fixed(layout.Row(2,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Settings")), Grow: 1},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(state, max(1, w-12)))},
	), layout.Size{W: w, H: 1})
}

func (m model) sidebar(th theme.Theme, h int) layout.Node {
	rule := ansi.NewStyle().Foreground(th.Muted).Render("│")
	var rows []layout.FlexChild
	for i, c := range m.cats {
		dot := ""
		if n := m.dirtyIn(c); n > 0 {
			dot = " ●"
		}
		id := "cat-" + strconv.Itoa(i)
		st := ansi.NewStyle()
		switch {
		case i == m.cat && m.pane == inCats:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case i == m.cat:
			st = ansi.NewStyle().Bold().Background(th.Selection)
		case m.hover == id:
			st = ansi.NewStyle().Underline()
		}
		rows = append(rows, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(fit(" "+c.name+dot, sidebarW-1))+rule))})
	}
	for len(rows) < h {
		rows = append(rows, block(strings.Repeat(" ", sidebarW-1)+rule))
	}
	return layout.Fixed(layout.Column(0, rows[:h]...), layout.Size{W: sidebarW, H: h})
}

// control draws the right-hand part of a row and gives its parts click ids.
func (m model) control(i int, s setting, th theme.Theme) layout.Node {
	n := strconv.Itoa(i)
	x := m.cur[s.key]
	off := !s.enabled(m.cur)
	hot := ansi.NewStyle().Foreground(th.Primary)
	if off {
		hot = ansi.NewStyle().Faint()
	}
	switch s.kind {
	case toggle:
		glyph := hot.Render("○━━") + " Off"
		if x == 1 {
			glyph = hot.Bold().Render("━━●") + " On"
		}
		return layout.Fixed(layout.Named("ctl-"+n, layout.Block(glyph)), layout.Size{W: ctrlW, H: 1})
	case choice:
		arrow := func(id, g string) layout.FlexChild {
			st := hot
			if m.hover == id {
				st = st.Bold().Underline()
			}
			return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(g)))}
		}
		label := fit(s.text(x), ctrlW-4)
		return layout.Fixed(layout.Row(0, arrow("dec-"+n, "‹ "), block(label), arrow("inc-"+n, " ›")), layout.Size{W: ctrlW, H: 1})
	}
	frac := (x - s.min) / (s.max - s.min)
	at := int(frac*float64(trackW-1) + 0.5)
	bar := hot.Render(strings.Repeat("━", at)) + hot.Bold().Render("●") + ansi.NewStyle().Faint().Render(strings.Repeat("─", trackW-1-at))
	return layout.Fixed(layout.Row(1,
		layout.FlexChild{Node: layout.Named("track-"+n, layout.Block(bar))},
		block(s.text(x)),
	), layout.Size{W: ctrlW, H: 1})
}

func (m model) rowNode(i int, s setting, labelW int, th theme.Theme) layout.Node {
	n := strconv.Itoa(i)
	st := ansi.NewStyle()
	switch {
	case !s.enabled(m.cur):
		st = ansi.NewStyle().Faint()
	case i == m.row && m.pane == inList:
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	case i == m.row:
		st = ansi.NewStyle().Bold()
	}
	mark := "  "
	if i == m.row && m.pane == inList {
		mark = "› "
	}
	dirty := " "
	if m.cur[s.key] != m.saved[s.key] {
		dirty = ansi.NewStyle().Foreground(th.Warning).Render("●")
	}
	return layout.Row(1,
		layout.FlexChild{Node: layout.Named("row-"+n, layout.Block(st.Render(mark+fit(s.label, labelW))))},
		layout.FlexChild{Node: m.control(i, s, th)},
		block(dirty),
	)
}

func (m model) list(th theme.Theme, w, h int) layout.Node {
	labelW := max(6, w-ctrlW-6)
	var kids []layout.FlexChild
	if !m.wide() {
		kids = append(kids, layout.FlexChild{Node: m.catSwitcher(th)})
	} else {
		kids = append(kids, block(ansi.NewStyle().Bold().Render(" "+m.cats[m.cat].name)))
	}
	for i, s := range m.items() {
		kids = append(kids, layout.FlexChild{Node: m.rowNode(i, s, labelW, th)})
	}
	for len(kids) < h {
		kids = append(kids, block(""))
	}
	return layout.Fixed(layout.Column(0, kids[:h]...), layout.Size{W: w, H: h})
}

func (m model) catSwitcher(th theme.Theme) layout.Node {
	arrow := func(id, g string) layout.FlexChild {
		st := ansi.NewStyle().Foreground(th.Primary)
		if m.hover == id {
			st = st.Bold().Underline()
		}
		return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(g)))}
	}
	name := fmt.Sprintf("%s (%d/%d)", m.cats[m.cat].name, m.cat+1, len(m.cats))
	return layout.Row(1, arrow("cat-prev", "‹"), block(ansi.NewStyle().Bold().Render(name)), arrow("cat-next", "›"))
}

func (m model) helpPanel(th theme.Theme, w, h int) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(fit(k, 16))+a, w))
	}
	kids := []layout.FlexChild{
		key("Tab", "swap categories and settings"), key("Up Down j k", "move"),
		key("Left Right h l", "change a value"), key("Space Enter", "flip or step forward"),
		key("PgUp PgDn", "previous or next category"), key("Ctrl+S", "apply the changes"),
		key("Ctrl+R", "reset the changes"), key("q Ctrl+Q", "quit, asking if changes would be lost"),
		key("F1", "close this help"),
	}
	for len(kids) < h {
		kids = append(kids, block(""))
	}
	return layout.Fixed(layout.Column(0, kids[:h]...), layout.Size{W: w, H: h})
}

func (m model) footer(th theme.Theme) layout.Node {
	w := m.size().W
	if m.confirm {
		warn := ansi.NewStyle().Bold().Foreground(th.Warning).Render(fmt.Sprintf("Discard %d unsaved changes? ", len(m.dirtyKeys())))
		yes := layout.Named("yes", layout.Block("[y] yes"))
		no := layout.Named("no", layout.Block("[n] no"))
		return layout.Row(2, layout.FlexChild{Node: layout.Block(warn)}, layout.FlexChild{Node: yes}, layout.FlexChild{Node: no})
	}
	f := widgets.Footer(w,
		widgets.Hint{Key: "tab", Action: "pane"}, widgets.Hint{Key: "←→", Action: "change"},
		widgets.Hint{Key: "ctrl+s", Action: "apply"}, widgets.Hint{Key: "ctrl+r", Action: "reset"},
		widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	return layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(f, w)))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	sz := m.size()
	h := max(1, sz.H-4) // all but the title, the help line, the buttons and the footer
	listW := sz.W
	if m.wide() {
		listW -= sidebarW
	}
	body := m.list(th, listW, h)
	if m.help {
		body = m.helpPanel(th, listW, h)
	}
	if m.wide() {
		body = layout.Row(0, layout.FlexChild{Node: m.sidebar(th, h)}, layout.FlexChild{Node: body})
	}
	line := m.status
	if line == "" && !m.help {
		s := m.selected()
		line = s.help
		if !s.enabled(m.cur) {
			line = "Turn " + m.needName(s) + " on to change this."
		}
	}
	buttons := layout.Row(2,
		m.button("apply", "Apply", len(m.dirtyKeys()) > 0, th),
		m.button("reset", "Reset", len(m.dirtyKeys()) > 0, th))
	return layout.Column(0,
		layout.FlexChild{Node: m.title(th)},
		layout.FlexChild{Node: body},
		block(ansi.NewStyle().Faint().Render(fit(" "+line, sz.W))),
		layout.FlexChild{Node: buttons},
		layout.FlexChild{Node: m.footer(th)},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	file := flag.String("file", "", "save applied settings to this file, and read them at start")
	flag.Parse()
	m := initialModel()
	if *file != "" {
		var err error
		if m, err = fromFile(m, *file); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
	p := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
