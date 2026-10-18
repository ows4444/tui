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

func (m model) rightW() int { return m.size().W - leftW - 1 }

func (m model) showRight() bool { return m.size().W >= 70 }

func (m model) tokenRows(ui theme.Theme) layout.Node {
	var rows []layout.FlexChild
	for i, tk := range tokens {
		id := "tok-" + strconv.Itoa(i)
		c := toRGB(tk.get(m.t))
		mark := " "
		if m.changedMark(i) {
			mark = "*"
		}
		sw := ansi.NewStyle().Foreground(c).Render("██")
		label := fmt.Sprintf("%s %s %-12s %s", mark, sw, tk.label, hex(c))
		st := ansi.NewStyle()
		switch {
		case i == m.sel:
			label = "▸" + label[1:]
			st = ansi.NewStyle().Bold().Foreground(ui.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(ui.Primary).Underline()
		}
		rows = append(rows, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(pad(label, leftW))))})
	}
	return layout.Fixed(layout.Column(0, rows...), layout.Size{W: leftW, H: len(tokens)})
}

func (m model) button(id, label string, ui theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(ui.Muted)
	if m.hover == id {
		st = ansi.NewStyle().Foreground(ui.Primary).Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render("["+label+"]")))}
}

func (m model) bar(c int, ui theme.Theme) layout.Node {
	rgb := m.current()
	v := int([]uint8{rgb.R, rgb.G, rgb.B}[c])
	bw := max(8, min(32, m.rightW()-12))
	filled := v * bw / 255
	tone := []ansi.Color{ui.Error, ui.Success, ui.Info}[c]
	name := "RGB"[c : c+1]
	label := name + " "
	if m.focus == c {
		label = "▸" + name
		label += " "
	}
	bar := ansi.NewStyle().Foreground(tone).Render(strings.Repeat("█", filled)) + ansi.NewStyle().Faint().Render(strings.Repeat("░", bw-filled))
	return layout.Row(0,
		layout.FlexChild{Node: layout.Block(label)},
		layout.FlexChild{Node: layout.Named("bar-"+strconv.Itoa(c), layout.Fixed(layout.Block(bar), layout.Size{W: bw, H: 1}))},
		layout.FlexChild{Node: layout.Block(fmt.Sprintf(" %3d", v))})
}

func (m model) editor(ui theme.Theme) layout.Node {
	w := m.rightW()
	cur := m.current()
	hexText := hex(cur)
	hexStyle := ansi.NewStyle().Bold()
	if m.typing {
		hexText, hexStyle = string(m.edit)+"▏", ansi.NewStyle().Bold().Foreground(ui.Primary).Underline()
	} else if m.focus == fHex {
		hexStyle = hexStyle.Foreground(ui.Primary)
	}
	r := m.ratio()
	grade := "✗ below 4.5:1"
	if r >= 7 {
		grade = "✓ AAA"
	} else if r >= 4.5 {
		grade = "✓ AA"
	}
	head := layout.Row(0,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Bold().Foreground(ui.Primary).Render(tokens[m.sel].label) + "  ")},
		layout.FlexChild{Node: layout.Named("hex", layout.Block(hexStyle.Render(hexText)))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Foreground(cur).Render("  ████") + ansi.NewStyle().Faint().Render(fmt.Sprintf("  %.2f:1 on background  %s", r, grade)))})
	buttons := layout.Row(1,
		m.button("btn-reset", "x reset", ui), m.button("btn-all", "0 all", ui),
		m.button("btn-border", "b border: "+orBase(borderNames[m.sp.border]), ui), m.button("btn-glyphs", "g glyphs: "+orBase(glyphNames[m.sp.glyphs]), ui), m.button("btn-save", "s save", ui))
	return layout.Fixed(layout.Column(0,
		layout.FlexChild{Node: head}, layout.FlexChild{Node: m.bar(0, ui)}, layout.FlexChild{Node: m.bar(1, ui)}, layout.FlexChild{Node: m.bar(2, ui)}, layout.FlexChild{Node: buttons}),
		layout.Size{W: w, H: 5})
}

