package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// Where the counter's buttons are in its content, counted from its top left.
const (
	counterRow = 1
	minusX     = 3
	plusX      = 16
	resetX     = 3
)

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return ansi.Truncate(s, w)
}

func block(lines []string, w, h int) layout.Node {
	for len(lines) < h {
		lines = append(lines, "")
	}
	for i := range lines[:h] {
		lines[i] = pad(lines[i], w)
	}
	return layout.Block(strings.Join(lines[:h], "\n"))
}

// content builds what goes inside window id, w by h cells.
func (m model) content(id string, w, h int) layout.Node {
	switch id {
	case "clock":
		big := ansi.NewStyle().Bold().Render(m.clock())
		return block([]string{"", "  " + big, "", "  up " + itoa(m.secs) + " s", "  " + itoa(len(m.desk.Windows)) + " windows open"}, w, h)
	case "counter":
		lines := []string{"", "   [ - ]  " + fmt.Sprintf("%4d", m.counter) + "  [ + ]", "", "   [ reset ]", "", "  + - 0 work from the keys too"}
		return block(lines, w, h)
	case "notes":
		var lines []string
		for _, l := range strings.Split(string(m.notes)+"▌", "\n") {
			lines = append(lines, wrap(l, max(4, w-2))...)
		}
		if len(lines) > h {
			lines = lines[len(lines)-h:]
		}
		for i := range lines {
			lines[i] = " " + lines[i]
		}
		return block(lines, w, h)
	}
	end := len(m.logs) - m.logOff
	start := max(0, end-h)
	var lines []string
	for _, l := range m.logs[start:max(start, end)] {
		lines = append(lines, " "+l)
	}
	return block(lines, w, h)
}

func wrap(s string, w int) []string {
	var out []string
	r := []rune(s)
	for len(r) > w {
		out = append(out, string(r[:w]))
		r = r[w:]
	}
	return append(out, string(r))
}

// deskNode is the desktop with every window's content filled in for this frame.
func (m model) deskNode() layout.Node {
	d := m.desk
	d.Windows = append(d.Windows[:0:0], d.Windows...)
	for i := range d.Windows {
		w := &d.Windows[i]
		title := w.Title
		if w.ID == "notes" && m.dirty {
			title += " *"
		}
		w.Title = title
		w.Content = m.content(w.ID, w.Rect.W-2, w.Rect.H-2)
	}
	d.Background = dots(d.Bounds.W, d.Bounds.H)
	return d.LayoutNode()
}

func dots(w, h int) layout.Node {
	line := string([]rune(strings.Repeat("· ", (w+1)/2))[:max(0, w)])
	lines := make([]string, h)
	for i := range lines {
		lines[i] = ansi.NewStyle().Faint().Render(line)
	}
	return layout.Block(strings.Join(lines, "\n"))
}

func (m model) taskbar(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	focus := m.desk.Focused()
	for _, a := range apps {
		id := "tb-" + a.id
		_, open := m.desk.Window(a.id)
		hidden := false
		for _, w := range m.hidden {
			hidden = hidden || w.ID == a.id
		}
		if !open && !hidden {
			continue
		}
		label, st := " ▢ "+a.title+" ", ansi.NewStyle().Foreground(th.Muted)
		switch {
		case hidden:
			label, st = " ▫ "+a.title+" (hidden) ", ansi.NewStyle().Faint()
		case a.id == focus:
			label, st = " ▣ "+a.title+" ", ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	if len(m.closed) > 0 {
		st := ansi.NewStyle().Foreground(th.Muted)
		if m.hover == "tb-add" {
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named("tb-add", layout.Block(st.Render(" + "+itoa(len(m.closed))+" closed ")))})
	}
	kids = append(kids, layout.FlexChild{Node: layout.Block(""), Grow: 1}, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(m.clock() + " "))})
	return layout.Fixed(layout.Row(0, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) dialog(th theme.Theme) layout.Node {
	btn := func(id, label string, primary bool) layout.FlexChild {
		st := ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Muted)
		if primary {
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		}
		return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(" "+label+" ")))}
	}
	body := layout.Column(1,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Bold().Render(m.modal.title))},
		layout.FlexChild{Node: layout.Block(m.modal.text)},
		layout.FlexChild{Node: layout.Row(2, btn("modal-yes", "y Yes", true), btn("modal-no", "n No", false))})
	return layout.BoxNode(layout.NewBox().Border(layout.DoubleBorder()).Padding(1, 3, 1, 3), body)
}

func (m model) helpLines(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 16)) + a
	}
	return []string{
		key("drag a title bar", "move a window"), key("drag an edge", "resize it (left, right and bottom edges)"), key("x in the title", "close the window"),
		key("Tab  Shift+Tab", "change window"), key("F2", "maximize or restore"), key("F3", "bring back a closed window"),
		key("F4  F5", "close, hide to the task bar"), key("F6", "move mode: arrows move, Shift+arrows resize"), key("Alt+arrows", "move without move mode"),
		key("task bar", "click: raise, hide the focused one, show a hidden one"), key("Ctrl+Q", "quit (asks first)"), "", " Press any key to close.",
	}
}

func (m model) status() string {
	if m.msg != "" {
		return m.msg
	}
	f := m.desk.Focused()
	if f == "" {
		return ansi.NewStyle().Faint().Render("No windows; F3 or the + in the task bar brings one back")
	}
	w, _ := m.desk.Window(f)
	return ansi.NewStyle().Faint().Render(fmt.Sprintf("%s  ·  %d,%d  %dx%d", titleOf(f), w.Rect.X, w.Rect.Y, w.Rect.W, w.Rect.H))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	s := m.size()
	hints := widgets.Footer(s.W, widgets.Hint{Key: "tab", Action: "window"}, widgets.Hint{Key: "f2", Action: "maximize"}, widgets.Hint{Key: "f3", Action: "reopen"},
		widgets.Hint{Key: "f4", Action: "close"}, widgets.Hint{Key: "f5", Action: "hide"}, widgets.Hint{Key: "f6", Action: "move"}, widgets.Hint{Key: "f1", Action: "help"})
	d := m.deskRect()
	var body layout.Node = layout.Fixed(m.deskNode(), layout.Size{W: d.W, H: d.H})
	if m.help {
		var lines []layout.FlexChild
		for _, l := range m.helpLines(th) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, s.W))})
		}
		body = layout.Fixed(layout.Column(0, lines...), layout.Size{W: d.W, H: d.H})
	}
	if m.modal != nil {
		body = layout.Stack(body, layout.Anchor(layout.Center, 0, 0, m.dialog(th)))
	}
	title := ansi.NewStyle().Bold().Foreground(th.Primary).Render("desktop") + ansi.NewStyle().Faint().Render("  "+itoa(len(m.desk.Windows))+" windows")
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(ansi.Truncate(title, s.W))},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: m.taskbar(th)},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
