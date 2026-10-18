// Command launcher is a command palette for a project: type to narrow the list
// of commands (a fuzzy match on name and description, the ones you ran last
// first), press Enter to run one, and read its output below as it arrives. The
// sample project plays scripted output and runs nothing; give it your own
// commands in a file and it runs them for real.
//
// Mouse: click a command to select it and double-click to run it, scroll the
// wheel over the list to move and over the output to scroll it, click Cancel
// or Rerun beside the output. Keyboard: type to filter, Up and Down move,
// Enter runs, PgUp and PgDn scroll the output, Ctrl+X cancels the run, Ctrl+R
// runs it again, Esc clears the filter, F1 shows help, Ctrl+Q or Ctrl+C quits.
//
//	go run ./examples/launcher                  # the sample project, scripted
//	go run ./examples/launcher -file cmds.txt   # runs your commands
//
// A command file has one command a line, "name | description | shell command",
// and # starts a comment. Commands run with sh -c, output and errors together.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/fuzzy"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	chrome     = 5 // title, input, output header, hints, and one row of slack
	maxRecents = 5
	scriptGap  = 120 * time.Millisecond
)

type entry struct {
	idx int
	pos []int // matched rune offsets in the name
}

// run is the latest run, whose output the lower pane shows.
type run struct {
	name    string
	lines   []string
	running bool
	code    int
	err     error
	elapsed time.Duration
	seq     int
	began   time.Time
	cancel  context.CancelFunc
}

type model struct {
	th      ui.Theming // the theme the Program chose; see SetTheme
	cmds    []command
	recents []string // names, newest first
	in      textinput.Model
	cur     int
	runner  func(c command) runner

	last   run
	scroll int // lines scrolled up from the newest output
	seq    int
	help   bool
	hover  string
	file   bool // the commands come from a file, so they really run

	width, height int
}

func newModel(cmds []command, file bool, gap time.Duration) model {
	in := textinput.New()
	in.Prompt = "› "
	in.Placeholder = "Type a command name…"
	in.Focus()
	m := model{cmds: cmds, in: in, file: file}
	m.runner = func(c command) runner {
		if c.shell != "" {
			return shellRunner
		}
		return scriptRunner(gap)
	}
	return m
}

func (m model) Init() tui.Cmd {
	in := m.in
	return in.Focus()
}

// ---- the list --------------------------------------------------------------

func (m model) recency(name string) int {
	for i, r := range m.recents {
		if r == name {
			return i
		}
	}
	return len(m.recents)
}

// matches are the commands that fit the query, best first; with no query, the
// recent ones in the order they were run and then the rest as given.
func (m model) matches() []entry {
	q := strings.TrimSpace(m.in.Value())
	if q == "" {
		out := make([]entry, len(m.cmds))
		for i := range m.cmds {
			out[i] = entry{idx: i}
		}
		sort.SliceStable(out, func(a, b int) bool { return m.recency(m.cmds[out[a].idx].name) < m.recency(m.cmds[out[b].idx].name) })
		return out
	}
	res := fuzzy.Find(q, len(m.cmds), func(i int) string { return m.cmds[i].name + " " + m.cmds[i].desc })
	sort.SliceStable(res, func(a, b int) bool {
		if res[a].Score != res[b].Score {
			return res[a].Score > res[b].Score
		}
		return m.recency(m.cmds[res[a].Index].name) < m.recency(m.cmds[res[b].Index].name)
	})
	out := make([]entry, len(res))
	for i, r := range res {
		out[i] = entry{idx: r.Index}
		n := len([]rune(m.cmds[r.Index].name))
		for _, p := range r.Positions {
			if p < n {
				out[i].pos = append(out[i].pos, p)
			}
		}
	}
	return out
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

// listRows is how many commands show at once.
func (m model) listRows() int { return max(1, min(8, (m.size().H-4)/3)) }

// outRows is how many output lines show.
func (m model) outRows() int { return max(1, m.size().H-3-m.listRows()-1) }

func (m model) clamp() model {
	m.cur = max(0, min(m.cur, len(m.matches())-1))
	m.scroll = max(0, min(m.scroll, max(0, len(m.last.lines)-m.outRows())))
	return m
}

func (m model) selected() (command, bool) {
	v := m.matches()
	if len(v) == 0 {
		return command{}, false
	}
	return m.cmds[v[min(m.cur, len(v)-1)].idx], true
}

// ---- running ---------------------------------------------------------------

func (m model) execute(c command) (model, tui.Cmd) {
	if m.last.cancel != nil {
		m.last.cancel() // a new run replaces the one under way
	}
	m.seq++
	began := time.Now()
	ch, cancel := start(context.Background(), m.runner(c), c)
	m.last = run{name: c.name, running: true, seq: m.seq, began: began, cancel: cancel}
	m.scroll = 0
	rec := []string{c.name}
	for _, r := range m.recents {
		if r != c.name && len(rec) < maxRecents {
			rec = append(rec, r)
		}
	}
	m.recents = rec
	return m, next(m.seq, ch, began)
}

func (m model) onOut(o outMsg) (tui.Model, tui.Cmd) {
	if o.seq != m.last.seq {
		return m, nil // a line from a run that was replaced
	}
	lines := append(append([]string(nil), m.last.lines...), o.line)
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	m.last.lines = lines
	if m.scroll > 0 {
		m.scroll++ // keep the place being read
	}
	return m.clamp(), next(o.seq, o.ch, o.began)
}

func (m model) onDone(d doneMsg) model {
	if d.seq != m.last.seq {
		return m
	}
	m.last.running, m.last.code, m.last.err, m.last.elapsed = false, d.code, d.err, d.elapsed
	return m
}

func (m model) stop() model {
	if m.last.running && m.last.cancel != nil {
		m.last.cancel()
	}
	return m
}

func (m model) quit() (tui.Model, tui.Cmd) {
	if m.last.cancel != nil {
		m.last.cancel()
	}
	return m, tui.Quit()
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.clamp(), nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	case outMsg:
		return m.onOut(msg)
	case doneMsg:
		return m.onDone(msg), nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(msg)
	return m, cmd
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyCtrlC:
		return m.quit()
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
	case tui.KeyUp:
		m.cur--
		return m.clamp(), nil
	case tui.KeyDown:
		m.cur++
		return m.clamp(), nil
	case tui.KeyPgUp:
		m.scroll += max(1, m.outRows()-1)
		return m.clamp(), nil
	case tui.KeyPgDown:
		m.scroll -= max(1, m.outRows()-1)
		return m.clamp(), nil
	case tui.KeyEnter:
		return m.runSelected()
	case tui.KeyEsc:
		m.in.Reset()
		m.cur = 0
		return m, nil
	}
	before := m.in.Value()
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	if m.in.Value() != before {
		m.cur = 0
	}
	return m, cmd
}

