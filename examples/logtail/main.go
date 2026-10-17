// Command logtail follows a log: a file that is still being written, a pipe,
// or (with neither) a made-up service. Mixed lines are read apart, whether
// JSON or text. Hide levels, filter with a regular expression, pause the
// stream while you read, and open any line to see all its fields.
//
// Mouse: click a line to select it, click it again to open or close it, scroll
// the wheel, click a level, Pause, Follow or the filter in the bar above the
// log. Keyboard: Up, Down, PgUp, PgDn move, g and G go to the first and last
// line (G also follows the end), Enter opens or closes a line, E closes them
// all; 1 to 5 hide or show TRACE, DEBUG, INFO, WARN and ERROR; / filters with a
// regular expression (Esc clears it, i switches case sensitivity); space or p
// pauses and resumes; f follows the end; y copies the line; C clears the log;
// F1 shows help; q or Ctrl+C quits.
//
//	go run ./examples/logtail                  # a made-up service
//	go run ./examples/logtail app.log          # follows a file
//	some-command 2>&1 | go run ./examples/logtail
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	maxEntries = 50000
	maxPending = 100000
	chrome     = 4 // title, bar, status, hints
	genEvery   = 150 * time.Millisecond
)

type genMsg struct{ seq int }

type model struct {
	ents    []entry
	vis     []int // indexes into ents of the entries that pass the filters
	nextID  int
	counts  [numLevels]int
	dropped int

	cur, top int
	follow   bool
	open     map[int]bool // entry ids that are opened

	levels [numLevels]bool
	re     *regexp.Regexp
	query  string
	nocase bool
	typing bool
	in     textinput.Model

	paused  bool
	pending []string
	lost    int // held lines that did not fit

	name  string
	eof   bool
	err   error
	gen   *generator
	seq   int
	start tui.Cmd // the first read, run by Init
	stop  context.CancelFunc
	msg   string
	help  bool
	hover string

	width, height int
}

func newModel(name string) model {
	in := textinput.New()
	in.Prompt = "Filter (regex) ▸ "
	in.Width = 30
	m := model{name: name, follow: true, nocase: true, open: map[int]bool{}, in: in}
	for i := range m.levels {
		m.levels[i] = true
	}
	return m
}

// fromReader reads r; follow keeps waiting for more at the end, as for a file.
func (m model) fromReader(r io.Reader, follow bool) model {
	m.start, m.stop = start(r, follow)
	return m
}

// fromGenerator makes up the log of an imaginary service.
func (m model) fromGenerator(base time.Time) model {
	m.gen = newGenerator(base)
	return m.ingest(m.gen.lines(40)) // some history, so the screen starts full
}

func (m model) Init() tui.Cmd {
	if m.gen != nil {
		return m.genTick()
	}
	return m.start
}

func (m model) genTick() tui.Cmd {
	seq := m.seq
	return tui.FromCtx(motion.After(genEvery, func(time.Time) tui.Msg { return genMsg{seq} }))
}

// ---- entries ---------------------------------------------------------------

func (m model) pass(e entry) bool {
	return m.levels[e.lvl] && (m.re == nil || m.re.MatchString(e.raw))
}

// selectedID is the id of the selected entry, or -1.
func (m model) selectedID() int {
	if m.cur >= 0 && m.cur < len(m.vis) {
		return m.ents[m.vis[m.cur]].id
	}
	return -1
}

// refilter rebuilds the list of shown entries, keeping the selected one.
func (m model) refilter() model { return m.refilterKeeping(m.selectedID()) }

// refilterKeeping rebuilds the list and puts the cursor on entry id keep, or the
// first shown entry after it.
func (m model) refilterKeeping(keep int) model {
	m.vis = m.vis[:0:0]
	m.cur = -1
	for i, e := range m.ents {
		if m.pass(e) {
			m.vis = append(m.vis, i)
			if m.cur < 0 && keep >= 0 && e.id >= keep {
				m.cur = len(m.vis) - 1
			}
		}
	}
	if m.cur < 0 {
		m.cur = len(m.vis) - 1
	}
	return m.keep()
}

