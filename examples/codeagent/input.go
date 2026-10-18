package main

import (
	"strings"
	"unicode"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
)

func (m model) promptEmpty() bool { return len(m.prompt) == 0 }

// asking is true when a key should answer the waiting call: one is waiting and
// nothing is typed in the prompt.
func (m model) asking() bool { return m.phase == waiting && m.promptEmpty() }

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if m.help {
		if k.Type == tui.KeyCtrlC {
			return m, tui.Quit()
		}
		m.help = false
		return m, nil
	}
	m.msg = ""
	switch k.Type {
	case tui.KeyCtrlC:
		if m.phase != idle {
			return m.interrupt(), nil
		}
		return m, tui.Quit()
	case tui.KeyEnter:
		if m.asking() {
			return m.decide(true, false)
		}
		return m.submit()
	case tui.KeyEsc:
		return m.escape()
	case tui.KeyPgUp:
		return m.scroll(m.pageRows()), nil
	case tui.KeyPgDown:
		return m.scroll(-m.pageRows()), nil
	case tui.KeyUp:
		return m.history(-1), nil
	case tui.KeyDown:
		return m.history(1), nil
	case tui.KeyLeft:
		m.cur = max(0, m.cur-1)
	case tui.KeyRight:
		m.cur = min(len(m.prompt), m.cur+1)
	case tui.KeyHome:
		m.cur = 0
	case tui.KeyEnd:
		m.cur = len(m.prompt)
	case tui.KeyBackspace:
		return m.deleteBack(), nil
	case tui.KeyDelete:
		return m.deleteFwd(), nil
	case tui.KeyCtrl:
		return m.ctrl(k.Code)
	case tui.KeyRunes, tui.KeySpace:
		return m.typed(k.Text)
	}
	return m, nil
}

func (m model) escape() (tui.Model, tui.Cmd) {
	switch {
	case m.asking():
		return m.decide(false, false)
	case m.phase != idle:
		return m.interrupt(), nil
	}
	m.prompt, m.cur = nil, 0
	return m, nil
}

func (m model) ctrl(c rune) (tui.Model, tui.Cmd) {
	switch c {
	case 'd':
		return m, tui.Quit()
	case 'l':
		m.blocks, m.off = nil, 0
	case 'u':
		m.prompt, m.cur = nil, 0
	case 'a':
		m.cur = 0
	case 'e':
		m.cur = len(m.prompt)
	case 'w':
		return m.deleteWord(), nil
	}
	return m, nil
}

// typed handles printable input: the answer keys when a call is waiting and the
// prompt is empty, otherwise text for the prompt.
func (m model) typed(s string) (tui.Model, tui.Cmd) {
	if m.asking() {
		switch s {
		case "y":
			return m.decide(true, false)
		case "n":
			return m.decide(false, false)
		case "a":
			return m.decide(true, true)
		case "d":
			return m.toggleDiff(), nil
		case "?":
			m.help = true
			return m, nil
		}
	}
	if s == "?" && m.promptEmpty() {
		m.help = true
		return m, nil
	}
	return m.insert(s), nil
}

func (m model) insert(s string) model {
	rs := []rune(s)
	p := make([]rune, 0, len(m.prompt)+len(rs))
	p = append(append(append(p, m.prompt[:m.cur]...), rs...), m.prompt[m.cur:]...)
	m.prompt, m.cur = p, m.cur+len(rs)
	m.histPos = len(m.hist)
	return m
}

// paste puts pasted text in the prompt, newlines turned to spaces. It never
// reaches the answer keys.
func (m model) paste(p tui.PasteEvent) model {
	text := strings.Join(strings.Fields(strings.ReplaceAll(p.Text, "\n", " ")), " ")
	if lines := strings.Count(p.Text, "\n") + 1; lines > 1 {
		m.msg = "Pasted " + itoa(lines) + " lines as one"
	}
	return m.insert(text)
}

func (m model) deleteBack() model {
	if m.cur == 0 {
		return m
	}
	m.prompt = append(append([]rune(nil), m.prompt[:m.cur-1]...), m.prompt[m.cur:]...)
	m.cur--
	return m
}

func (m model) deleteFwd() model {
	if m.cur >= len(m.prompt) {
		return m
	}
	m.prompt = append(append([]rune(nil), m.prompt[:m.cur]...), m.prompt[m.cur+1:]...)
	return m
}

func (m model) deleteWord() model {
	i := m.cur
	for i > 0 && unicode.IsSpace(m.prompt[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(m.prompt[i-1]) {
		i--
	}
	m.prompt = append(append([]rune(nil), m.prompt[:i]...), m.prompt[m.cur:]...)
	m.cur = i
	return m
}

func (m model) submit() (tui.Model, tui.Cmd) {
	text := strings.TrimSpace(string(m.prompt))
	if text == "" {
		return m, nil
	}
	if m.phase != idle {
		m.msg = "The agent is busy; Esc stops it"
		return m, nil
	}
	m.prompt, m.cur = nil, 0
	if len(m.hist) == 0 || m.hist[len(m.hist)-1] != text {
		m.hist = append(append([]string(nil), m.hist...), text)
	}
	m.histPos, m.draft = len(m.hist), ""
	if strings.HasPrefix(text, "/") {
		return m.command(text)
	}
	return m.send(text)
}

// history walks through earlier prompts; the unsent text comes back at the end.
func (m model) history(d int) model {
	if len(m.hist) == 0 {
		return m
	}
	if m.histPos == len(m.hist) {
		m.draft = string(m.prompt)
	}
	m.histPos = max(0, min(len(m.hist), m.histPos+d))
	text := m.draft
	if m.histPos < len(m.hist) {
		text = m.hist[m.histPos]
	}
	m.prompt = []rune(text)
	m.cur = len(m.prompt)
	return m
}

func (m model) pageRows() int { return max(1, m.size().H-chromeRows-1) }

func (m model) scroll(d int) model {
	m.off = max(0, min(max(0, len(m.lines())-m.pageRows()-1), m.off+d))
	return m
}

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"btn-approve", "btn-deny", "btn-always", "btn-diff", "prompt"}
	for i, b := range m.blocks {
		if b.kind == bTool {
			ids = append(ids, "card-"+itoa(i))
		}
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
	case ev.Button == tui.MouseButtonWheelUp:
		return m.scroll(3), nil
	case ev.Button == tui.MouseButtonWheelDown:
		return m.scroll(-3), nil
	case ev.Button != tui.MouseButtonLeft || ev.Action != tui.MouseActionPress:
		return m, nil
	}
	switch {
	case id == "btn-approve":
		return m.decide(true, false)
	case id == "btn-deny":
		return m.decide(false, false)
	case id == "btn-always":
		return m.decide(true, true)
	case id == "btn-diff":
		return m.toggleDiff(), nil
	case strings.HasPrefix(id, "card-"):
		return m.edit(atoi(id[5:]), func(b *block) { b.open = !b.open }), nil
	}
	return m, nil
}
