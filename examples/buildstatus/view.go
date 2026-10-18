package main

import (
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
)

func (m model) spin() string {
	if m.still {
		return "…"
	}
	return m.frames[m.b.tick%len(m.frames)]
}

func (m model) width0() int {
	if m.width <= 0 {
		return 80
	}
	return m.width
}

func bar(done, total, w int) string {
	filled := min(w, done*w/max(1, total))
	return strings.Repeat("█", filled) + strings.Repeat("░", w-filled)
}

func (m model) row(i int, th theme.Theme) string {
	s, j := m.b.steps[i], m.b.jobs[i]
	name := pad(s.name, 16)
	switch j.status {
	case running:
		pct := j.progress * 100 / s.dur
		line := ""
		if m.verbose && len(j.log) > 0 {
			line = "  " + ansi.NewStyle().Faint().Render(j.log[len(j.log)-1])
		}
		return ansi.NewStyle().Foreground(th.Primary).Render(m.spin()) + " " + name + bar(j.progress, s.dur, 10) + " " + pad(itoa(pct)+"%", 5) + line
	case done:
		return ansi.NewStyle().Foreground(th.Success).Render("✓") + " " + name + seconds(j.ended-j.started)
	case failed:
		return ansi.NewStyle().Foreground(th.Error).Render("✗") + " " + name + "failed"
	case skipped:
		return ansi.NewStyle().Faint().Render("- " + name + "skipped (" + j.reason + ")")
	case cancelled:
		return ansi.NewStyle().Faint().Render("- " + name + "cancelled")
	}
	return ansi.NewStyle().Faint().Render("· " + name + "waits for " + strings.Join(s.deps, ", "))
}

func (m model) View() string {
	th := theme.DarkTheme()
	w := m.width0()
	if m.help {
		return strings.Join([]string{
			ansi.Truncate("Ctrl+C  cancel the build; what finished stays printed", w), ansi.Truncate("+  -    steps to run at once", w),
			ansi.Truncate("v       show or hide the output lines", w), ansi.Truncate("Enter   leave when the build is over", w), ansi.Truncate("Esc     close this help", w),
		}, "\n")
	}
	var rows []string
	if m.b.over {
		rows = append(rows, ansi.NewStyle().Bold().Render(m.summary()))
		return ansi.Truncate(rows[0], w)
	}
	head := "Building  " + seconds(m.b.tick) + "  ·  " + itoa(m.b.count(done)) + "/" + itoa(len(m.b.steps)) + " done  ·  " + itoa(m.b.count(running)) + " running, up to " + itoa(m.b.max)
	rows = append(rows, ansi.NewStyle().Bold().Render(head))
	for i, j := range m.b.jobs {
		// finished steps are already in scrollback; the live region shows the rest
		if j.status == done || j.status == failed {
			continue
		}
		rows = append(rows, m.row(i, th))
	}
	rows = append(rows, ansi.NewStyle().Faint().Render("ctrl+c cancel  ·  +/- jobs  ·  v output  ·  ? help"))
	for i, r := range rows {
		rows[i] = ansi.Truncate(r, w)
	}
	return strings.Join(rows, "\n")
}
