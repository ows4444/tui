// Command colorlab is a colour workbench: set a colour with hue, saturation
// and lightness sliders or type its hex code, see how readable text in that
// colour is on white and on black, pick a shade or the complement from a
// derived palette, copy the hex code, and export the palette as CSS custom
// properties. Nothing is written to disk unless you pass -out.
//
// Mouse: click or drag along a slider, wheel over it to nudge, click the hex
// code to type one, click a palette swatch to take it, click Copy, Export or
// Random. Keyboard: Tab and Shift+Tab move between the sliders, the hex field
// and the palette; Left and Right change a value (PgUp and PgDn by ten); in
// the palette Left, Right and Enter pick a swatch; C copies the hex code, E
// copies the palette as CSS, R tries a random colour, Esc leaves the hex field,
// F1 shows help, Q or Ctrl+C quits. Copying uses the terminal's clipboard
// escape (OSC 52), which not every terminal allows.
//
//	go run ./examples/colorlab
//	go run ./examples/colorlab -out palette.css   # E also writes the file
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

type focus int

const (
	onHue focus = iota
	onSat
	onLight
	onHex
	onPalette
	numFocus
)

var sliderNames = [...]string{"Hue", "Saturation", "Lightness"}
var sliderMax = [...]float64{360, 100, 100}
var sliderUnit = [...]string{"°", "%", "%"}

type model struct {
	th      ui.Theming // the theme the Program chose; see SetTheme
	rgb     ansi.RGB
	h, s, l float64 // the sliders, kept apart from rgb so a grey does not lose its hue
	hexIn   textinput.Model
	focus   focus
	cursor  int // palette swatch under the keyboard
	drag    int // slider being dragged, or -1
	hover   string
	status  string
	help    bool
	out     string // file E also writes, or ""
	rng     *rand.Rand

	width, height int
}

func initialModel() model {
	in := textinput.New()
	in.CharLimit = 7
	in.Placeholder = "#RRGGBB"
	m := model{hexIn: in, drag: -1, cursor: 2, rng: rand.New(rand.NewSource(1))} // #nosec G404 -- a random colour is not security-sensitive
	return m.setRGB(ansi.RGB{R: 0x3A, G: 0x7B, B: 0xD5}, true)
}

func (m model) Init() tui.Cmd { return nil }

// ---- colour state ----------------------------------------------------------

// setRGB makes c the colour and moves the sliders to it. writeHex also
// rewrites the hex field, which is left alone while the person types in it.
func (m model) setRGB(c ansi.RGB, writeHex bool) model {
	m.rgb = c
	h, sat, l := c.HSL()
	if sat > 0 {
		m.h = h // a grey has no hue of its own: the sliders keep the one they had
	}
	m.s, m.l = sat*100, l*100
	if writeHex {
		m.hexIn.SetValue(hex(c))
	}
	return m
}

func (m model) setHSL(h, s, l float64) model {
	m.h = max(0, min(h, 359))
	m.s, m.l = max(0, min(s, 100)), max(0, min(l, 100))
	m.rgb = ansi.FromHSL(m.h, m.s/100, m.l/100)
	m.hexIn.SetValue(hex(m.rgb))
	return m
}

func (m model) slider(i int) float64 { return [...]float64{m.h, m.s, m.l}[i] }

func (m model) setSlider(i int, v float64) model {
	h, s, l := m.h, m.s, m.l
	switch i {
	case 0:
		h = float64(int(max(0, min(v, 359)) + 0.5))
	case 1:
		s = float64(int(max(0, min(v, 100)) + 0.5))
	default:
		l = float64(int(max(0, min(v, 100)) + 0.5))
	}
	return m.setHSL(h, s, l)
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyCtrlC:
		return m, tui.Quit()
	case k.Type == tui.KeyF1:
		m.help = !m.help
		return m, nil
	case m.help:
		m.help = k.Type != tui.KeyEsc && k.Type != tui.KeyEnter
		return m, nil
	case k.Type == tui.KeyTab:
		step := focus(1)
		if k.Mod.Shift() {
			step = numFocus - 1
		}
		return m.focusOn((m.focus + step) % numFocus)
	case m.focus == onHex:
		return m.hexKey(k)
	}
	return m.navKey(k)
}

func (m model) focusOn(f focus) (tui.Model, tui.Cmd) {
	m.hexIn.Blur()
	m.focus = f
	if f == onHex {
		return m, m.hexIn.Focus()
	}
	return m, nil
}

