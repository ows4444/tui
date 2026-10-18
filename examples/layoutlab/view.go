package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return ansi.Truncate(s, w)
}

func (m model) showRight() bool { return m.size().W >= 80 }

func (m model) tabs(th theme.Theme) layout.Node {
	kids := []layout.FlexChild{}
	for i, n := range modeNames {
		id := "mode-" + strconv.Itoa(i)
		label, st := " "+strconv.Itoa(i+1)+" "+n+" ", ansi.NewStyle().Foreground(th.Muted)
		switch {
		case mode(i) == m.mode:
			label, st = "▸"+strconv.Itoa(i+1)+" "+n+" ", ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) paramRows(th theme.Theme, h int) layout.Node {
	var rows []layout.FlexChild
	bw := 10
	top := max(0, m.sel-(h-1)) // keep the selected setting in view
	for i, p := range m.params() {
		if i < top || i >= top+h {
			continue
		}
		s := strconv.Itoa(i)
		v := m.vals[p.key]
		span := max(1, p.hi-p.lo)
		filled := (v - p.lo) * bw / span
		bar := ansi.NewStyle().Foreground(th.Primary).Render(strings.Repeat("█", filled)) + ansi.NewStyle().Faint().Render(strings.Repeat("░", bw-filled))
		mark, st := " ", ansi.NewStyle()
		switch {
		case i == m.sel:
			mark, st = "▸", ansi.NewStyle().Bold().Foreground(th.Primary)
		case m.hover == "row-"+s:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		rows = append(rows, layout.FlexChild{Node: layout.Row(0,
			layout.FlexChild{Node: layout.Named("row-"+s, layout.Block(st.Render(mark+pad(p.label, 17))))},
			layout.FlexChild{Node: layout.Named("dec-"+s, layout.Block("◂"))},
			layout.FlexChild{Node: layout.Named("bar-"+s, layout.Fixed(layout.Block(bar), layout.Size{W: bw, H: 1}))},
			layout.FlexChild{Node: layout.Named("inc-"+s, layout.Block("▸"))},
			layout.FlexChild{Node: layout.Block(" " + st.Render(p.text(v)))})})
	}
	return layout.Fixed(layout.Column(0, rows...), layout.Size{W: leftW, H: len(rows)})
}

func (m model) rectLines(th theme.Theme, s layout.Size, h int) []string {
	boxes := m.boxes(th, s)
	if len(boxes) == 0 {
		return []string{" This container reports no rectangles for its children."}
	}
	var out []string
	sel := "c" + strconv.Itoa(m.vals["child"])
	for _, p := range boxes {
		line := fmt.Sprintf(" %-6s x=%-3d y=%-3d w=%-3d h=%-3d", p.Name, p.Rect.X, p.Rect.Y, p.Rect.W, p.Rect.H)
		if p.Rect.W == 0 || p.Rect.H == 0 {
			line += " hidden"
		}
		if m.mode == mFlex && p.Name == sel {
			line = ansi.NewStyle().Bold().Foreground(th.Primary).Render(line)
		}
		out = append(out, line)
	}
	if len(out) > h {
		out = append(out[:h-1], fmt.Sprintf(" … %d more", len(out)-h+1))
	}
	return out
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14)) + a
	}
	return []string{
		key("1 to 4  Tab", "Flex, Grid, Wrap, Anchor"), key("Up Down", "pick a setting"), key("Left Right", "change it (PgUp PgDn: five)"),
		key("Home", "put the setting back"), key("r", "reset the whole mode"), key("c", "rectangles or code"),
		key("mouse", "click a mode, slider or box; drag a slider; wheel nudges"), key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
	}
}

func (m model) right(th theme.Theme, bodyH int) layout.Node {
	w := m.size().W - leftW - 1
	panelMin := 5
	s := m.frame(w-2, bodyH-2-panelMin-1)
	stage := layout.Fixed(m.stage(th), s)
	top := "┌" + strings.Repeat("─", s.W) + "┐"
	label := fmt.Sprintf(" %dx%d ", s.W, s.H)
	if s.W != m.vals["width"] || s.H != m.vals["height"] {
		label = fmt.Sprintf(" %dx%d (cut from %dx%d) ", s.W, s.H, m.vals["width"], m.vals["height"])
	}
	if ansi.Width(label)+2 <= s.W {
		top = "┌" + label + strings.Repeat("─", s.W-ansi.Width(label)) + "┐"
	}
	side := func() layout.FlexChild {
		col := make([]layout.FlexChild, s.H)
		for i := range col {
			col[i] = layout.FlexChild{Node: layout.Block("│")}
		}
		return layout.FlexChild{Node: layout.Fixed(layout.Column(0, col...), layout.Size{W: 1, H: s.H})}
	}
	frame := layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(top))},
		layout.FlexChild{Node: layout.Row(0, side(), layout.FlexChild{Node: stage}, side())},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("└" + strings.Repeat("─", s.W) + "┘"))})
	panelH := bodyH - (s.H + 2) - 1
	head, lines := "Rectangles  [c] code", m.rectLines(th, s, max(1, panelH))
	if m.code {
		head, lines = "Code  [c] rectangles", strings.Split(strings.TrimRight(code(m.mode, m.vals), "\n"), "\n")
	}
	rows := []layout.FlexChild{
		{Node: frame},
		{Node: layout.Named("view", layout.Block(ansi.NewStyle().Bold().Foreground(th.Muted).Render(head)))},
	}
	for i := 0; i < max(0, panelH); i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		rows = append(rows, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, w))})
	}
	return layout.Fixed(layout.Column(0, rows...), layout.Size{W: w, H: bodyH})
}

func (m model) status() string {
	if m.msg != "" {
		return m.msg
	}
	p := m.params()[m.sel]
	return ansi.NewStyle().Faint().Render(p.label + " = " + p.text(m.vals[p.key]) + "  ·  " + modeNames[m.mode])
}

func (m model) screen() layout.Node {
	th := m.theme()
	s := m.size()
	bodyH := max(1, s.H-chromeRows)
	hints := widgets.Footer(s.W, widgets.Hint{Key: "1-4", Action: "mode"}, widgets.Hint{Key: "↑↓", Action: "setting"}, widgets.Hint{Key: "←→", Action: "change"},
		widgets.Hint{Key: "c", Action: "code"}, widgets.Hint{Key: "r", Action: "reset"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	var body layout.Node
	switch {
	case m.help:
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, s.W))})
		}
		body = layout.Column(0, lines...)
	case !m.showRight():
		body = m.paramRows(th, bodyH)
	default:
		sep := make([]layout.FlexChild, bodyH)
		for i := range sep {
			sep[i] = layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("│"))}
		}
		body = layout.Row(0, layout.FlexChild{Node: m.paramRows(th, bodyH)},
			layout.FlexChild{Node: layout.Fixed(layout.Column(0, sep...), layout.Size{W: 1, H: bodyH})},
			layout.FlexChild{Node: m.right(th, bodyH)})
	}
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("layoutlab"))},
		layout.FlexChild{Node: m.tabs(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
