// Command charts explores a few made-up data sets as line, scatter, bar and
// histogram charts: toggle series on and off, zoom into the x axis, drag to
// pan, and hover to read the values at one place. The charts are drawn on a
// braille canvas, so a cell holds eight dots.
//
// Mouse: hover the chart for a cursor and the values under it, scroll the wheel
// to zoom around the pointer, drag to pan, double-click to reset, click a data
// set, a chart type or a series in the legend. Keyboard: Left and Right move
// the cursor, PgUp and PgDn pan, + and - zoom, 0 resets the zoom, Tab and
// Shift+Tab change the data set, t changes the chart type, 1 to 6 hide or show
// a series, y starts the y axis at zero, [ and ] change the histogram's bins,
// F1 shows help, q or Ctrl+C quits.
//
//	go run ./examples/charts
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

type kind int

const (
	kLine kind = iota
	kScatter
	kBars
	kHist
	numKinds
)

var kindNames = [numKinds]string{"Line", "Scatter", "Bars", "Histogram"}

const chrome = 8 // title, two tab rows, the x axis and its labels, legend, status, hints

type model struct {
	th   ui.Theming // the theme the Program chose; see SetTheme
	sets []dataset
	ds   int
	kind kind
	vis  [][]bool // per data set, which series show
	wins []window // per data set, the x range on screen
	bins int

	cursor int // point under the hover cursor, or -1
	zeroY  bool
	dragX  int // pointer column while dragging, or -1
	hover  string
	help   bool
	msg    string

	width, height int
}

func newModel() model {
	m := model{sets: datasets(), bins: 10, cursor: -1, dragX: -1}
	for _, d := range m.sets {
		v := make([]bool, len(d.series))
		for i := range v {
			v[i] = true
		}
		m.vis = append(m.vis, v)
		m.wins = append(m.wins, full(d.points()))
	}
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) cur() dataset { return m.sets[m.ds] }

func (m model) win() window { return m.wins[m.ds] }

func (m model) setWin(w window) model {
	ws := append([]window(nil), m.wins...)
	ws[m.ds] = w
	m.wins = ws
	return m
}

// ---- geometry --------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) plotH() int { return max(2, m.size().H-chrome) }

// scale is the y range and tick step the current chart is drawn with.
func (m model) scale() (lo, hi, step float64, ok bool) {
	lo, hi, ok = bounds(m.cur().values(), m.vis[m.ds], m.win())
	if !ok {
		return 0, 1, 1, false
	}
	if m.kind == kBars || m.zeroY {
		lo, hi = min(lo, 0), max(hi, 0)
	}
	lo, hi, step = niceRange(lo, hi, max(2, min(5, m.plotH()/2)))
	return lo, hi, step, true
}

// yLabelW is the width of the y axis labels.
func (m model) yLabelW() int {
	lo, hi, step, ok := m.scale()
	if !ok {
		return 1
	}
	w := 1
	for v := lo; v <= hi+step/2; v += step {
		w = max(w, len(label(v, step)))
	}
	return min(w, 9)
}

func (m model) plotX() int { return m.yLabelW() + 1 }

func (m model) plotW() int { return max(8, m.size().W-m.plotX()-1) }

func (m model) plotRect() layout.Rect {
	return layout.Rect{X: m.plotX(), Y: 3, W: m.plotW(), H: m.plotH()}
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
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
	switch k.Type {
	case tui.KeyTab:
		if k.Mod.Shift() {
			return m.setDataset(m.ds - 1), nil
		}
		return m.setDataset(m.ds + 1), nil
	case tui.KeyLeft:
		return m.moveCursor(-1), nil
	case tui.KeyRight:
		return m.moveCursor(1), nil
	case tui.KeyPgUp:
		return m.setWin(m.win().pan(-m.win().span() / 2)), nil
	case tui.KeyPgDown:
		return m.setWin(m.win().pan(m.win().span() / 2)), nil
	case tui.KeyEsc:
		m.cursor = -1
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
	case "t":
		m.kind = (m.kind + 1) % numKinds
	case "+", "=":
		return m.zoomAround(0.7, 0.5), nil
	case "-":
		return m.zoomAround(1/0.7, 0.5), nil
	case "0":
		return m.setWin(full(m.cur().points())), nil
	case "y":
		m.zeroY = !m.zeroY
	case "[":
		m.bins = max(3, m.bins-1)
	case "]":
		m.bins = min(m.plotH(), m.bins+1)
	default:
		if n, err := strconv.Atoi(r); err == nil && n >= 1 && n <= len(m.cur().series) {
			return m.toggle(n - 1), nil
		}
	}
	return m, nil
}

