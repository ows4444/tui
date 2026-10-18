package main

import (
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
)

type tickMsg struct{ seq int }

const chromeRows = 5 // title bar, top border, bottom border, status, hints

type model struct {
	g             game
	lv            int
	cx, cy        int // the keyboard cursor
	secs          int
	seq           int // tells a stale timer from the current one
	seed          uint64
	hover         string
	help          bool
	msg           string
	hx, hy        int // cell under the pointer, or -1
	width, height int
}

func newModel() model {
	m := model{seed: 0x2545f4914f6cdd1d, hx: -1, hy: -1}
	m.g = newGame(levels[0], m.seed)
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) fits() bool {
	return 2*m.g.lv.w+2 <= m.size().W && m.g.lv.h+chromeRows <= m.size().H
}

func (m model) tick() tui.Cmd {
	seq := m.seq
	return tui.FromCtx(motion.After(time.Second, func(time.Time) tui.Msg { return tickMsg{seq} }))
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tickMsg:
		if msg.seq != m.seq || m.g.state != playing {
			return m, nil
		}
		m.secs++
		return m, m.tick()
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// act applies a game move and starts the clock when the first cell opens and
// stops it when the game ends.
func (m model) act(next game) (model, tui.Cmd) {
	was := m.g.state
	m.g = next
	switch {
	case was == fresh && m.g.state == playing:
		m.seq++
		return m, m.tick()
	case m.g.state == won:
		m.msg = "Cleared in " + secs(m.secs)
	case m.g.state == lost:
		m.msg = "Boom. n starts a new game"
	}
	return m, nil
}

func secs(n int) string {
	if n == 1 {
		return "1 second"
	}
	return itoa(n) + " seconds"
}

func (m model) newGame(lv int) (model, tui.Cmd) {
	m.lv = lv
	m.seed = m.seed*6364136223846793005 + 1442695040888963407
	m.g = newGame(levels[lv], m.seed)
	m.secs, m.msg = 0, ""
	m.seq++
	m.cx, m.cy = min(m.cx, levels[lv].w-1), min(m.cy, levels[lv].h-1)
	return m, nil
}

func (m model) open(x, y int) (model, tui.Cmd) {
	if m.g.in(x, y) && m.g.at(x, y).open {
		return m.act(m.g.chord(x, y))
	}
	return m.act(m.g.reveal(x, y))
}