// ingest adds lines to the log.
func (m model) ingest(lines []string) model {
	ents := m.ents
	for _, l := range lines {
		e := parseLine(m.nextID, l)
		m.nextID++
		ents = append(ents, e)
		m.counts[e.lvl]++
		if m.pass(e) {
			m.vis = append(m.vis, len(ents)-1)
		}
	}
	m.ents = ents
	if len(m.ents) > maxEntries {
		m = m.trim()
	}
	return m.keep()
}

// trim drops the oldest tenth of the log.
func (m model) trim() model {
	keep := m.selectedID() // before the entries the indexes point at move
	drop := len(m.ents) / 10
	for _, e := range m.ents[:drop] {
		m.counts[e.lvl]--
		delete(m.open, e.id)
	}
	m.ents = append([]entry(nil), m.ents[drop:]...)
	m.dropped += drop
	return m.refilterKeeping(keep)
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 28}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) bodyH() int { return max(1, m.size().H-chrome) }

// height is how many screen lines vis[i] takes.
func (m model) lineCount(i int) int {
	e := m.ents[m.vis[i]]
	if !m.open[e.id] {
		return 1
	}
	return 1 + len(e.detail(m.size().W-6))
}

// keep clamps the cursor and scrolls so it is on screen.
func (m model) keep() model {
	if len(m.vis) == 0 {
		m.cur, m.top = 0, 0
		return m
	}
	if m.follow {
		m.cur = len(m.vis) - 1
	}
	m.cur = max(0, min(m.cur, len(m.vis)-1))
	m.top = max(0, min(m.top, m.cur))
	// The lowest top that still shows the cursor's whole line: walk back from
	// the cursor until the window is full. This costs a window, not the log.
	used, lowest := 0, m.cur
	for i := m.cur; i >= 0; i-- {
		used += m.lineCount(i)
		if used > m.bodyH() && i < m.cur {
			break
		}
		lowest = i
	}
	m.top = max(m.top, lowest)
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.keep(), nil
	case linesMsg:
		return m.receive(msg.lines), next(msg.ch)
	case eofMsg:
		m.eof, m.err = true, msg.err
		return m, nil
	case genMsg:
		if msg.seq != m.seq || m.gen == nil {
			return m, nil
		}
		return m.receive(m.gen.lines(1 + m.gen.n%3)), m.genTick()
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// receive takes new lines: into the log, or aside while paused.
func (m model) receive(lines []string) model {
	if !m.paused {
		return m.ingest(lines)
	}
	room := maxPending - len(m.pending)
	if len(lines) > room {
		m.lost += len(lines) - max(0, room)
		lines = lines[:max(0, room)]
	}
	m.pending = append(append([]string(nil), m.pending...), lines...)
	return m
}

func (m model) setPaused(on bool) model {
	m.paused = on
	if !on {
		held := m.pending
		m.pending, m.lost = nil, 0
		return m.ingest(held)
	}
	return m
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m.quit()
	}
	if m.typing {
		return m.filterKey(k)
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
	switch k.Type {
	case tui.KeyEnter:
		return m.toggleOpen(m.cur), nil
	case tui.KeyEsc:
		return m.setFilter(""), nil
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) quit() (tui.Model, tui.Cmd) {
	if m.stop != nil {
		m.stop()
	}
	return m, tui.Quit()
}

func (m model) moveKey(k tui.Key) (model, bool) {
	page := max(1, m.bodyH()-1)
	from := m.cur
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
		m.cur = len(m.vis) - 1
	case tui.KeyRunes:
		switch k.Text {
		case "k":
			m.cur--
		case "j":
			m.cur++
		case "g":
			m.cur = 0
		case "G":
			m.cur, m.follow = len(m.vis)-1, true
			return m, true
		default:
			return m, false
		}
	default:
		return m, false
	}
	m.cur = max(0, min(m.cur, len(m.vis)-1))
	m.follow = m.cur == len(m.vis)-1 && from <= m.cur && m.follow
	if m.cur == len(m.vis)-1 && !m.follow && k.Type != tui.KeyUp {
		m.follow = true // moving down onto the last line goes back to following
	}
	return m, true
}

func (m model) runeKey(r string) (tui.Model, tui.Cmd) {
	switch r {
	case "q":
		return m.quit()
	case "?":
		m.help = true
	case " ", "p":
		return m.setPaused(!m.paused), nil
	case "f":
		m.follow = !m.follow
		return m.keep(), nil
	case "/":
		m.typing = true
		m.in.SetValue(m.query)
		m.in.CursorEnd()
		return m, m.in.Focus()
	case "i":
		m.nocase = !m.nocase
		return m.setFilter(m.query), nil
	case "E":
		m.open = map[int]bool{}
		return m.keep(), nil
	case "C":
		return m.clear(), nil
	case "y":
		return m.copyLine()
	default:
		if n, err := strconv.Atoi(r); err == nil && n >= 1 && n <= int(numLevels) {
			return m.toggleLevel(level(n - 1)), nil
		}
	}
	return m, nil
}

func (m model) toggleLevel(l level) model {
	m.levels[l] = !m.levels[l]
	return m.refilter()
}

func (m model) toggleOpen(i int) model {
	if i < 0 || i >= len(m.vis) {
		return m
	}
	id := m.ents[m.vis[i]].id
	o := make(map[int]bool, len(m.open)+1)
	for k, v := range m.open {
		o[k] = v
	}
	o[id] = !o[id]
	m.open = o
	return m.keep()
}

func (m model) clear() model {
	m.ents, m.vis, m.counts, m.open = nil, nil, [numLevels]int{}, map[int]bool{}
	m.cur, m.top, m.msg = 0, 0, "Log cleared"
	return m
}

func (m model) copyLine() (tui.Model, tui.Cmd) {
	if m.cur < 0 || m.cur >= len(m.vis) {
		m.msg = "No line to copy"
		return m, nil
	}
	m.msg = "Copied the line (if your terminal allows it)"
	return m, tui.WriteClipboard(m.ents[m.vis[m.cur]].raw)
}

// setFilter compiles q as the filter ("" clears it).
func (m model) setFilter(q string) model {
	if q == "" {
		m.re, m.query = nil, ""
		return m.refilter()
	}
	expr := q
	if m.nocase {
		expr = "(?i)" + q
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		m.msg = "Not a valid regular expression: " + strings.TrimPrefix(err.Error(), "error parsing regexp: ")
		return m
	}
	m.re, m.query = re, q
	return m.refilter()
}

func (m model) filterKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.typing = false
		m.in.Blur()
		return m, nil
	case tui.KeyEnter:
		m.typing = false
		m.in.Blur()
		return m.setFilter(strings.TrimSpace(m.in.Value())), nil
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"pause", "follow", "filter", "log"}
	for i := 0; i < int(numLevels); i++ {
		ids = append(ids, "lvl-"+strconv.Itoa(i))
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
		m.cur = max(0, m.cur-3)
		m.follow = false
		return m.keep(), nil
	case tui.MouseButtonWheelDown:
		m.cur = min(len(m.vis)-1, m.cur+3)
		m.follow = m.cur == len(m.vis)-1
		return m.keep(), nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	switch {
	case strings.HasPrefix(id, "lvl-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(id, "lvl-"))
		return m.toggleLevel(level(n)), nil
	case id == "pause":
		return m.setPaused(!m.paused), nil
	case id == "follow":
		m.follow = !m.follow
		return m.keep(), nil
	case id == "filter":
		return m.runeKey("/")
	case id == "log":
		return m.clickLine(ev.Y), nil
	}
	return m, nil
}

// clickLine selects the entry drawn at screen row y; a click on the selected
// one opens or closes it.
func (m model) clickLine(y int) model {
	r, ok := layout.RectOf(m.screen(), m.size(), "log")
	if !ok {
		return m
	}
	line := y - r.Y
	for i := m.top; i < len(m.vis); i++ {
		h := m.lineCount(i)
		if line < h {
			if i == m.cur {
				return m.toggleOpen(i)
			}
			m.cur = i
			m.follow = i == len(m.vis)-1
			return m.keep()
		}
		line -= h
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

func (m model) levelStyle(l level, th theme.Theme) ansi.Style {
	switch l {
	case lError:
		return ansi.NewStyle().Bold().Foreground(th.Error)
	case lWarn:
		return ansi.NewStyle().Foreground(th.Warning)
	case lDebug, lTrace:
		return ansi.NewStyle().Faint()
	}
	return ansi.NewStyle().Foreground(th.Info)
}

// mark underlines every match of the filter in s.
func (m model) mark(s string, st ansi.Style) string {
	if m.re == nil {
		return st.Render(s)
	}
	var b strings.Builder
	last := 0
	for _, ix := range m.re.FindAllStringIndex(s, -1) {
		if ix[1] == ix[0] {
			continue
		}
		b.WriteString(st.Render(s[last:ix[0]]) + st.Bold().Underline().Reverse().Render(s[ix[0]:ix[1]]))
		last = ix[1]
	}
	return b.String() + st.Render(s[last:])
}

func (m model) lines(th theme.Theme) []string {
	w, h := m.size().W, m.bodyH()
	var out []string
	for i := m.top; i < len(m.vis) && len(out) < h; i++ {
		e := m.ents[m.vis[i]]
		cursor, fold := " ", " "
		if i == m.cur {
			cursor = "›"
		}
		if m.open[e.id] {
			fold = "▾"
		}
		badge := "  ·  "
		if e.hasLvl {
			badge = fmt.Sprintf("%-5s", levelNames[e.lvl])
		}
		head := cursor + fold + ansi.NewStyle().Faint().Render(fmt.Sprintf("%-12s", e.ts)) + " " + m.levelStyle(e.lvl, th).Render(badge) + " "
		body := m.mark(ansi.Truncate(e.summary(), max(1, w-ansi.Width(head))), ansi.NewStyle())
		line := ansi.Truncate(head+body, w)
		if i == m.cur {
			line = ansi.NewStyle().Background(th.Selection).Render(pad(ansi.StripANSI(line), w))
			line = m.selectedLine(e, cursor+fold, w, th)
		}
		out = append(out, line)
		if m.open[e.id] {
			for _, d := range e.detail(w - 6) {
				if len(out) >= h {
					break
				}
				out = append(out, ansi.NewStyle().Faint().Render("    "+ansi.Truncate(d, max(1, w-4))))
			}
		}
	}
	return out
}

// selectedLine is the cursor's line on the selection colour, plain so the
// highlight reads against it.
func (m model) selectedLine(e entry, mark string, w int, th theme.Theme) string {
	badge := "  ·  "
	if e.hasLvl {
		badge = fmt.Sprintf("%-5s", levelNames[e.lvl])
	}
	line := mark + fmt.Sprintf("%-12s", e.ts) + " " + badge + " " + e.summary()
	return ansi.NewStyle().Bold().Background(th.Selection).Render(pad(ansi.Truncate(line, w), w))
}

func (m model) barNode(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	chip := func(id, text string, st ansi.Style) {
		if m.hover == id {
			st = st.Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(text)))})
	}
	for l := lTrace; l < numLevels; l++ {
		box := "☑"
		st := m.levelStyle(l, th)
		if !m.levels[l] {
			box, st = "☐", ansi.NewStyle().Faint()
		}
		chip("lvl-"+strconv.Itoa(int(l)), fmt.Sprintf("%s%d %s %d", box, l+1, levelNames[l], m.counts[l]), st)
	}
	f := "/ filter"
	if m.query != "" {
		f = "/" + m.query
	}
	chip("filter", "["+f+"]", ansi.NewStyle().Foreground(th.Warning))
	p := "[‖ pause]"
	if m.paused {
		p = fmt.Sprintf("[▶ resume %d held]", len(m.pending))
	}
	chip("pause", p, ansi.NewStyle().Foreground(th.Primary))
	fl := "[☐ follow]"
	if m.follow {
		fl = "[☑ follow]"
	}
	chip("follow", fl, ansi.NewStyle().Foreground(th.Primary))
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) statusLine(th theme.Theme) string {
	w := m.size().W
	if m.typing {
		return ansi.Truncate(m.in.View(), w)
	}
	if m.msg != "" {
		return ansi.Truncate(m.msg, w)
	}
	s := fmt.Sprintf("%s · %d of %d lines", m.name, len(m.vis), len(m.ents))
	if m.dropped > 0 {
		s += fmt.Sprintf(" · %d older dropped", m.dropped)
	}
	if m.paused {
		s += " · PAUSED"
		if m.lost > 0 {
			s += fmt.Sprintf(" · %d lost", m.lost)
		}
	}
	switch {
	case m.err != nil:
		s += " · ✗ " + m.err.Error()
	case m.eof:
		s += " · end of input"
	}
	return ansi.NewStyle().Faint().Render(ansi.Truncate(s, w))
}