func (m model) hexKey(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyEsc {
		return m.focusOn(onHue)
	}
	var cmd tui.Cmd
	m.hexIn, cmd = m.hexIn.Update(k)
	if c, ok := parseHex(m.hexIn.Value()); ok {
		in := m.hexIn
		m = m.setRGB(c, false)
		m.hexIn = in
		m.status = ""
	} else {
		m.status = "Type a hex code like #3A7BD5"
	}
	return m, cmd
}

func (m model) navKey(k tui.Key) (tui.Model, tui.Cmd) {
	step := 0.0
	switch k.Type {
	case tui.KeyRight:
		step = 1
	case tui.KeyLeft:
		step = -1
	case tui.KeyPgUp:
		step = 10
	case tui.KeyPgDown:
		step = -10
	case tui.KeyUp:
		return m.focusOn(max(onHue, m.focus-1))
	case tui.KeyDown:
		return m.focusOn(min(onPalette, m.focus+1))
	case tui.KeyEnter:
		if m.focus == onPalette {
			return m.adopt(m.cursor), nil
		}
	case tui.KeyRunes:
		return m.runeKey(k.Text)
	}
	if step == 0 {
		return m, nil
	}
	if m.focus == onPalette {
		m.cursor = max(0, min(5, m.cursor+int(step/abs(step))))
		return m, nil
	}
	return m.setSlider(int(m.focus), m.slider(int(m.focus))+step), nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func (m model) runeKey(r string) (tui.Model, tui.Cmd) {
	switch strings.ToLower(r) {
	case "q":
		return m, tui.Quit()
	case "c":
		return m.copyHex()
	case "e":
		return m.export()
	case "r":
		return m.random(), nil
	case "h":
		return m.navKey(tui.Key{Type: tui.KeyLeft})
	case "l":
		return m.navKey(tui.Key{Type: tui.KeyRight})
	case "j":
		return m.navKey(tui.Key{Type: tui.KeyDown})
	case "k":
		return m.navKey(tui.Key{Type: tui.KeyUp})
	}
	return m, nil
}

func (m model) adopt(i int) model {
	m = m.setRGB(palette(m.rgb)[i].c, true)
	m.cursor = 2
	m.status = "Took " + palette(m.rgb)[2].name
	return m
}

func (m model) random() model {
	m = m.setHSL(float64(m.rng.Intn(360)), float64(30+m.rng.Intn(60)), float64(25+m.rng.Intn(50)))
	m.status = "Random colour " + hex(m.rgb)
	return m
}

func (m model) copyHex() (tui.Model, tui.Cmd) {
	m.status = "Copied " + hex(m.rgb) + " (if your terminal allows it)"
	return m, tui.WriteClipboard(hex(m.rgb))
}

func (m model) export() (tui.Model, tui.Cmd) {
	text := css(palette(m.rgb))
	m.status = "Copied the palette as CSS (if your terminal allows it)"
	cmd := tui.WriteClipboard(text)
	if m.out != "" {
		m.status = "Copied the palette as CSS and wrote " + m.out
		cmd = tui.Batch(cmd, writeFile(m.out, text))
	}
	return m, cmd
}

func writeFile(path, text string) tui.Cmd {
	return func() tui.Msg {
		_ = os.WriteFile(path, []byte(text), 0o600)
		return nil
	}
}

// ---- mouse -----------------------------------------------------------------

func inside(r layout.Rect, x, y int) bool { return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H }

func (m model) ids() []string {
	ids := []string{"copy", "export", "random", "hex", "tr-0", "tr-1", "tr-2"}
	for i := 0; i < 6; i++ {
		ids = append(ids, fmt.Sprintf("sw-%d", i))
	}
	return ids
}

func (m model) hit(x, y int) string {
	for _, id := range m.ids() {
		if r, ok := layout.RectOf(m.screen(), m.size(), id); ok && inside(r, x, y) {
			return id
		}
	}
	return ""
}

// slideTo sets slider i from a pointer at column x of its track.
func (m model) slideTo(i, x int) model {
	r, ok := layout.RectOf(m.screen(), m.size(), fmt.Sprintf("tr-%d", i))
	if !ok || r.W < 2 {
		return m
	}
	frac := float64(max(0, min(x-r.X, r.W-1))) / float64(r.W-1)
	return m.setSlider(i, frac*sliderMax[i])
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		return m, nil
	}
	id := m.hit(ev.X, ev.Y)
	switch ev.Action {
	case tui.MouseActionMotion:
		if m.drag >= 0 {
			return m.slideTo(m.drag, ev.X), nil
		}
		m.hover = id
		return m, nil
	case tui.MouseActionRelease:
		m.drag = -1
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		return m.wheel(id, ev.Button == tui.MouseButtonWheelUp), nil
	case tui.MouseButtonLeft:
		return m.click(id, ev.X)
	}
	return m, nil
}

