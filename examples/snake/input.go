package main

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter && !(k.Type == tui.KeyRunes && (k.Text == "?" || k.Text == "q"))
		return m, nil
	}
	m.msg = ""
	switch k.Type {
	case tui.KeyUp:
		return m.steerTo(up)
	case tui.KeyRight:
		return m.steerTo(right)
	case tui.KeyDown:
		return m.steerTo(down)
	case tui.KeyLeft:
		return m.steerTo(left)
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	return m, nil
}

func (m model) runeKey(r string) (tui.Model, tui.Cmd) {
	switch strings.ToLower(r) {
	case "q":
		return m, tui.Quit()
	case "?":
		m.help = true
		return m.pauseForHelp()
	case "w", "k":
		return m.steerTo(up)
	case "d", "l":
		return m.steerTo(right)
	case "s", "j":
		return m.steerTo(down)
	case "a", "h":
		return m.steerTo(left)
	case "p", " ":
		return m.togglePause()
	case "r":
		return m.restart()
	case "+", "=":
		return m.setLevel(m.level + 1)
	case "-":
		return m.setLevel(m.level - 1)
	}
	return m, nil
}

func (m model) pauseForHelp() (tui.Model, tui.Cmd) {
	if m.g.state == running {
		m.g.state = paused
		m.seq++
	}
	return m, nil
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

var buttons = []string{"pause", "restart", "slower", "faster"}

func (m model) hit(x, y int) string {
	scr := m.screen()
	for _, id := range buttons {
		if r, ok := layout.RectOf(scr, m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	if r, ok := layout.RectOf(scr, m.size(), "arena"); ok && inside(r, x, y) {
		return "arena"
	}
	return ""
}

// cellAt is the arena cell under screen position (x, y).
func (m model) cellAt(x, y int) pt {
	r, _ := layout.RectOf(m.screen(), m.size(), "arena")
	return pt{(x - r.X) / 2, y - r.Y}
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
	if ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	switch id {
	case "arena":
		if m.g.state == paused || m.g.state == over || m.g.state == won {
			return m, nil
		}
		was := m.g.state
		m.g = m.g.steer(m.cellAt(ev.X, ev.Y))
		if was == ready && m.g.state == running {
			return m.start()
		}
	case "pause":
		return m.togglePause()
	case "restart":
		return m.restart()
	case "slower":
		return m.setLevel(m.level - 1)
	case "faster":
		return m.setLevel(m.level + 1)
	}
	return m, nil
}
