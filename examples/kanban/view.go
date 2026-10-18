package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// cardTop is the screen row of the first card: the title and the column heads
// come first.
const cardTop = 2

// areaH is the number of rows from the first card to the foot row.
func (m model) areaH() int { return max(2, m.size().H-4) }

// top is the first card row shown in column c; the selected card stays in view.
func (m model) top(c int) int {
	if c != m.col || m.row < 0 {
		return 0
	}
	return max(0, m.row-(m.areaH()-2))
}

func (m model) colW(c int) int {
	w := m.size().W / 3
	if c == 2 {
		return m.size().W - 2*w
	}
	return w
}

func fit(s string, w int) string {
	s = ansi.Truncate(s, w)
	if n := ansi.Width(s); n < w {
		s += strings.Repeat(" ", w-n)
	}
	return s
}

// cell fits s to a column's width, leaving one blank column between columns.
func cell(s string, w int) string { return fit(s, w-1) + " " }

func (m model) head(c int, th theme.Theme) layout.Node {
	w := m.colW(c)
	st := ansi.NewStyle().Bold().Foreground(th.Muted)
	switch {
	case m.dragging && m.moved && m.overCol == c:
		st = ansi.NewStyle().Bold().Underline().Foreground(th.Success)
	case c == m.col:
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	text := " " + columnNames[c] + " (" + strconv.Itoa(len(m.b.cols[c])) + ")"
	return layout.Named("head-"+strconv.Itoa(c), layout.Block(st.Render(cell(text, w))))
}

func (m model) cardLine(c, r int, th theme.Theme) string {
	w := m.colW(c)
	k := m.b.cols[c][r]
	title := " " + k.title
	switch {
	case m.dragging && m.moved && c == m.dragCol && r == m.dragRow:
		return ansi.NewStyle().Faint().Italic().Render(cell("◇"+k.title, w))
	case c == m.col && r == m.row:
		return ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary).Render(cell("▌"+k.title, w))
	case m.hover == cardID(c, r):
		return ansi.NewStyle().Foreground(th.Primary).Render(cell(title, w))
	}
	return cell(title, w)
}

func (m model) column(c int, th theme.Theme) layout.Node {
	w, h := m.colW(c), m.areaH()
	var kids []layout.FlexChild
	cs := m.b.cols[c]
	switch {
	case m.help:
	case len(cs) == 0:
		kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(cell(" (empty)", w)))})
	}
	for r := m.top(c); r < len(cs) && len(kids) < h-1 && !m.help; r++ {
		kids = append(kids, layout.FlexChild{Node: layout.Named(cardID(c, r), layout.Block(m.cardLine(c, r, th)))})
	}
	for len(kids) < h-1 {
		kids = append(kids, layout.FlexChild{Node: layout.Block(strings.Repeat(" ", w))})
	}
	add := ansi.NewStyle().Faint()
	if m.hover == "add-"+strconv.Itoa(c) {
		add = ansi.NewStyle().Foreground(th.Primary)
	}
	kids = append(kids, layout.FlexChild{Node: layout.Named("add-"+strconv.Itoa(c), layout.Block(add.Render(cell(" + add", w))))})
	return layout.Fixed(layout.Column(0, kids...), layout.Size{W: w, H: h})
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(fit(k, 16)) + a
	}
	return []string{
		key("arrows h j k l", "move the selection"), key("Shift+←→  < >", "move the card to the next column"),
		key("Shift+↑↓  K J", "move the card up or down"), key("a", "add a card to this column"),
		key("Enter  e", "rename the card"), key("d  u", "delete, undo the delete"),
		key("drag", "drop a card on a column, at the pointer's row"), key("double-click", "rename"),
		key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
	}
}

func (m model) status(th theme.Theme) string {
	w := m.size().W
	if e := m.ed; e != nil {
		verb := "Rename"
		if e.row < 0 {
			verb = "Add to " + columnNames[e.col]
		}
		return ansi.Truncate(ansi.NewStyle().Foreground(th.Primary).Render(verb+": ")+string(e.text)+"▏", w)
	}
	if m.dragging && m.moved {
		return ansi.Truncate("Drop on "+columnNames[m.overCol]+" at row "+strconv.Itoa(m.overRow+1), w)
	}
	if m.msg != "" {
		return ansi.Truncate(m.msg, w)
	}
	return ansi.NewStyle().Faint().Render(ansi.Truncate(strconv.Itoa(m.b.total())+" cards  ·  drag a card to another column", w))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("kanban") + ansi.NewStyle().Faint().Render("  "+strconv.Itoa(len(m.b.cols[2]))+" of "+strconv.Itoa(m.b.total())+" done")
	var heads, cols []layout.FlexChild
	for c := 0; c < 3; c++ {
		heads = append(heads, layout.FlexChild{Node: layout.Fixed(m.head(c, th), layout.Size{W: m.colW(c), H: 1})})
		cols = append(cols, layout.FlexChild{Node: m.column(c, th)})
	}
	body := layout.Node(layout.Row(0, cols...))
	if m.help {
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, w))})
		}
		for len(lines) < m.areaH() {
			lines = append(lines, layout.FlexChild{Node: layout.Block("")})
		}
		body = layout.Column(0, lines[:m.areaH()]...)
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "←→↑↓", Action: "select"}, widgets.Hint{Key: "<>", Action: "move"}, widgets.Hint{Key: "a", Action: "add"},
		widgets.Hint{Key: "e", Action: "rename"}, widgets.Hint{Key: "d", Action: "delete"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	if m.ed != nil {
		hints = widgets.Footer(w, widgets.Hint{Key: "enter", Action: "save"}, widgets.Hint{Key: "esc", Action: "cancel"})
	}
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, w))},
		layout.FlexChild{Node: layout.Row(0, heads...)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(m.status(th))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
