package main

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m.cancel()
	}
	if m.confirm != "" {
		return m.confirmKey(k)
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter && !(k.Type == tui.KeyRunes && k.Text == "?")
		return m, nil
	}
	m.msg = ""
	switch k.Type {
	case tui.KeyF2:
		return m.setMode(!m.save), nil
	case tui.KeyF3:
		return m.setFilter(m.filter + 1), nil
	case tui.KeyEsc:
		return m.cancel()
	}
	if m.onName {
		return m.nameKey(k)
	}
	return m.listKey(k)
}

func (m model) cancel() (tui.Model, tui.Cmd) {
	m.result = ""
	return m, tui.Quit()
}

func (m model) confirmKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyEnter, k.Type == tui.KeyRunes && strings.EqualFold(k.Text, "y"):
		return m.choose(m.confirm)
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && strings.EqualFold(k.Text, "n"):
		m.confirm = ""
	}
	return m, nil
}

func (m model) nameKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEnter:
		return m.accept()
	case tui.KeyTab, tui.KeyUp, tui.KeyDown:
		m.onName = false
		if k.Type != tui.KeyTab {
			return m.listKey(k)
		}
	case tui.KeyBackspace:
		if len(m.name) > 0 {
			m.name = append([]rune(nil), m.name[:len(m.name)-1]...)
		}
	case tui.KeyRunes, tui.KeySpace:
		m.name = append(append([]rune(nil), m.name...), []rune(k.Text)...)
	}
	return m, nil
}

func (m model) listKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyTab:
		if m.save {
			m.onName = true
		}
		return m, nil
	case tui.KeyEnter:
		return m.activate()
	case tui.KeyPgUp:
		return m.page(-1), nil
	case tui.KeyPgDown:
		return m.page(1), nil
	case tui.KeyRunes:
		switch k.Text {
		case "?":
			m.help = true
			return m, nil
		case ".":
			return m.toggleHidden(), nil
		}
	}
	d := 1
	if k.Type == tui.KeyUp {
		d = -1
	}
	m.fp, _ = m.fp.Update(k)
	m.lastClick = ""
	return m.fix(d), nil
}

// page moves the cursor ten visible entries.
func (m model) page(d int) model {
	for range 10 {
		before := m.fp.Cursor()
		m.fp, _ = m.fp.Update(arrow(d))
		m = m.fix(d)
		if m.fp.Cursor() == before {
			break
		}
	}
	return m
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

// crumbs are the folders of the path, each with the path that leads to it.
func (m model) crumbs() (names, paths []string) {
	dir := filepath.Clean(m.fp.Dir)
	sep := string(filepath.Separator)
	start := filepath.VolumeName(dir) + sep
	names = []string{sep}
	if m.root != "" && strings.HasPrefix(dir, m.root) {
		start, names = m.root, []string{filepath.Base(m.root)}
	}
	paths = []string{start}
	rest := strings.TrimPrefix(strings.TrimPrefix(dir, start), sep)
	acc := start
	for _, part := range strings.Split(rest, sep) {
		if part == "" {
			continue
		}
		acc = filepath.Join(acc, part)
		names, paths = append(names, part), append(paths, acc)
	}
	return names, paths
}

func (m model) ids() []string {
	ids := []string{"hidden", "btn-ok", "btn-cancel", "name"}
	for i := range filters {
		ids = append(ids, "filter-"+strconv.Itoa(i))
	}
	names, _ := m.crumbs()
	for i := range names {
		ids = append(ids, "crumb-"+strconv.Itoa(i))
	}
	for _, i := range m.visible() {
		ids = append(ids, "row-"+strconv.Itoa(i))
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
	if m.help || m.confirm != "" {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion:
		m.hover = id
		return m, nil
	case ev.Button == tui.MouseButtonWheelUp:
		return m.page3(-1), nil
	case ev.Button == tui.MouseButtonWheelDown:
		return m.page3(1), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	m.msg = ""
	return m.click(id, ev.Clicks)
}

func (m model) page3(d int) model {
	for range 3 {
		m.fp, _ = m.fp.Update(arrow(d))
		m = m.fix(d)
	}
	return m
}

func (m model) click(id string, clicks int) (tui.Model, tui.Cmd) {
	kind, n := split(id)
	switch {
	case id == "hidden":
		return m.toggleHidden(), nil
	case id == "btn-ok":
		return m.accept()
	case id == "btn-cancel":
		return m.cancel()
	case id == "name":
		m.onName = m.save
	case kind == "filter":
		return m.setFilter(n), nil
	case kind == "crumb":
		_, paths := m.crumbs()
		return m.goTo(paths[n]), nil
	case kind == "row":
		m.onName = false
		if clicks >= 2 || m.lastClick == id {
			m = m.moveTo(n)
			return m.activate()
		}
		m = m.moveTo(n)
		m.lastClick = id
	}
	return m, nil
}

func split(id string) (string, int) {
	if i := strings.IndexByte(id, '-'); i >= 0 {
		if n, err := strconv.Atoi(id[i+1:]); err == nil {
			return id[:i], n
		}
	}
	return id, -1
}
