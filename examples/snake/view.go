package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func (m model) button(id, label string, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == id {
		st = ansi.NewStyle().Foreground(th.Primary).Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render("["+label+"]")))}
}

func (m model) bar(th theme.Theme) layout.Node {
	w := m.size().W
	pause := "pause"
	if m.g.state == paused {
		pause = "resume"
	}
	kids := []layout.FlexChild{
		{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("snake"))},
		{Node: layout.Block("score " + strconv.Itoa(m.g.score) + "  best " + strconv.Itoa(m.best) + "  speed " + strconv.Itoa(m.level+1) + "/5")},
		m.button("pause", pause, th), m.button("restart", "restart", th), m.button("slower", "-", th), m.button("faster", "+", th),
	}
	return layout.Fixed(layout.Row(2, kids...), layout.Size{W: w, H: 1})
}

func (m model) cellStr(p pt, th theme.Theme) string {
	switch {
	case p == m.g.body[0]:
		return ansi.NewStyle().Bold().Foreground(th.Primary).Render("██")
	case m.g.on(p):
		return ansi.NewStyle().Foreground(th.Success).Render("▓▓")
	case p == m.g.food:
		return ansi.NewStyle().Foreground(th.Error).Render("◆ ")
	}
	return ansi.NewStyle().Faint().Render("· ")
}

func (m model) arenaRows(th theme.Theme) []string {
	rows := make([]string, m.g.h)
	for y := range rows {
		var b strings.Builder
		for x := 0; x < m.g.w; x++ {
			b.WriteString(m.cellStr(pt{x, y}, th))
		}
		rows[y] = b.String()
	}
	return rows
}

// overlay writes a centred message over the upper part of the arena, clear of
// the snake in the middle.
func (m model) overlay(rows []string, lines []string, th theme.Theme) []string {
	w := 2 * m.g.w
	top := (len(rows) - len(lines)) / 4
	for i, l := range lines {
		if top+i < 0 || top+i >= len(rows) {
			continue
		}
		l = " " + ansi.Truncate(l, w-2) + " "
		pad := (w - ansi.Width(l)) / 2
		rows[top+i] = strings.Repeat(" ", pad) + ansi.NewStyle().Bold().Reverse().Render(l) + strings.Repeat(" ", w-pad-ansi.Width(l))
	}
	return rows
}

func (m model) banner(th theme.Theme) []string {
	switch m.g.state {
	case ready:
		return []string{"Press an arrow key or click", "to start"}
	case paused:
		return []string{"Paused", "p or Space resumes"}
	case over:
		return []string{"Game over: " + strconv.Itoa(m.g.score), "r plays again"}
	case won:
		return []string{"You filled the board", "r plays again"}
	}
	return nil
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 16)) + a
	}
	return []string{
		key("arrows w a s d", "steer (also h j k l)"), key("click in the arena", "turn toward the pointer"),
		key("p  Space", "pause or resume"), key("r", "restart"), key("+  -", "faster, slower; an apple is worth the speed"),
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
	if m.msg != "" {
		return m.msg
	}
	if !m.fits() {
		return "Window too small for the board; make it bigger"
	}
	return ansi.NewStyle().Faint().Render("eat ◆ to grow; walls and your own body end the game")
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	w := m.size().W
	hints := widgets.Footer(w, widgets.Hint{Key: "←↑↓→", Action: "steer"}, widgets.Hint{Key: "p", Action: "pause"}, widgets.Hint{Key: "r", Action: "restart"},
		widgets.Hint{Key: "+/-", Action: "speed"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	var body layout.Node
	switch {
	case !m.fits():
		body = layout.Block("")
	default:
		rows := m.arenaRows(th)
		if m.help {
			rows = m.overlay(rows, m.helpLines(th), th)
		} else if b := m.banner(th); b != nil {
			rows = m.overlay(rows, b, th)
		}
		var kids []layout.FlexChild
		inner := 2 * m.g.w
		kids = append(kids, layout.FlexChild{Node: layout.Block("┌" + strings.Repeat("─", inner) + "┐")})
		var lines []layout.FlexChild
		for _, r := range rows {
			lines = append(lines, layout.FlexChild{Node: layout.Block(r)})
		}
		arena := layout.Named("arena", layout.Column(0, lines...))
		side := layout.Fixed(layout.Column(0, repeat("│", len(rows))...), layout.Size{W: 1, H: len(rows)})
		kids = append(kids, layout.FlexChild{Node: layout.Row(0, layout.FlexChild{Node: side}, layout.FlexChild{Node: arena}, layout.FlexChild{Node: side})})
		kids = append(kids, layout.FlexChild{Node: layout.Block("└" + strings.Repeat("─", inner) + "┘")})
		body = layout.Column(0, kids...)
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
