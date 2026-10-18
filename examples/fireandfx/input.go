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
	case tui.KeyTab:
		d := 1
		if k.Mod.Shift() {
			d = int(numScenes) - 1
		}
		return m.setScene((m.scene + scene(d)) % numScenes), nil
	case tui.KeyLeft:
		m.wind = max(-1, m.wind-1)
	case tui.KeyRight:
		m.wind = min(1, m.wind+1)
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
	case "1", "2", "3":
		return m.setScene(scene(r[0] - '1')), nil
	case " ":
		m.paused = !m.paused
	case "+", "=":
		m.speed = min(4, m.speed+1)
	case "-":
		m.speed = max(1, m.speed-1)
	case "e":
		m.fx = (m.fx + 1) % len(effects)
		m.since = m.frame
	case "i":
		m.fire.on = !m.fire.on
	case "h":
		m.wind = max(-1, m.wind-1)
	case "l":
		m.wind = min(1, m.wind+1)
	case "r":
		return m.restart(), nil
	case "n":
		return m.stepByHand(), nil
	}
	return m, nil
}

// stepByHand advances the scene eight steps; it is how a reduced-motion
// viewer moves things on.
func (m model) stepByHand() model {
	for range 8 {
		m = m.step()
	}
	if m.scene == sType && m.chars < scriptLen() && m.still {
		m.chars = min(scriptLen(), m.chars+20)
	}
	return m
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

var hitIDs = []string{"scene-0", "scene-1", "scene-2", "effect", "body"}

func (m model) hit(x, y int) string {
	scr := m.screen()
	for _, id := range hitIDs {
		if r, ok := layout.RectOf(scr, m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (m model) bodyPoint(x, y int) (int, int) {
	r, _ := layout.RectOf(m.screen(), m.size(), "body")
	return x - r.X, y - r.Y
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
		if id == "body" {
			m.mx, m.my = m.bodyPoint(ev.X, ev.Y)
			if ev.Button == tui.MouseButtonLeft && m.scene == sFire {
				m.fire = m.fire.ignite(m.mx, m.my, 1)
			}
		}
		return m, nil
	}
	if ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress {
		return m, nil
	}
	switch {
	case strings.HasPrefix(id, "scene-"):
		return m.setScene(scene(id[6] - '0')), nil
	case id == "effect":
		m.fx = (m.fx + 1) % len(effects)
		m.since = m.frame
	case id == "body":
		return m.clickBody(ev), nil
	}
	return m, nil
}

func (m model) clickBody(ev tui.MouseEvent) model {
	x, y := m.bodyPoint(ev.X, ev.Y)
	switch m.scene {
	case sFire:
		m.fire = m.fire.ignite(x, y, 2)
	case sType:
		if m.chars >= scriptLen() {
			return m.restart()
		}
		m.chars = scriptLen()
	}
	return m
}
