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

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

var buttonIDs = []string{"pause", "clear", "coalesce", "kind-0", "kind-1", "kind-2", "kind-3"}

func (m model) hitButton(x, y int) string {
	scr := m.screen()
	for _, id := range buttonIDs {
		if r, ok := layout.RectOf(scr, m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

func (m model) button(id, label string, on bool, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Muted)
	if on {
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))}
}

func mark(on bool) string {
	if on {
		return "●"
	}
	return "○"
}

func (m model) bar(th theme.Theme) layout.Node {
	kids := []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("inputlab"))}}
	for k := kind(0); k < numKinds; k++ {
		kids = append(kids, m.button("kind-"+strconv.Itoa(int(k)), mark(m.shown[k])+" "+kindNames[k]+" "+strconv.Itoa(m.counts[k]), m.shown[k], th))
	}
	pause := "[pause]"
	if m.paused {
		pause = "[resume]"
	}
	kids = append(kids, m.button("coalesce", mark(m.coalesce)+" join motion", m.coalesce, th), m.button("pause", pause, m.paused, th), m.button("clear", "[clear]", false, th))
	return layout.Fixed(layout.Row(2, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) logLines(w, rows int, th theme.Theme) []string {
	vis := m.visible(true)
	end := len(vis) - m.back
	start := max(0, end-rows)
	var out []string
	for _, e := range vis[start:end] {
		tag := ansi.NewStyle().Foreground(th.Muted).Render(fmt.Sprintf("%4d ", e.n))
		body := e.text
		if e.count > 1 {
			body += ansi.NewStyle().Faint().Render(" ×" + strconv.Itoa(e.count))
		}
		out = append(out, ansi.Truncate(tag+body, w))
	}
	if len(vis) == 0 {
		out = append(out, ansi.NewStyle().Faint().Render(" Waiting for input: press a key, click, scroll or paste."))
	}
	return out
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func (m model) detailLines(th theme.Theme) []string {
	head := func(s string) string { return ansi.NewStyle().Bold().Foreground(th.Primary).Render(s) }
	s := m.size()
	out := []string{
		head("Last key"), " " + orDash(m.lastKey), head("Last mouse"), " " + orDash(m.lastMouse), head("Last paste"), " " + orDash(m.lastPaste),
		"", head("State"),
		" window  " + strconv.Itoa(s.W) + "x" + strconv.Itoa(s.H), " focus   " + m.focus, " scheme  " + m.scheme, " events  " + strconv.Itoa(m.total),
		"", head("Terminal"),
	}
	if m.caps == nil {
		return append(out, " probing…")
	}
	c := m.caps
	return append(out,
		" kitty keys      "+yesNo(c.KittyKeyboard), " kitty graphics  "+yesNo(c.KittyGraphics), " sixel           "+yesNo(c.Sixel), " inline images   "+yesNo(c.InlineImages),
		" sync output     "+yesNo(c.SyncOutput), " grapheme width  "+yesNo(c.GraphemeClusters), " styled underl.  "+yesNo(c.StyledUnderline), " notifications   "+yesNo(c.Notifications),
		" version         "+orDash(c.XTVersion))
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 12)) + a
	}
	return []string{
		"Every key, click and paste is logged. These also do something:", "",
		key("Ctrl+C", "quit"), key("Ctrl+P", "pause or resume the log"), key("Ctrl+L", "clear the log"),
		key("PgUp PgDn", "scroll back (pauses the log)"), key("F2 to F5", "show or hide keys, mouse, paste, other"),
		key("F6", "join runs of mouse motion"), key("F1", "close this help"), "",
		"Release and repeat lines need a terminal with the kitty keyboard protocol.",
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
	case m.paused && m.back > 0:
		return "Paused, " + strconv.Itoa(m.back) + " lines back; " + strconv.Itoa(len(m.log)-m.frozen) + " new since the pause"
	case m.paused:
		return "Paused: " + strconv.Itoa(len(m.log)-m.frozen) + " new events since the pause"
	}
	return ansi.NewStyle().Faint().Render(strconv.Itoa(m.total) + " events  ·  last gap " + lastGap(m))
}

func lastGap(m model) string {
	if len(m.log) == 0 {
		return "-"
	}
	return m.log[len(m.log)-1].gap.Round(1e6).String()
}

func (m model) screen() layout.Node {
	th := m.theme()
	s := m.size()
	rows := m.logRows()
	hints := widgets.Footer(s.W, widgets.Hint{Key: "ctrl+c", Action: "quit"}, widgets.Hint{Key: "ctrl+p", Action: "pause"}, widgets.Hint{Key: "ctrl+l", Action: "clear"},
		widgets.Hint{Key: "f2-f6", Action: "filters"}, widgets.Hint{Key: "f1", Action: "help"})
	var left, right []string
	switch {
	case m.help:
		left = m.helpLines(th)
	case s.W >= 90:
		left, right = m.logLines(s.W-detailW-1, rows, th), m.detailLines(th)
	default:
		left = m.logLines(s.W, rows, th)
	}
	lw := s.W
	if len(right) > 0 {
		lw = s.W - detailW - 1
	}
	var lines []layout.FlexChild
	for i := 0; i < rows; i++ {
		l, r := "", ""
		if i < len(left) {
			l = left[i]
		}
		line := pad(ansi.Truncate(l, lw), lw)
		if len(right) > 0 {
			if i < len(right) {
				r = right[i]
			}
			line += ansi.NewStyle().Faint().Render("│") + ansi.Truncate(r, detailW)
		}
		lines = append(lines, layout.FlexChild{Node: layout.Block(line)})
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.bar(th)},
		layout.FlexChild{Node: layout.Column(0, lines...)},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
