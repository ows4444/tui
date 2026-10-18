package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func fit(s string, w int) string {
	s = ansi.Truncate(s, w)
	if n := ansi.Width(s); n < w {
		s += strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) tabBar(p int, w int, th theme.Theme) layout.Node {
	focused := p == m.focus
	var kids []layout.FlexChild
	for t, name := range paneTabs[p] {
		id := "tab-" + strconv.Itoa(p) + "-" + strconv.Itoa(t)
		label, st := " "+name+" ", ansi.NewStyle().Foreground(th.Muted)
		switch {
		case t == m.tab[p] && focused:
			label, st = "▸"+name+" ", ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case t == m.tab[p]:
			label, st = "▸"+name+" ", ansi.NewStyle().Bold().Foreground(th.Text)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	pos := ansi.NewStyle().Faint().Render(strconv.Itoa(m.off[p][m.tab[p]]+1) + "/" + strconv.Itoa(max(1, len(m.lines(p)))) + " ")
	kids = append(kids, layout.FlexChild{Node: layout.Block(""), Grow: 1}, layout.FlexChild{Node: layout.Block(pos)})
	return layout.Fixed(layout.Row(0, kids...), layout.Size{W: w, H: 1})
}

// pane draws pane p in its rectangle: the tab bar, then the text.
func (m model) pane(p int, r layout.Rect, th theme.Theme) layout.Node {
	rows := []layout.FlexChild{{Node: m.tabBar(p, r.W, th)}}
	lines := m.lines(p)
	off := m.off[p][m.tab[p]]
	for i := 0; i < r.H-1; i++ {
		l := ""
		if off+i < len(lines) {
			l = lines[off+i]
		}
		rows = append(rows, layout.FlexChild{Node: layout.Block(fit(l, r.W))})
	}
	return layout.Named("pane-"+strconv.Itoa(p), layout.Fixed(layout.Column(0, rows...), layout.Size{W: r.W, H: r.H}))
}

func (m model) divStyle(id string, th theme.Theme) ansi.Style {
	switch {
	case m.drag != noDrag && ((id == "vdiv") == (m.drag == dragV)):
		return ansi.NewStyle().Bold().Foreground(th.Primary)
	case m.hover == id:
		return ansi.NewStyle().Foreground(th.Primary)
	}
	return ansi.NewStyle().Foreground(th.Muted)
}

func (m model) vdiv(g geometry, th theme.Theme) layout.Node {
	r := g.vdiv
	ch := "│"
	if m.drag == dragV {
		ch = "┃"
	}
	if g.strip[0] {
		ch = "▸"
	}
	rows := make([]layout.FlexChild, r.H)
	for i := range rows {
		rows[i] = layout.FlexChild{Node: layout.Block(m.divStyle("vdiv", th).Render(ch))}
	}
	return layout.Named("vdiv", layout.Fixed(layout.Column(0, rows...), layout.Size{W: 1, H: r.H}))
}

func (m model) hdiv(g geometry, th theme.Theme) layout.Node {
	r := g.hdiv
	line := strings.Repeat("─", r.W)
	if m.drag == dragH {
		line = strings.Repeat("━", r.W)
	}
	if g.strip[1] {
		line = "▴ " + paneNames[2] + " " + strings.Repeat("─", max(0, r.W-len(paneNames[2])-3))
	}
	return layout.Named("hdiv", layout.Fixed(layout.Block(m.divStyle("hdiv", th).Render(ansi.Truncate(line, r.W))), layout.Size{W: r.W, H: 1}))
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(fit(k, 16)) + a
	}
	return []string{
		key("Tab  Shift+Tab", "move between panes"), key(".  ,", "next, previous tab"),
		key("↑ ↓ PgUp PgDn", "scroll the focused pane"), key("[  ]", "move the vertical divider"),
		key("-  =", "move the horizontal divider"), key("c", "collapse the explorer or the panel"),
		key("z  Esc", "zoom the focused pane, back"), key("0", "reset the layout"),
		key("drag a divider", "resize; double-click collapses"), key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
	}
}

func (m model) status() string {
	g, w := m.geo(), m.size().W
	switch {
	case m.msg != "":
		return m.msg
	case m.drag == dragV:
		return "Explorer " + strconv.Itoa(g.pane[0].W) + " columns"
	case m.drag == dragH:
		return "Panel " + strconv.Itoa(g.pane[2].H) + " rows"
	case m.size().W < 50 || m.size().H-chromeRows < 12:
		return ansi.NewStyle().Faint().Render(ansi.Truncate("Small window: some panes hidden", w))
	}
	return ansi.NewStyle().Faint().Render(ansi.Truncate(paneNames[m.focus]+" focused  ·  "+paneTabs[m.focus][m.tab[m.focus]], w))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	s := m.size()
	g := m.geo()
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("workspace")
	if m.zoom {
		title += ansi.NewStyle().Faint().Render("  zoomed: z or Esc goes back")
	}
	hints := widgets.Footer(s.W, widgets.Hint{Key: "tab", Action: "pane"}, widgets.Hint{Key: ".,", Action: "tab"}, widgets.Hint{Key: "[ ]", Action: "resize"},
		widgets.Hint{Key: "c", Action: "collapse"}, widgets.Hint{Key: "z", Action: "zoom"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	var body layout.Node
	if m.help {
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, s.W))})
		}
		body = layout.Column(0, lines...)
	} else {
		body = m.workspace(g, th)
	}
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, s.W))},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) workspace(g geometry, th theme.Theme) layout.Node {
	var cols []layout.FlexChild
	if g.shown(0) {
		cols = append(cols, layout.FlexChild{Node: m.pane(0, g.pane[0], th)})
	}
	if g.vdiv.W > 0 {
		cols = append(cols, layout.FlexChild{Node: m.vdiv(g, th)})
	}
	var right []layout.FlexChild
	if g.shown(1) {
		right = append(right, layout.FlexChild{Node: m.pane(1, g.pane[1], th)})
	}
	if g.hdiv.W > 0 {
		right = append(right, layout.FlexChild{Node: m.hdiv(g, th)})
	}
	if g.shown(2) {
		right = append(right, layout.FlexChild{Node: m.pane(2, g.pane[2], th)})
	}
	if m.zoom {
		for p := 0; p < 3; p++ {
			if g.shown(p) {
				return m.pane(p, g.pane[p], th)
			}
		}
	}
	rw := m.size().W - g.pane[1].X
	cols = append(cols, layout.FlexChild{Node: layout.Fixed(layout.Column(0, right...), layout.Size{W: rw, H: max(1, m.size().H-chromeRows)})})
	return layout.Row(0, cols...)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
