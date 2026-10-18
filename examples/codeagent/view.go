package main

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/spinner"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

func wrap(text string, w int) []string {
	w = max(8, w)
	var out []string
	for _, para := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			switch {
			case line == "":
				line = word
			case ansi.Width(line)+1+ansi.Width(word) <= w:
				line += " " + word
			default:
				out = append(out, line)
				line = word
			}
		}
		out = append(out, line)
	}
	return out
}

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

var frames = spinner.Frames()

func (m model) spin() string {
	if m.still {
		return "…"
	}
	return frames[(len(m.blocks)+m.wait+m.tokensTick())%len(frames)]
}

// tokensTick varies the spinner frame from state that changes each tick.
func (m model) tokensTick() int {
	n := 0
	for _, b := range m.blocks {
		n += b.ticks + b.shown
	}
	return n / 2
}

func block1(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func (m model) cardStatus(b block, th theme.Theme) string {
	switch b.st {
	case pending:
		return ansi.NewStyle().Bold().Foreground(th.Warning).Render("⚠ " + riskNames[b.t.risk] + " · waiting for you")
	case running:
		return ansi.NewStyle().Foreground(th.Primary).Render(m.spin() + " running")
	case done:
		s := "✓ done"
		if b.auto {
			s += " (ran without asking)"
		}
		return ansi.NewStyle().Foreground(th.Success).Render(s)
	case failed:
		return ansi.NewStyle().Foreground(th.Error).Render("✗ failed")
	}
	return ansi.NewStyle().Foreground(th.Error).Render("✗ denied")
}

func diffLine(l string, th theme.Theme) string {
	switch {
	case strings.HasPrefix(l, "+"):
		return ansi.NewStyle().Foreground(th.Success).Render(l)
	case strings.HasPrefix(l, "-"):
		return ansi.NewStyle().Foreground(th.Error).Render(l)
	case strings.HasPrefix(l, "@"):
		return ansi.NewStyle().Foreground(th.Info).Render(l)
	}
	return ansi.NewStyle().Faint().Render(l)
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

// toolLines draws a tool call as a card.
func (m model) toolLines(i int, b block, th theme.Theme, w int) []layout.FlexChild {
	edge := ansi.NewStyle().Faint()
	head := edge.Render("┌ ") + ansi.NewStyle().Bold().Render(b.t.icon+" "+b.t.name) + " " + ansi.NewStyle().Faint().Render(b.t.arg)
	status := m.cardStatus(b, th)
	gap := max(1, w-ansi.Width(head)-ansi.Width(status)-1)
	if m.hover == "card-"+itoa(i) {
		head = ansi.NewStyle().Underline().Render(ansi.StripANSI(head))
	}
	out := []layout.FlexChild{{Node: layout.Named("card-"+itoa(i), layout.Block(ansi.Truncate(head+strings.Repeat(" ", gap)+status, w)))}}
	body := func(s string) {
		s = strings.ReplaceAll(s, "\t", "    ")
		out = append(out, block1(ansi.Truncate(edge.Render("│ ")+s, w)))
	}
	limit := 6
	if b.open {
		limit = 1 << 20
	}
	for k, l := range b.t.diff {
		if k >= limit {
			body(ansi.NewStyle().Faint().Render("… " + itoa(len(b.t.diff)-limit) + " more lines (d shows all)"))
			break
		}
		body(diffLine(l, th))
	}
	if b.st == running {
		body(ansi.NewStyle().Foreground(th.Primary).Render(strings.Repeat("█", b.ticks*12/toolTicks)) + ansi.NewStyle().Faint().Render(strings.Repeat("░", 12-b.ticks*12/toolTicks)))
	}
	rlimit := 2
	if b.open {
		rlimit = 1 << 20
	}
	for k, l := range b.result {
		if k >= rlimit {
			body(ansi.NewStyle().Faint().Render("… " + itoa(len(b.result)-rlimit) + " more lines (click the title to open)"))
			break
		}
		body(l)
	}
	if b.st == pending {
		kids := []layout.FlexChild{{Node: layout.Block(edge.Render("│ "))}, m.button("btn-approve", "y Approve", th, true), {Node: layout.Block(" ")}, m.button("btn-deny", "n Deny", th, false)}
		if b.t.risk != high {
			kids = append(kids, layout.FlexChild{Node: layout.Block(" ")}, m.button("btn-always", "a Always allow "+b.t.name, th, false))
		}
		if len(b.t.diff) > 0 {
			kids = append(kids, layout.FlexChild{Node: layout.Block(" ")}, m.button("btn-diff", "d Diff", th, false))
		}
		out = append(out, layout.FlexChild{Node: layout.Fixed(layout.Row(0, kids...), layout.Size{W: w, H: 1})})
	}
	return append(out, block1(edge.Render("└")))
}

// lines is the whole transcript, one node per row.
func (m model) lines() []layout.FlexChild {
	th := m.theme()
	w := m.size().W
	var out []layout.FlexChild
	for i, b := range m.blocks {
		if i > 0 {
			out = append(out, block1(""))
		}
		switch b.kind {
		case bUser:
			for k, l := range wrap(b.text, w-3) {
				prefix := "  "
				if k == 0 {
					prefix = ansi.NewStyle().Bold().Foreground(th.Primary).Render("❯ ")
				}
				out = append(out, block1(prefix+ansi.NewStyle().Bold().Render(l)))
			}
		case bAgent:
			text := string([]rune(b.text)[:min(len([]rune(b.text)), b.shown)])
			if i == len(m.blocks)-1 && m.phase == saying {
				text += "▌"
			}
			for k, l := range wrap(text, w-3) {
				prefix := "  "
				if k == 0 {
					prefix = ansi.NewStyle().Foreground(th.Secondary).Render("● ")
				}
				out = append(out, block1(prefix+l))
			}
		case bNote:
			for _, l := range wrap(b.text, w-3) {
				out = append(out, block1(ansi.NewStyle().Faint().Render("· "+l)))
			}
		case bTool:
			out = append(out, m.toolLines(i, b, th, w)...)
		}
	}
	return out
}

func (m model) stateText() string {
	switch m.phase {
	case thinking:
		return m.spin() + " thinking"
	case saying:
		return m.spin() + " writing"
	case toolRunning:
		return m.spin() + " running a tool"
	case waiting:
		return "⚠ waiting for your approval"
	}
	return "ready"
}

func (m model) status(th theme.Theme) string {
	w := m.size().W
	if m.msg != "" {
		return ansi.Truncate(m.msg, w)
	}
	parts := []string{m.stateText(), itoa(m.tokens) + " tokens", "scripted-1"}
	if len(m.always) > 0 {
		parts = append(parts, "always allowed: "+strings.Join(sortedKeys(m.always), ","))
	}
	if m.off > 0 {
		parts = append(parts, "↓ "+itoa(m.off)+" lines below (PgDn)")
	}
	return ansi.NewStyle().Faint().Render(ansi.Truncate(strings.Join(parts, "  ·  "), w))
}

func sortedKeys(set map[string]bool) []string {
	var out []string
	for k := range set {
		out = append(out, k)
	}
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (m model) promptRow(th theme.Theme) string {
	w := m.size().W - 2
	rs := m.prompt
	start := max(0, m.cur-(w-2)) // keep the cursor in view
	rs = rs[min(start, len(rs)):]
	cur := m.cur - start
	before := string(rs[:min(cur, len(rs))])
	at, after := " ", ""
	if cur < len(rs) {
		at, after = string(rs[cur]), string(rs[cur+1:])
	}
	label := ansi.NewStyle().Bold().Foreground(th.Primary).Render("❯ ")
	if m.asking() {
		label = ansi.NewStyle().Bold().Foreground(th.Warning).Render("? ")
	}
	if len(m.prompt) == 0 && !m.asking() {
		return label + ansi.NewStyle().Reverse().Render(" ") + ansi.NewStyle().Faint().Render("Ask the agent… (/help)")
	}
	if len(m.prompt) == 0 {
		return label + ansi.NewStyle().Reverse().Render(" ") + ansi.NewStyle().Faint().Render("y approve · n deny · a always · d diff")
	}
	return ansi.Truncate(label+before+ansi.NewStyle().Reverse().Render(at)+after, m.size().W)
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 16)) + a
	}
	return []string{
		key("Enter", "send; or approve a waiting call when the prompt is empty"), key("y  n  a  d", "approve, deny, always allow, whole diff (prompt empty)"),
		key("Esc", "deny a waiting call, stop a reply, clear the prompt"), key("Up  Down", "earlier prompts"), key("PgUp  PgDn", "scroll the transcript"),
		key("Ctrl+L", "clear the transcript"), key("Ctrl+C", "stop; quit when idle"), key("Ctrl+D  /quit", "quit"),
		key("/clear /history", "/allow /reset"), key("click", "the buttons, or a card's title to open its output"), "", " Press any key to close.",
	}
}

