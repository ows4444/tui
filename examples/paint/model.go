package main

import (
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

type tool int

const (
	pen tool = iota
	eraser
	line
	box
	fill
	numTools
)

var toolNames = [numTools]string{"Pen", "Eraser", "Line", "Rect", "Fill"}
var toolKeys = [numTools]string{"p", "e", "l", "r", "f"}

const (
	chromeRows = 6 // title and tools, palette, top border, bottom border, status, hints
	numColours = 8
	maxUndo    = 60
)

type model struct {
	th     ui.Theming // the theme the Program chose; see SetTheme
	g      grid
	tool   tool
	colour uint8 // 1 to 8
	brush  int   // brush size, 1 to 3
	undo   []grid
	redo   []grid

	anchor  *pt  // first corner of a line or rectangle in progress
	base    grid // the picture a shape in progress is drawn over
	last    *pt  // the previous point of a pen stroke, while drawing
	erasing bool // the stroke in progress is the right button's
	cur     pt   // keyboard cursor
	hover   string
	kbd     bool // the keyboard cursor, not the pointer, is in use
	hx, hy  int
	dirty   bool // changed since the last save
	msg     string
	help    bool
	path    string

	width, height int
}

func newModel() model {
	return model{colour: 1, brush: 1, hx: -1, hy: -1}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

// canvasSize is the picture's size in pixels: two columns wide, one row high.
func (m model) canvasSize() (w, h int) {
	s := m.size()
	return max(1, (s.W-2)/2), max(1, s.H-chromeRows)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		w, h := m.canvasSize()
		if m.g.w != w || m.g.h != h {
			m.g = m.g.resize(w, h)
			m.cur = pt{min(m.cur.x, w-1), min(m.cur.y, h-1)}
		}
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// ---- editing ---------------------------------------------------------------

// begin saves the picture for undo before a stroke changes it.
func (m model) begin() model {
	m.undo = append(append([]grid(nil), m.undo...), m.g)
	if len(m.undo) > maxUndo {
		m.undo = m.undo[1:]
	}
	m.redo = nil
	m.g = m.g.clone()
	m.dirty = true
	return m
}

func (m model) paint(c uint8) uint8 {
	if m.tool == eraser || m.erasing {
		return 0
	}
	return c
}

// shape is the picture with the line or rectangle from the anchor to p drawn
// on it.
func (m model) shape(p pt) grid {
	g := m.base.clone()
	c := m.paint(m.colour)
	if m.tool == box {
		g.rect(*m.anchor, p, m.brush, c)
	} else {
		g.line(*m.anchor, p, m.brush, c)
	}
	return g
}

// shown is what the canvas displays: the picture, or the shape in progress.
func (m model) shown(p pt) grid {
	if m.anchor != nil && m.g.in(p) {
		return m.shape(p)
	}
	return m.g
}

func (m model) press(p pt, erase bool) model {
	if !m.g.in(p) {
		return m
	}
	m.erasing = erase
	m.msg = ""
	switch {
	case m.tool == fill && !erase:
		m = m.begin()
		m.g.fill(p, m.colour)
	case m.tool == fill:
		m = m.begin()
		m.g.fill(p, 0)
	case (m.tool == line || m.tool == box) && !erase:
		m.anchor, m.base = &p, m.g
	default:
		m = m.begin()
		m.g.stamp(p, m.brush, m.paint(m.colour))
		m.last = &p
	}
	return m
}

func (m model) drag(p pt) model {
	p = pt{max(0, min(m.g.w-1, p.x)), max(0, min(m.g.h-1, p.y))}
	if m.last != nil {
		m.g.line(*m.last, p, m.brush, m.paint(m.colour))
		m.last = &p
	}
	return m
}

// release ends a stroke; a shape in progress becomes part of the picture.
func (m model) release(p pt) model {
	if m.anchor != nil {
		p = pt{max(0, min(m.g.w-1, p.x)), max(0, min(m.g.h-1, p.y))}
		shape := m.shape(p)
		m = m.begin()
		m.g = shape
		m.anchor = nil
	}
	m.last, m.erasing = nil, false
	return m
}

func (m model) undoStroke() model {
	if len(m.undo) == 0 {
		m.msg = "Nothing to undo"
		return m
	}
	m.redo = append(append([]grid(nil), m.redo...), m.g)
	m.g = m.undo[len(m.undo)-1]
	m.undo = m.undo[:len(m.undo)-1]
	m.dirty = true
	return m
}

func (m model) redoStroke() model {
	if len(m.redo) == 0 {
		m.msg = "Nothing to redo"
		return m
	}
	m.undo = append(append([]grid(nil), m.undo...), m.g)
	m.g = m.redo[len(m.redo)-1]
	m.redo = m.redo[:len(m.redo)-1]
	m.dirty = true
	return m
}

func (m model) clear() model {
	if m.g.empty() {
		return m
	}
	m = m.begin()
	m.g = newGrid(m.g.w, m.g.h)
	m.msg = "Cleared (u undoes)"
	return m
}

func (m model) save() model {
	if m.path == "" {
		m.msg = "Not saved: run with -out FILE to say where"
		return m
	}
	if err := os.WriteFile(m.path, []byte(m.g.text()), 0o600); err != nil {
		m.msg = "Could not save: " + err.Error()
		return m
	}
	m.dirty = false
	m.msg = "Saved " + m.path
	return m
}