func (m model) setDataset(i int) model {
	n := len(m.sets)
	m.ds, m.cursor = ((i%n)+n)%n, -1
	return m
}

func (m model) toggle(s int) model {
	vis := make([][]bool, len(m.vis))
	copy(vis, m.vis)
	vis[m.ds] = append([]bool(nil), m.vis[m.ds]...)
	vis[m.ds][s] = !vis[m.ds][s]
	any := false
	for _, v := range vis[m.ds] {
		any = any || v
	}
	if !any {
		m.msg = "Keep at least one series showing"
		return m
	}
	m.vis = vis
	return m
}

func (m model) moveCursor(d int) model {
	if m.cursor < 0 {
		m.cursor = max(m.win().first(), min(m.win().last(), int(m.win().lo+m.win().span()/2)))
		return m
	}
	m.cursor = max(0, min(m.cur().points()-1, m.cursor+d))
	w := m.win()
	switch { // scroll to keep the cursor in view
	case float64(m.cursor) < w.lo:
		m = m.setWin(w.pan(float64(m.cursor) - w.lo))
	case float64(m.cursor) > w.hi:
		m = m.setWin(w.pan(float64(m.cursor) - w.hi))
	}
	return m
}

// zoomAround zooms in (factor below 1) or out around fraction f of the plot.
func (m model) zoomAround(factor, f float64) model {
	if m.cursor >= 0 && f == 0.5 {
		w := m.win()
		f = max(0, min(1, (float64(m.cursor)-w.lo)/w.span()))
	}
	return m.setWin(m.win().zoom(factor, f))
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	var ids []string
	for i := range m.sets {
		ids = append(ids, "ds-"+strconv.Itoa(i))
	}
	for i := 0; i < int(numKinds); i++ {
		ids = append(ids, "kind-"+strconv.Itoa(i))
	}
	for i := range m.cur().series {
		ids = append(ids, "leg-"+strconv.Itoa(i))
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

// pointerIndex is the data point nearest the pointer, and the fraction of the
// plot's width it is at.
func (m model) pointerIndex(x int) (int, float64) {
	r := m.plotRect()
	dots := 2 * r.W
	col := max(0, min(dots-1, 2*(x-r.X)))
	w := m.win()
	return w.nearest(w.at(col, dots)), float64(col) / float64(max(1, dots-1))
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	r := m.plotRect()
	in := inside(r, ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionRelease:
		m.dragX = -1
		return m, nil
	case ev.Action == tui.MouseActionMotion:
		return m.motion(ev, in), nil
	case ev.Button == tui.MouseButtonWheelUp && in:
		_, f := m.pointerIndex(ev.X)
		return m.zoomAround(0.8, f), nil
	case ev.Button == tui.MouseButtonWheelDown && in:
		_, f := m.pointerIndex(ev.X)
		return m.zoomAround(1.25, f), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	case in && ev.Clicks >= 2:
		return m.setWin(full(m.cur().points())), nil
	case in:
		m.dragX = ev.X
		m.cursor, _ = m.pointerIndex(ev.X)
		return m, nil
	}
	return m.clickControl(m.hit(ev.X, ev.Y)), nil
}

func (m model) motion(ev tui.MouseEvent, in bool) model {
	if m.dragX >= 0 && in {
		dx := m.dragX - ev.X // dragging left moves the window right
		m.dragX = ev.X
		cells := float64(m.plotW())
		return m.setWin(m.win().pan(float64(dx) / cells * m.win().span()))
	}
	if in {
		m.cursor, _ = m.pointerIndex(ev.X)
		return m
	}
	m.hover = m.hit(ev.X, ev.Y)
	m.cursor = -1
	return m
}

func (m model) clickControl(id string) model {
	var n int
	switch {
	case strings.HasPrefix(id, "ds-"):
		n, _ = strconv.Atoi(strings.TrimPrefix(id, "ds-"))
		return m.setDataset(n)
	case strings.HasPrefix(id, "kind-"):
		n, _ = strconv.Atoi(strings.TrimPrefix(id, "kind-"))
		m.kind = kind(n)
	case strings.HasPrefix(id, "leg-"):
		n, _ = strconv.Atoi(strings.TrimPrefix(id, "leg-"))
		return m.toggle(n)
	}
	return m
}

// ---- drawing ---------------------------------------------------------------

func palette(th theme.Theme) []ansi.Color {
	return []ansi.Color{th.Primary, th.Secondary, th.Success, th.Warning, th.Error, th.Info}
}

func (m model) color(s int, th theme.Theme) ansi.Color {
	p := palette(th)
	return p[s%len(p)]
}

func (m model) visibleSeries() []int {
	var out []int
	for i, v := range m.vis[m.ds] {
		if v {
			out = append(out, i)
		}
	}
	return out
}

// canvasRows draws the data in the current window as the rows of the plot.
func (m model) canvasRows(lo, hi float64, th theme.Theme) []string {
	w, h := m.plotW(), m.plotH()
	cv := chart.NewCanvas(w, h)
	dw, dh := cv.DotSize()
	win := m.win()
	vis := m.visibleSeries()
	data := m.cur().values()
	if m.cursor >= 0 && float64(m.cursor) >= win.lo && float64(m.cursor) <= win.hi {
		x := win.col(float64(m.cursor), dw)
		cv.Line(x, 0, x, dh-1, th.Muted)
	}
	first, last := max(0, win.first()-1), min(win.n-1, win.last()+1)
	for k, s := range vis {
		col := m.color(s, th)
		vals := data[s]
		px, py := 0, 0
		for i := first; i <= last && i < len(vals); i++ {
			x, y := win.col(float64(i), dw), row(vals[i], lo, hi, dh)
			switch m.kind {
			case kLine:
				if i > first {
					cv.Line(px, py, x, y, col)
				}
				cv.Set(x, y, col)
			case kScatter:
				cv.Set(x, y, col)
				cv.Set(x+1, y, col)
				cv.Set(x, y+1, col)
				cv.Set(x+1, y+1, col)
			case kBars:
				m.bar(cv, win, i, k, len(vis), vals[i], lo, hi, col)
			}
			px, py = x, y
		}
		if i := m.cursor; i >= 0 && i < len(vals) && float64(i) >= win.lo && float64(i) <= win.hi {
			cv.Circle(win.col(float64(i), dw), row(vals[i], lo, hi, dh), 2, col)
		}
	}
	return cv.Rows(th)
}

// bar draws the k-th of n bars at point i.
func (m model) bar(cv *chart.Canvas, win window, i, k, n int, v, lo, hi float64, col ansi.Color) {
	dw, dh := cv.DotSize()
	per := max(1, int(float64(dw)/(win.span()+1)*0.8)/n)
	x0 := win.col(float64(i), dw) - per*n/2 + k*per
	base := row(max(lo, min(0, hi)), lo, hi, dh)
	top := row(v, lo, hi, dh)
	for x := x0; x < x0+per; x++ {
		cv.Line(x, base, x, top, col)
	}
}

func (m model) yAxis(lo, hi, step float64, th theme.Theme) []string {
	h, lw := m.plotH(), m.yLabelW()
	ticks := map[int]string{}
	for v := lo; v <= hi+step/2; v += step {
		r := row(v, lo, hi, 4*h) / 4
		if _, taken := ticks[r]; !taken {
			ticks[r] = label(v, step)
		}
	}
	out := make([]string, h)
	for r := range out {
		if t, ok := ticks[r]; ok {
			out[r] = ansi.NewStyle().Faint().Render(fmt.Sprintf("%*s", lw, t)) + "┤"
		} else {
			out[r] = strings.Repeat(" ", lw) + "│"
		}
	}
	return out
}

func (m model) xAxis(th theme.Theme) (string, string) {
	w, lw := m.plotW(), m.yLabelW()
	axis := strings.Repeat(" ", lw) + "└" + strings.Repeat("─", w)
	labels := make([]rune, w)
	for i := range labels {
		labels[i] = ' '
	}
	win, d := m.win(), m.cur()
	last := -100
	for k := 0; k <= 4; k++ {
		col := k * (w - 1) / 4
		text := d.x(win.nearest(win.at(col*2, 2*w)))
		start := col - len([]rune(text))/2
		if k == 4 {
			start = w - len([]rune(text))
		}
		start = max(0, start)
		if start < last+1 || start+len([]rune(text)) > w {
			continue
		}
		copy(labels[start:], []rune(text))
		last = start + len([]rune(text))
	}
	return ansi.NewStyle().Faint().Render(axis), strings.Repeat(" ", lw+1) + ansi.NewStyle().Faint().Render(string(labels))
}

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) plotLines(th theme.Theme) []string {
	h := m.plotH()
	if m.kind == kHist {
		return m.histLines(th)
	}
	lo, hi, step, ok := m.scale()
	if !ok {
		return append([]string{ansi.NewStyle().Faint().Render(" Nothing to show.")}, make([]string, h-1)...)
	}
	axis, rows := m.yAxis(lo, hi, step, th), m.canvasRows(lo, hi, th)
	out := make([]string, h)
	for i := range out {
		r := ""
		if i < len(rows) {
			r = rows[i]
		}
		out[i] = axis[i] + r
	}
	return out
}

func (m model) histLines(th theme.Theme) []string {
	h, w := m.plotH(), m.size().W-2
	vis := m.visibleSeries()
	vals := m.cur().series[vis[0]].values
	win := m.win()
	vals = vals[win.first() : win.last()+1]
	text := strings.TrimRight(chart.Histogram(vals, m.bins, w, th), "\n")
	lines := []string{ansi.NewStyle().Faint().Render(fmt.Sprintf(" %s, %d values, %d bins ([ ] change)", m.cur().series[vis[0]].name, len(vals), m.bins))}
	if text != "" {
		lines = append(lines, strings.Split(text, "\n")...)
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return lines[:h]
}

// ---- view ------------------------------------------------------------------

func (m model) tabs(names []string, cur int, prefix string, th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	for i, n := range names {
		id := prefix + strconv.Itoa(i)
		st := ansi.NewStyle().Foreground(th.Muted)
		switch {
		case i == cur:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(" "+n+" ")))})
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) legend(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	for i, s := range m.cur().series {
		on := m.vis[m.ds][i]
		mark, st := "●", ansi.NewStyle().Foreground(m.color(i, th))
		text := fmt.Sprintf("%d %s", i+1, s.name)
		if !on {
			mark, st, text = "○", ansi.NewStyle().Faint(), text+" (off)"
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named("leg-"+strconv.Itoa(i), layout.Block(st.Render(mark)+" "+text))})
	}
	return layout.Fixed(layout.Row(2, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) status(th theme.Theme) string {
	w := m.size().W
	if m.msg != "" {
		return ansi.Truncate(m.msg, w)
	}
	d := m.cur()
	if m.cursor >= 0 && m.kind != kHist {
		var b strings.Builder
		b.WriteString(ansi.NewStyle().Bold().Render(d.x(m.cursor)) + " ")
		for _, s := range m.visibleSeries() {
			b.WriteString(" " + ansi.NewStyle().Foreground(m.color(s, th)).Render("●") + " " + d.series[s].name + " " + strconv.FormatFloat(d.series[s].values[m.cursor], 'g', 4, 64) + d.unit)
		}
		return ansi.Truncate(b.String(), w)
	}
	zoom := "all " + strconv.Itoa(d.points()) + " points"
	if win := m.win(); !win.isFull() {
		zoom = fmt.Sprintf("points %d–%d of %d", win.first()+1, win.last()+1, d.points())
	}
	return ansi.NewStyle().Faint().Render(ansi.Truncate(zoom+"  ·  hover for values, wheel to zoom, drag to pan", w))
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14)) + a
	}
	return []string{
		key("Left Right", "move the cursor along the chart"), key("PgUp PgDn", "pan when zoomed"), key("+ - 0", "zoom in, out, reset"),
		key("Tab", "next data set"), key("t", "next chart type"), key("1 2 3", "show or hide a series"),
		key("y", "start the y axis at zero"), key("[ ]", "fewer or more histogram bins"), key("wheel drag", "zoom around the pointer, pan"),
		key("F1", "close help"), key("q", "quit"),
	}
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	var names []string
	for _, d := range m.sets {
		names = append(names, d.name)
	}
	lines := m.plotLines(th)
	axis, labels := m.xAxis(th)
	if m.help {
		lines = m.helpLines(th)
		for len(lines) < m.plotH() {
			lines = append(lines, "")
		}
		lines = lines[:m.plotH()]
		axis, labels = "", ""
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "←→", Action: "cursor"}, widgets.Hint{Key: "+/-", Action: "zoom"}, widgets.Hint{Key: "tab", Action: "data"},
		widgets.Hint{Key: "t", Action: "type"}, widgets.Hint{Key: "1-3", Action: "series"}, widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("charts") + ansi.NewStyle().Faint().Render("  "+m.cur().name+m.unit())
	rows := []layout.FlexChild{
		{Node: layout.Block(ansi.Truncate(title, w))},
		{Node: m.tabs(names, m.ds, "ds-", th)},
		{Node: m.tabs(kindNames[:], int(m.kind), "kind-", th)},
	}
	for _, l := range lines {
		rows = append(rows, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, w))})
	}
	rows = append(rows,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(axis, w))}, layout.FlexChild{Node: layout.Block(ansi.Truncate(labels, w))},
		layout.FlexChild{Node: m.legend(th)}, layout.FlexChild{Node: layout.Block(m.status(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))})
	return layout.Column(0, rows...)
}

func (m model) unit() string {
	if u := m.cur().unit; u != "" {
		return " (" + u + ")"
	}
	return ""
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	if _, err := ui.NewProgram(newModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