func (m model) ctrlKey(code rune) (tui.Model, tui.Cmd) {
	switch code {
	case 'q':
		return m.quit()
	case 'x':
		return m.stop(), nil
	case 'r':
		return m.rerun()
	}
	return m, nil
}

func (m model) runSelected() (tui.Model, tui.Cmd) {
	c, ok := m.selected()
	if !ok {
		return m, nil
	}
	m.in.Reset()
	m.cur = 0
	return m.execute(c)
}

func (m model) rerun() (tui.Model, tui.Cmd) {
	for _, c := range m.cmds {
		if c.name == m.last.name {
			return m.execute(c)
		}
	}
	return m, nil
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"cancel", "rerun", "out"}
	for i := 0; i < m.listRows(); i++ {
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

// top is the first command shown: the list scrolls to keep the selection in view.
func (m model) top() int {
	rows := m.listRows()
	return max(0, min(m.cur-rows+1, max(0, len(m.matches())-rows)))
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		d := 1
		if ev.Button == tui.MouseButtonWheelUp {
			d = -1
		}
		if id == "out" {
			m.scroll -= 3 * d
		} else {
			m.cur += d
		}
		return m.clamp(), nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	switch {
	case strings.HasPrefix(id, "row-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "row-"))
		m.cur = m.top() + n
		m = m.clamp()
		if ev.Clicks >= 2 {
			return m.runSelected()
		}
	case id == "cancel":
		return m.stop(), nil
	case id == "rerun":
		return m.rerun()
	}
	return m, nil
}

// ---- view ------------------------------------------------------------------

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) listNode(th theme.Theme) layout.Node {
	w, rows := m.size().W, m.listRows()
	v, top := m.matches(), m.top()
	var kids []layout.FlexChild
	for i := 0; i < rows; i++ {
		line := ""
		switch {
		case top+i < len(v):
			line = m.rowLine(v[top+i], top+i == m.cur, w, th)
		case i == 0 && len(v) == 0:
			line = ansi.NewStyle().Faint().Render("  No command matches.")
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named("row-"+strconv.Itoa(i), layout.Block(pad(ansi.Truncate(line, w), w)))})
	}
	return layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: rows})
}

func (m model) rowLine(e entry, selected bool, w int, th theme.Theme) string {
	c := m.cmds[e.idx]
	recent := "  "
	if m.recency(c.name) < len(m.recents) {
		recent = "↺ " // a mark as well as the order, so recency does not rest on position alone
	}
	nameW := min(24, max(6, w/3))
	name := c.name
	if !selected {
		name = fuzzy.Highlight(c.name, e.pos, func(x string) string { return ansi.NewStyle().Bold().Underline().Render(x) })
	}
	desc := ansi.Truncate(c.desc, max(0, w-nameW-6))
	line := " " + recent + pad(ansi.Truncate(name, nameW), nameW) + " " + ansi.NewStyle().Faint().Render(desc)
	if selected {
		plain := "›" + recent + pad(ansi.Truncate(c.name, nameW), nameW) + " " + desc
		return ansi.NewStyle().Bold().Background(th.Selection).Render(pad(plain, w))
	}
	return line
}