func (m model) wheel(id string, up bool) model {
	var i int
	if _, err := fmt.Sscanf(id, "tr-%d", &i); err != nil {
		return m
	}
	d := -1.0
	if up {
		d = 1
	}
	m.focus = focus(i)
	return m.setSlider(i, m.slider(i)+d)
}

func (m model) click(id string, x int) (tui.Model, tui.Cmd) {
	var i int
	switch {
	case id == "copy":
		return m.copyHex()
	case id == "export":
		return m.export()
	case id == "random":
		return m.random(), nil
	case id == "hex":
		return m.focusOn(onHex)
	case strings.HasPrefix(id, "tr-"):
		_, _ = fmt.Sscanf(id, "tr-%d", &i)
		m.focus, m.drag = focus(i), i
		return m.slideTo(i, x), nil
	case strings.HasPrefix(id, "sw-"):
		_, _ = fmt.Sscanf(id, "sw-%d", &i)
		m.focus = onPalette
		return m.adopt(i), nil
	}
	return m, nil
}

// ---- view ------------------------------------------------------------------

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) compact() bool { return m.size().H < 16 }

func (m model) trackW() int { return max(8, min(30, m.size().W-22)) }

func block(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

func bg(c ansi.RGB) ansi.Style { return ansi.NewStyle().Background(c) }

// track draws slider i: a bar coloured with what the slider would give at each
// cell, and a marker, so the marker still shows with colours off.
func (m model) track(i int, th theme.Theme) string {
	w := m.trackW()
	at := int(m.slider(i)/sliderMax[i]*float64(w-1) + 0.5)
	var b strings.Builder
	for x := 0; x < w; x++ {
		f := float64(x) / float64(w-1)
		var c ansi.RGB
		switch i {
		case 0:
			c = ansi.FromHSL(f*359, max(m.s, 50)/100, 0.5)
		case 1:
			c = ansi.FromHSL(m.h, f, max(m.l, 30)/100)
		default:
			c = ansi.FromHSL(m.h, max(m.s, 30)/100, f)
		}
		cell := ansi.NewStyle().Foreground(c).Render("━")
		if x == at {
			cell = ansi.NewStyle().Bold().Render("●")
		}
		b.WriteString(cell)
	}
	return b.String()
}

func (m model) sliderRow(i int, th theme.Theme) layout.Node {
	st := ansi.NewStyle().Faint()
	if m.focus == focus(i) {
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	val := fmt.Sprintf("%3d%s", int(m.slider(i)+0.5), sliderUnit[i])
	return layout.Row(1,
		layout.FlexChild{Node: layout.Block(st.Render(fmt.Sprintf("%-10s", sliderNames[i])))},
		layout.FlexChild{Node: layout.Named(fmt.Sprintf("tr-%d", i), layout.Block(m.track(i, th)))},
		block(val),
	)
}

func (m model) hexRow(th theme.Theme) layout.Node {
	st := ansi.NewStyle().Faint()
	if m.focus == onHex {
		st = ansi.NewStyle().Bold().Foreground(th.Primary)
	}
	field := m.hexIn.View()
	if m.focus != onHex {
		field = hex(m.rgb)
	}
	return layout.Row(1,
		layout.FlexChild{Node: layout.Block(st.Render(fmt.Sprintf("%-10s", "Hex")))},
		layout.FlexChild{Node: layout.Named("hex", layout.Block(ansi.Truncate(field, 10)))},
	)
}

func (m model) block(c ansi.RGB, w, h int) []layout.FlexChild {
	var rows []layout.FlexChild
	label := ansi.NewStyle().Background(c).Foreground(c.Readable()).Bold().Render(fmt.Sprintf("%-*s", w, " "+hex(c)))
	fill := bg(c).Render(strings.Repeat(" ", w))
	for r := 0; r < h; r++ {
		line := fill
		if r == h/2 {
			line = label
		}
		rows = append(rows, block(line))
	}
	return rows
}

func (m model) contrastLine(name string, other ansi.RGB, th theme.Theme) string {
	r := contrast(m.rgb, other)
	g := grade(r)
	mark, st := "✓", ansi.NewStyle().Foreground(th.Success)
	if g == "fail" {
		mark, st = "✗", ansi.NewStyle().Foreground(th.Error)
	}
	sample := ansi.NewStyle().Background(other).Foreground(m.rgb).Render(" Aa ")
	return fmt.Sprintf("%s %-5s %5.2f %s", sample, name, r, st.Render(mark+" "+g))
}

func (m model) palette(th theme.Theme) layout.Node {
	var kids []layout.FlexChild
	for i, p := range palette(m.rgb) {
		st := bg(p.c).Foreground(p.c.Readable())
		label := "  "
		switch {
		case i == m.cursor && m.focus == onPalette:
			st, label = st.Bold().Underline(), "▲ "
		case i == 2:
			label = "• "
		}
		if m.hover == fmt.Sprintf("sw-%d", i) {
			st = st.Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(fmt.Sprintf("sw-%d", i), layout.Block(st.Render(label+"  ")))})
	}
	return layout.Row(0, kids...)
}

func (m model) button(id, label string, th theme.Theme) layout.FlexChild {
	st := ansi.NewStyle().Foreground(th.Primary)
	if m.hover == id {
		st = st.Bold().Underline()
	}
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render("[ "+label+" ]")))}
}

