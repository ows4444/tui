package main

import (
	"strconv"
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
		if k.Mod.Shift() {
			return m.nextFocus(-1), nil
		}
		return m.nextFocus(1), nil
	case tui.KeyUp:
		return m.scroll(m.focus, -1), nil
	case tui.KeyDown:
		return m.scroll(m.focus, 1), nil
	case tui.KeyPgUp:
		return m.scroll(m.focus, -max(1, m.visible(m.focus)-1)), nil
	case tui.KeyPgDown:
		return m.scroll(m.focus, max(1, m.visible(m.focus)-1)), nil
	case tui.KeyHome:
		return m.scroll(m.focus, -1<<20), nil
	case tui.KeyEnd:
		return m.scroll(m.focus, 1<<20), nil
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
	case ".":
		return m.setTab(m.focus, m.tab[m.focus]+1), nil
	case ",":
		return m.setTab(m.focus, m.tab[m.focus]-1), nil
	case "j":
		return m.scroll(m.focus, 1), nil
	case "k":
		return m.scroll(m.focus, -1), nil
	case "[":
		m.collapsedL = false
		m.lw = max(minLeft, m.lwNow()-2)
	case "]":
		m.collapsedL = false
		m.lw = m.lwNow() + 2
	case "-":
		m.collapsedB = false
		m.bh = m.bhNow() - 1
	case "=":
		m.collapsedB = false
		m.bh = m.bhNow() + 1
	case "c":
		return m.collapse(m.focus), nil
	case "z":
		m.zoom = !m.zoom
	case "0":
		return m.reset(), nil
	}
	return m, nil
}

// lwNow and bhNow are the sizes in use, which clamping may have changed from
// the wanted ones.
func (m model) lwNow() int {
	if w := m.geo().pane[0].W; w > 0 {
		return w
	}
	return m.lw
}

func (m model) bhNow() int {
	if h := m.geo().pane[2].H; h > 0 {
		return h
	}
	return m.bh
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// hit names what is at (x, y): "tab-P-T", "vdiv", "hdiv", "pane-P" or "".
func (m model) hit(x, y int) string {
	scr := m.screen()
	var ids []string
	for p := 0; p < 3; p++ {
		for t := range paneTabs[p] {
			ids = append(ids, "tab-"+strconv.Itoa(p)+"-"+strconv.Itoa(t))
		}
	}
	ids = append(ids, "vdiv", "hdiv", "pane-0", "pane-1", "pane-2")
	for _, id := range ids {
		if r, ok := layout.RectOf(scr, m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionRelease:
		m.drag = noDrag
		return m, nil
	case ev.Action == tui.MouseActionMotion:
		return m.motion(ev, id), nil
	case ev.Button == tui.MouseButtonWheelUp || ev.Button == tui.MouseButtonWheelDown:
		d := 3
		if ev.Button == tui.MouseButtonWheelUp {
			d = -3
		}
		if strings.HasPrefix(id, "pane-") {
			return m.scroll(int(id[5]-'0'), d), nil
		}
		if strings.HasPrefix(id, "tab-") {
			return m.scroll(int(id[4]-'0'), d), nil
		}
		return m, nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	return m.press(ev, id), nil
}

func (m model) press(ev tui.MouseEvent, id string) model {
	m.msg = ""
	switch {
	case id == "vdiv" && ev.Clicks >= 2:
		return m.collapse(0)
	case id == "hdiv" && ev.Clicks >= 2:
		return m.collapse(2)
	case id == "vdiv":
		m.drag = dragV
		if m.collapsedL {
			m.collapsedL = false
		}
	case id == "hdiv":
		m.drag = dragH
		if m.collapsedB {
			m.collapsedB = false
		}
	case strings.HasPrefix(id, "tab-"):
		p, t := int(id[4]-'0'), int(id[6]-'0')
		m = m.setFocus(p)
		m.tab[p] = t
	case strings.HasPrefix(id, "pane-"):
		m = m.setFocus(int(id[5] - '0'))
	}
	return m
}

func (m model) motion(ev tui.MouseEvent, id string) model {
	m.hover = id
	switch m.drag {
	case dragV:
		m.lw = ev.X
	case dragH:
		m.bh = m.size().H - chromeRows - ev.Y
	}
	return m
}
