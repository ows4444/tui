// Command dashboard is a system monitor: CPU, memory, disks, network and the
// busiest processes in panels that update every second. The numbers are made
// up (a seeded simulation), so it is safe to run anywhere and reads nothing
// from the machine.
//
// Mouse: click a panel to focus it, double-click its title bar to zoom it,
// drag a title bar onto another panel to swap their places, click "paused" or
// the interval in the status line. Keyboard: Tab and the arrow keys move
// between panels, Enter or z zooms the focused one (Esc comes back), < and >
// move it earlier or later, space pauses, + and - change how often it updates,
// F1 shows help, q or Ctrl+C quits. Where there is no room for the grid the
// focused panel fills the screen and Tab moves to the next.
//
//	go run ./examples/dashboard
package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

const (
	chrome   = 3 // title, status, hints
	minPanel = 7 // fewest rows a panel gets before the grid gives way to one panel
	gridAt   = 84
)

var intervals = []time.Duration{500 * time.Millisecond, time.Second, 2 * time.Second, 5 * time.Second}

type tickMsg struct{ seq int }

type model struct {
	th       ui.Theming // the theme the Program chose; see SetTheme
	w        *world
	order    []int // panel ids, in the order they are shown
	focus    int   // index into order
	zoom     bool
	paused   bool
	interval int // index into intervals
	seq      int // tells a stale timer from the current one
	ticks    int

	dragFrom, dragOver int // panel indexes while a title bar is dragged, else -1
	msg                string
	help               bool
	hover              string

	width, height int
}

func newModel() model {
	m := model{w: newWorld(), interval: 1, dragFrom: -1, dragOver: -1}
	for i := 0; i < numPanels; i++ {
		m.order = append(m.order, i)
	}
	return m
}

func (m model) Init() tui.Cmd { return m.tick() }

func (m model) tick() tui.Cmd {
	seq := m.seq
	return tui.FromCtx(motion.After(intervals[m.interval], func(time.Time) tui.Msg { return tickMsg{seq} }))
}

// ---- layout ----------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) bodyH() int { return max(1, m.size().H-chrome) }

// rects are the panels on screen: the grid, or only the focused one when it is
// zoomed or there is no room for the grid.
func (m model) rects() map[int]layout.Rect {
	w, h := m.size().W, m.bodyH()
	n := len(m.order)
	rows := (n + 1) / 2
	if m.zoom || w < gridAt || h/rows < minPanel {
		return map[int]layout.Rect{m.focus: {X: 0, Y: 1, W: w, H: h}}
	}
	out := map[int]layout.Rect{}
	left := w / 2
	rowH := h / rows
	for i := 0; i < n; i++ {
		r, c := i/2, i%2
		y, rh := 1+r*rowH, rowH
		if r == rows-1 {
			rh = h - r*rowH
		}
		x, rw := c*left, left
		if c == 1 {
			rw = w - left
		}
		if i == n-1 && n%2 == 1 {
			x, rw = 0, w // the odd one out spans the row
		}
		out[i] = layout.Rect{X: x, Y: y, W: rw, H: rh}
	}
	return out
}

func (m model) grid() bool { return len(m.rects()) > 1 }

