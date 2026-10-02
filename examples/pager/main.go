package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/viewport"
)

// contentWidth and windowHeight are the pager's natural content width and
// scrolling window; a smaller terminal shrinks both (see model.resize).
const (
	contentWidth = 60
	windowHeight = 10
)

// chordTop is the name of the vim-style "gg" chord that jumps to the top.
const chordTop = "top"

var paragraphs = []string{
	"This from-scratch TUI framework talks to the terminal directly " +
		"via raw syscall ioctls, no golang.org/x/term and no Bubble Tea.",
	"The event loop follows the Elm Architecture: a Model implements " +
		"Init, Update, and View, and Program owns the terminal, the " +
		"input parsing, and a line-level diffing renderer.",
	"This viewport widget scrolls a window over content taller than " +
		"it. Try the up/down arrows, page up/down, home/end, or the " +
		"mouse wheel.",
	"Every widget in this framework - textinput, and now viewport - " +
		"lives in its own package depending on the root tui package, " +
		"composed into a parent Model the same way this example does.",
	"Long lines wrap into separate paragraph lines here just so " +
		"there's enough content to make scrolling worth demonstrating; " +
		"a real pager would usually reflow text to the viewport's " +
		"width instead of relying on pre-wrapped input.",
}

func buildContent(contentWidth int) string {
	var b strings.Builder
	for i, p := range paragraphs {
		b.WriteString(fmt.Sprintf("Paragraph %d\n", i+1))
		for _, line := range wrap(p, contentWidth) {
			b.WriteString(line + "\n")
		}
		if i < len(paragraphs)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// wrap does simple word wrapping to width columns.
func wrap(s string, width int) []string {
	words := strings.Fields(s)
	var lines []string
	var line string
	for _, w := range words {
		if line == "" {
			line = w
			continue
		}
		if len(line)+1+len(w) > width {
			lines = append(lines, line)
			line = w
			continue
		}
		line += " " + w
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

type model struct {
	vp            viewport.Model
	width, height int
}

func initialModel() model {
	vp := viewport.New(contentWidth, windowHeight)
	vp.SetContent(buildContent(contentWidth))
	return model{vp: vp}
}

// padding is the vertical padding inside the box: one row, or none when the
// terminal is too short to spare it.
func (m model) padding() int {
	if m.height > 0 && m.height < 16 {
		return 0
	}
	return 1
}

// gap is the blank row between the title and the window, dropped with the
// padding on a short terminal.
func (m model) gap() int { return m.padding() }

// resize fits the content width and the window height to the terminal: the
// box takes 2 border and 2 padding columns, and 2 border rows, the padding,
// the title row, the gap and the help line under it.
func (m model) resize() model {
	w, h := contentWidth, windowHeight
	if m.width > 0 {
		w = max(10, min(contentWidth, m.width-4))
	}
	if m.height > 0 {
		h = max(1, min(windowHeight, m.height-2-2*m.padding()-1-m.gap()-1))
	}
	if w != m.vp.Width {
		m.vp.Width = w
		m.vp.SetContent(buildContent(w))
	}
	m.vp.Height = h
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m = m.resize()
	case tui.ChordMsg:
		// Delivered by WithChords (see main): "g" "g" pressed in sequence.
		if msg.Name == chordTop {
			m.vp.GotoTop()
		}
		return m, nil
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC, tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyRunes:
			if msg.Text == "q" {
				return m, tui.Quit()
			}
			if msg.Text == "G" {
				m.vp.GotoBottom()
				return m, nil
			}
		}
	}

	var cmd tui.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

var (
	title   = ansi.NewStyle().Bold().Underline()
	heading = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	dim     = ansi.NewStyle().Faint()
)

// screen is the pager as a layout.Node: a rounded box holding the title and the
// scrolling window, and the help line under it.
func (m model) screen() layout.Node {
	scrollbar := fmt.Sprintf("%3.0f%%", m.vp.ScrollPercent()*100)
	if m.vp.AtTop() && m.vp.AtBottom() {
		scrollbar = "all"
	}

	body := layout.Column(m.gap(),
		layout.FlexChild{Node: layout.Block(title.Render("Pager"))},
		// The viewport scrolls, so it would measure to its whole content: give it
		// the fixed window it always had.
		layout.FlexChild{Node: layout.Fixed(m.vp.LayoutNode(), layout.Size{H: m.vp.Height})},
	)
	sized := layout.Fixed(body, layout.Size{W: m.vp.Width})
	p := m.padding()
	box := layout.BoxNode(layout.NewBox().Border(layout.RoundedBorder()).Padding(p, 1, p, 1), sized)

	keys := "up/down/pgup/pgdown/home/end/wheel scroll  gg top  G bottom  q quit"
	if m.width > 0 && ansi.Width(keys)+2+ansi.Width(scrollbar) > m.width {
		keys = "scroll: arrows pgup/dn  gg G q"
	}
	help := dim.Render(keys) + "  " + heading.Render(scrollbar)
	if m.width > 0 {
		help = ansi.Truncate(help, m.width)
	}
	return layout.Column(0,
		layout.FlexChild{Node: box, CrossAlign: layout.CrossStart}, // keep the box's own width, not the help line's
		layout.FlexChild{Node: layout.Block(help)},
	)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	// WithChords turns the two keypresses "g" "g" into one ChordMsg; a lone "g"
	// (or any other key) is delivered as an ordinary Key after the timeout.
	program := tui.NewProgram(initialModel(),
		tui.WithMouse(tui.MouseClick),
		tui.WithChords(tui.ChordDef{Name: chordTop, Keys: []string{"g", "g"}}),
	)
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