func (m model) helpNode(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return layout.FlexChild{Node: layout.Block(ansi.Truncate(" "+ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14))+a, m.size().W))}
	}
	return layout.Fixed(layout.Column(0,
		key("j k ↑ ↓", "move through the lines"), key("Enter", "open or close a line (E closes all)"),
		key("1-5", "show or hide TRACE DEBUG INFO WARN ERROR"), key("/", "filter with a regular expression"),
		key("i", "case sensitive or not"), key("space p", "pause and resume (new lines wait)"),
		key("f G", "follow the end"), key("g", "first line"), key("y", "copy the line"),
		key("C", "clear the log"), key("F1", "close help"), key("q", "quit"),
	), layout.Size{W: m.size().W, H: m.bodyH()})
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	var body layout.Node
	if m.help {
		body = m.helpNode(th)
	} else {
		var kids []layout.FlexChild
		lines := m.lines(th)
		if len(m.vis) == 0 {
			msg := " Waiting for log lines…"
			if len(m.ents) > 0 {
				msg = " Every line is hidden by the levels or the filter."
			}
			lines = []string{ansi.NewStyle().Faint().Render(msg)}
		}
		for i := 0; i < m.bodyH(); i++ {
			l := ""
			if i < len(lines) {
				l = lines[i]
			}
			kids = append(kids, layout.FlexChild{Node: layout.Block(pad(ansi.Truncate(l, w), w))})
		}
		body = layout.Named("log", layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: m.bodyH()}))
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "enter", Action: "open"}, widgets.Hint{Key: "1-5", Action: "levels"},
		widgets.Hint{Key: "/", Action: "filter"}, widgets.Hint{Key: "space", Action: "pause"}, widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("logtail")
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(title)},
		layout.FlexChild{Node: m.barNode(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(m.statusLine(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))},
	)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	m := newModel("sample service")
	opts := []tui.ProgramOption{tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)}
	switch {
	case len(os.Args) > 1 && os.Args[1] != "-":
		f, err := os.Open(os.Args[1]) // #nosec G304 G703 -- a log viewer opens the file its user names
		if err != nil {
			fmt.Fprintln(os.Stderr, "logtail:", err)
			os.Exit(1)
		}
		defer f.Close()
		m.name = os.Args[1]
		m = m.fromReader(f, true)
	case len(os.Args) > 1 || stdinIsPipe():
		m.name = "stdin"
		m = m.fromReader(os.Stdin, false)
		if tty, err := os.Open("/dev/tty"); err == nil {
			defer tty.Close() // the keys come from the terminal, not from the pipe
			opts = append(opts, tui.WithInput(tty))
		}
	default:
		m = m.fromGenerator(time.Date(2026, 10, 18, 9, 30, 0, 0, time.UTC))
	}
	if _, err := tui.NewProgram(m, opts...).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func stdinIsPipe() bool {
	info, err := os.Stdin.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice == 0
}
