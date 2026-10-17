// Command procmon is a process monitor: live CPU and memory, and a table you
// sort by clicking a column header. The numbers are simulated, so it is safe
// to run and to "kill" a process.
//
// Mouse: click a header to sort by it (again to reverse), click a row to select
// it, scroll to move, click Kill, Pause or Filter at the bottom.
// Keyboard: arrows or j/k move, c m p n sort by CPU, memory, PID, name, / filters,
// k or Delete kills (asks first), space pauses, q quits.
//
//	go run ./examples/procmon
package main

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

const (
	history  = 40 // samples the sparklines keep
	interval = time.Second
	chrome   = 6 // title, two stat lines, header, buttons, status
)

type tickMsg time.Time

type model struct {
	procs []proc
	rng   *rand.Rand

	sortCol  int
	desc     bool
	query    string
	typing   bool
	selected int // pid of the selected process, 0 for none
	top      int // first row shown
	paused   bool

	cpuHist, memHist []float64
	askKill          int // pid the kill dialog is about, 0 when closed
	status           string
	hover, pressed   string

	width, height int
}

func initialModel() model {
	m := model{procs: sample(), rng: rand.New(rand.NewSource(7)), sortCol: colCPU, desc: true} // #nosec G404 -- simulated data, not security-sensitive
	for i := 0; i < history; i++ {                                                             // warm up, so the graphs start with a past
		var cpu, mem float64
		m.procs, cpu, mem = step(m.procs, m.rng)
		m.cpuHist = append(m.cpuHist, cpu)
		m.memHist = append(m.memHist, mem)
	}
	sortProcs(m.procs, m.sortCol, m.desc)
	m.selected = m.procs[0].pid
	return m
}

func (m model) Init() tui.Cmd { return m.tick() }

func (m model) tick() tui.Cmd {
	return tui.FromCtx(motion.After(interval, func(t time.Time) tui.Msg { return tickMsg(t) }))
}

// shown is the processes that pass the filter, in sort order.
func (m model) shown() []proc {
	if m.query == "" {
		return m.procs
	}
	var out []proc
	for _, p := range m.procs {
		if match(p, m.query) {
			out = append(out, p)
		}
	}
	return out
}

func (m model) rows() int {
	if m.height == 0 {
		return 18
	}
	return max(1, m.height-chrome)
}

func (m model) index(pid int) int {
	for i, p := range m.shown() {
		if p.pid == pid {
			return i
		}
	}
	return -1
}

// move puts the selection n rows down (negative is up) and scrolls to it.
func (m model) move(n int) model {
	rows := m.shown()
	if len(rows) == 0 {
		m.selected = 0
		return m
	}
	i := max(0, min(len(rows)-1, max(m.index(m.selected), 0)+n))
	m.selected = rows[i].pid
	return m.reveal()
}

// reveal scrolls so the selected row is on screen.
func (m model) reveal() model {
	n := len(m.shown())
	m.top = max(0, min(m.top, n-m.rows()))
	if i := m.index(m.selected); i >= 0 {
		if i < m.top {
			m.top = i
		}
		if i >= m.top+m.rows() {
			m.top = i - m.rows() + 1
		}
	}
	return m
}

