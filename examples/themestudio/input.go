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
	if m.typing {
		return m.typeKey(k), nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter && !(k.Type == tui.KeyRunes && (k.Text == "?" || k.Text == "q"))
		return m, nil
	}
	m.msg = ""
	switch k.Type {
	case tui.KeyUp:
		return m.pick(-1), nil
	case tui.KeyDown:
		return m.pick(1), nil
	case tui.KeyTab:
		d := 1
		if k.Mod.Shift() {
			d = 3
		}
		m.focus = (m.focus + d) % 4
	case tui.KeyLeft:
		return m.nudge(-1), nil
	case tui.KeyRight:
		return m.nudge(1), nil
	case tui.KeyPgUp:
		return m.nudge(16), nil
	case tui.KeyPgDown:
		return m.nudge(-16), nil
	case tui.KeyEnter:
		return m.startTyping(), nil
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
	case "j":
		return m.pick(1), nil
	case "k":
		return m.pick(-1), nil
	case "h":
		return m.nudge(-1), nil
	case "l":
		return m.nudge(1), nil
	case "[", "p":
		return m.setBase(m.baseIndex() - 1), nil
	case "]", "n":
		return m.setBase(m.baseIndex() + 1), nil
	case "x":
		return m.resetToken(), nil
	case "0":
		return m.resetAll(), nil
	case "b":
		return m.cycleBorder(), nil
	case "g":
		return m.cycleGlyphs(), nil
	case "s":
		return m.save(), nil
	}
	return m, nil
}

func (m model) pick(d int) model {
	m.sel = max(0, min(len(tokens)-1, m.sel+d))
	return m
}

// nudge changes the focused slider by d; on the hex code it starts typing.
func (m model) nudge(d int) model {
	if m.focus == fHex {
		return m.startTyping()
	}
	return m.channel(m.focus, d)
}

func (m model) startTyping() model {
	m.typing, m.focus, m.edit = true, fHex, []rune(hex(m.current()))
	return m
}

func (m model) typeKey(k tui.Key) model {
	switch k.Type {
	case tui.KeyEsc:
		m.typing = false
	case tui.KeyEnter:
		return m.commitHex()
	case tui.KeyBackspace:
		if len(m.edit) > 0 {
			m.edit = m.edit[:len(m.edit)-1]
		}
	case tui.KeyRunes:
		for _, r := range k.Text {
			if len(m.edit) < 7 && (r == '#' || r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
				m.edit = append(append([]rune(nil), m.edit...), r)
			}
		}
	}
	return m
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"prev", "next", "hex", "btn-reset", "btn-all", "btn-border", "btn-glyphs", "btn-save", "bar-0", "bar-1", "bar-2"}
	for i := range tokens {
		ids = append(ids, "tok-"+strconv.Itoa(i))
	}
	return ids
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
	if m.help || m.typing {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion:
		m.hover = id
		if ev.Button == tui.MouseButtonLeft && strings.HasPrefix(id, "bar-") {
			return m.dragBar(ev, id), nil
		}
		return m, nil
	case ev.Button == tui.MouseButtonWheelUp && strings.HasPrefix(id, "bar-"):
		return m.channel(int(id[4]-'0'), 4), nil
	case ev.Button == tui.MouseButtonWheelDown && strings.HasPrefix(id, "bar-"):
		return m.channel(int(id[4]-'0'), -4), nil
	case ev.Button == tui.MouseButtonWheelUp:
		return m.pick(-1), nil
	case ev.Button == tui.MouseButtonWheelDown:
		return m.pick(1), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	return m.click(ev, id), nil
}

func (m model) dragBar(ev tui.MouseEvent, id string) model {
	c := int(id[4] - '0')
	r, _ := layout.RectOf(m.screen(), m.size(), id)
	m.focus = c
	return m.setChannel(c, (ev.X-r.X)*255/max(1, r.W-1))
}

func (m model) click(ev tui.MouseEvent, id string) model {
	m.msg = ""
	switch {
	case id == "prev":
		return m.setBase(m.baseIndex() - 1)
	case id == "next":
		return m.setBase(m.baseIndex() + 1)
	case id == "hex":
		return m.startTyping()
	case id == "btn-reset":
		return m.resetToken()
	case id == "btn-all":
		return m.resetAll()
	case id == "btn-border":
		return m.cycleBorder()
	case id == "btn-glyphs":
		return m.cycleGlyphs()
	case id == "btn-save":
		return m.save()
	case strings.HasPrefix(id, "bar-"):
		return m.dragBar(ev, id)
	case strings.HasPrefix(id, "tok-"):
		m.sel, _ = strconv.Atoi(id[4:])
	}
	return m
}
