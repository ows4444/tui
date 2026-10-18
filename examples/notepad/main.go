// Command notepad is a small text editor. It has a menu bar (File, Edit,
// View), line numbers, bracket matching, find and replace, undo, and asks
// before it throws away changes.
//
// Mouse: click the menu bar and pick an item; click in the text to place the
// cursor; drag to select; scroll with the wheel. Keyboard: Ctrl+S save, Ctrl+O
// open, Ctrl+N new, Ctrl+F find, Ctrl+R replace, Ctrl+Z undo, Ctrl+Q quit,
// Alt+F / Alt+E / Alt+V open the menus.
//
//	go run ./examples/notepad [file]
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/examples/internal/ui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/menubar"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

type mode int

const (
	modeEdit mode = iota
	modeOpen
	modeSaveAs
	modeFind
	modeReplaceFind
	modeReplaceWith
	modeDiscard // asking whether to throw away changes
)

type model struct {
	th    ui.Theming // the theme the Program chose; see SetTheme
	help  bool
	st    store
	path  string
	saved string // the text as last saved or opened

	ed  textarea.Model
	bar menubar.Model
	in  textinput.Model

	mode    mode
	then    string // what to do once changes are discarded: "new", "open" or "quit"
	query   string // the last thing searched for
	found   string // the query the editor has already selected a match for
	replace string // the text found, while its replacement is typed
	msg     string

	numbers, wrap, space, curline bool

	width, height int
}

func newModel(st store, path string) model {
	m := model{st: st, numbers: true, wrap: false, curline: true}
	m.ed = textarea.New()
	m.ed.Mouse, m.ed.AutoIndent, m.ed.MatchBrackets = true, true, true
	m.ed.Focus()
	m.in = textinput.New()
	m.in.Width = 40
	m.bar = menubar.New(
		menubar.Menu{Title: "File", Accel: 'f', Items: []menubar.Item{
			{Label: "New", Value: "new", Shortcut: "ctrl+n", Accel: 'n'},
			{Label: "Open…", Value: "open", Shortcut: "ctrl+o", Accel: 'o'},
			{Label: "Save", Value: "save", Shortcut: "ctrl+s", Accel: 's'},
			{Label: "Save as…", Value: "saveas", Accel: 'a'},
			{Separator: true},
			{Label: "Quit", Value: "quit", Shortcut: "ctrl+q", Accel: 'q'},
		}},
		menubar.Menu{Title: "Edit", Accel: 'e', Items: []menubar.Item{
			{Label: "Undo", Value: "undo", Shortcut: "ctrl+z", Accel: 'u'},
			{Label: "Redo", Value: "redo", Shortcut: "ctrl+y", Accel: 'd'},
			{Label: "Select all", Value: "selectall", Shortcut: "ctrl+a", Accel: 'a'},
			{Separator: true},
			{Label: "Find…", Value: "find", Shortcut: "ctrl+f", Accel: 'f'},
			{Label: "Replace…", Value: "replace", Shortcut: "ctrl+r", Accel: 'p'},
		}},
		menubar.Menu{Title: "View", Accel: 'v', Items: []menubar.Item{
			{Label: "Line numbers", Value: "numbers", Accel: 'l'},
			{Label: "Word wrap", Value: "wrap", Accel: 'w'},
			{Label: "Show spaces", Value: "space", Accel: 's'},
			{Label: "Highlight line", Value: "curline", Accel: 'h'},
		}},
	)
	m.bar.Mouse = true
	if path != "" {
		m = m.load(path)
		m.msg = ""
	}
	return m.applyView()
}

func (m model) Init() tui.Cmd { return m.ed.Focus() }

func (m model) dirty() bool { return m.ed.Value() != m.saved }

