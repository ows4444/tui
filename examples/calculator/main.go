// Command calculator is a pocket calculator for the terminal. Click the
// buttons or type: digits, + - * / % ^ and parentheses, Enter or = for the
// result, Backspace to delete, c to clear, y (or a click on the display) to copy the result, q to quit.
// A result shows as you type, finished ones land in the history on the right,
// and clicking a history line puts its result back into the expression.
//
//	go run ./examples/calculator
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// keys is the button grid, row by row.
var keys = [][]string{
	{"C", "(", ")", "÷"},
	{"7", "8", "9", "×"},
	{"4", "5", "6", "−"},
	{"1", "2", "3", "+"},
	{"0", ".", "⌫", "="},
}

const (
	cellW      = 6 // width of one button
	gridW      = 4*cellW + 3
	historyMax = 8  // lines the history shows
	wideLayout = 64 // columns needed for the history beside the keypad
)

type entry struct{ expr, result string }

type model struct {
	expr    string
	done    bool // expr is a finished result, so a digit starts afresh
	err     string
	history []entry

	hover, pressed string
	copied         bool

	width, height int
}

func initialModel() model { return model{} }

func (m model) Init() tui.Cmd { return nil }

func isDigit(s string) bool { return len(s) == 1 && (s[0] >= '0' && s[0] <= '9' || s[0] == '.') }

// press runs a button or key by its label.
func (m model) press(label string) (model, tui.Cmd) {
	m.err, m.copied = "", false
	switch label {
	case "C":
		m.expr, m.done = "", false
	case "⌫":
		if r := []rune(m.expr); len(r) > 0 {
			m.expr, m.done = string(r[:len(r)-1]), false
		}
	case "=":
		return m.equals(), nil
	default:
		if m.done && isDigit(label) {
			m.expr = ""
		}
		if m.expr == "" && !isDigit(label) && label != "−" && label != "(" {
			return m, nil // nothing for an operator to work on yet
		}
		m.expr, m.done = m.expr+label, false
	}
	return m, nil
}

func (m model) equals() model {
	if strings.TrimSpace(m.expr) == "" || m.done {
		return m
	}
	v, err := eval(m.expr)
	if err != nil {
		m.err = err.Error()
		return m
	}
	r := format(v)
	m.history = append(m.history, entry{m.expr, r})
	if len(m.history) > historyMax {
		m.history = m.history[len(m.history)-historyMax:]
	}
	m.expr, m.done = r, true
	return m
}

// preview is the value of the expression as typed, when it is complete.
func (m model) preview() (string, bool) {
	if m.expr == "" || m.done {
		return "", false
	}
	v, err := eval(m.expr)
	if err != nil {
		return "", false
	}
	return format(v), true
}

// result is what y copies: the finished result, or the preview.
func (m model) result() string {
	if m.done {
		return m.expr
	}
	r, _ := m.preview()
	return r
}

var rune2label = map[string]string{"-": "−", "*": "×", "/": "÷", "x": "×"}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.MouseEvent:
		return m.mouse(msg)
	case tui.Key:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyCtrlC:
		return m, tui.Quit()
	case tui.KeyEnter:
		return m.press("=")
	case tui.KeyBackspace:
		return m.press("⌫")
	case tui.KeyEsc:
		if m.expr == "" {
			return m, tui.Quit()
		}
		return m.press("C")
	case tui.KeyRunes:
		s := k.Text
		switch s {
		case "q":
			return m, tui.Quit()
		case "c", "C":
			return m.press("C")
		case "y":
			return m.copy()
		case "=":
			return m.press("=")
		case "%", "^":
			return m.press(s) // no button, but the evaluator knows them
		}
		if l, ok := rune2label[s]; ok && s != "x" {
			s = l
		}
		for _, row := range keys {
			for _, key := range row {
				if key == s && key != "C" && key != "⌫" && key != "=" {
					return m.press(s)
				}
			}
		}
	}
	return m, nil
}

func (m model) copy() (model, tui.Cmd) {
	r := m.result()
	if r == "" {
		return m, nil
	}
	m.copied = true
	return m, tui.WriteClipboard(r)
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	hit, ok := hittest.HitMap(m.screen(), m.size()).At(ev.X, ev.Y)
	id := ""
	if ok {
		id = hit.ID
	}
	m.hover = id
	if ev.Button != tui.MouseButtonLeft {
		return m, nil
	}
	switch ev.Action {
	case tui.MouseActionPress:
		m.pressed = id
	case tui.MouseActionRelease:
		pressed := m.pressed
		m.pressed = ""
		if pressed == "" || pressed != id {
			return m, nil
		}
		return m.activate(id)
	}
	return m, nil
}

