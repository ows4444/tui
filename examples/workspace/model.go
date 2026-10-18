package main

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

const (
	chromeRows = 3 // title, status, hints
	minPane    = 3 // rows, tab bar included
	minLeft    = 10
	minRight   = 20
	defaultL   = 24
	defaultB   = 8
)

// drag says which divider the pointer holds.
type drag int

const (
	noDrag drag = iota
	dragV
	dragH
)

type model struct {
	th                     ui.Theming // the theme the Program chose; see SetTheme
	focus                  int
	tab                    [3]int
	off                    [3][3]int // scroll offset per pane and tab
	lw, bh                 int       // wanted width of the explorer and height of the panel
	collapsedL, collapsedB bool
	zoom                   bool
	drag                   drag
	hover                  string
	msg                    string
	help                   bool

	width, height int
}

func newModel() model { return model{focus: 1, lw: defaultL, bh: defaultB} }

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

// geometry is where everything is, worked out from the wanted sizes and the
// window. A zero-sized pane is hidden.
type geometry struct {
	pane       [3]layout.Rect
	vdiv, hdiv layout.Rect // zero when there is no divider
	strip      [2]bool     // the dividers are collapse strips: left, bottom
}

func clamp(v, lo, hi int) int { return max(lo, min(hi, v)) }

func (m model) geo() geometry {
	s := m.size()
	bodyH := max(1, s.H-chromeRows)
	var g geometry
	if m.zoom {
		g.pane[m.focus] = layout.Rect{X: 0, Y: 1, W: s.W, H: bodyH}
		return g
	}
	showL, showB := s.W >= 50, bodyH >= 12
	lw, vd := 0, 0
	if showL {
		vd = 1
		if !m.collapsedL {
			lw = clamp(m.lw, minLeft, s.W-vd-minRight)
		}
		g.strip[0] = m.collapsedL
		g.vdiv = layout.Rect{X: lw, Y: 1, W: 1, H: bodyH}
	}
	x0 := lw + vd
	rw := s.W - x0
	th, bh := bodyH, 0
	if showB {
		bh = 0
		if !m.collapsedB {
			bh = clamp(m.bh, minPane, bodyH-1-minPane)
		}
		th = bodyH - bh - 1
		g.strip[1] = m.collapsedB
		g.hdiv = layout.Rect{X: x0, Y: 1 + th, W: rw, H: 1}
	}
	if lw > 0 {
		g.pane[0] = layout.Rect{X: 0, Y: 1, W: lw, H: bodyH}
	}
	g.pane[1] = layout.Rect{X: x0, Y: 1, W: rw, H: th}
	if bh > 0 {
		g.pane[2] = layout.Rect{X: x0, Y: 2 + th, W: rw, H: bh}
	}
	return g
}

func (g geometry) shown(p int) bool { return g.pane[p].W > 0 && g.pane[p].H > 0 }

func (m model) lines(p int) []string { return content(p, m.tab[p]) }

// visible is the number of text rows of pane p (its tab bar takes one).
func (m model) visible(p int) int { return max(0, m.geo().pane[p].H-1) }

func (m model) maxOff(p int) int { return max(0, len(m.lines(p))-m.visible(p)) }

func (m model) scroll(p, d int) model {
	m.off[p][m.tab[p]] = clamp(m.off[p][m.tab[p]]+d, 0, m.maxOff(p))
	return m
}

func (m model) setTab(p, t int) model {
	n := len(paneTabs[p])
	m.tab[p] = ((t % n) + n) % n
	return m
}

func (m model) setFocus(p int) model {
	if m.geo().shown(p) {
		m.focus = p
	}
	return m
}

// nextFocus moves focus to the next pane that is on screen.
func (m model) nextFocus(d int) model {
	g := m.geo()
	for i := 1; i <= 3; i++ {
		p := ((m.focus+d*i)%3 + 3) % 3
		if g.shown(p) {
			m.focus = p
			return m
		}
	}
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if !m.geo().shown(m.focus) {
			m = m.nextFocus(1)
		}
		for p := range m.off {
			for t := range m.off[p] {
				m.off[p][t] = min(m.off[p][t], m.maxOffFor(p, t))
			}
		}
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) maxOffFor(p, t int) int {
	return max(0, len(content(p, t))-m.visible(p))
}

// collapse toggles the explorer or the panel; the editor cannot collapse.
func (m model) collapse(p int) model {
	switch p {
	case 0:
		m.collapsedL = !m.collapsedL
	case 2:
		m.collapsedB = !m.collapsedB
	default:
		m.msg = "The editor stays; collapse the explorer or the panel"
		return m
	}
	if !m.geo().shown(m.focus) {
		m.focus = 1
	}
	return m
}

func (m model) reset() model {
	m.lw, m.bh, m.collapsedL, m.collapsedB, m.zoom = defaultL, defaultB, false, false, false
	m.msg = "Layout reset"
	return m
}
