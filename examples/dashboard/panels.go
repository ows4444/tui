package main

import (
	"fmt"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets/chart"
)

// The panels, by id.
const (
	pCPU = iota
	pMem
	pDisk
	pNet
	pProcs
	numPanels
)

var panelNames = [numPanels]string{"CPU", "Memory", "Disks", "Network", "Processes"}

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

// fit cuts or pads lines to exactly w by h.
func fit(lines []string, w, h int) []string {
	out := make([]string, max(0, h))
	for i := range out {
		l := ""
		if i < len(lines) {
			l = ansi.Truncate(lines[i], w)
		}
		out[i] = pad(l, w)
	}
	return out
}

func split(s string) []string { return strings.Split(strings.TrimRight(s, "\n"), "\n") }

func rate(b float64) string {
	switch {
	case b >= 1e6:
		return fmt.Sprintf("%.1f MB/s", b/1e6)
	case b >= 1e3:
		return fmt.Sprintf("%.0f KB/s", b/1e3)
	}
	return fmt.Sprintf("%.0f B/s", b)
}

func bytesTotal(b float64) string {
	switch {
	case b >= 1e9:
		return fmt.Sprintf("%.1f GB", b/1e9)
	case b >= 1e6:
		return fmt.Sprintf("%.0f MB", b/1e6)
	}
	return fmt.Sprintf("%.0f KB", b/1e3)
}

// level colours a percentage by how worrying it is, and says it in words too
// when the colours are off: the caller prints the number beside it.
func level(p float64, th theme.Theme) ansi.Style {
	switch {
	case p >= 90:
		return ansi.NewStyle().Foreground(th.Error)
	case p >= 75:
		return ansi.NewStyle().Foreground(th.Warning)
	}
	return ansi.NewStyle().Foreground(th.Success)
}

func tail(v []float64, n int) []float64 {
	if n < 1 {
		return nil
	}
	return v[max(0, len(v)-n):]
}

// plot draws series in a w by h area, or a one-line sparkline of the first when
// there is no room for axes.
func plot(series []chart.Series, w, h int, yfmt func(float64) string, th theme.Theme) []string {
	if h < 4 || w < 16 {
		if len(series) == 0 || h < 1 {
			return nil
		}
		return []string{chart.SparklineWith(tail(series[0].Values, w), th)}
	}
	for i := range series {
		series[i].Values = tail(series[i].Values, w)
	}
	o := chart.PlotOptions{Width: w - 8, Height: h - 2, YTicks: 3, YFormat: yfmt, XLabels: []string{"-2m", "now"}}
	return fit(split(chart.Plot(series, o, th)), w, h)
}

func (m model) cpuLines(w, h int, th theme.Theme) []string {
	c := m.w.cur
	lines := []string{
		fmt.Sprintf("%s %3.0f%%   load %.2f %.2f %.2f", ansi.NewStyle().Bold().Render("Total"), c.cpu, m.w.load[0], m.w.load[1], m.w.load[2]),
		level(c.cpu, th).Render(chart.SparklineWith(tail(m.w.series(func(s sample) float64 { return s.cpu }), w), th)),
	}
	switch {
	case h-2 >= cores:
		items := make([]chart.BarItem, cores)
		for i := range items {
			items[i] = chart.BarItem{Label: fmt.Sprintf("core %d", i), Value: c.core[i]}
		}
		lines = append(lines, split(chart.BarChart(items, w, th))...)
	case h >= 3:
		lines = append(lines, "cores "+chart.SparklineWith(c.core[:], th))
	}
	return fit(lines, w, h)
}

func (m model) memLines(w, h int, th theme.Theme) []string {
	c := m.w.cur
	pct := 100 * c.memGB / memTotalGB
	head := fmt.Sprintf("Used %.1f / %.0f GB (%.0f%%)  cached %.1f  swap %.1f/%.0f", c.memGB, memTotalGB, pct, c.cached, c.swapGB, swapTotalGB)
	head = level(pct, th).Render(ansi.Truncate(head, w))
	series := []chart.Series{
		{Name: "used", Values: m.w.series(func(s sample) float64 { return s.memGB })},
		{Name: "cached", Values: m.w.series(func(s sample) float64 { return s.cached })},
	}
	lines := append([]string{head}, plot(series, w, h-1, func(v float64) string { return fmt.Sprintf("%.0fG", v) }, th)...)
	return fit(lines, w, h)
}

