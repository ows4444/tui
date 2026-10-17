// Command todo is a task list you can drive with the mouse or the keyboard.
// Click a tab to filter, click a checkbox or a row to tick it, click the cross
// to delete, click the input to type a new task, scroll the wheel to move.
// From the keyboard: up and down move, Space ticks, d deletes, Tab changes the
// filter, a or / starts a new task, Enter adds it, Esc stops typing, q quits.
// Every clickable thing is a named layout node, so the click targets are read
// from the same tree View draws and cannot drift from what is on screen.
//
//	go run ./examples/todo              # in memory
//	go run ./examples/todo -file t.txt  # saved after every change
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

type filter int

const (
	all filter = iota
	active
	done
)

var filterNames = [...]string{"All", "Active", "Done"}

type task struct {
	text string
	done bool
}

type model struct {
	tasks  []task
	filter filter
	cursor int // index into the visible tasks
	top    int // first visible task shown
	hover  string
	input  textinput.Model
	typing bool
	path   string // file the list is saved to; empty keeps it in memory
	saved  string // what the file holds, so only a change writes

	width, height int
}

func initialModel() model {
	in := textinput.New()
	in.Placeholder = "Add a task and press Enter"
	return model{
		tasks: []task{
			{"Read the tutorial", true},
			{"Try the mouse: click a checkbox", false},
			{"Scroll with the wheel", false},
			{"Delete this one with its cross", false},
			{"Write a first tui program", false},
			{"Ship it", false},
		},
		input: in,
	}
}

func (m model) Init() tui.Cmd { return nil }

// encode is the file format: one task a line, "x " or "- " then the text.
func encode(ts []task) string {
	var b strings.Builder
	for _, t := range ts {
		mark := "- "
		if t.done {
			mark = "x "
		}
		b.WriteString(mark + t.text + "\n")
	}
	return b.String()
}

func decode(s string) []task {
	var ts []task
	for _, l := range strings.Split(s, "\n") {
		if len(l) > 2 && (l[0] == 'x' || l[0] == '-') && l[1] == ' ' {
			ts = append(ts, task{text: l[2:], done: l[0] == 'x'})
		}
	}
	return ts
}

// fromFile loads the list from path, or starts the sample list when the file
// does not exist yet. Later changes are written back to it.
func fromFile(path string) (model, error) {
	m := initialModel()
	m.path = path
	b, err := os.ReadFile(path) // #nosec G304 -- the path is the user's own -file flag
	switch {
	case err == nil:
		m.tasks = decode(string(b))
	case !os.IsNotExist(err):
		return m, err
	}
	m.saved = encode(m.tasks)
	return m, nil
}

// visible returns the indexes into m.tasks that pass the filter.
func (m model) visible() []int {
	var out []int
	for i, t := range m.tasks {
		if m.filter == all || (m.filter == active && !t.done) || (m.filter == done && t.done) {
			out = append(out, i)
		}
	}
	return out
}

func (m model) counts() (total, open int) {
	for _, t := range m.tasks {
		if !t.done {
			open++
		}
	}
	return len(m.tasks), open
}

// listHeight is how many task rows fit on screen.
func (m model) listHeight() int {
	if m.height == 0 {
		return 8
	}
	if m.compact() {
		return max(2, m.height-5)
	}
	return max(3, min(12, m.height-16))
}

// compact reports a terminal too short for the roomy layout: the padding, the
// title, the gaps and the key hints go, so every control stays on screen.
func (m model) compact() bool { return m.height > 0 && m.height < 20 }

func (m model) clamp() model {
	n := len(m.visible())
	m.cursor = max(0, min(m.cursor, n-1))
	h := m.listHeight()
	if m.cursor < m.top {
		m.top = m.cursor
	}
	if m.cursor >= m.top+h {
		m.top = m.cursor - h + 1
	}
	m.top = max(0, min(m.top, max(0, n-h)))
	return m
}

func (m model) toggle(vis int) model {
	v := m.visible()
	if vis >= 0 && vis < len(v) {
		m.tasks[v[vis]].done = !m.tasks[v[vis]].done
	}
	return m.clamp()
}

func (m model) remove(vis int) model {
	v := m.visible()
	if vis >= 0 && vis < len(v) {
		i := v[vis]
		m.tasks = append(m.tasks[:i:i], m.tasks[i+1:]...)
	}
	return m.clamp()
}

// Update handles msg, then saves the list if it changed. The write runs as a
// Cmd, off the Update path.
func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	next, cmd := m.update(msg)
	n := next.(model)
	if n.path == "" {
		return n, cmd
	}
	if s := encode(n.tasks); s != n.saved {
		n.saved = s
		return n, tui.Batch(cmd, save(n.path, s))
	}
	return n, cmd
}

// save returns a Cmd that writes s to path.
func save(path, s string) tui.Cmd {
	return func() tui.Msg {
		_ = os.WriteFile(path, []byte(s), 0o600)
		return nil
	}
}

func (m model) update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.clamp(), nil
	case tui.MouseEvent:
		return m.mouse(msg)
	case tui.Key:
		if m.typing {
			return m.typingKey(msg)
		}
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyCtrlC, tui.KeyEsc:
		return m, tui.Quit()
	case tui.KeyUp:
		m.cursor--
	case tui.KeyDown:
		m.cursor++
	case tui.KeyTab:
		m.filter = (m.filter + 1) % 3
		m.cursor, m.top = 0, 0
	case tui.KeyDelete:
		return m.remove(m.cursor), nil
	case tui.KeyEnter:
		return m.toggle(m.cursor), nil
	case tui.KeyRunes:
		switch k.Text {
		case "q":
			return m, tui.Quit()
		case " ":
			return m.toggle(m.cursor), nil
		case "d":
			return m.remove(m.cursor), nil
		case "j":
			m.cursor++
		case "k":
			m.cursor--
		case "a", "/":
			m.typing = true
			return m, m.input.Focus()
		}
	}
	return m.clamp(), nil
}

func (m model) typingKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.typing = false
		m.input.Blur()
		return m, nil
	case tui.KeyEnter:
		if s := strings.TrimSpace(m.input.Value()); s != "" {
			m.tasks = append(m.tasks, task{text: s})
			m.input.Reset()
			if m.filter == done {
				m.filter = all
			}
			m.cursor = len(m.visible()) - 1
		}
		return m.clamp(), nil
	}
	next, cmd := m.input.Update(k)
	m.input = next
	return m, cmd
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	hit, ok := hittest.HitMap(m.screen(), m.size()).At(ev.X, ev.Y)
	if ev.Action == tui.MouseActionMotion {
		m.hover = ""
		if ok {
			m.hover = hit.ID
		}
		return m, nil
	}
	switch ev.Button {
	case tui.MouseButtonWheelUp, tui.MouseButtonWheelDown:
		d := 1
		if ev.Button == tui.MouseButtonWheelUp {
			d = -1
		}
		m.top += d
		m.cursor += d
		return m.clamp(), nil
	case tui.MouseButtonLeft:
		if ev.Action != tui.MouseActionPress {
			return m, nil
		}
	default:
		return m, nil
	}
	if !ok {
		if m.typing {
			m.typing = false
			m.input.Blur()
		}
		return m, nil
	}
	switch {
	case strings.HasPrefix(hit.ID, "tab-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(hit.ID, "tab-"))
		m.filter, m.cursor, m.top = filter(n), 0, 0
	case strings.HasPrefix(hit.ID, "del-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(hit.ID, "del-"))
		return m.remove(m.top + n), nil
	case strings.HasPrefix(hit.ID, "row-"):
		n, _ := strconv.Atoi(strings.TrimPrefix(hit.ID, "row-"))
		m.cursor = m.top + n
		return m.toggle(m.cursor), nil
	case hit.ID == "input":
		m.typing = true
		return m, m.input.Focus()
	case hit.ID == "clear":
		kept := m.tasks[:0:0]
		for _, t := range m.tasks {
			if !t.done {
				kept = append(kept, t)
			}
		}
		m.tasks = kept
	}
	return m.clamp(), nil
}

// size is the layout size the screen is hit-tested at.
func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

func (m model) panelWidth() int {
	w := m.size().W
	return max(24, min(64, w-4))
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	inner := m.panelWidth() - 4 // border and padding
	if m.compact() {
		inner = m.panelWidth() - 2
	}

	var tabs []layout.FlexChild
	total, open := m.counts()
	counts := [...]int{total, open, total - open}
	for i, name := range filterNames {
		label := fmt.Sprintf(" %s %d ", name, counts[i])
		st := ansi.NewStyle().Foreground(th.Muted)
		switch {
		case filter(i) == m.filter:
			st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
		case m.hover == "tab-"+strconv.Itoa(i):
			st = ansi.NewStyle().Foreground(th.Primary).Underline()
		}
		tabs = append(tabs, layout.FlexChild{Node: layout.Named("tab-"+strconv.Itoa(i), layout.Block(st.Render(label)))})
	}

	vis := m.visible()
	h := m.listHeight()
	var rows []layout.FlexChild
	for r := 0; r < h; r++ {
		idx := m.top + r
		if idx >= len(vis) {
			if len(vis) == 0 && r == 0 {
				rows = append(rows, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("Nothing here. Press a to add a task."))})
				continue
			}
			rows = append(rows, layout.FlexChild{Node: layout.Block(" ")})
			continue
		}
		rows = append(rows, layout.FlexChild{Node: m.row(r, m.tasks[vis[idx]], idx == m.cursor, inner, th)})
	}

	input := m.input.View()
	if !m.typing && m.input.Value() == "" {
		input = ansi.NewStyle().Faint().Render("＋ Add a task (click, or press a)")
	}
	inputRow := layout.Named("input", layout.Block(ansi.Truncate(input, inner)))

	clear := ansi.NewStyle().Foreground(th.Muted)
	if m.hover == "clear" {
		clear = ansi.NewStyle().Foreground(th.Error).Underline()
	}
	status := fmt.Sprintf("%d left", open)
	footer := layout.Row(2,
		layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(status)), Grow: 1},
		layout.FlexChild{Node: layout.Named("clear", layout.Block(clear.Render("Clear done")))},
	)

	help := widgets.KeyHints("  ",
		widgets.Hint{Key: "click", Action: "tick"},
		widgets.Hint{Key: "↑↓", Action: "move"},
		widgets.Hint{Key: "a", Action: "add"},
		widgets.Hint{Key: "q", Action: "quit"},
	)

	gap, box := 1, layout.NewBox().Border(th.Border).PaddingAll(1)
	kids := []layout.FlexChild{
		{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Tasks"))},
		{Node: layout.Row(1, tabs...)},
		{Node: layout.Column(0, rows...)},
		{Node: inputRow},
		{Node: footer},
		{Node: layout.Block(ansi.Truncate(help, inner))},
	}
	if m.compact() {
		gap, box = 0, layout.NewBox().Border(th.Border)
		kids = kids[1:5]
	}
	body := layout.Column(gap, kids...)
	panel := layout.Fixed(layout.BoxNode(box, layout.Fixed(body, layout.Size{W: inner})), layout.Size{W: m.panelWidth()})
	return layout.Stack(layout.Anchor(layout.Center, 0, 0, panel))
}