func orBase(s string) string {
	if s == "" {
		return "base"
	}
	return s
}

func (m model) helpLines(ui theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(ui.Primary).Render(pad(k, 16)) + a
	}
	return []string{
		key("Up Down  j k", "pick a colour"), key("Tab  Shift+Tab", "pick a slider or the hex code"),
		key("Left Right", "change the slider by one (PgUp PgDn: sixteen)"), key("Enter", "type a hex code"),
		key("[ ]  p n", "change the base theme (clears the changes)"), key("x  0", "reset this colour, everything"),
		key("b  g", "cycle the border, the glyph set"), key("s", "save the theme file"),
		key("mouse", "click a colour or slider, drag a slider, wheel nudges"), key("?  Esc", "close this help"), key("q  Ctrl+C", "quit"),
	}
}

func (m model) status() string {
	switch {
	case m.msg != "":
		return m.msg
	case m.err != nil:
		return "Theme error: " + m.err.Error()
	case m.typing:
		return "Type #rrggbb, Enter to apply, Esc to cancel"
	}
	n := m.sp.changes()
	if n == 0 {
		return ansi.NewStyle().Faint().Render("No changes to " + m.sp.base + " yet; saving writes the base alone")
	}
	return ansi.NewStyle().Faint().Render(strconv.Itoa(n) + " changes to " + m.sp.base + "  ·  " + strconv.Itoa(len(m.t.Check(4.5))) + " roles below 4.5:1")
}

func (m model) title(ui theme.Theme) layout.Node {
	name := ansi.NewStyle().Bold().Render(m.sp.base)
	if m.hover == "prev" || m.hover == "next" {
		name = ansi.NewStyle().Bold().Foreground(ui.Primary).Render(m.sp.base)
	}
	kids := []layout.FlexChild{
		{Node: layout.Block(ansi.NewStyle().Bold().Foreground(ui.Primary).Render("themestudio"))},
		{Node: layout.Named("prev", layout.Block("◂"))}, {Node: layout.Block(name)}, {Node: layout.Named("next", layout.Block("▸"))},
		{Node: layout.Block(ansi.NewStyle().Faint().Render(strconv.Itoa(m.baseIndex()+1) + "/" + strconv.Itoa(len(m.names))))},
	}
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) screen() layout.Node {
	ui := theme.DarkTheme()
	s := m.size()
	bodyH := max(1, s.H-chromeRows)
	hints := widgets.Footer(s.W, widgets.Hint{Key: "↑↓", Action: "colour"}, widgets.Hint{Key: "tab", Action: "slider"}, widgets.Hint{Key: "←→", Action: "change"},
		widgets.Hint{Key: "[ ]", Action: "base"}, widgets.Hint{Key: "s", Action: "save"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	var body layout.Node
	switch {
	case m.help:
		var lines []layout.FlexChild
		for _, l := range m.helpLines(ui) {
			lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, s.W))})
		}
		body = layout.Column(0, lines...)
	case !m.showRight():
		body = layout.Column(0, layout.FlexChild{Node: m.tokenRows(ui)})
	default:
		sep := make([]layout.FlexChild, bodyH)
		for i := range sep {
			sep[i] = layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("│"))}
		}
		right := layout.Column(0,
			layout.FlexChild{Node: m.editor(ui)}, layout.FlexChild{Node: layout.Block("")},
			layout.FlexChild{Node: m.preview(ui, bodyH-6)})
		body = layout.Row(0,
			layout.FlexChild{Node: m.tokenRows(ui)},
			layout.FlexChild{Node: layout.Fixed(layout.Column(0, sep...), layout.Size{W: 1, H: bodyH})},
			layout.FlexChild{Node: layout.Fixed(right, layout.Size{W: m.rightW(), H: bodyH})})
	}
	return layout.Column(0,
		layout.FlexChild{Node: m.title(ui)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), s.W))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, s.W)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