func (m model) helpPanel(th theme.Theme) layout.Node {
	key := func(k, a string) layout.FlexChild {
		return block(ansi.NewStyle().Bold().Foreground(th.Primary).Render(fmt.Sprintf("%-14s", k)) + a)
	}
	return layout.Column(0,
		key("Tab", "next control"), key("Left Right", "change a value"), key("PgUp PgDn", "change by ten"),
		key("Enter", "take the palette swatch"), key("C", "copy the hex code"), key("E", "copy the palette as CSS"),
		key("R", "random colour"), key("Esc", "leave the hex field"), key("F1", "close help"), key("Q", "quit"))
}

func (m model) screen() layout.Node {
	th := m.theme()
	sz := m.size()
	if m.help {
		return layout.Column(0, block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Colour lab: keys")), layout.FlexChild{Node: m.helpPanel(th)})
	}
	rows := []layout.FlexChild{
		{Node: m.sliderRow(0, th)}, {Node: m.sliderRow(1, th)}, {Node: m.sliderRow(2, th)}, {Node: m.hexRow(th)},
	}
	if !m.compact() {
		rows = append([]layout.FlexChild{block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Colour lab"))}, rows...)
		rows = append(rows, m.block(m.rgb, min(30, sz.W), 3)...)
	}
	buttons := []layout.FlexChild{m.button("copy", "Copy", th), m.button("export", "Export", th), m.button("random", "Random", th)}
	rows = append(rows, block(m.contrastLine("white", white, th)), block(m.contrastLine("black", black, th)))
	if sz.W < 60 { // not wide enough for the buttons beside the palette
		rows = append(rows, layout.FlexChild{Node: m.palette(th)}, layout.FlexChild{Node: layout.Row(2, buttons...)})
	} else {
		rows = append(rows, layout.FlexChild{Node: layout.Row(2, append([]layout.FlexChild{{Node: m.palette(th)}}, buttons...)...)})
	}
	status := m.status
	if status == "" {
		status = hex(m.rgb)
	}
	rows = append(rows, block(ansi.NewStyle().Faint().Render(ansi.Truncate(status, sz.W))))
	if !m.compact() || sz.H >= 11 {
		f := widgets.Footer(sz.W, widgets.Hint{Key: "tab", Action: "control"}, widgets.Hint{Key: "←→", Action: "change"},
			widgets.Hint{Key: "c", Action: "copy"}, widgets.Hint{Key: "e", Action: "export"}, widgets.Hint{Key: "f1", Action: "help"})
		rows = append(rows, block(ansi.NewStyle().Faint().Render(ansi.Truncate(f, sz.W))))
	}
	return layout.Column(0, rows...)
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	out := flag.String("out", "", "also write the exported CSS to this file")
	flag.Parse()
	m := initialModel()
	m.out = *out
	if _, err := ui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
