package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func palette(th theme.Theme) [numColours + 1]ansi.Color {
	return [numColours + 1]ansi.Color{nil, th.Primary, th.Secondary, th.Success, th.Warning, th.Error, th.Info, th.Text, th.Muted}
}

func (m model) tools(th theme.Theme) layout.Node {
	kids := []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("paint"))}}
	for i := 0; i < int(numTools); i++ {
		id := "tool-" + strconv.Itoa(i)
		label, st := " "+toolNames[i]+" ", ansi.NewStyle().Foreground(th.Muted)
		switch {
		case tool(i) == m.tool:
			label, st = "▸"+toolNames[i]+" ", ansi.NewStyle().Bold().Foreground(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("brush " + strconv.Itoa(m.brush)))})
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) swatches(th theme.Theme) layout.Node {
	pal := palette(th)
	var kids []layout.FlexChild
	for i := 1; i <= numColours; i++ {
		id := "col-" + strconv.Itoa(i)
		sw := ansi.NewStyle().Foreground(pal[i]).Render("██")
		label := " " + strconv.Itoa(i) + sw + " "
		if uint8(i) == m.colour {
			label = "[" + strconv.Itoa(i) + sw + "]"
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(label))})
	}
	return layout.Fixed(layout.Row(0, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) pixelStr(g grid, p pt, pal [numColours + 1]ansi.Color) string {
	c := g.at(p)
	s := "  "
	if c != 0 {
		s = ansi.NewStyle().Foreground(pal[c]).Render("██")
	}
	cursor := m.kbd && p == m.cur
	hover := !m.kbd && p.x == m.hx && p.y == m.hy
	switch {
	case cursor || hover:
		if c == 0 {
			return ansi.NewStyle().Faint().Render("░░")
		}
		return ansi.NewStyle().Foreground(pal[c]).Render("▓▓")
	}
	return s
}

func (m model) canvas(th theme.Theme) layout.Node {
	pal := palette(th)
	g := m.shown(m.cur)
	if !m.kbd && m.hx >= 0 {
		g = m.shown(pt{m.hx, m.hy})
	}
	rows := make([]layout.FlexChild, g.h)
	for y := range rows {
		var b strings.Builder
		for x := 0; x < g.w; x++ {
			b.WriteString(m.pixelStr(g, pt{x, y}, pal))
		}
		rows[y] = layout.FlexChild{Node: layout.Block(b.String())}
	}
	return layout.Named("canvas", layout.Column(0, rows...))
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 16)) + a
	}
	return []string{
		key("left drag", "draw; right drag erases"), key("wheel", "brush size"),
		key("p e l r f", "pen, eraser, line, rectangle, fill"), key("1 to 8", "colour"),
		key("arrows", "move the cursor (Shift: five at a time)"), key("Space  Enter", "start and end a stroke or shape"),
		key("[  ]", "smaller, larger brush"), key("u  U", "undo, redo"), key("c  s", "clear, save as text"),
		key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
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
	case m.msg != "":
		return m.msg
	case m.anchor != nil:
		return "Drag to the other corner, release to draw"
	}
	pos := ""
	switch {
	case m.kbd:
		pos = "  ·  " + strconv.Itoa(m.cur.x) + "," + strconv.Itoa(m.cur.y)
	case m.hx >= 0:
		pos = "  ·  " + strconv.Itoa(m.hx) + "," + strconv.Itoa(m.hy)
	}
	state := "saved"
	if m.dirty {
		state = "not saved"
	}
	return ansi.NewStyle().Faint().Render(strconv.Itoa(m.g.w) + "x" + strconv.Itoa(m.g.h) + " pixels  ·  " + state + pos)
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	inner := 2 * m.g.w
	hints := widgets.Footer(w, widgets.Hint{Key: "p e l r f", Action: "tool"}, widgets.Hint{Key: "1-8", Action: "colour"}, widgets.Hint{Key: "space", Action: "draw"},
		widgets.Hint{Key: "u", Action: "undo"}, widgets.Hint{Key: "s", Action: "save"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	body := layout.Node(layout.Row(0,
		layout.FlexChild{Node: layout.Fixed(layout.Column(0, repeat("│", m.g.h)...), layout.Size{W: 1, H: m.g.h})},
		layout.FlexChild{Node: m.canvas(th)},
		layout.FlexChild{Node: layout.Fixed(layout.Column(0, repeat("│", m.g.h)...), layout.Size{W: 1, H: m.g.h})}))
	if m.help {
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, w))})
		}
		body = layout.Column(0, lines...)
	}
	top, bottom := layout.Node(layout.Block("┌"+strings.Repeat("─", inner)+"┐")), layout.Node(layout.Block("└"+strings.Repeat("─", inner)+"┘"))
	if m.help {
		top, bottom = layout.Block(""), layout.Block("")
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.tools(th)},
		layout.FlexChild{Node: m.swatches(th)},
		layout.FlexChild{Node: top},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: bottom},
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