// row draws one task: a checkbox, its text and a cross, each its own click
// target so a click on the box or the text ticks and a click on the cross
// deletes.
func (m model) row(r int, t task, selected bool, width int, th theme.Theme) layout.Node {
	box, text := "☐", t.text
	st := ansi.NewStyle()
	if t.done {
		box = "☑"
		st = ansi.NewStyle().Faint().Strikethrough()
	}
	rowID, delID := "row-"+strconv.Itoa(r), "del-"+strconv.Itoa(r)
	bg := ansi.NewStyle()
	switch {
	case selected:
		bg = ansi.NewStyle().Background(th.Selection)
	case m.hover == rowID || m.hover == delID:
		bg = ansi.NewStyle().Background(th.Selection).Faint()
	}
	cross := ansi.NewStyle().Foreground(th.Muted).Render("✕")
	if m.hover == delID {
		cross = ansi.NewStyle().Bold().Foreground(th.Error).Render("✕")
	}
	textW := max(1, width-6)
	line := bg.Render(box+" ") + bg.Render(st.Render(pad(ansi.Truncate(text, textW), textW))) + bg.Render(" ")
	return layout.Row(1,
		layout.FlexChild{Node: layout.Named(rowID, layout.Block(line))},
		layout.FlexChild{Node: layout.Named(delID, layout.Block(cross))},
	)
}

func pad(s string, w int) string {
	if n := ansi.Width(s); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}

func (m model) View() string {
	return layout.DrawTight(m.screen(), m.size())
}

func main() {
	file := flag.String("file", "", "save the list to this file")
	flag.Parse()
	m := initialModel()
	if *file != "" {
		var err error
		if m, err = fromFile(*file); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	}
	p := tui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
