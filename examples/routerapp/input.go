package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func itoa(n int) string { return strconv.Itoa(n) }

func padTo(s string, w int) string {
	r := []rune(s)
	if len(r) >= w {
		return string(r[:w-1]) + " "
	}
	return s + strings.Repeat(" ", w-len(r))
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter && !(k.Type == tui.KeyRunes && (k.Text == "?" || k.Text == "q"))
		return m, nil
	}
	switch k.Type {
	case tui.KeyTab:
		m.side = !m.side
	case tui.KeyUp:
		return m.move(-1), nil
	case tui.KeyDown:
		return m.move(1), nil
	case tui.KeyEnter:
		return m.activate()
	case tui.KeyEsc, tui.KeyBackspace:
		return m.back()
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
		return m.move(1), nil
	case "k":
		return m.move(-1), nil
	case "s":
		return m.star()
	case "m":
		return m.ship()
	}
	return m, nil
}

func (m model) move(d int) model {
	if m.side {
		m.sideCur = max(0, min(len(sections)-1, m.sideCur+d))
		return m
	}
	n := len(m.rows(m.top()))
	if n == 0 {
		return m
	}
	t := m.top()
	t.cur = max(0, min(n-1, t.cur+d))
	return m.withTop(t)
}

func (m model) activate() (tui.Model, tui.Cmd) {
	if m.side {
		m.side = false
		return m.goSection(sections[m.sideCur].kind), nil
	}
	return m.open(m.top().cur)
}

// open acts on row i of the current screen.
func (m model) open(i int) (model, tui.Cmd) {
	rows := m.rows(m.top())
	if i < 0 || i >= len(rows) {
		return m, nil
	}
	r := rows[i]
	switch {
	case r.toggle == 0:
		m.compact = !m.compact
	case r.toggle == 1:
		m.notify = !m.notify
	case r.to != nil && r.reset:
		return m.goSection(r.to.kind), nil
	case r.to != nil:
		return m.push(*r.to), nil
	}
	return m, nil
}

func (m model) star() (model, tui.Cmd) {
	t := m.top()
	if t.kind != rCustomer {
		return m.notice("Open a customer to star them")
	}
	st := make(map[int]bool, len(m.starred)+1)
	for k, v := range m.starred {
		st[k] = v
	}
	st[t.id] = !st[t.id]
	m.starred = st
	if st[t.id] {
		return m.notice("Starred " + customerByID(t.id).name)
	}
	return m.notice("Unstarred " + customerByID(t.id).name)
}

func (m model) ship() (model, tui.Cmd) {
	t := m.top()
	switch {
	case t.kind != rOrder:
		return m.notice("Open an order to ship it")
	case m.shipped[t.id]:
		return m.notice("Order #" + itoa(t.id) + " has already shipped")
	}
	sh := make(map[int]bool, len(m.shipped)+1)
	for k, v := range m.shipped {
		sh[k] = v
	}
	sh[t.id] = true
	m.shipped = sh
	return m.notice("Order #" + itoa(t.id) + " shipped")
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"back"}
	for i := range m.stack {
		ids = append(ids, "crumb-"+itoa(i))
	}
	for i := range sections {
		ids = append(ids, "side-"+itoa(i))
	}
	for i := range m.toasts {
		ids = append(ids, "toast-"+itoa(i))
	}
	for i := range m.rows(m.top()) {
		ids = append(ids, "row-"+itoa(i))
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

func split(id string) (string, int) {
	if i := strings.IndexByte(id, '-'); i >= 0 {
		if n, err := strconv.Atoi(id[i+1:]); err == nil {
			return id[:i], n
		}
	}
	return id, -1
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	if m.top().kind == rCustomer && m.size().W >= 30 {
		a := m.avatarOf(m.top())
		over := a.Bounds().Contains(ev.X, ev.Y)
		if over || m.faceModel.Name == a.Name && (a.Idling() || a.Reacting()) {
			var cmd tui.Cmd
			m.faceModel, cmd = a.Update(ev)
			return m, cmd
		}
	}
	id := m.hit(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion:
		m.hover = id
		return m, nil
	case ev.Button == tui.MouseButtonWheelUp:
		return m.move(-1), nil
	case ev.Button == tui.MouseButtonWheelDown:
		return m.move(1), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	kind, n := split(id)
	switch kind {
	case "back":
		return m.back()
	case "crumb":
		return m.popTo(n + 1), nil
	case "side":
		m.side = false
		m.sideCur = n
		return m.goSection(sections[n].kind), nil
	case "toast":
		m.toasts = append(append([]toast(nil), m.toasts[:n]...), m.toasts[n+1:]...)
	case "row":
		m.side = false
		if t := m.top(); t.cur == n || ev.Clicks >= 2 {
			t.cur = n
			return m.withTop(t).open(n)
		}
		t := m.top()
		t.cur = n
		return m.withTop(t), nil
	}
	return m, nil
}