func (m model) screen() layout.Node {
	th := m.theme()
	s := m.size()
	n := max(1, s.H-chromeRows)
	hints := widgets.Footer(s.W, widgets.Hint{Key: "enter", Action: "send"}, widgets.Hint{Key: "y/n", Action: "approve/deny"}, widgets.Hint{Key: "↑↓", Action: "history"},
		widgets.Hint{Key: "pgup", Action: "scroll"}, widgets.Hint{Key: "esc", Action: "stop"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "ctrl+d", Action: "quit"})
	var rows []layout.FlexChild
	if m.help {
		for _, l := range m.helpLines(th) {
			rows = append(rows, block1(ansi.Truncate(l, s.W)))
		}
	} else {
		all := m.lines()
		end := len(all) - m.off
		start := max(0, end-n)
		if end > 0 {
			rows = all[start:min(end, len(all))]
		}
	}
	for len(rows) < n {
		rows = append(rows, block1(""))
	}
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("codeagent") + ansi.NewStyle().Faint().Render("  ~/tui  ·  layout/")
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, s.W))},
		layout.FlexChild{Node: layout.Fixed(layout.Column(0, rows[:n]...), layout.Size{W: s.W, H: n})},
		layout.FlexChild{Node: layout.Block(m.status(th))},
		layout.FlexChild{Node: layout.Named("prompt", layout.Block(m.promptRow(th)))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