func (m model) outHeader(th theme.Theme) layout.Node {
	w := m.size().W
	if m.last.name == "" {
		return layout.Fixed(layout.Block(ansi.NewStyle().Faint().Render("Output appears here")), layout.Size{W: w, H: 1})
	}
	var state string
	switch {
	case m.last.running:
		state = ansi.NewStyle().Foreground(th.Warning).Render("▶ running")
	case m.last.err != nil || m.last.code != 0:
		state = ansi.NewStyle().Foreground(th.Error).Render("✗ " + describe(m.last.code, m.last.err))
	default:
		state = ansi.NewStyle().Foreground(th.Success).Render("✓ " + describe(m.last.code, m.last.err))
	}
	if !m.last.running {
		state += ansi.NewStyle().Faint().Render(fmt.Sprintf(" in %.1fs", m.last.elapsed.Seconds()))
	}
	btn := func(id, label string, on bool) layout.FlexChild {
		st := ansi.NewStyle().Foreground(th.Muted)
		switch {
		case !on:
			st = ansi.NewStyle().Faint()
		case m.hover == id:
			st = ansi.NewStyle().Bold().Foreground(th.Primary).Underline()
		}
		return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render("["+label+"]")))}
	}
	head := ansi.NewStyle().Bold().Render(m.last.name) + "  " + state
	return layout.Fixed(layout.Row(1,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(head, max(1, w-18))), Grow: 1},
		btn("cancel", "Cancel", m.last.running), btn("rerun", "Rerun", !m.last.running),
	), layout.Size{W: w, H: 1})
}

func (m model) outNode(th theme.Theme) layout.Node {
	w, rows := m.size().W, m.outRows()
	lines := m.last.lines
	end := max(0, len(lines)-m.scroll)
	start := max(0, end-rows)
	var kids []layout.FlexChild
	for i := 0; i < rows; i++ {
		l := ""
		if start+i < end {
			l = strings.ReplaceAll(lines[start+i], "\t", "    ")
		}
		kids = append(kids, block(pad(ansi.Truncate(l, w), w)))
	}
	if m.scroll > 0 {
		note := ansi.NewStyle().Foreground(th.TextInverse).Background(th.Primary).Render(fmt.Sprintf(" ↓ %d newer lines ", m.scroll))
		kids[len(kids)-1] = block(pad(ansi.Truncate(note, w), w))
	}
	return layout.Named("out", layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: rows}))
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.Truncate("  "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 12))+a, m.size().W))
	}
	return layout.Fixed(layout.Column(0,
		key("type", "narrow the commands"), key("Up Down", "choose one"), key("Enter", "run it"),
		key("PgUp PgDn", "scroll the output"), key("Ctrl+X", "cancel the run"), key("Ctrl+R", "run the last one again"),
		key("Esc", "clear the filter"), key("F1", "close help"), key("Ctrl+Q", "quit"),
	), layout.Size{W: m.size().W, H: m.listRows() + 1 + m.outRows()})
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	src := "sample project, scripted"
	if m.file {
		src = "your commands, run for real"
	}
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("launcher") + "  " + ansi.NewStyle().Faint().Render(src)
	hints := widgets.Footer(w, widgets.Hint{Key: "enter", Action: "run"}, widgets.Hint{Key: "ctrl+x", Action: "cancel"},
		widgets.Hint{Key: "ctrl+r", Action: "rerun"}, widgets.Hint{Key: "pgup/pgdn", Action: "scroll"},
		widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "ctrl+q", Action: "quit"})
	in := m.in
	in.Width = max(4, w-4)
	rows := []layout.FlexChild{
		block(ansi.Truncate(title, w)),
		block(ansi.Truncate(in.View(), w)),
	}
	if m.help {
		rows = append(rows, layout.FlexChild{Node: m.helpNode(th)})
	} else {
		rows = append(rows, layout.FlexChild{Node: m.listNode(th)}, layout.FlexChild{Node: m.outHeader(th)}, layout.FlexChild{Node: m.outNode(th)})
	}
	rows = append(rows, block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w))))
	return layout.Column(0, rows...)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	file := flag.String("file", "", "a command file: \"name | description | shell command\" per line; its commands really run")
	flag.Parse()
	m := newModel(sampleCommands(), false, scriptGap)
	if *file != "" {
		b, err := os.ReadFile(*file) // #nosec G304 G703 -- the user names their own command file
		if err == nil {
			var cmds []command
			if cmds, err = parseCommands(string(b)); err == nil {
				m = newModel(cmds, true, 0)
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "launcher: %s: %v\n", *file, err)
			os.Exit(1)
		}
	}
	if _, err := ui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
