// Command splitedit is a small editor: a splitpane holds the text on the left
// and a live outline of it on the right. The textarea colours Go-style
// keywords through its Highlighter, indents new lines with AutoIndent, and
// F2 searches (Find, then FindNext) for the word the outline shows.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/splitpane"
	"github.com/ows4444/tui/textarea"
)

const sample = `func main() {
  for i := 0; i < 3; i++ {
    if i == 1 {
      return
    }
  }
}

func helper() int {
  return 42
}
`

var keyword = ansi.NewStyle().Bold().Foreground(ansi.BrightMagenta)

// highlight colours the keywords of one line.
func highlight(line string) ansi.Text {
	var t ansi.Text
	for _, w := range strings.SplitAfter(line, " ") {
		switch strings.TrimSpace(w) {
		case "func", "for", "if", "return":
			t = append(t, ansi.Span{Text: w, Style: keyword})
		default:
			t = append(t, ansi.Span{Text: w})
		}
	}
	return t
}

type model struct {
	edit  textarea.Model
	split splitpane.Model
	query string
	found int
	w, h  int // terminal size from the last ResizeMsg; 80x24 until known
}

func initialModel() model {
	e := textarea.New()
	e.Highlighter = highlight
	e.AutoIndent = true
	e.SetValue(sample)
	e.Focus()
	m := model{edit: e, w: 80, h: 24}
	return m.fit()
}

// outline lists the func lines of the text with their line numbers.
func (m model) outline() string {
	var b strings.Builder
	b.WriteString("Outline\n")
	for i, l := range strings.Split(m.edit.Value(), "\n") {
		if strings.HasPrefix(l, "func ") {
			fmt.Fprintf(&b, "%3d  %s\n", i+1, strings.TrimSuffix(strings.TrimPrefix(l, "func "), " {"))
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// fit rebuilds the split so both panes see the current text and size.
func (m model) fit() model {
	m.split = splitpane.New(m.edit.LayoutNode(), layout.Block(m.outline()))
	m.split.SetTotal(m.w)
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m.fit(), nil
	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc:
			return m, tui.Quit()
		case msg.Type == tui.KeyF2:
			m.query = "return"
			m.found = m.edit.Find(m.query)
			return m.fit(), nil
		case msg.Type == tui.KeyF3:
			m.edit.FindNext(m.query)
			return m.fit(), nil
		}
	}
	var cmd tui.Cmd
	m.edit, cmd = m.edit.Update(msg)
	return m.fit(), cmd
}

func (m model) screen() layout.Node {
	status := "f2: find return   f3: next   esc: quit"
	if m.query != "" {
		status = fmt.Sprintf("%d matches for %q   f3: next   esc: quit", m.found, m.query)
	}
	return layout.Column(0,
		layout.Fill(m.split.LayoutNode()),
		layout.FlexChild{Node: layout.Text(status, layout.WithEllipsis("…"))},
	)
}

func (m model) View() string {
	return layout.Draw(m.screen(), layout.Tight(layout.Size{W: m.w, H: m.h}))
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