func (m model) diskLines(w, h int, th theme.Theme) []string {
	var lines []string
	for _, d := range m.w.disks {
		p := 100 * d.usedGB / d.totalGB
		text := fmt.Sprintf(" %3.0f%% %s/%s", p, gb(d.usedGB), gb(d.totalGB))
		if w < 36 {
			text = fmt.Sprintf(" %3.0f%%", p)
		}
		barW := max(4, w-9-ansi.Width(text))
		fill := int(p/100*float64(barW) + 0.5)
		bar := level(p, th).Render(strings.Repeat("━", fill)) + ansi.NewStyle().Faint().Render(strings.Repeat("─", barW-fill))
		lines = append(lines, fmt.Sprintf("%-8s", d.mount)+bar+text)
		if h >= 2*len(m.w.disks) {
			lines = append(lines, "")
		}
	}
	return fit(lines, w, h)
}

func gb(v float64) string {
	if v >= 1000 {
		return fmt.Sprintf("%.1fT", v/1000)
	}
	return fmt.Sprintf("%.0fG", v)
}

func (m model) netLines(w, h int, th theme.Theme) []string {
	c := m.w.cur
	lines := []string{
		fmt.Sprintf("▼ %s  ▲ %s", ansi.NewStyle().Foreground(th.Primary).Render(rate(c.rx)), ansi.NewStyle().Foreground(th.Secondary).Render(rate(c.tx))),
	}
	if h >= 7 {
		lines = append(lines, ansi.NewStyle().Faint().Render(fmt.Sprintf("since start  ▼ %s  ▲ %s", bytesTotal(m.w.rxSum), bytesTotal(m.w.txSum))))
	}
	series := []chart.Series{
		{Name: "down", Values: m.w.series(func(s sample) float64 { return s.rx }), Color: th.Primary},
		{Name: "up", Values: m.w.series(func(s sample) float64 { return s.tx }), Color: th.Secondary},
	}
	lines = append(lines, plot(series, w, h-len(lines), func(v float64) string { return rate(v)[:min(len(rate(v)), 6)] }, th)...)
	return fit(lines, w, h)
}

func (m model) procLines(w, h int, th theme.Theme) []string {
	head := ansi.NewStyle().Faint().Render(fmt.Sprintf("%5s %-*s %5s %5s", "PID", max(8, w-20), "NAME", "CPU%", "MEM%"))
	lines := []string{head}
	for _, p := range m.w.top(max(0, h-1)) {
		lines = append(lines, fmt.Sprintf("%5d %-*s %s %5.1f", p.pid, max(8, w-20), ansi.Truncate(p.name, max(8, w-20)),
			level(p.cpu*1.2, th).Render(fmt.Sprintf("%5.1f", p.cpu)), p.memP))
	}
	return fit(lines, w, h)
}

// lines is the content of panel id in a w by h area.
func (m model) lines(id, w, h int, th theme.Theme) []string {
	switch id {
	case pCPU:
		return m.cpuLines(w, h, th)
	case pMem:
		return m.memLines(w, h, th)
	case pDisk:
		return m.diskLines(w, h, th)
	case pNet:
		return m.netLines(w, h, th)
	}
	return m.procLines(w, h, th)
}

// summary is the number shown in a panel's title bar.
func (m model) summary(id int) string {
	c := m.w.cur
	switch id {
	case pCPU:
		return fmt.Sprintf("%.0f%%", c.cpu)
	case pMem:
		return fmt.Sprintf("%.0f%%", 100*c.memGB/memTotalGB)
	case pDisk:
		worst := 0.0
		for _, d := range m.w.disks {
			worst = max(worst, 100*d.usedGB/d.totalGB)
		}
		return fmt.Sprintf("worst %.0f%%", worst)
	case pNet:
		return "▼" + rate(c.rx)
	}
	return fmt.Sprintf("%d", len(m.w.procs))
}

// box draws lines inside a rounded border w by h, the title in the top edge
// and right (if any) at its far end.
func box(title, right string, lines []string, w, h int, edge ansi.Style) []string {
	inner := w - 2
	left := edge.Render("╭─ ") + ansi.NewStyle().Bold().Render(ansi.Truncate(title, max(1, w-8))) + " "
	end := ""
	if right != "" {
		end = " " + right + " "
	}
	fill := max(0, w-1-ansi.Width(left)-ansi.Width(end))
	top := left + edge.Render(strings.Repeat("─", fill)) + end
	out := []string{pad(ansi.Truncate(top, max(0, w-1)), max(0, w-1)) + edge.Render("╮")}
	if h < 2 {
		return out[:max(0, h)]
	}
	for i := 0; i < h-2; i++ {
		l := ""
		if i < len(lines) {
			l = lines[i]
		}
		out = append(out, edge.Render("│")+pad(ansi.Truncate(l, inner), inner)+edge.Render("│"))
	}
	return append(out, edge.Render("╰"+strings.Repeat("─", max(0, w-2))+"╯"))
}
