package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func itoa(n int) string { return strconv.Itoa(n) }

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
	case tui.KeyLeft:
		return m.move(-1, 0), nil
	case tui.KeyRight:
		return m.move(1, 0), nil
	case tui.KeyUp:
		return m.move(0, -1), nil
	case tui.KeyDown:
		return m.move(0, 1), nil
	case tui.KeyEnter:
		return m.open(m.cx, m.cy)
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
	case "h":
		return m.move(-1, 0), nil
	case "l":
		return m.move(1, 0), nil
	case "k":
		return m.move(0, -1), nil
	case "j":
		return m.move(0, 1), nil
	case " ":
		return m.open(m.cx, m.cy)
	case "f":
		return m.act(m.g.toggleFlag(m.cx, m.cy))
	case "c":
		return m.act(m.g.chord(m.cx, m.cy))
	case "n":
		return m.newGame(m.lv)
	case "1", "2", "3":
		n, _ := strconv.Atoi(r)
		return m.newGame(n - 1)
	}
	return m, nil
}

func (m model) move(dx, dy int) model {
	m.cx = max(0, min(m.g.lv.w-1, m.cx+dx))
	m.cy = max(0, min(m.g.lv.h-1, m.cy+dy))
	return m
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

var buttons = []string{"new", "size-0", "size-1", "size-2"}

// hit names a button under (x, y), or "board".
func (m model) hit(x, y int) string {
	scr := m.screen()
	for _, id := range buttons {
		if r, ok := layout.RectOf(scr, m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	if r, ok := layout.RectOf(scr, m.size(), "board"); ok && inside(r, x, y) {
		return "board"
	}
	return ""
}

// cellAt is the cell under (x, y), or (-1, -1) off the board.
func (m model) cellAt(x, y int) (int, int) {
	r, ok := layout.RectOf(m.screen(), m.size(), "board")
	if !ok || !inside(r, x, y) {
		return -1, -1
	}
	return (x - r.X) / 2, y - r.Y
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	if ev.Action == tui.MouseActionMotion {
		m.hover = m.hit(ev.X, ev.Y)
		m.hx, m.hy = m.cellAt(ev.X, ev.Y)
		return m, nil
	}
	if ev.Action != tui.MouseActionPress {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch {
	case id == "new":
		return m.newGame(m.lv)
	case strings.HasPrefix(id, "size-"):
		return m.newGame(int(id[5] - '0'))
	case id != "board":
		return m, nil
	}
	x, y := m.cellAt(ev.X, ev.Y)
	m.cx, m.cy, m.msg = x, y, ""
	switch {
	case ev.Button == tui.MouseButtonRight:
		return m.act(m.g.toggleFlag(x, y))
	case ev.Button == tui.MouseButtonMiddle:
		return m.act(m.g.chord(x, y))
	case ev.Button == tui.MouseButtonLeft && ev.Clicks >= 2:
		return m.act(m.g.chord(x, y))
	case ev.Button == tui.MouseButtonLeft:
		return m.act(m.g.reveal(x, y))
	}
	return m, nil
}