// applyView copies the View menu's switches onto the editor.
func (m model) applyView() model {
	m.ed.ShowLineNumbers = m.numbers
	m.ed.SoftWrap = m.wrap
	m.ed.ShowWhitespace = m.space
	m.ed.HighlightCursorLine = m.curline
	return m.resize()
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

// resize fits the editor between the menu bar and the two lines below it.
func (m model) resize() model {
	sz := m.size()
	gutter := 0
	if m.numbers {
		gutter = len(fmt.Sprint(max(99, m.ed.LineCount()))) + 1
	}
	m.ed.Width = max(10, sz.W-gutter)
	m.ed.Height = max(1, sz.H-3)
	m.ed.Bounds = layout.Rect{X: 0, Y: 1, W: sz.W, H: m.ed.Height}
	m.bar.Bounds = layout.Rect{X: 0, Y: 0, W: sz.W, H: 1}
	m.bar.ScreenW, m.bar.ScreenH = sz.W, sz.H
	return m
}

// ---- documents ------------------------------------------------------------

func (m model) load(path string) model {
	text, err := m.st.read(path)
	if err != nil {
		m.msg = err.Error()
		return m
	}
	m.msg = "Opened " + path
	if strings.Contains(text, "\t") {
		// The text area draws a tab as one character, so tabs become spaces.
		text = strings.ReplaceAll(text, "\t", "    ")
		m.msg += " (tabs became 4 spaces)"
	}
	m.ed.SetValue(text)
	m.ed.SetCursor(0)
	m.path, m.saved = path, text
	return m.resize()
}

func (m model) save(path string) model {
	text := m.ed.Value()
	if err := m.st.write(path, text); err != nil {
		m.msg = "Cannot save: " + err.Error()
		return m
	}
	m.path, m.saved, m.msg = path, text, "Saved "+path
	return m
}

func (m model) newDoc() model {
	m.ed.Reset()
	m.path, m.saved, m.msg = "", "", "New document"
	return m.resize()
}

// guard runs action now, or after the user agrees to drop unsaved changes.
func (m model) guard(action string) (model, tui.Cmd) {
	if m.dirty() {
		m.mode, m.then = modeDiscard, action
		return m, nil
	}
	return m.perform(action)
}

func (m model) perform(action string) (model, tui.Cmd) {
	switch action {
	case "new":
		return m.newDoc(), nil
	case "open":
		return m.prompt(modeOpen, "Open: ", ""), nil
	case "quit":
		return m, tui.Quit()
	}
	return m, nil
}

// prompt opens the one-line input under the editor.
func (m model) prompt(md mode, label, initial string) model {
	m.mode = md
	m.in.Prompt = label
	m.in.SetValue(initial)
	m.in.CursorEnd()
	m.in.Focus()
	m.ed.Blur()
	return m
}

func (m model) closePrompt() model {
	m.mode = modeEdit
	m.in.Blur()
	m.ed.Focus()
	return m
}

// ---- update ----------------------------------------------------------------

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.resize(), nil
	case menubar.SelectedMsg:
		return m.menu(msg.Item.Value)
	case menubar.ClosedMsg:
		return m, nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	var cmd tui.Cmd
	m.ed, cmd = m.ed.Update(msg)
	return m, cmd
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if m.help {
		if k.Type == tui.KeyCtrlC {
			return m, tui.Quit()
		}
		m.help = false
		return m, nil
	}
	if k.Type == tui.KeyF1 {
		m.help = true
		return m, nil
	}
	switch m.mode {
	case modeDiscard:
		return m.discardKey(k)
	case modeEdit:
	default:
		return m.promptKey(k)
	}
	if m.bar.Open() || k.Mod.Alt() {
		before := m.bar.Open()
		next, cmd := m.bar.Update(k)
		if next.Open() || before {
			m.bar = next
			return m, cmd
		}
	}
	if k.Type == tui.KeyCtrl {
		switch k.Code {
		case 's':
			return m.menu("save")
		case 'o':
			return m.menu("open")
		case 'n':
			return m.menu("new")
		case 'f':
			return m.menu("find")
		case 'r':
			return m.menu("replace")
		case 'q':
			return m.menu("quit")
		case 'g':
			return m.findNext()
		}
	}
	m.msg = ""
	var cmd tui.Cmd
	m.ed, cmd = m.ed.Update(k)
	return m, cmd
}

func (m model) discardKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch {
	case k.Type == tui.KeyRunes && (k.Text == "y" || k.Text == "Y"):
		m.mode = modeEdit
		m.saved = m.ed.Value() // what is left is no longer unsaved
		return m.perform(m.then)
	case k.Type == tui.KeyEsc, k.Type == tui.KeyRunes && (k.Text == "n" || k.Text == "N"):
		m.mode = modeEdit
	}
	return m, nil
}

func (m model) menu(action string) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	switch action {
	case "new", "open", "quit":
		return m.guard(action)
	case "save":
		if m.path == "" {
			return m.prompt(modeSaveAs, "Save as: ", ""), nil
		}
		return m.save(m.path), nil
	case "saveas":
		return m.prompt(modeSaveAs, "Save as: ", m.path), nil
	case "undo":
		m.ed, cmd = m.ed.Update(tui.Key{Type: tui.KeyCtrl, Code: 'z'})
	case "redo":
		m.ed, cmd = m.ed.Update(tui.Key{Type: tui.KeyCtrl, Code: 'y'})
	case "selectall":
		m.ed.SelectAll()
	case "find":
		return m.prompt(modeFind, "Find: ", m.query), nil
	case "replace":
		return m.prompt(modeReplaceFind, "Replace: ", m.query), nil
	case "numbers":
		m.numbers = !m.numbers
	case "wrap":
		m.wrap = !m.wrap
	case "space":
		m.space = !m.space
	case "curline":
		m.curline = !m.curline
	}
	return m.applyView(), cmd
}