// sortBy sorts by col, or reverses the order when it is the sort column.
func (m model) sortBy(col int) model {
	if col == m.sortCol {
		m.desc = !m.desc
	} else {
		m.sortCol = col
		m.desc = col == colCPU || col == colMem // biggest first for numbers of load
	}
	sortProcs(m.procs, m.sortCol, m.desc)
	return m.reveal()
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.reveal(), nil
	case tickMsg:
		if m.paused {
			return m, m.tick()
		}
		var cpu, mem float64
		m.procs, cpu, mem = step(m.procs, m.rng)
		sortProcs(m.procs, m.sortCol, m.desc)
		m.cpuHist = append(m.cpuHist[1:], cpu)
		m.memHist = append(m.memHist[1:], mem)
		return m.reveal(), m.tick()
	case tui.MouseEvent:
		return m.mouse(msg), nil
	case tui.Key:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if m.askKill != 0 {
		return m.killKey(k), nil
	}
	if m.typing {
		return m.typingKey(k), nil
	}
	switch k.Type {
	case tui.KeyEsc:
		if m.query != "" {
			m.query = ""
			return m.move(0), nil
		}
		return m, tui.Quit()
	case tui.KeyUp:
		return m.move(-1), nil
	case tui.KeyDown:
		return m.move(1), nil
	case tui.KeyPgUp:
		return m.move(-m.rows()), nil
	case tui.KeyPgDown:
		return m.move(m.rows()), nil
	case tui.KeyHome:
		return m.move(-1 << 20), nil
	case tui.KeyEnd:
		return m.move(1 << 20), nil
	case tui.KeyDelete:
		return m.ask(), nil
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) runeKey(s string) (tui.Model, tui.Cmd) {
	switch s {
	case "q":
		return m, tui.Quit()
	case "j":
		return m.move(1), nil
	case "k":
		return m.ask(), nil
	case "c":
		return m.sortBy(colCPU), nil
	case "m":
		return m.sortBy(colMem), nil
	case "p":
		return m.sortBy(colPID), nil
	case "n":
		return m.sortBy(colName), nil
	case " ":
		m.paused = !m.paused
	case "/":
		m.typing = true
	}
	return m, nil
}

func (m model) typingKey(k tui.Key) model {
	switch k.Type {
	case tui.KeyEsc:
		m.typing, m.query = false, ""
	case tui.KeyEnter:
		m.typing = false
	case tui.KeyBackspace:
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
	case tui.KeyRunes:
		m.query += k.Text
	}
	return m.move(0)
}

// ask opens the kill dialog for the selected process.
func (m model) ask() model {
	if m.index(m.selected) >= 0 {
		m.askKill = m.selected
	}
	return m
}

func (m model) killKey(k tui.Key) model {
	switch {
	case k.Type == tui.KeyRunes && (k.Text == "y" || k.Text == "Y"):
		return m.kill()
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && (k.Text == "n" || k.Text == "N"):
		m.askKill = 0
	}
	return m
}

func (m model) kill() model {
	pid := m.askKill
	m.askKill = 0
	i := m.index(pid)
	if i < 0 {
		return m
	}
	name := m.shown()[i].name
	var kept []proc
	for _, p := range m.procs {
		if p.pid != pid {
			kept = append(kept, p)
		}
	}
	m.procs = kept
	m.status = fmt.Sprintf("Killed %s (pid %d)", name, pid)
	rows := m.shown()
	if len(rows) > 0 {
		m.selected = rows[min(i, len(rows)-1)].pid
	} else {
		m.selected = 0
	}
	return m.reveal()
}

func (m model) mouse(ev tui.MouseEvent) model {
	hit, ok := hittest.HitMap(m.screen(), m.size()).At(ev.X, ev.Y)
	id := ""
	if ok {
		id = hit.ID
	}
	m.hover = id
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		if m.askKill != 0 {
			return m
		}
		d := 3
		if ev.Button == tui.MouseButtonWheelUp {
			d = -3
		}
		m.top = max(0, min(m.top+d, len(m.shown())-m.rows()))
		return m
	case tui.MouseButtonLeft:
	default:
		return m
	}
	switch ev.Action {
	case tui.MouseActionPress:
		m.pressed = id
	case tui.MouseActionRelease:
		pressed := m.pressed
		m.pressed = ""
		if pressed != "" && pressed == id {
			return m.activate(id)
		}
	}
	return m
}

func (m model) activate(id string) model {
	if m.askKill != 0 {
		switch id {
		case "kill-yes":
			return m.kill()
		case "kill-no":
			m.askKill = 0
		}
		return m
	}
	switch {
	case strings.HasPrefix(id, "col-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "col-"))
		return m.sortBy(n)
	case strings.HasPrefix(id, "row-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "row-"))
		if rows := m.shown(); m.top+n < len(rows) {
			m.selected = rows[m.top+n].pid
		}
	case id == "kill":
		return m.ask()
	case id == "pause":
		m.paused = !m.paused
	case id == "filter":
		m.typing = true
	}
	return m
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

// widths gives the column widths for a terminal w cells wide; the name column
// takes what the others leave, and USER goes first when space is short.
func widths(w int) [numCols]int {
	cw := [numCols]int{colPID: 6, colUser: 9, colCPU: 6, colMem: 6, colState: 5}
	if w < 64 {
		cw[colUser] = 0
	}
	rest := w - 1
	for _, n := range cw {
		if n > 0 {
			rest -= n + 1
		}
	}
	cw[colName] = max(8, rest)
	return cw
}

func fit(s string, w int, right bool) string {
	s = ansi.Truncate(s, w)
	pad := strings.Repeat(" ", max(0, w-ansi.Width(s)))
	if right {
		return pad + s
	}
	return s + pad
}

func (m model) button(id, label string, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Text)
	switch {
	case m.pressed == id && m.hover == id:
		st = ansi.NewStyle().Bold().Reverse()
	case m.hover == id:
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render("[ "+label+" ]")))}
}

func (m model) header(cw [numCols]int, th theme.Theme) layout.Node {
	var cells []layout.FlexChild
	for c := 0; c < numCols; c++ {
		if cw[c] == 0 {
			continue
		}
		label := colNames[c]
		if c == m.sortCol {
			if m.desc {
				label += "▼"
			} else {
				label += "▲"
			}
		}
		st := ansi.NewStyle().Bold().Foreground(th.Muted)
		if c == m.sortCol {
			st = ansi.NewStyle().Bold().Foreground(th.Primary)
		}
		if m.hover == "col-"+strconv.Itoa(c) {
			st = st.Underline()
		}
		right := c == colPID || c == colCPU || c == colMem
		cells = append(cells, layout.FlexChild{Node: layout.Named("col-"+strconv.Itoa(c), layout.Block(st.Render(fit(label, cw[c], right))))})
	}
	return layout.Row(1, cells...)
}

