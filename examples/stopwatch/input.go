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
			d = int(numTabs) - 1
		}
		m.tab = (m.tab + tab(d)) % numTabs
	case tui.KeySpace:
		return m.primary()
	case tui.KeyUp:
		return m.adjust(1), nil
	case tui.KeyDown:
		return m.adjust(-1), nil
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
		m.tab = tab(r[0] - '1')
	case " ":
		return m.primary()
	case "l":
		if m.tab == tStopwatch {
			return m.lap(), nil
		}
	case "r":
		return m.reset(), nil
	case "p":
		if m.tab == tTimer {
			return m.nextPreset(), nil
		}
	case "+", "=":
		return m.adjust(1), nil
	case "-":
		return m.adjust(-1), nil
	}
	return m, nil
}

// primary is the Start/Stop action of the tab.
func (m model) primary() (tui.Model, tui.Cmd) {
	switch m.tab {
	case tStopwatch:
		return m.toggleWatch()
	case tTimer:
		return m.toggleTimer()
	}
	return m, nil
}

func (m model) reset() model {
	switch m.tab {
	case tStopwatch:
		return m.resetWatch()
	case tTimer:
		return m.resetTimer()
	}
	return m
}

// adjust changes the timer length, or scrolls the laps on the stopwatch tab.
func (m model) adjust(d int) model {
	switch m.tab {
	case tTimer:
		return m.setMinutes(m.minutes + d)
	case tStopwatch:
		m.lapAt = max(0, min(max(0, len(m.w.laps)-1), m.lapAt-d))
	}
	return m
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"btn-start", "btn-lap", "btn-reset", "minus", "plus", "laps"}
	for i := range tabNames {
		ids = append(ids, "tab-"+strconv.Itoa(i))
	}
	for _, p := range presets {
		ids = append(ids, "preset-"+strconv.Itoa(p))
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
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch {
	case ev.Action == tui.MouseActionMotion:
		m.hover = id
		return m, nil
	case ev.Button == tui.MouseButtonWheelUp && id == "laps":
		return m.adjust(1), nil
	case ev.Button == tui.MouseButtonWheelDown && id == "laps":
		return m.adjust(-1), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	m.msg = ""
	switch {
	case strings.HasPrefix(id, "tab-"):
		n, _ := strconv.Atoi(id[4:])
		m.tab = tab(n)
	case id == "btn-start":
		return m.primary()
	case id == "btn-lap":
		return m.lap(), nil
	case id == "btn-reset":
		return m.reset(), nil
	case id == "minus":
		return m.setMinutes(m.minutes - 1), nil
	case id == "plus":
		return m.setMinutes(m.minutes + 1), nil
	case strings.HasPrefix(id, "preset-"):
		n, _ := strconv.Atoi(id[7:])
		return m.setMinutes(n), nil
	}
	return m, nil
}