func (m model) findNext() (model, tui.Cmd) {
	if m.query == "" {
		return m.prompt(modeFind, "Find: ", ""), nil
	}
	if !m.ed.FindNext(m.query) {
		m.msg = "Not found: " + m.query
	}
	return m, nil
}

func (m model) promptKey(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyEsc:
		m.msg = ""
		return m.closePrompt(), nil
	case tui.KeyEnter:
		return m.submit()
	}
	var cmd tui.Cmd
	m.in, cmd = m.in.Update(k)
	return m, cmd
}

func (m model) submit() (tui.Model, tui.Cmd) {
	v := strings.TrimSpace(m.in.Value())
	switch m.mode {
	case modeOpen:
		m = m.closePrompt()
		if v != "" {
			m = m.load(v)
		}
	case modeSaveAs:
		m = m.closePrompt()
		if v != "" {
			m = m.save(v)
		}
	case modeFind:
		m.query = m.in.Value()
		switch {
		case m.query == "":
			m.msg = ""
		case m.query == m.found:
			m.ed.FindNext(m.query) // the same query again: the next match
		default:
			if n := m.ed.Find(m.query); n == 0 {
				m.msg, m.found = "Not found: "+m.query, ""
			} else {
				m.msg, m.found = fmt.Sprintf("%d matches", n), m.query
			}
		}
	case modeReplaceFind:
		if m.in.Value() == "" {
			return m.closePrompt(), nil
		}
		m.query = m.in.Value()
		m = m.prompt(modeReplaceWith, "With: ", "")
	case modeReplaceWith:
		n := m.ed.ReplaceAll(m.query, m.in.Value())
		m = m.closePrompt()
		m.msg = fmt.Sprintf("Replaced %d", n)
	}
	return m, nil
}

func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	if m.help {
		if ev.Action == tui.MouseActionPress {
			m.help = false
		}
		return m, nil
	}
	var cmd tui.Cmd
	if m.mode == modeDiscard {
		return m, nil
	}
	if m.bar.Open() || ev.Y == 0 {
		m.bar, cmd = m.bar.Update(ev)
		return m, cmd
	}
	if m.mode != modeEdit {
		return m, nil
	}
	m.ed, cmd = m.ed.Update(ev)
	return m, cmd
}

// ---- view ------------------------------------------------------------------

func (m model) status(th theme.Theme) string {
	name := m.path
	if name == "" {
		name = "untitled"
	}
	if m.dirty() {
		name += " ●"
	}
	pos := fmt.Sprintf("Ln %d, Col %d", m.ed.Line()+1, m.ed.Column()+1)
	right := fmt.Sprintf("%s  %d lines", pos, m.ed.LineCount())
	w := m.size().W
	if w < 60 {
		right = pos // the position matters more than the line count
	}
	if m.msg != "" && w >= 60 {
		right = m.msg + "   " + right
	}
	gap := max(1, w-ansi.Width(name)-ansi.Width(right)-2)
	line := " " + ansi.NewStyle().Bold().Render(name) + strings.Repeat(" ", gap) + ansi.NewStyle().Faint().Render(right)
	return ansi.NewStyle().Background(th.Surface).Render(ansi.Truncate(line, w) + strings.Repeat(" ", max(0, w-ansi.Width(line))))
}

func (m model) bottom(th theme.Theme) string {
	w := m.size().W
	switch m.mode {
	case modeDiscard:
		return ansi.NewStyle().Bold().Foreground(th.Warning).Render("Discard unsaved changes? ") + "[y] yes  [n] no"
	case modeEdit:
		return ansi.NewStyle().Faint().Render(widgets.Footer(w,
			widgets.Hint{Key: "ctrl+s", Action: "save"}, widgets.Hint{Key: "ctrl+o", Action: "open"},
			widgets.Hint{Key: "ctrl+f", Action: "find"}, widgets.Hint{Key: "f1", Action: "help"}, widgets.Hint{Key: "ctrl+q", Action: "quit"}))
	}
	return m.in.View()
}

func (m model) mainScreen() layout.Node {
	th := m.theme()
	return layout.Column(0,
		layout.FlexChild{Node: layout.Block(m.bar.View())},
		layout.Fill(m.ed.LayoutNode()),
		layout.FlexChild{Node: layout.Block(m.status(th))},
		layout.FlexChild{Node: layout.Block(ansi.Truncate(m.bottom(th), m.size().W))},
	)
}

func (m model) View() string {
	out := layout.DrawTight(m.screen(), m.size())
	if m.bar.Open() {
		return m.bar.Render(out)
	}
	return out
}

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	}
	m := newModel(diskStore{}, path)
	if _, err := ui.NewProgram(m, tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
