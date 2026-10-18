package main

import (
	"strconv"
	"strings"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion/fx"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// ramp is the glyphs from cold to hot, so the picture reads without colour.
const ramp = " .:-=+*#%@"

func glyph(v, max int) byte { return ramp[min(len(ramp)-1, v*len(ramp)/(max+1))] }

// runs draws one row from cell glyphs and styles, joining cells of one style.
func runs(cells []byte, styles []ansi.Style) string {
	var b strings.Builder
	for i := 0; i < len(cells); {
		j := i
		for j < len(cells) && styles[j] == styles[i] {
			j++
		}
		b.WriteString(styles[i].Render(string(cells[i:j])))
		i = j
	}
	return b.String()
}

func (m model) fireRows(th theme.Theme) []string {
	f := m.fire
	cold := ansi.NewStyle().Foreground(th.Error)
	warm := ansi.NewStyle().Foreground(th.Warning)
	hot := ansi.NewStyle().Bold().Foreground(th.Text)
	rows := make([]string, f.h)
	for y := range rows {
		cells, styles := make([]byte, f.w), make([]ansi.Style, f.w)
		for x := range cells {
			h := int(f.heat[y*f.w+x])
			cells[x] = glyph(h, maxHeat)
			switch {
			case h < maxHeat/3:
				styles[x] = cold
			case h < 2*maxHeat/3:
				styles[x] = warm
			default:
				styles[x] = hot
			}
		}
		rows[y] = runs(cells, styles)
	}
	return rows
}

func (m model) plasmaRows(th theme.Theme) []string {
	w, h := m.bodySize()
	tones := []ansi.Style{ansi.NewStyle().Foreground(th.Info), ansi.NewStyle().Foreground(th.Primary), ansi.NewStyle().Foreground(th.Secondary), ansi.NewStyle().Bold().Foreground(th.Success)}
	rows := make([]string, h)
	for y := range rows {
		cells, styles := make([]byte, w), make([]ansi.Style, w)
		for x := range cells {
			v := plasma(x, y, m.sim, m.mx, m.my)
			cells[x] = glyph(v, 255)
			styles[x] = tones[min(len(tones)-1, v*len(tones)/256)]
		}
		rows[y] = runs(cells, styles)
	}
	return rows
}

func (m model) typeRows(th theme.Theme) []string {
	w, h := m.bodySize()
	lines := typed(m.chars)
	cursor := ansi.NewStyle().Foreground(th.Primary).Render("▌")
	if !m.still && (m.frame/12)%2 == 1 && m.chars < scriptLen() {
		cursor = " "
	}
	rows := make([]string, h)
	top := max(0, (h-len(script))/2)
	for i := range rows {
		li := i - top
		if li < 0 || li >= len(script) {
			continue
		}
		if li < len(lines) {
			rows[i] = " " + ansi.Truncate(lines[li], w-2)
			if li == len(lines)-1 && m.chars < scriptLen() {
				rows[i] += cursor
			}
		} else if li == len(lines) && m.chars < scriptLen() {
			rows[i] = " " + cursor
		}
	}
	if m.chars >= scriptLen() && top+len(script) < h {
		rows[top+len(script)] = ansi.NewStyle().Faint().Render(" click or r to type it again")
	}
	return rows
}

func (m model) sceneRows(th theme.Theme) []string {
	switch m.scene {
	case sPlasma:
		return m.plasmaRows(th)
	case sType:
		return m.typeRows(th)
	}
	return m.fireRows(th)
}

func (m model) tabs(th theme.Theme) layout.Node {
	kids := []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("fireandfx"))}}
	for i, n := range sceneNames {
		id := "scene-" + strconv.Itoa(i)
		label, st := " "+strconv.Itoa(i+1)+" "+n+" ", ansi.NewStyle().Foreground(th.Muted)
		switch {
		case scene(i) == m.scene:
			label, st = "▸"+strconv.Itoa(i+1)+" "+n+" ", ansi.NewStyle().Bold().Foreground(th.Primary)
		case m.hover == id:
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(label)))})
	}
	st := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == "effect" {
		st = ansi.NewStyle().Foreground(th.Primary).Underline()
	}
	kids = append(kids, layout.FlexChild{Node: layout.Named("effect", layout.Block(st.Render("[e "+effects[m.fx].name+"]")))})
	return layout.Fixed(layout.Row(1, kids...), layout.Size{W: m.size().W, H: 1})
}

func (m model) helpRows(th theme.Theme) []string {
	key := func(k, a string) string {
		return " " + ansi.NewStyle().Bold().Foreground(th.Primary).Render(pad(k, 14)) + a
	}
	return []string{
		key("1 2 3  Tab", "fire, plasma, typewriter"), key("Space", "pause the scene (transitions still play)"),
		key("+  -", "steps per frame"), key("e", "next transition: dissolve, fade, slide, sweep"),
		key("i", "fire source on or off"), key("Left Right", "blow the fire"), key("r", "restart the scene"),
		key("n", "step by hand (for reduced motion)"), key("mouse", "light the fire, move the ripple, finish the text"),
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
	switch {
	case m.msg != "":
		return m.msg
	case m.still:
		return "Reduced motion: still frames; n steps by hand"
	}
	state := "playing"
	if m.paused {
		state = "paused"
	}
	return ansi.NewStyle().Faint().Render(state + "  ·  speed " + strconv.Itoa(m.speed) + "/4  ·  " + effects[m.fx].name + " " + strconv.Itoa(int(m.progress()*100)) + "%")
}

func (m model) screen() layout.Node {
	th := m.theme()
	w := m.size().W
	rows := m.sceneRows(th)
	if m.help {
		rows = m.helpRows(th)
	}
	_, bh := m.bodySize()
	var lines []layout.FlexChild
	for i := 0; i < bh; i++ {
		l := ""
		if i < len(rows) {
			l = rows[i]
		}
		lines = append(lines, layout.FlexChild{Node: layout.Block(ansi.Truncate(l, w))})
	}
	body := layout.Node(layout.Named("body", layout.Fixed(layout.Column(0, lines...), layout.Size{W: w, H: bh})))
	if p := m.progress(); p < 1 && !m.help {
		body = layout.Named("body", fx.Wrap(layout.Fixed(layout.Column(0, lines...), layout.Size{W: w, H: bh}), effects[m.fx].fx, p))
	}
	hints := widgets.Footer(w, widgets.Hint{Key: "1-3", Action: "scene"}, widgets.Hint{Key: "e", Action: "effect"}, widgets.Hint{Key: "space", Action: "pause"},
		widgets.Hint{Key: "+/-", Action: "speed"}, widgets.Hint{Key: "n", Action: "step"}, widgets.Hint{Key: "?", Action: "help"}, widgets.Hint{Key: "q", Action: "quit"})
	return layout.Column(0,
		layout.FlexChild{Node: m.tabs(th)},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.status(), w))},
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(ansi.Truncate(hints, w)))})
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }
