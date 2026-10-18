package main

import (
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
)

// edit is the single-line editor for a card title. row is -1 for a new card.
type edit struct {
	col, row int
	text     []rune
}

type undo struct {
	col, row int
	k        card
}

type model struct {
	th       ui.Theming // the theme the Program chose; see SetTheme
	b        board
	col, row int // the selected card; row -1 when the column is empty
	ed       *edit
	dragging bool
	dragCol  int // column the card was lifted from
	dragRow  int
	overCol  int // column under the pointer while dragging
	overRow  int // row the card would land at
	moved    bool
	trash    []undo
	hover    string
	msg      string
	help     bool

	width, height int
}

func newModel() model {
	m := model{b: sampleBoard()}
	return m.clamp()
}

func (m model) Init() tui.Cmd { return nil }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 100, H: 30}
	}
	return layout.Size{W: m.width, H: m.height}
}

// clamp keeps the selection on a card, or row -1 on an empty column.
func (m model) clamp() model {
	m.col = max(0, min(2, m.col))
	m.row = min(m.row, len(m.b.cols[m.col])-1)
	if len(m.b.cols[m.col]) > 0 {
		m.row = max(0, m.row)
	}
	return m
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

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	if m.ed != nil {
		return m.editKey(k), nil
	}
	if m.help {
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter && !(k.Type == tui.KeyRunes && (k.Text == "?" || k.Text == "q"))
		return m, nil
	}
	m.msg = ""
	switch k.Type {
	case tui.KeyLeft, tui.KeyRight:
		d := 1
		if k.Type == tui.KeyLeft {
			d = -1
		}
		if k.Mod.Shift() {
			return m.shift(d), nil
		}
		return m.goCol(d), nil
	case tui.KeyUp, tui.KeyDown:
		d := 1
		if k.Type == tui.KeyUp {
			d = -1
		}
		if k.Mod.Shift() {
			return m.reorder(d), nil
		}
		return m.goRow(d), nil
	case tui.KeyEnter:
		return m.startEdit(false), nil
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
	case "h":
		return m.goCol(-1), nil
	case "l":
		return m.goCol(1), nil
	case "k":
		return m.goRow(-1), nil
	case "j":
		return m.goRow(1), nil
	case "<":
		return m.shift(-1), nil
	case ">":
		return m.shift(1), nil
	case "K":
		return m.reorder(-1), nil
	case "J":
		return m.reorder(1), nil
	case "a":
		return m.startEdit(true), nil
	case "e":
		return m.startEdit(false), nil
	case "d":
		return m.delete(), nil
	case "u":
		return m.undelete(), nil
	}
	return m, nil
}

func (m model) goCol(d int) model {
	m.col = max(0, min(2, m.col+d))
	return m.clamp()
}

func (m model) goRow(d int) model {
	if len(m.b.cols[m.col]) == 0 {
		return m
	}
	m.row = max(0, min(len(m.b.cols[m.col])-1, m.row+d))
	return m
}

// shift moves the selected card one column over, to the end of that column.
func (m model) shift(d int) model {
	to := m.col + d
	if m.row < 0 || to < 0 || to > 2 {
		m.msg = "No column that way"
		return m
	}
	m.b, m.row = m.b.move(m.col, m.row, to, len(m.b.cols[to]))
	m.col = to
	m.msg = "Moved to " + columnNames[to]
	return m
}

func (m model) reorder(d int) model {
	if m.row < 0 {
		return m
	}
	m.b, m.row = m.b.swap(m.col, m.row, d)
	return m
}

func (m model) delete() model {
	if m.row < 0 {
		return m
	}
	var k card
	m.b, k = m.b.remove(m.col, m.row)
	m.trash = append(append([]undo(nil), m.trash...), undo{m.col, m.row, k})
	m.msg = "Deleted \"" + k.title + "\" (u undoes)"
	return m.clamp()
}

func (m model) undelete() model {
	if len(m.trash) == 0 {
		m.msg = "Nothing to undo"
		return m
	}
	u := m.trash[len(m.trash)-1]
	m.trash = m.trash[:len(m.trash)-1]
	m.b, m.row = m.b.insert(u.col, u.row, u.k)
	m.col = u.col
	m.msg = "Brought back \"" + u.k.title + "\""
	return m
}

// ---- editing ---------------------------------------------------------------

func (m model) startEdit(add bool) model {
	switch {
	case add:
		m.ed = &edit{col: m.col, row: -1}
	case m.row >= 0:
		m.ed = &edit{col: m.col, row: m.row, text: []rune(m.b.cols[m.col][m.row].title)}
	default:
		m.msg = "Nothing to rename; a adds a card"
	}
	return m
}

func (m model) editKey(k tui.Key) model {
	e := *m.ed
	switch k.Type {
	case tui.KeyEsc:
		m.ed = nil
	case tui.KeyBackspace:
		if len(e.text) > 0 {
			e.text = e.text[:len(e.text)-1]
		}
		m.ed = &e
	case tui.KeyEnter:
		return m.commit()
	case tui.KeyRunes:
		e.text = append(append([]rune(nil), e.text...), []rune(k.Text)...)
		m.ed = &e
	}
	return m
}

func (m model) commit() model {
	e := *m.ed
	m.ed = nil
	title := strings.TrimSpace(string(e.text))
	switch {
	case title == "":
		m.msg = "A card needs a title"
	case e.row < 0:
		m.b, m.row = m.b.add(e.col, title)
		m.col = e.col
		m.msg = "Added to " + columnNames[e.col]
	default:
		m.b = m.b.rename(e.col, e.row, title)
	}
	return m
}
