package main

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/window"
)

// altKey is the Alt chord package window listens for.
func altKey(t tui.KeyType, text string, shift bool) tui.Key {
	mod := input.ModAlt
	if shift {
		mod |= input.ModShift
	}
	return tui.Key{Type: t, Text: text, Mod: mod}
}

func (m model) forward(k tui.Key) (model, tui.Cmd) {
	d, cmd := m.desk.Update(k)
	m.desk = d
	return m, cmd
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if m.modal != nil {
		return m.modalKey(k)
	}
	if m.help {
		m.help = false
		return m, nil
	}
	m.msg = ""
	if m.moving {
		return m.moveKey(k)
	}
	switch k.Type {
	case tui.KeyCtrl:
		if k.Code == 'q' {
			m.modal = &modal{title: "Quit?", text: "Close all windows and leave.", yes: "quit"}
		}
		return m, nil
	case tui.KeyCtrlC:
		m.modal = &modal{title: "Quit?", text: "Close all windows and leave.", yes: "quit"}
		return m, nil
	case tui.KeyF1:
		m.help = true
	case tui.KeyF2:
		return m.forward(altKey(tui.KeyRunes, "m", false))
	case tui.KeyF3:
		return m.reopen(), nil
	case tui.KeyF4:
		return m.askClose(m.desk.Focused()), nil
	case tui.KeyF5:
		return m.hide(m.desk.Focused()), nil
	case tui.KeyF6:
		if m.desk.Focused() != "" {
			m.moving = true
			m.msg = "Move mode: arrows move, Shift+arrows resize, Esc leaves"
		}
	case tui.KeyTab:
		if k.Mod.Shift() {
			return m.forward(altKey(tui.KeyRunes, "[", false))
		}
		return m.forward(altKey(tui.KeyRunes, "]", false))
	default:
		if k.Mod.Alt() {
			return m.forward(k)
		}
		return m.typeInWindow(k), nil
	}
	return m, nil
}

// moveKey turns plain arrows into the Alt chords the desktop moves and resizes
// with.
func (m model) moveKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc, tui.KeyEnter, tui.KeyF6:
		m.moving, m.msg = false, ""
		return m, nil
	case tui.KeyLeft, tui.KeyRight, tui.KeyUp, tui.KeyDown:
		return m.forward(altKey(k.Type, "", k.Mod.Shift()))
	}
	return m, nil
}

// typeInWindow gives a key to the window with focus: the notepad takes text.
func (m model) typeInWindow(k tui.Key) model {
	switch m.desk.Focused() {
	case "notes":
		return m.typeNotes(k)
	case "counter":
		switch {
		case k.Type == tui.KeyRunes && (k.Text == "+" || k.Text == "="), k.Type == tui.KeyUp:
			m.counter++
		case k.Type == tui.KeyRunes && k.Text == "-", k.Type == tui.KeyDown:
			m.counter--
		case k.Type == tui.KeyRunes && k.Text == "0":
			m.counter = 0
		}
	case "log":
		switch k.Type {
		case tui.KeyUp, tui.KeyPgUp:
			m.logOff = min(max(0, len(m.logs)-1), m.logOff+1)
		case tui.KeyDown, tui.KeyPgDown:
			m.logOff = max(0, m.logOff-1)
		}
	}
	return m
}

func (m model) typeNotes(k tui.Key) model {
	switch k.Type {
	case tui.KeyRunes, tui.KeySpace:
		m.notes = append(append([]rune(nil), m.notes...), []rune(k.Text)...)
	case tui.KeyEnter:
		m.notes = append(append([]rune(nil), m.notes...), '\n')
	case tui.KeyBackspace:
		if len(m.notes) > 0 {
			m.notes = append([]rune(nil), m.notes[:len(m.notes)-1]...)
		}
	default:
		return m
	}
	m.dirty = true
	return m
}

func (m model) modalKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyEnter, k.Type == tui.KeyRunes && strings.EqualFold(k.Text, "y"):
		return m.answer(true)
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && strings.EqualFold(k.Text, "n"):
		return m.answer(false)
	}
	return m, nil
}

func (m model) answer(yes bool) (tui.Model, tui.Cmd) {
	md := m.modal
	m.modal = nil
	if !yes {
		return m, nil
	}
	switch {
	case md.yes == "quit":
		return m, tui.Quit()
	case strings.HasPrefix(md.yes, "close:"):
		return m.closeWindow(strings.TrimPrefix(md.yes, "close:")), nil
	}
	return m, nil
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

var fixedIDs = []string{"modal-yes", "modal-no", "tb-add"}

func (m model) ids() []string {
	ids := append([]string(nil), fixedIDs...)
	for _, a := range apps {
		ids = append(ids, "tb-"+a.id)
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
	id := m.hit(ev.X, ev.Y)
	if m.modal != nil {
		if ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft {
			switch id {
			case "modal-yes":
				return m.answer(true)
			case "modal-no":
				return m.answer(false)
			}
		}
		return m, nil
	}
	if m.help {
		return m, nil
	}
	if ev.Action == tui.MouseActionMotion {
		m.hover = id
	}
	press := ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft
	switch {
	case press && id == "tb-add":
		return m.reopen(), nil
	case press && strings.HasPrefix(id, "tb-"):
		return m.taskClick(id[3:]), nil
	case ev.Button == tui.MouseButtonWheelUp || ev.Button == tui.MouseButtonWheelDown:
		return m.wheel(ev), nil
	case press:
		return m.press(ev)
	}
	d, cmd := m.desk.Update(ev)
	m.desk = d
	return m, cmd
}

// taskClick handles a click on a task bar entry: the focused window hides, another comes to the front, a hidden
// one is shown.
func (m model) taskClick(id string) model {
	if _, open := m.desk.Window(id); !open {
		return m.show(id)
	}
	if m.desk.Focused() == id {
		return m.hide(id)
	}
	m.desk.Raise(id)
	return m.log(titleOf(id) + " has focus")
}

func (m model) wheel(ev tui.MouseEvent) model {
	if id, part := m.desk.Hit(ev.X, ev.Y); id == "log" && part == window.PartBody {
		if ev.Button == tui.MouseButtonWheelUp {
			m.logOff = min(max(0, len(m.logs)-1), m.logOff+3)
		} else {
			m.logOff = max(0, m.logOff-3)
		}
	}
	return m
}

// press looks at what the press lands on before the desktop raises it: the
// notepad's close box asks first, and a click in the counter's body is a button.
func (m model) press(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	id, part := m.desk.Hit(ev.X, ev.Y)
	switch {
	case part == window.PartClose && id == "notes" && m.dirty:
		m.desk.Raise(id)
		return m.askClose(id), nil
	case part == window.PartBody && id == "counter":
		m = m.counterClick(ev)
	}
	d, cmd := m.desk.Update(ev)
	m.desk = d
	return m, cmd
}

// counterClick finds which of the counter's buttons a click is on, from the
// click's place in the window's content.
func (m model) counterClick(ev tui.MouseEvent) model {
	cr, ok := m.desk.ContentRect("counter")
	if !ok {
		return m
	}
	x, y := ev.X-cr.X, ev.Y-cr.Y
	switch {
	case y == counterRow && x >= minusX && x < minusX+5:
		m.counter--
	case y == counterRow && x >= plusX && x < plusX+5:
		m.counter++
	case y == counterRow+2 && x >= resetX && x < resetX+11:
		m.counter = 0
	}
	return m
}