// activate runs the target a click landed on.
func (m model) activate(id string) (tui.Model, tui.Cmd) {
	switch {
	case strings.HasPrefix(id, "k-"):
		return m.press(strings.TrimPrefix(id, "k-"))
	case strings.HasPrefix(id, "hist-"):
		i, _ := strconv.Atoi(strings.TrimPrefix(id, "hist-"))
		if i >= 0 && i < len(m.history) {
			if m.done {
				m.expr, m.done = "", false
			}
			m.expr += m.history[i].result
		}
	case id == "copy":
		return m.copy()
	}
	return m, nil
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) wide() bool  { return m.size().W >= wideLayout }
func (m model) roomy() bool { return m.height == 0 || m.height >= 12 }

// button draws one key of the grid as its own click target.
func (m model) button(label string, th theme.Theme) layout.FlexChild {
	id := "k-" + label
	st := ansi.NewStyle().Foreground(th.Text)
	switch label {
	case "÷", "×", "−", "+", "(", ")":
		st = ansi.NewStyle().Foreground(th.Warning)
	case "C":
		st = ansi.NewStyle().Foreground(th.Error)
	case "=":
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Success)
	}
	switch {
	case m.pressed == id && m.hover == id:
		st = ansi.NewStyle().Bold().Reverse()
	case m.hover == id:
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	}
	text := widgets.Center(label, cellW, 1)
	return layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(text)))}
}

func right(s string, w int) string {
	n := ansi.Width(s)
	if n > w {
		return "…" + ansi.TruncateStart(s, w-1, "")
	}
	return strings.Repeat(" ", w-n) + s
}

func (m model) display(th theme.Theme) layout.Node {
	expr := m.expr
	if expr == "" {
		expr = "0"
	}
	line1 := ansi.NewStyle().Bold().Render(right(expr, gridW))
	var line2 string
	switch {
	case m.err != "":
		line2 = ansi.NewStyle().Foreground(th.Error).Render(right(m.err, gridW))
	case m.copied:
		line2 = ansi.NewStyle().Foreground(th.Success).Render(right("copied", gridW))
	default:
		if p, ok := m.preview(); ok {
			line2 = ansi.NewStyle().Faint().Render(right("= "+p, gridW))
		} else {
			line2 = strings.Repeat(" ", gridW)
		}
	}
	return layout.Named("copy", layout.Block(line1+"\n"+line2))
}

func (m model) keypad(th theme.Theme) layout.Node {
	var rows []layout.FlexChild
	for _, row := range keys {
		var cells []layout.FlexChild
		for _, k := range row {
			cells = append(cells, m.button(k, th))
		}
		rows = append(rows, layout.FlexChild{Node: layout.Row(1, cells...)})
	}
	kids := []layout.FlexChild{{Node: m.display(th)}}
	if m.roomy() {
		kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(strings.Repeat("─", gridW)))})
	}
	kids = append(kids, rows...)
	if m.roomy() {
		kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(
			widgets.Footer(gridW, widgets.Hint{Key: "y", Action: "copy"}, widgets.Hint{Key: "q", Action: "quit"})))})
	}
	return layout.BoxNode(layout.NewBox().Border(th.Border), layout.Fixed(layout.Column(0, kids...), layout.Size{W: gridW}))
}

func (m model) historyPanel(th theme.Theme) layout.Node {
	const w = 28
	kids := []layout.FlexChild{{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("History"))}}
	if len(m.history) == 0 {
		kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("Results appear here."))})
	}
	for i := len(m.history) - 1; i >= 0; i-- {
		e := m.history[i]
		id := "hist-" + strconv.Itoa(i)
		line := ansi.Truncate(e.expr+" = "+e.result, w)
		st := ansi.NewStyle()
		if m.hover == id {
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(id, layout.Block(st.Render(line)))})
	}
	return layout.BoxNode(layout.NewBox().Border(th.Border), layout.Fixed(layout.Column(0, kids...), layout.Size{W: w}))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	var content layout.Node = m.keypad(th)
	if m.wide() {
		content = layout.Row(2, layout.FlexChild{Node: content}, layout.FlexChild{Node: m.historyPanel(th)})
	}
	return layout.Stack(layout.Anchor(layout.Center, 0, 0, content))
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
