package main

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func (m model) button(id, label string, on bool, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	switch {
	case on:
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	case m.hover == id:
		st = ansi.NewStyle().Foreground(th.Primary).Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))}
}

func (m model) face() string {
	switch m.g.state {
	case won:
		return "B)"
	case lost:
		return ":("
	}
	return ":)"
}

func (m model) bar(th theme.Theme) layout.Node {
	kids := []layout.FlexChild{
		{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("minesweeper"))},
		{Node: layout.Block("mines " + itoa(m.g.minesLeft()) + "  time " + itoa(m.secs))},
		m.button("new", "["+m.face()+" new]", false, th),
	}
	for i, lv := range levels {
		kids = append(kids, m.button("size-"+itoa(i), "["+itoa(i+1)+" "+itoa(lv.w)+"x"+itoa(lv.h)+"]", i == m.lv, th))
	}
	return layout.Fixed(layout.Row(2, kids...), layout.Size{W: m.size().W, H: 1})
}

var numberColour = func(th theme.Theme) []ansi.Color {
	return []ansi.Color{th.Muted, th.Info, th.Success, th.Error, th.Primary, th.Warning, th.Secondary, th.Warning, th.Error}
}

func (m model) cellStr(x, y int, th theme.Theme) string {
	c := m.g.at(x, y)
	var s string
	switch {
	case c.open && c.mine && c.boom:
		s = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Error).Render("* ")
	case c.open && c.mine:
		s = ansi.NewStyle().Foreground(th.Error).Render("* ")
	case c.flag && m.g.state == lost && !c.mine:
		s = ansi.NewStyle().Foreground(th.Error).Render("x ")
	case c.flag:
		s = ansi.NewStyle().Foreground(th.Warning).Render("⚑ ")
	case !c.open:
		s = ansi.NewStyle().Foreground(th.Muted).Render("■ ")
	case c.adj == 0:
		s = "  "
	default:
		s = ansi.NewStyle().Bold().Foreground(numberColour(th)[c.adj]).Render(itoa(c.adj) + " ")
	}
	switch {
	case x == m.cx && y == m.cy:
		return ansi.NewStyle().Reverse().Render(ansi.StripANSI(s))
	case x == m.hx && y == m.hy && !c.open:
		return ansi.NewStyle().Underline().Render(ansi.StripANSI(s))
	}
	return s
}

func (m model) boardRows(th theme.Theme) []layout.FlexChild {
	var rows []layout.FlexChild
	for y := 0; y < m.g.lv.h; y++ {
		var b strings.Builder
		for x := 0; x < m.g.lv.w; x++ {
			b.WriteString(m.cellStr(x, y, th))
		}
		rows = append(rows, layout.FlexChild{Node: layout.Block(b.String())})
	}
	return rows
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 18)) + a
	}
	return []string{
		key("arrows h j k l", "move the cursor"), key("Space  Enter", "open a cell; on a number, open around it"),
		key("f", "flag or unflag"), key("c", "open around a number whose flags match"),
		key("left click", "open"), key("right click", "flag"), key("middle, double-click", "open around a number"),
		key("1 2 3  n", "pick a size, new game"), key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
	}
}

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) status() string {
	switch {
	case !m.fits():
		return "Too small for this size: needs " + itoa(2*m.g.lv.w+2) + "x" + itoa(m.g.lv.h+chromeRows) + "; try a smaller size"
	case m.msg != "":
		return m.msg
	case m.g.state == fresh:
		return m.g.lv.name + ": " + itoa(m.g.lv.mine) + " mines. The first click is safe"
	}
	return ansi.NewStyle().Faint().Render(itoa(m.g.left) + " cells to open")
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	hints := widgets.Footer(w, widgets.Hint{Key: "←↑↓→", Action: "move"}, widgets.Hint{Key: "space", Action: "open"}, widgets.Hint{Key: "f", Action: "flag"},
		widgets.Hint{Key: "c", Action: "chord"}, widgets.Hint{Key: "n", Action: "new"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	body := layout.Node(layout.Block(""))
	if m.fits() {
		inner := 2 * m.g.lv.w
		rows := m.boardRows(th)
		side := layout.Fixed(layout.Column(0, repeat("│", len(rows))...), layout.Size{W: 1, H: len(rows)})
		mid := layout.Node(layout.Named("board", layout.Column(0, rows...)))
		body = layout.Column(0,
			layout.FlexChild{Node: layout.Block("┌" + strings.Repeat("─", inner) + "┐")},
			layout.FlexChild{Node: layout.Row(0, layout.FlexChild{Node: side}, layout.FlexChild{Node: mid}, layout.FlexChild{Node: side})},
			layout.FlexChild{Node: layout.Block("└" + strings.Repeat("─", inner) + "┘")})
	}
	if m.help {
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, w))})
		}
		body = layout.Column(0, lines...)
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.bar(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), w))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))})
}

func repeat(s string, n int) []layout.FlexChild {
	out := make([]layout.FlexChild, n)
	for i := range out {
		out[i] = layout.FlexChild{Node: layout.Block(s)}
	}
	return out
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
