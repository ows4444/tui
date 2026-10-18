package main

import (
	"os"
	"sort"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
)

const (
	chromeRows = 3 // title, status, hints
	leftW      = 30
	// focus targets in the editor
	fR, fG, fB, fHex = 0, 1, 2, 3
)

type model struct {
	names []string
	sp    spec
	t     theme.Theme
	err   error

	sel    int // selected token
	focus  int // slider or hex code
	edit   []rune
	typing bool
	hover  string
	msg    string
	help   bool
	path   string

	width, height int
}

func newModel() model {
	m := model{names: theme.PresetNames(), path: "theme.json"}
	m.sp = spec{base: "tokyo-night", over: map[string]string{}}
	return m.rebuild()
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 110, H: 32}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) rebuild() model {
	m.t, m.err = m.sp.build()
	return m
}

func (m model) baseIndex() int {
	return sort.SearchStrings(m.names, m.sp.base)
}

func (m model) setBase(i int) model {
	n := len(m.names)
	m.sp.base = m.names[((i%n)+n)%n]
	cleared := m.sp.changes() > 0
	m.sp.over, m.sp.border, m.sp.glyphs = map[string]string{}, 0, 0
	m.msg = ""
	if cleared {
		m.msg = "Changes cleared with the new base"
	}
	return m.rebuild()
}

// current is the selected token's colour in the theme as it stands.
func (m model) current() ansi.RGB { return toRGB(tokens[m.sel].get(m.t)) }

func (m model) baseColour(i int) ansi.RGB {
	base, _ := theme.Preset(m.sp.base)
	return toRGB(tokens[i].get(base))
}

func (m model) set(c ansi.RGB) model {
	k := tokens[m.sel].key
	if c == m.baseColour(m.sel) {
		delete(m.sp.over, k)
	} else {
		m.sp.over = cloneMap(m.sp.over)
		m.sp.over[k] = hex(c)
	}
	return m.rebuild()
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+1)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func (m model) channel(c, d int) model {
	rgb := m.current()
	p := []*uint8{&rgb.R, &rgb.G, &rgb.B}[c]
	*p = uint8(max(0, min(255, int(*p)+d)))
	return m.set(rgb)
}

func (m model) setChannel(c, v int) model {
	rgb := m.current()
	p := []*uint8{&rgb.R, &rgb.G, &rgb.B}[c]
	*p = uint8(max(0, min(255, v)))
	return m.set(rgb)
}

func (m model) resetToken() model {
	m.sp.over = cloneMap(m.sp.over)
	delete(m.sp.over, tokens[m.sel].key)
	return m.rebuild()
}

func (m model) resetAll() model {
	m.sp.over, m.sp.border, m.sp.glyphs = map[string]string{}, 0, 0
	m.msg = "Back to " + m.sp.base
	return m.rebuild()
}

func (m model) cycleBorder() model {
	m.sp.border = (m.sp.border + 1) % len(borderNames)
	return m.rebuild()
}

func (m model) cycleGlyphs() model {
	m.sp.glyphs = (m.sp.glyphs + 1) % len(glyphNames)
	return m.rebuild()
}

func (m model) save() model {
	if err := os.WriteFile(m.path, m.sp.document(), 0o600); err != nil {
		m.msg = "Could not save: " + err.Error()
		return m
	}
	m.msg = "Saved " + m.path
	return m
}

func (m model) commitHex() model {
	m.typing = false
	c, ok := parseHex(string(m.edit))
	if !ok {
		m.msg = "\"" + string(m.edit) + "\" is not a colour; use #rrggbb"
		return m
	}
	m.msg = ""
	return m.set(c)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) changedMark(i int) bool {
	_, ok := m.sp.over[tokens[i].key]
	return ok
}

func (m model) ratio() float64 {
	bg := m.t.Background
	if bg == nil {
		bg = m.t.TextInverse
	}
	return theme.Contrast(tokens[m.sel].get(m.t), bg)
}
