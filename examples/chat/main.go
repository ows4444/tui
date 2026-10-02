// Command chat is a manual showcase of the InkUI-parity widgets added
// after the original dashboard/form/list examples: Gauge, CodeBlock,
// DiffView, markdown.Render, streamtext (Typewriter), TokenCounter and
// layout.Column and layout.Row. It doubles as a manual check of Dark and Light
// themes — 't' swaps between them live. The boxes shrink to a narrow terminal,
// and when the chat is taller than the window Up/Down/PgUp/PgDn scroll it.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/markdown"
	"github.com/ows4444/tui/streamtext"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
	"github.com/ows4444/tui/widgets/chart"
)

// boxWidth is the widest the message boxes get; on a narrower terminal they
// shrink to the window width (see model.boxWidth).
const boxWidth = 60

const userPrompt = "Refactor **parseArgs** to return an `error` instead of calling `os.Exit`.\n\n" +
	"- keep the flag order\n- add a regression test"

const assistantExplain = "Sure — here's the change. parseArgs now returns (Config, error); " +
	"main is the only place left that can exit."

const codeSample = `func parseArgs(args []string) (Config, error) {
	if len(args) < 2 {
		return Config{}, errors.New("usage: prog <file>")
	}
	return Config{Path: args[1]}, nil
}`

const diffSample = `--- a/args.go
+++ b/args.go
@@ -1,6 +1,6 @@
-func parseArgs(args []string) Config {
+func parseArgs(args []string) (Config, error) {
 	if len(args) < 2 {
-		os.Exit(1)
+		return Config{}, errors.New("usage: prog <file>")
 	}
-	return Config{Path: args[1]}
+	return Config{Path: args[1]}, nil
 }`

type model struct {
	light bool
	reply streamtext.Model

	usedTokens  int
	tokenLimit  int
	contextUsed float64 // 0..1, shown in the header Gauge

	width, height int // terminal size from the last ResizeMsg; 0 until known
	top           int // first screen line shown, when the chat is taller than the window
}

// boxWidth is the width of the message boxes: boxWidth, or the terminal width
// when that is narrower.
func (m model) boxWidth() int {
	if m.width > 0 {
		return max(8, min(boxWidth, m.width))
	}
	return boxWidth
}

// bodyHeight is how many lines of the chat the window shows, or 0 (all of
// them) before the terminal size is known.
func (m model) bodyHeight() int { return max(m.height, 0) }

func initialModel() model {
	m := model{
		usedTokens:  2350,
		tokenLimit:  8000,
		contextUsed: 0.35,
	}
	m.reply = streamtext.NewTypewriter()
	m.reply.Width = boxWidth - 4
	m.reply.SetText(assistantExplain)
	return m
}

func (m model) Init() tui.Cmd {
	return m.reply.Start()
}

func (m model) theme() theme.Theme {
	if m.light {
		return theme.LightTheme()
	}
	return theme.DarkTheme()
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width, m.height = rs.Width, rs.Height
		m.reply.Width = m.boxWidth() - 4
		m.top = m.clampTop(m.top)
		return m, nil
	}
	if key, ok := msg.(tui.Key); ok {
		switch key.Type {
		case tui.KeyCtrlC, tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyUp:
			m.top = m.clampTop(m.top - 1)
			return m, nil
		case tui.KeyDown:
			m.top = m.clampTop(m.top + 1)
			return m, nil
		case tui.KeyPgUp:
			m.top = m.clampTop(m.top - max(1, m.bodyHeight()-1))
			return m, nil
		case tui.KeyPgDown:
			m.top = m.clampTop(m.top + max(1, m.bodyHeight()-1))
			return m, nil
		case tui.KeyRunes:
			if key.Text == "" {
				break
			}
			switch []rune(key.Text)[0] {
			case 'q':
				return m, tui.Quit()
			case 't':
				m.light = !m.light
				m.reply.Theme = m.theme()
				return m, nil
			case 'r':
				m.usedTokens = 0
				return m, m.reply.SetText(assistantExplain)
			}
		}
	}

	var cmd tui.Cmd
	m.reply, cmd = m.reply.Update(msg)
	if !m.reply.Done() {
		m.usedTokens += 4 // approximate live token counter while streaming
	}
	return m, cmd
}

// bubble is a bordered message box exactly w cells wide (the border and
// padding included) around content.
func bubble(t theme.Theme, w int, content string) layout.FlexChild {
	box := layout.BoxNode(layout.NewBox().Border(t.Border).PaddingAll(1), layout.Block(content))
	return layout.FlexChild{Node: layout.Fixed(box, layout.Size{W: w})}
}

// screen stacks the header, the two message bubbles, the code, the diff and
// the footer, one blank row apart, as a layout.Node.
func (m model) screen() layout.Node {
	t := m.theme()
	bw := m.boxWidth()
	block := func(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }

	header := layout.Row(2,
		layout.FlexChild{Node: layout.Block(widgets.HeaderWithAccessory("Chat", "session #1", max(1, bw-18), t)), CrossAlign: layout.CrossStart},
		layout.FlexChild{Node: layout.Block(chart.Gauge(m.contextUsed, 14, t)), CrossAlign: layout.CrossStart},
	)

	counter := layout.FlexChild{Node: layout.Block(widgets.TokenCounter(m.usedTokens, m.tokenLimit, t)), CrossAlign: layout.CrossStart}
	hints := layout.FlexChild{Node: layout.Block(widgets.KeyHints("  ",
		widgets.Hint{Key: "r", Action: "replay"},
		widgets.Hint{Key: "t", Action: "theme"},
		widgets.Hint{Key: "q", Action: "quit"})), CrossAlign: layout.CrossStart}
	footer := layout.Row(3, counter, hints)
	if bw < boxWidth {
		footer = layout.Column(0, counter, hints) // too narrow for one row
	}

	return layout.Column(1,
		layout.FlexChild{Node: header, CrossAlign: layout.CrossStart},
		bubble(t, bw, markdown.Render(userPrompt, bw-4, t)),
		bubble(t, bw, m.reply.View()),
		block(widgets.CodeBlock(codeSample, bw, true, t)),
		block(widgets.DiffView(diffSample, bw, t)),
		layout.FlexChild{Node: footer, CrossAlign: layout.CrossStart},
	)
}

// content is the whole chat, one string per line, each cut to the window
// width as a last resort for anything that ignores its width.
func (m model) content() []string {
	lines := strings.Split(layout.Draw(m.screen(), layout.Unconstrained()), "\n")
	if m.width > 0 {
		for i, l := range lines {
			lines[i] = ansi.Truncate(l, m.width)
		}
	}
	return lines
}

// clampTop limits a scroll offset to the lines that exist.
func (m model) clampTop(top int) int {
	h := m.bodyHeight()
	if h == 0 {
		return 0
	}
	return max(0, min(top, len(m.content())-h))
}

// View draws the chat; when it is taller than the window, the part from the
// scroll offset (Up/Down/PgUp/PgDn) that fits.
func (m model) View() string {
	lines := m.content()
	if h := m.bodyHeight(); h > 0 && len(lines) > h {
		top := max(0, min(m.top, len(lines)-h))
		lines = lines[top : top+h]
	}
	return strings.Join(lines, "\n")
}

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
