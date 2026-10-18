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
	step := 1
	if k.Mod.Shift() {
		step = 5
	}
	switch k.Type {
	case tui.KeyLeft:
		return m.moveCursor(-step, 0), nil
	case tui.KeyRight:
		return m.moveCursor(step, 0), nil
	case tui.KeyUp:
		return m.moveCursor(0, -step), nil
	case tui.KeyDown:
		return m.moveCursor(0, step), nil
	case tui.KeyEnter:
		return m.space(), nil
	case tui.KeyEsc:
		m.anchor, m.last, m.erasing = nil, nil, false
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
	case " ":
		return m.space(), nil
	case "u":
		return m.undoStroke(), nil
	case "U":
		return m.redoStroke(), nil
	case "c":
		return m.clear(), nil
	case "s":
		return m.save(), nil
	case "[":
		m.brush = max(1, m.brush-1)
	case "]":
		m.brush = min(3, m.brush+1)
	default:
		return m.pick(r), nil
	}
	return m, nil
}

// pick handles the tool letters and the colour digits.
func (m model) pick(r string) model {
	for t, k := range toolKeys {
		if r == k {
			m.anchor, m.last = nil, nil
			m.tool = tool(t)
			return m
		}
	}
	if n, err := strconv.Atoi(r); err == nil && n >= 1 && n <= numColours {
		m.colour = uint8(max(1, min(numColours, n)))
	}
	return m
}

func (m model) moveCursor(dx, dy int) model {
	m.kbd = true
	m.cur = pt{max(0, min(m.g.w-1, m.cur.x+dx)), max(0, min(m.g.h-1, m.cur.y+dy))}
	if m.last != nil {
		m = m.drag(m.cur)
	}
	return m
}

// space is the keyboard's mouse button: it starts a stroke or shape and, on the
// next press, ends it.
func (m model) space() model {
	m.kbd = true
	switch {
	case m.last != nil || m.anchor != nil:
		return m.release(m.cur)
	case m.tool == fill:
		return m.press(m.cur, false).release(m.cur)
	}
	return m.press(m.cur, false)
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// hit names a tool button "tool-N", a swatch "col-N", the "canvas", or "".
func (m model) hit(x, y int) string {
	scr := m.screen()
	var ids []string
	for i := 0; i < int(numTools); i++ {
		ids = append(ids, "tool-"+strconv.Itoa(i))
	}
	for i := 1; i <= numColours; i++ {
		ids = append(ids, "col-"+strconv.Itoa(i))
	}
	ids = append(ids, "canvas")
	for _, id := range ids {
		if r, ok := layout.RectOf(scr, m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

// pixel is the canvas pixel under (x, y), clamped to the canvas.
func (m model) pixel(x, y int) pt {
	r, _ := layout.RectOf(m.screen(), m.size(), "canvas")
	return pt{max(0, min(m.g.w-1, (x-r.X)/2)), max(0, min(m.g.h-1, y-r.Y))}
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion:
		return m.motion(ev, id), nil
	case ev.Action == tui.MouseActionRelease:
		if m.last != nil || m.anchor != nil {
			return m.release(m.pixel(ev.X, ev.Y)), nil
		}
		return m, nil
	case ev.Button == tui.MouseButtonWheelUp && id == "canvas":
		m.brush = min(3, m.brush+1)
		return m, nil
	case ev.Button == tui.MouseButtonWheelDown && id == "canvas":
		m.brush = max(1, m.brush-1)
		return m, nil
	case ev.Action != tui.MouseActionPress:
		return m, nil
	}
	return m.click(ev, id), nil
}

func (m model) click(ev tui.MouseEvent, id string) model {
	m.kbd = false
	switch {
	case strings.HasPrefix(id, "tool-"):
		n, _ := strconv.Atoi(id[5:])
		m.tool = tool(n)
	case strings.HasPrefix(id, "col-"):
		if c := id[4] - '0'; c >= 1 && c <= numColours { // "col-N", N one digit
			m.colour = c
		}
		if m.tool == eraser {
			m.tool = pen
		}
	case id == "canvas" && ev.Button == tui.MouseButtonLeft:
		return m.press(m.pixel(ev.X, ev.Y), false)
	case id == "canvas" && ev.Button == tui.MouseButtonRight:
		return m.press(m.pixel(ev.X, ev.Y), true)
	}
	return m
}

func (m model) motion(ev tui.MouseEvent, id string) model {
	m.hover, m.kbd = id, false
	if id == "canvas" {
		p := m.pixel(ev.X, ev.Y)
		m.hx, m.hy = p.x, p.y
	} else {
		m.hx, m.hy = -1, -1
	}
	if m.last != nil || m.anchor != nil {
		p := m.pixel(ev.X, ev.Y)
		m.cur = p
		return m.drag(p)
	}
	return m
}
