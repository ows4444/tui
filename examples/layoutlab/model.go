package main

import (
	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

const (
	chromeRows = 4 // title, mode tabs, status, hints
	leftW      = 46
)

type model struct {
	th    ui.Theming // the theme the Program chose; see SetTheme
	mode  mode
	vals  map[string]int
	saved [numModes]map[string]int // each mode keeps its settings
	sel   int                      // selected setting
	code  bool
	hover string
	help  bool
	msg   string

	width, height int
}

func newModel() model {
	m := model{}
	for md := range m.saved {
		m.saved[md] = defaults(mode(md))
	}
	m.vals = m.saved[mFlex]
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 120, H: 36}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) params() []param { return params(m.mode, m.vals) }

func (m model) setMode(md mode) model {
	m.saved[m.mode] = m.vals
	m.mode, m.vals, m.sel, m.msg = md, m.saved[md], 0, ""
	return m
}

// put stores value n for the selected setting, within its range; it copies the
// map first so earlier values stay as they were.
func (m model) put(i, n int) model {
	ps := m.params()
	if i < 0 || i >= len(ps) {
		return m
	}
	p := ps[i]
	vals := make(map[string]int, len(m.vals)+1)
	for k, x := range m.vals {
		vals[k] = x
	}
	vals[p.key] = max(p.lo, min(p.hi, n))
	m.vals, m.sel = vals, i
	return m.settle()
}

// settle keeps the selectors inside the counts they select from.
func (m model) settle() model {
	switch m.mode {
	case mFlex:
		m.vals["child"] = min(m.vals["child"], m.vals["count"]-1)
	case mGrid:
		m.vals["col"] = min(m.vals["col"], m.vals["cols"]-1)
	}
	if m.sel >= len(m.params()) {
		m.sel = len(m.params()) - 1
	}
	return m
}

func (m model) nudge(d int) model {
	return m.put(m.sel, m.vals[m.params()[m.sel].key]+d)
}

func (m model) reset(i int) model {
	ps := m.params()
	return m.put(i, defaults(m.mode)[ps[i].key])
}

func (m model) resetMode() model {
	m.vals = defaults(m.mode)
	m.sel, m.msg = 0, "Settings put back"
	return m.settle()
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
