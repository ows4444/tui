package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

// fit cuts s to w cells and pads it to w.
func fit(s string, w int) string {
	s = ansi.Truncate(s, w)
	if n := ansi.Width(s); n < w {
		s += strings.Repeat(" ", w-n)
	}
	return s
}

// block is lines cut to w columns and h rows, padded with blanks.
func block(lines []string, w, h int) []string {
	out := make([]string, h)
	for i := range out {
		if i < len(lines) {
			out[i] = fit(lines[i], w)
		} else {
			out[i] = strings.Repeat(" ", w)
		}
	}
	return out
}

// split is s as lines.
func split(s string) []string { return strings.Split(s, "\n") }

// beside puts right to the right of left, left padded to leftW.
func beside(left, right []string, leftW int) []string {
	n := max(len(left), len(right))
	out := make([]string, n)
	for i := range out {
		var l, r string
		if i < len(left) {
			l = left[i]
		}
		if i < len(right) {
			r = right[i]
		}
		out[i] = fit(l, leftW) + "  " + r
	}
	return out
}

func (m model) tabBar(th theme.Theme) string {
	var b strings.Builder
	b.WriteString(" ")
	for i, name := range pageNames {
		label := " " + name + " "
		if i == m.page {
			label = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary).Render(label)
		}
		b.WriteString(label + " ")
	}
	return b.String()
}

func (m model) pageLines(th theme.Theme) []string {
	bold := ansi.NewStyle().Bold().Render
	faint := ansi.NewStyle().Faint().Render
	switch m.page {
	case pageMenus:
		chose := "nothing chosen yet"
		if m.chose != "" {
			chose = "chose " + m.chose
		}
		menu := m.menu.SetTheme(th).View()
		right := []string{bold("Overlays"), faint("d dialog   w drawer"), faint("p popover  t toast"), "", chose}
		return append([]string{bold("Menu"), ""}, beside(split(menu), right, 24)...)
	case pageLoading:
		rows := split(m.list.View())
		bar := split(m.scroll.SetTheme(th).View())
		list := make([]string, len(rows))
		for i := range rows {
			sb := " "
			if i < len(bar) {
				sb = bar[i]
			}
			list[i] = fit(rows[i], 12) + sb
		}
		left := []string{bold("Loading bar"), m.bar.SetTheme(th).View(), "", bold("Skeleton")}
		left = append(left, split(m.skel.SetTheme(th).View())...)
		return beside(left, append([]string{bold("1,000 rows, drawn as needed")}, list...), 30)
	case pageFields:
		mark := func(i int) string {
			if m.field == i {
				return "> "
			}
			return "  "
		}
		valid := "invalid"
		if m.email.Valid() {
			valid = "valid"
		}
		lines := []string{
			bold("Email ") + faint("("+valid+")"), mark(0) + m.email.SetTheme(th).View(),
			bold("Secret"), mark(1) + m.secret.SetTheme(th).View(),
			bold("Word"), mark(2) + m.complete.SetTheme(th).View(),
		}
		return beside(lines, append([]string{bold("Colour")}, split(m.color.SetTheme(th).View())...), 34)
	}
	lines := []string{m.steps.View(th), "", faint("n next   b back")}
	if m.stepErr != "" {
		lines = append(lines, "", ansi.NewStyle().Foreground(th.Error).Render(m.stepErr))
	}
	return lines
}

func (m model) helpLines() []string {
	return []string{
		"F1 to F4, Ctrl+N, Ctrl+P  change the page",
		"Menus     Enter choose   d w p t overlays   Esc close",
		"Loading   Up Down PgUp PgDn scroll the list",
		"Fields    Tab next field   type in the focused one",
		"Wizard    n next   b back",
		"Ctrl+C quits.  Any key closes this help.",
	}
}

func (m model) View() string {
	if m.w <= 0 || m.h <= 0 {
		return ""
	}
	th := m.theme()
	bodyH := m.bodyRows()
	var body []string
	if m.help {
		body = m.helpLines()
	} else {
		body = m.pageLines(th)
	}
	lines := []string{
		fit(ansi.NewStyle().Bold().Render(" Gallery")+ansi.NewStyle().Faint().Render("  "+fmt.Sprintf("page %d of %d", m.page+1, pageCount)), m.w),
		fit(m.tabBar(th), m.w),
		ansi.NewStyle().Faint().Render(strings.Repeat("─", m.w)),
	}
	lines = append(lines, block(indent(body), m.w, bodyH)...)
	hint := " F1-F4 page  ? keys  Ctrl+C quit"
	lines = append(lines, ansi.NewStyle().Faint().Render(fit(hint, m.w)))
	base := strings.Join(lines[:min(len(lines), m.h)], "\n")
	// Overlays are drawn over the page while they are open.
	if m.dialog.Open() {
		base = m.dialog.SetTheme(th).Render(base)
	}
	if m.drawer.Open() {
		base = m.drawer.SetTheme(th).Render(base)
	}
	if m.popover.Open() {
		base = m.popover.SetTheme(th).Render(base)
	}
	if m.toast.Open() {
		base = m.toast.SetTheme(th).Render(base)
	}
	return base
}

func indent(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = " " + l
	}
	return out
}
