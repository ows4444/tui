package main

import (
	"fmt"
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

func (m model) tabs(th theme.Theme) layout.Node {
	kids := []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("stopwatch"))}}
	for i, n := range tabNames {
		id := fmt.Sprintf("tab-%d", i)
		label, st := fmt.Sprintf(" %d %s ", i+1, n), ansi.NewStyle().Foreground(th.Muted)
		switch {
		case tab(i) == m.tab:
			label, st = fmt.Sprintf("▸%d %s ", i+1, n), ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) button(id, label string, th theme.Theme, primary bool) layout.FlexChild {
	st := ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Muted)
	if primary {
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Success)
	}
	if m.hover == id {
		st = st.Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(" "+label+" ")))}
}

// digits draws a time big when it fits the width and plain otherwise.
func (m model) digits(s string, th theme.Theme) []layout.FlexChild {
	st := ansi.NewStyle().Bold().Foreground(th.Primary)
	if m.size().W < bigWidth(s)+4 {
		return []layout.FlexChild{{Node: layout.Block("  " + st.Render(s))}}
	}
	var out []layout.FlexChild
	for _, r := range big(s) {
		out = append(out, layout.FlexChild{Node: layout.Block("  " + st.Render(r))})
	}
	return out
}

func (m model) stopwatchBody(th theme.Theme, h int) []layout.FlexChild {
	var out []layout.FlexChild
	out = append(out, layout.FlexChild{Node: layout.Block("")})
	out = append(out, m.digits(stamp(m.w.elapsed(m.now())), th)...)
	start := "Start"
	if m.w.running {
		start = "Stop"
	}
	out = append(out, layout.FlexChild{Node: layout.Block("")},
		layout.FlexChild{Node: layout.Row(1, layout.FlexChild{Node: layout.Block(" ")}, m.button("btn-start", "Space "+start, th, true), m.button("btn-lap", "l Lap", th, false), m.button("btn-reset", "r Reset", th, false))},
		layout.FlexChild{Node: layout.Block("")})
	return append(out, m.lapRows(th, h-len(out))...)
}

func (m model) lapRows(th theme.Theme, h int) []layout.FlexChild {
	faint := ansi.NewStyle().Faint()
	if h < 2 {
		return nil
	}
	rows := []layout.FlexChild{{Node: layout.Block(faint.Render(pad("  Lap", 8) + pad("Time", 12) + pad("Total", 12) + "Note"))}}
	if len(m.w.laps) == 0 {
		return append(rows, layout.FlexChild{Node: layout.Block(faint.Render("  No laps yet; press l while it runs"))})
	}
	fast, slow := m.w.extremes()
	vis := h - 1
	var lines []layout.FlexChild
	for i := len(m.w.laps) - 1 - m.lapAt; i >= 0 && len(lines) < vis; i-- {
		l := m.w.laps[i]
		note, st := "", ansi.NewStyle()
		switch i {
		case fast:
			note, st = "fastest", ansi.NewStyle().Foreground(th.Success)
		case slow:
			note, st = "slowest", ansi.NewStyle().Foreground(th.Error)
		}
		lines = append(lines, layout.FlexChild{Node: layout.Block(st.Render(pad(fmt.Sprintf("  %d", i+1), 8) + pad(stamp(l.length), 12) + pad(stamp(l.total), 12) + note))})
	}
	return append(rows, layout.FlexChild{Node: layout.Named("laps", layout.Fixed(layout.Column(0, lines...), layout.Size{W: m.size().W, H: max(1, len(lines))}))})
}

func (m model) timerBody(th theme.Theme) []layout.FlexChild {
	out := []layout.FlexChild{{Node: layout.Block("")}}
	out = append(out, m.digits(m.timer.View(), th)...)
	state := "ready"
	switch {
	case m.timerDone():
		state = "Time's up"
	case m.timer.Running():
		state = "running"
	case m.started:
		state = "paused"
	}
	total := m.timer.Duration
	frac := 0
	if total > 0 {
		frac = int(int64(total-m.remaining()) * 30 / int64(total))
	}
	bar := ansi.NewStyle().Foreground(th.Primary).Render(strings.Repeat("█", frac)) + ansi.NewStyle().Faint().Render(strings.Repeat("░", 30-frac))
	start := "Start"
	if m.timer.Running() {
		start = "Pause"
	}
	out = append(out, layout.FlexChild{Node: layout.Block("")},
		layout.FlexChild{Node: layout.Block("  " + bar + "  " + state)},
		layout.FlexChild{Node: layout.Block("")},
		layout.FlexChild{Node: layout.Row(1, layout.FlexChild{Node: layout.Block(" ")}, m.button("btn-start", "Space "+start, th, true), m.button("btn-reset", "r Reset", th, false))},
		layout.FlexChild{Node: layout.Block("")})
	row := []layout.FlexChild{{Node: layout.Block("  Length")}, m.button("minus", "-", th, false), {Node: layout.Block(fmt.Sprintf("%3d min", m.minutes))}, m.button("plus", "+", th, false), {Node: layout.Block("  ")}}
	for _, p := range presets {
		label := fmt.Sprintf("%d", p)
		row = append(row, m.button(fmt.Sprintf("preset-%d", p), label, th, p == m.minutes))
	}
	return append(out, layout.FlexChild{Node: layout.Row(1, row...)})
}

func (m model) clockBody(th theme.Theme) []layout.FlexChild {
	out := []layout.FlexChild{{Node: layout.Block("")}}
	for _, z := range m.zones {
		out = append(out, layout.FlexChild{Node: layout.Block("  " + ansi.NewStyle().Foreground(th.Muted).Render(pad(z.name, 12)) + ansi.NewStyle().Bold().Render(z.clk.View()))})
	}
	date := m.now().Format("Monday, 2 January 2006")
	return append(out, layout.FlexChild{Node: layout.Block("")}, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("  " + date))})
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14)) + a
	}
	return []string{
		key("1 2 3  Tab", "stopwatch, timer, clock"), key("Space", "start and stop (pause on the timer)"), key("l", "take a lap"), key("r", "reset"),
		key("Up Down  + -", "timer length by a minute; scroll the laps"), key("p", "next timer length: 1, 5, 10, 25"),
		key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
	}
}

func (m model) status() string {
	if m.msg != "" {
		return m.msg
	}
	switch m.tab {
	case tStopwatch:
		return ansi.NewStyle().Faint().Render(fmt.Sprintf("%d laps", len(m.w.laps)))
	case tTimer:
		if m.timerDone() {
			return "Time's up after " + fmt.Sprintf("%d min", m.minutes)
		}
	}
	return ""
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	s := m.size()
	bodyH := max(1, s.H-chromeRows)
	hints := widgets.Footer(s.W, widgets.Hint{Key: "1-3", Action: "tab"}, widgets.Hint{Key: "space", Action: "start/stop"}, widgets.Hint{Key: "l", Action: "lap"},
		widgets.Hint{Key: "r", Action: "reset"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	var rows []layout.FlexChild
	switch {
	case m.help:
		for _, l := range m.helpLines(th) {
			rows = append(rows, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, s.W))})
		}
	case m.tab == tStopwatch:
		rows = m.stopwatchBody(th, bodyH)
	case m.tab == tTimer:
		rows = m.timerBody(th)
	default:
		rows = m.clockBody(th)
	}
	for len(rows) < bodyH {
		rows = append(rows, layout.FlexChild{Node: layout.Block("")})
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.tabs(th)},
		layout.FlexChild{Node: layout.Fixed(layout.Column(0, rows[:bodyH]...), layout.Size{W: s.W, H: bodyH})},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
