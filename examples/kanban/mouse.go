package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// hit names what is under (x, y), read from the same tree View draws: a card
// "card-C-R", a column foot "add-C", a column head "head-C", or "".
func (m model) hit(x, y int) string {
	scr := m.screen()
	for c := 0; c < 3; c++ {
		ids := []string{"add-" + strconv.Itoa(c), "head-" + strconv.Itoa(c)}
		for r := range m.b.cols[c] {
			ids = append(ids, cardID(c, r))
		}
		for _, id := range ids {
			if rc, ok := layout.RectOf(scr, m.size(), id); ok && inside(rc, x, y) {
				return id
			}
		}
	}
	return ""
}

func cardID(c, r int) string { return "card-" + strconv.Itoa(c) + "-" + strconv.Itoa(r) }

// parse reads the two numbers of a "card-C-R" id.
func parse(id string) (c, r int) {
	parts := strings.Split(id, "-")
	if len(parts) == 3 {
		c, _ = strconv.Atoi(parts[1])
		r, _ = strconv.Atoi(parts[2])
	}
	return c, r
}

func (m model) colAt(x int) int { return max(0, min(2, x/max(1, m.size().W/3))) }

// rowAt is the row a card dropped at row y of column c would take.
func (m model) rowAt(c, y int) int {
	return max(0, min(len(m.b.cols[c]), y-cardTop+m.top(c)))
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help || m.ed != nil {
		return m, nil
	}
	switch {
	case ev.Action == tui.MouseActionRelease:
		return m.release(), nil
	case ev.Action == tui.MouseActionMotion:
		return m.motion(ev), nil
	case ev.Button == tui.MouseButtonWheelUp:
		return m.goRow(-1), nil
	case ev.Button == tui.MouseButtonWheelDown:
		return m.goRow(1), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	return m.press(ev), nil
}

func (m model) press(ev tui.MouseEvent) model {
	id := m.hit(ev.X, ev.Y)
	var c, r int
	switch {
	case len(id) > 4 && id[:4] == "add-":
		m.col = int(id[4] - '0')
		return m.clamp().startEdit(true)
	case len(id) > 5 && id[:5] == "head-":
		m.col = int(id[5] - '0')
		return m.clamp()
	case len(id) > 5 && id[:5] == "card-":
		c, r = parse(id)
		m.col, m.row = c, r
		if ev.Clicks >= 2 {
			return m.startEdit(false)
		}
		m.dragging, m.moved = true, false
		m.dragCol, m.dragRow, m.overCol, m.overRow = c, r, c, r
	}
	return m
}

func (m model) motion(ev tui.MouseEvent) model {
	if !m.dragging {
		m.hover = m.hit(ev.X, ev.Y)
		return m
	}
	m.overCol = m.colAt(ev.X)
	m.overRow = m.rowAt(m.overCol, ev.Y)
	m.moved = m.moved || m.overCol != m.dragCol || m.overRow != m.dragRow
	return m
}

func (m model) release() model {
	if !m.dragging {
		return m
	}
	m.dragging = false
	if !m.moved {
		return m
	}
	at := m.overRow
	if m.overCol == m.dragCol {
		at = min(at, len(m.b.cols[m.dragCol])-1)
	}
	if m.overCol == m.dragCol && at == m.dragRow {
		return m
	}
	m.b, m.row = m.b.move(m.dragCol, m.dragRow, m.overCol, at)
	m.col = m.overCol
	m.msg = "Moved to " + columnNames[m.col]
	return m
}
