package main

import (
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

// levels are the time between steps at speed 1 to 5.
var levels = [5]time.Duration{240 * time.Millisecond, 180 * time.Millisecond, 130 * time.Millisecond, 90 * time.Millisecond, 60 * time.Millisecond}

type tickMsg struct{ seq int }

const (
	chromeRows = 5 // title, top border, bottom border, status, hints
	maxW, maxH = 40, 20
	minW, minH = 8, 5
)

type model struct {
	th    ui.Theming // the theme the Program chose; see SetTheme
	g     game
	level int // 0 to 4
	best  int
	seq   int // tells a stale timer from the current one
	seed  uint64
	hover string
	help  bool
	msg   string

	width, height int
}

func newModel() model {
	m := model{level: 1, seed: 0x9e3779b97f4a7c15}
	m.g = newGame(maxW, maxH, m.seed)
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

// arena is the playing field in cells (each cell is two columns wide) that
// fits the terminal.
func (m model) arena() (w, h int) {
	s := m.size()
	return max(minW, min(maxW, (s.W-2)/2)), max(minH, min(maxH, s.H-chromeRows))
}

func (m model) fits() bool {
	w, h := m.arena()
	return 2*w+2 <= m.size().W && h+chromeRows <= m.size().H
}

func (m model) tick() tui.Cmd {
	seq := m.seq
	return tui.FromCtx(motion.After(levels[m.level], func(time.Time) tui.Msg { return tickMsg{seq} }))
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.g.state == ready {
			w, h := m.arena()
			m.g = newGame(w, h, m.seed)
		}
		if m.g.state == running && !m.fits() {
			m.g.state = paused
			m.seq++
		}
	case tickMsg:
		if msg.seq != m.seq || m.g.state != running {
			return m, nil
		}
		m.g = m.g.step(m.level + 1)
		return m.afterStep(), m.cmdIf(m.g.state == running)
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) cmdIf(ok bool) tui.Cmd {
	if !ok {
		return nil
	}
	return m.tick()
}

func (m model) afterStep() model {
	m.best = max(m.best, m.g.score)
	return m
}

// start begins a run from ready or resumes a paused one, with a fresh timer.
func (m model) start() (model, tui.Cmd) {
	m.seq++
	return m, m.tick()
}

func (m model) restart() (model, tui.Cmd) {
	w, h := m.arena()
	m.seed = m.seed*6364136223846793005 + 1442695040888963407
	m.g = newGame(w, h, m.seed)
	m.seq++
	m.msg = ""
	return m, nil
}

func (m model) steerTo(d dir) (model, tui.Cmd) {
	was := m.g.state
	m.g = m.g.turn(d)
	if was == ready {
		return m.start()
	}
	return m, nil
}

func (m model) togglePause() (model, tui.Cmd) {
	switch m.g.state {
	case running:
		m.g.state = paused
		m.seq++
		return m, nil
	case paused:
		if !m.fits() {
			m.msg = "Make the window bigger first"
			return m, nil
		}
		m.g.state = running
		return m.start()
	}
	return m, nil
}

func (m model) setLevel(l int) (model, tui.Cmd) {
	l = max(0, min(len(levels)-1, l))
	if l == m.level {
		return m, nil
	}
	m.level = l
	if m.g.state == running {
		return m.start()
	}
	return m, nil
}