func (m model) focusOn(i int) model {
	n := len(m.order)
	m.focus = ((i % n) + n) % n
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		if msg.seq != m.seq || m.paused {
			return m, nil
		}
		m.w.step()
		m.ticks++
		return m, m.tick()
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
	if k.Type == tui.KeyF1 {
		m.help = !m.help
		return m, nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	}
	m.msg = ""
	cols := 2
	if !m.grid() {
		cols = 1
	}
	switch k.Type {
	case tui.KeyTab:
		if k.Mod.Shift() {
			return m.focusOn(m.focus - 1), nil
		}
		return m.focusOn(m.focus + 1), nil
	case tui.KeyLeft:
		return m.focusOn(m.focus - 1), nil
	case tui.KeyRight:
		return m.focusOn(m.focus + 1), nil
	case tui.KeyUp:
		return m.focusOn(m.focus - cols), nil
	case tui.KeyDown:
		return m.focusOn(m.focus + cols), nil
	case tui.KeyEnter:
		m.zoom = !m.zoom
		return m, nil
	case tui.KeyEsc:
		m.zoom = false
		return m, nil
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) runeKey(r string) (tui.Model, tui.Cmd) {
	switch r {
	case "q":
		return m, tui.Quit()
	case "?":
		m.help = true
	case "z":
		m.zoom = !m.zoom
	case "<":
		return m.move(-1), nil
	case ">":
		return m.move(1), nil
	case " ":
		return m.togglePause()
	case "+", "=":
		return m.setInterval(m.interval - 1)
	case "-":
		return m.setInterval(m.interval + 1)
	}
	return m, nil
}

// move swaps the focused panel with its neighbour and keeps it focused.
func (m model) move(d int) model {
	j := m.focus + d
	if j < 0 || j >= len(m.order) {
		m.msg = "Already at the " + map[bool]string{true: "start", false: "end"}[d < 0]
		return m
	}
	return m.swap(m.focus, j)
}

func (m model) swap(i, j int) model {
	o := append([]int(nil), m.order...)
	o[i], o[j] = o[j], o[i]
	m.order, m.focus = o, j
	return m
}

func (m model) togglePause() (tui.Model, tui.Cmd) {
	m.paused = !m.paused
	if m.paused {
		m.seq++ // the timer in flight is for nobody
		return m, nil
	}
	m.seq++
	return m, m.tick()
}

func (m model) setInterval(i int) (tui.Model, tui.Cmd) {
	if i < 0 || i >= len(intervals) {
		m.msg = map[bool]string{true: "Already as fast as it goes", false: "Already as slow as it goes"}[i < 0]
		return m, nil
	}
	m.interval = i
	m.seq++
	if m.paused {
		return m, nil
	}
	return m, m.tick()
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// panelAt is the index of the panel under x, y, or -1.
func (m model) panelAt(x, y int) int {
	for i, r := range m.rects() {
		if inside(r, x, y) {
			return i
		}
	}
	return -1
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	switch {
	case ev.Action == tui.MouseActionMotion:
		if m.dragFrom >= 0 {
			m.dragOver = m.panelAt(ev.X, ev.Y)
		}
		return m, nil
	case ev.Action == tui.MouseActionRelease:
		return m.drop(), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	for id, f := range map[string]func(model) (tui.Model, tui.Cmd){"pause": model.togglePause, "speed": func(x model) (tui.Model, tui.Cmd) { return x.setInterval((x.interval + 1) % len(intervals)) }} {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, ev.X, ev.Y) {
			return f(m)
		}
	}
	i := m.panelAt(ev.X, ev.Y)
	if i < 0 {
		return m, nil
	}
	m.focus = i
	r := m.rects()[i]
	if ev.Y == r.Y { // the title bar
		if ev.Clicks >= 2 {
			m.zoom = !m.zoom
			return m, nil
		}
		if m.grid() {
			m.dragFrom, m.dragOver = i, i
		}
	}
	return m, nil
}

// drop ends a drag: if it ended over another panel, they swap places.
func (m model) drop() model {
	from, over := m.dragFrom, m.dragOver
	m.dragFrom, m.dragOver = -1, -1
	if from >= 0 && over >= 0 && over != from {
		return m.swap(from, over)
	}
	return m
}

// ---- view ------------------------------------------------------------------

func (m model) panelBox(i int, r layout.Rect, th theme.Theme) []string {
	id := m.order[i]
	edge := ansi.NewStyle().Foreground(th.Muted)
	title := panelNames[id]
	switch {
	case i == m.dragOver && m.dragFrom >= 0 && i != m.dragFrom:
		edge, title = ansi.NewStyle().Bold().Foreground(th.Warning), title+" ⇄"
	case i == m.dragFrom:
		edge = ansi.NewStyle().Faint()
	case i == m.focus:
		edge = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	if i == m.focus && m.zoom {
		title += " (zoomed)"
	}
	body := m.lines(id, r.W-2, r.H-2, th)
	return box(title, ansi.NewStyle().Faint().Render(m.summary(id)), body, r.W, r.H, edge)
}

func (m model) body(th theme.Theme) []string {
	w, h := m.size().W, m.bodyH()
	type placed struct {
		r     layout.Rect
		lines []string
	}
	var ps []placed
	for i, r := range m.rects() {
		ps = append(ps, placed{r, m.panelBox(i, r, th)}) // each panel is drawn once
	}
	sort.Slice(ps, func(a, b int) bool {
		if ps[a].r.Y != ps[b].r.Y {
			return ps[a].r.Y < ps[b].r.Y
		}
		return ps[a].r.X < ps[b].r.X
	})
	canvas := make([]string, h)
	for y := range canvas {
		line := ""
		for _, p := range ps {
			if row := y + 1 - p.r.Y; row >= 0 && row < p.r.H {
				line = pad(line, p.r.X) + p.lines[row]
			}
		}
		canvas[y] = pad(ansi.Truncate(line, w), w)
	}
	return canvas
}

func (m model) statusNode(th theme.Theme) layout.Node {
	w := m.size().W
	state := ansi.NewStyle().Foreground(th.Success).Render("● live")
	if m.paused {
		state = ansi.NewStyle().Foreground(th.Warning).Render("‖ paused")
	}
	st := func(id, s string) layout.FlexChild {
		style := ansi.NewStyle()
		if m.hover == id {
			style = style.Underline()
		}
		return layout.FlexChild{Node: layout.Named(id, layout.Block(style.Render(s)))}
	}
	every := fmt.Sprintf("every %s", intervals[m.interval])
	right := panelNames[m.order[m.focus]]
	if m.msg != "" {
		right = m.msg
	}
	return layout.Fixed(layout.Row(2,
		st("pause", state), st("speed", every),
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(fmt.Sprintf("%d samples", m.ticks+history))), Grow: 1},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(right, max(1, w/3)))},
	), layout.Size{W: w, H: 1})
}

func (m model) helpBody(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14)) + a
	}
	return fit([]string{
		key("Tab arrows", "move between panels"), key("Enter z", "zoom the panel (Esc comes back)"),
		key("< >", "move the panel earlier or later"), key("space", "pause or resume"), key("+ -", "update faster or slower"),
		key("mouse", "click focuses, double-click a title zooms, drag a title onto another panel swaps them"),
		key("F1", "close help"), key("q", "quit"),
	}, m.size().W, m.bodyH())
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	lines := m.body(th)
	if m.help {
		lines = m.helpBody(th)
	}
	var body []layout.FlexChild
	for _, l := range lines {
		body = append(body, layout.FlexChild{Node: layout.Block(l)})
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "tab", Action: "panel"}, widgets.Hint{Key: "enter", Action: "zoom"},
		widgets.Hint{Key: "< >", Action: "move"}, widgets.Hint{Key: "space", Action: "pause"}, widgets.Hint{Key: "+/-", Action: "speed"},
		widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("dashboard") + ansi.NewStyle().Faint().Render("  simulated machine")
	return layout.Column(0,
		append([]layout.FlexChild{{Node: layout.Block(ansi.Truncate(title, w))}},
			append(body,
				layout.FlexChild{Node: m.statusNode(th)},
				layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))})...)...)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	if _, err := ui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
