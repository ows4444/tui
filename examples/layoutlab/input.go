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
		d := 1
		if k.Mod.Shift() {
			d = int(numModes) - 1
		}
		return m.setMode((m.mode + mode(d)) % numModes), nil
	case tui.KeyUp:
		return m.move(-1), nil
	case tui.KeyDown:
		return m.move(1), nil
	case tui.KeyLeft:
		return m.nudge(-1), nil
	case tui.KeyRight:
		return m.nudge(1), nil
	case tui.KeyPgUp:
		return m.nudge(5), nil
	case tui.KeyPgDown:
		return m.nudge(-5), nil
	case tui.KeyHome:
		return m.reset(m.sel), nil
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
	case "1", "2", "3", "4":
		return m.setMode(mode(r[0] - '1')), nil
	case "j":
		return m.move(1), nil
	case "k":
		return m.move(-1), nil
	case "h":
		return m.nudge(-1), nil
	case "l":
		return m.nudge(1), nil
	case "r":
		return m.resetMode(), nil
	case "c":
		m.code = !m.code
	}
	return m, nil
}

func (m model) move(d int) model {
	m.sel = max(0, min(len(m.params())-1, m.sel+d))
	return m
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	var ids []string
	for i := range m.params() {
		s := strconv.Itoa(i)
		ids = append(ids, "dec-"+s, "inc-"+s, "bar-"+s, "row-"+s)
	}
	for i := 0; i < int(numModes); i++ {
		ids = append(ids, "mode-"+strconv.Itoa(i))
	}
	for i := 0; i < 24; i++ {
		ids = append(ids, "c"+strconv.Itoa(i))
	}
	return append(ids, "view")
}

func (m model) hit(x, y int) string {
	scr := m.screen()
	for _, id := range m.ids() {
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
	kind, n := split(id)
	switch {
	case ev.Action == tui.MouseActionMotion:
		m.hover = id
		if ev.Button == tui.MouseButtonLeft && kind == "bar" {
			return m.dragBar(ev, n), nil
		}
		return m, nil
	case ev.Button == tui.MouseButtonWheelUp && (kind == "bar" || kind == "row"):
		return m.put(n, m.vals[m.params()[n].key]+1), nil
	case ev.Button == tui.MouseButtonWheelDown && (kind == "bar" || kind == "row"):
		return m.put(n, m.vals[m.params()[n].key]-1), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	return m.click(ev, id), nil
}

// split reads "bar-3" as ("bar", 3); an id with no number gives -1.
func split(id string) (string, int) {
	if i := strings.IndexByte(id, '-'); i >= 0 {
		n, err := strconv.Atoi(id[i+1:])
		if err == nil {
			return id[:i], n
		}
	}
	return id, -1
}

func (m model) dragBar(ev tui.MouseEvent, n int) model {
	r, _ := layout.RectOf(m.screen(), m.size(), "bar-"+strconv.Itoa(n))
	p := m.params()[n]
	return m.put(n, p.lo+(ev.X-r.X)*(p.hi-p.lo+1)/max(1, r.W))
}

func (m model) click(ev tui.MouseEvent, id string) model {
	m.msg = ""
	kind, n := split(id)
	switch {
	case kind == "mode":
		return m.setMode(mode(n))
	case kind == "dec":
		return m.put(n, m.vals[m.params()[n].key]-1)
	case kind == "inc":
		return m.put(n, m.vals[m.params()[n].key]+1)
	case kind == "bar":
		return m.dragBar(ev, n)
	case kind == "row":
		m.sel = n
	case id == "view":
		m.code = !m.code
	case strings.HasPrefix(id, "c") && n < 0:
		return m.selectBox(id)
	}
	return m
}

// selectBox points the mode's selector at the clicked box.
func (m model) selectBox(id string) model {
	i, err := strconv.Atoi(id[1:])
	if err != nil {
		return m
	}
	for pi, p := range m.params() {
		switch {
		case m.mode == mFlex && p.key == "child" && i < m.vals["count"]:
			return m.put(pi, i)
		case m.mode == mGrid && p.key == "col":
			return m.put(pi, i%m.vals["cols"])
		}
	}
	return m
}