func (m model) row(i int, p proc, cw [numCols]int, th theme.Theme) layout.Node {
	id := "row-" + strconv.Itoa(i)
	cpu := ansi.NewStyle()
	switch {
	case p.cpu >= 40:
		cpu = ansi.NewStyle().Bold().Foreground(th.Error)
	case p.cpu >= 15:
		cpu = ansi.NewStyle().Foreground(th.Warning)
	}
	bg := ansi.NewStyle()
	switch {
	case p.pid == m.selected:
		bg = ansi.NewStyle().Background(th.Selection).Bold()
	case m.hover == id:
		bg = ansi.NewStyle().Background(th.Selection).Faint()
	}
	var parts []string
	for c := 0; c < numCols; c++ {
		if cw[c] == 0 {
			continue
		}
		var text string
		switch c {
		case colPID:
			text = fit(strconv.Itoa(p.pid), cw[c], true)
		case colName:
			text = fit(p.name, cw[c], false)
		case colUser:
			text = fit(p.user, cw[c], false)
		case colCPU:
			text = cpu.Render(fit(fmt.Sprintf("%.1f", p.cpu), cw[c], true))
		case colMem:
			text = fit(fmt.Sprintf("%.1f", p.mem), cw[c], true)
		case colState:
			text = fit(p.state, cw[c], false)
		}
		parts = append(parts, bg.Render(text))
	}
	return layout.Named(id, layout.Block(strings.Join(parts, bg.Render(" "))+bg.Render(" ")))
}

func (m model) stat(label string, value float64, hist []float64, th theme.Theme) string {
	w := m.size().W
	bar := 20
	if w < 60 {
		bar = 10
	}
	line := fmt.Sprintf("%-4s %s %5.1f%%", label, widgets.ProgressBar(value/100, bar, th), value)
	if w >= 50 {
		line += "  " + chart.SparklineWith(hist[max(0, len(hist)-(w-ansi.Width(line)-4)):], th)
	}
	return ansi.Truncate(line, w)
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	cw := widths(w)
	cpu, mem := m.cpuHist[len(m.cpuHist)-1], m.memHist[len(m.memHist)-1]

	title := ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary).Render(" Processes ")
	state := fmt.Sprintf("  %d running", len(m.procs))
	if m.paused {
		state = ansi.NewStyle().Foreground(th.Warning).Render("  paused")
	}
	if m.query != "" || m.typing {
		state += ansi.NewStyle().Foreground(th.Muted).Render("  filter: ") + m.query
		if m.typing {
			state += "▏"
		}
	}

	kids := []layout.FlexChild{
		{Node: layout.Block(ansi.Truncate(title+state, w))},
		{Node: layout.Block(m.stat("CPU", cpu, m.cpuHist, th))},
		{Node: layout.Block(m.stat("MEM", mem, m.memHist, th))},
		{Node: m.header(cw, th)},
	}
	rows := m.shown()
	for r := 0; r < m.rows(); r++ {
		if m.top+r < len(rows) {
			kids = append(kids, layout.FlexChild{Node: m.row(r, rows[m.top+r], cw, th)})
		} else if len(rows) == 0 && r == 0 {
			kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("No process matches."))})
		} else {
			kids = append(kids, layout.FlexChild{Node: layout.Block(" ")})
		}
	}
	pause := "Pause"
	if m.paused {
		pause = "Resume"
	}
	kids = append(kids,
		layout.FlexChild{Node: layout.Row(1, m.button("kill", "Kill", th), m.button("pause", pause, th), m.button("filter", "Filter /", th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(m.footer(), w)))},
	)
	root := layout.Column(0, kids...)
	if m.askKill != 0 {
		return layout.Stack(root, layout.Layer(5, layout.Anchor(layout.Center, 0, 0, m.dialog(th))))
	}
	return root
}

func (m model) footer() string {
	if m.status != "" {
		return m.status
	}
	return widgets.Footer(m.size().W, widgets.Hint{Key: "click header", Action: "sort"}, widgets.Hint{Key: "k", Action: "kill"},
		widgets.Hint{Key: "space", Action: "pause"}, widgets.Hint{Key: "q", Action: "quit"})
}

func (m model) dialog(th theme.Theme) layout.Node {
	name := ""
	if i := m.index(m.askKill); i >= 0 {
		name = m.shown()[i].name
	}
	body := layout.Column(1,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Bold().Render("Kill " + name + "?"))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("pid " + strconv.Itoa(m.askKill)))},
		layout.FlexChild{Node: layout.Row(2, m.button("kill-yes", "Yes (y)", th), m.button("kill-no", "No (n)", th))},
	)
	return layout.BoxNode(layout.NewBox().Border(th.Border).PaddingAll(1), body)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
