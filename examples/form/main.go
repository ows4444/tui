package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/autocomplete"
	"github.com/ows4444/tui/confirm"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/wizard"
)

type field int

const (
	fieldName field = iota
	fieldEmail
	fieldLanguage
	fieldCount
)

// Step indices into the wizard. These mirror the original stepEditing /
// stepConfirming states one-for-one; stepDone isn't a step you walk back
// to, so it stays a separate bool (model.done) rather than a wizard step.
const (
	stepEditing = iota
	stepConfirm
)

type model struct {
	wizard wizard.Model
	done   bool // "submitted" isn't a step you walk back to, so it lives outside the wizard.

	name     textinput.Model
	email    textinput.Model
	language autocomplete.Model
	confirm  confirm.Model

	focused field

	width, height int // terminal size from the last ResizeMsg; 0 until known

	// Captured once at construction so Init can hand it back — Init can
	// only return a Cmd, not a mutated Model, so the initial Focus() call
	// (which needs to happen on the persisted model, not a throwaway copy)
	// has to happen in initialModel instead.
	focusCmd tui.Cmd
}

func initialModel() model {
	var m model

	m.wizard = wizard.New("Edit", "Confirm")

	m.name = textinput.New()
	m.name.Prompt = "Name:     "
	m.name.Placeholder = "Ada Lovelace"
	m.name.Width = 24
	m.name.CharLimit = 40

	m.email = textinput.New()
	m.email.Prompt = "Email:    "
	m.email.Placeholder = "ada@example.com"
	m.email.Width = 24
	m.email.CharLimit = 40

	m.language = autocomplete.New(
		"Go", "Python", "Rust", "JavaScript", "TypeScript",
		"Ruby", "Java", "C++", "C#", "Swift", "Kotlin",
	)
	m.language.Input.Prompt = "Language: "
	m.language.Input.Placeholder = "start typing..."
	m.language.Input.Width = 24
	m.language.Blur() // autocomplete.New focuses by default; name should have it instead

	m.confirm = confirm.New("Submit with these details?")

	m.focusCmd = m.name.Focus()
	return m
}

func (m model) Init() tui.Cmd { return m.focusCmd }

func (m model) switchFocus(delta int) (model, tui.Cmd) {
	switch m.focused {
	case fieldName:
		m.name.Blur()
	case fieldEmail:
		m.email.Blur()
	case fieldLanguage:
		m.language.Blur()
	}

	m.focused = (m.focused + field(delta) + fieldCount) % fieldCount

	var cmd tui.Cmd
	switch m.focused {
	case fieldName:
		cmd = m.name.Focus()
	case fieldEmail:
		cmd = m.email.Focus()
	case fieldLanguage:
		cmd = m.language.Focus()
	}
	return m, cmd
}

// forward routes msg to whichever widget is active for the current step.
func (m model) forward(msg tui.Msg) (model, tui.Cmd) {
	var cmd tui.Cmd
	switch m.wizard.Current() {
	case stepConfirm:
		m.confirm, cmd = m.confirm.Update(msg)
	default:
		switch m.focused {
		case fieldName:
			m.name, cmd = m.name.Update(msg)
		case fieldEmail:
			m.email, cmd = m.email.Update(msg)
		case fieldLanguage:
			m.language, cmd = m.language.Update(msg)
		}
	}
	return m, cmd
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tui.Key:
		if msg.Type == tui.KeyCtrlC {
			return m, tui.Quit()
		}
		return m.handleKey(msg)
	case confirm.ConfirmedMsg:
		if msg.Yes {
			m.done = true
		} else {
			m.wizard.Back()
		}
		return m, nil
	}
	return m.forward(msg)
}

func (m model) handleKey(key tui.Key) (tui.Model, tui.Cmd) {
	if m.done {
		if key.Type == tui.KeyEnter || (key.Type == tui.KeyRunes && key.Text == "q") {
			return m, tui.Quit()
		}
		return m, nil
	}

	if m.wizard.Current() == stepConfirm {
		if key.Type == tui.KeyEsc {
			m.wizard.Back()
			return m, nil
		}
		return m.forward(key)
	}

	// editing a field
	switch key.Type {
	case tui.KeyEsc:
		return m, tui.Quit()
	case tui.KeyTab:
		return m.switchFocus(1)
	case tui.KeyEnter:
		switch m.focused {
		case fieldName, fieldEmail:
			return m.switchFocus(1)
		case fieldLanguage:
			if !m.language.IsOpen() {
				_ = m.wizard.Next(nil)
				return m, nil
			}
			// dropdown open: fall through, forward Enter so
			// autocomplete accepts the highlighted suggestion
		}
	}
	return m.forward(key)
}

var (
	title   = ansi.NewStyle().Bold().Underline()
	dimSty  = ansi.NewStyle().Faint()
	doneSty = ansi.NewStyle().Foreground(ansi.BrightGreen).Bold()
)

// tallBody is how many rows the framed, airy layout needs; on a shorter
// terminal the blank separator rows and the frame's padding are dropped.
const tallBody = 14

// compact reports whether the terminal is too short for the airy layout.
func (m model) compact() bool { return m.height > 0 && m.height < tallBody }

// sep joins the sections of a screen: a blank row between them, or none when
// compact.
func (m model) sep() string {
	if m.compact() {
		return "\n"
	}
	return "\n\n"
}

// frame draws body in a rounded box padded by one cell (none vertically when
// compact). On a terminal narrower than the body the content is clipped to
// fit rather than overflowing it.
func (m model) frame(body string) string {
	pad := 1
	box := layout.NewBox().Border(layout.RoundedBorder()).Padding(pad, 1, pad, 1)
	if m.compact() {
		box = box.Padding(0, 1, 0, 1)
	}
	if room := m.width - 4; m.width > 0 {
		widest := 0
		for _, l := range strings.Split(body, "\n") {
			widest = max(widest, ansi.Width(l))
		}
		if widest > room {
			box = box.Width(max(1, room))
		}
	}
	return layout.Draw(layout.BoxNode(box, layout.Block(body)), layout.Unconstrained())
}

// hint picks the longer key legend when it fits the terminal.
func (m model) hint(long, short string) string {
	if m.width > 0 && ansi.Width(long)+4 > m.width {
		return dimSty.Render(short)
	}
	return dimSty.Render(long)
}

func (m model) View() string {
	sep := m.sep()
	if m.done {
		body := strings.Join([]string{
			doneSty.Render("Submitted!"),
			"Name:     " + m.name.Value() + "\nEmail:    " + m.email.Value() + "\nLanguage: " + m.language.Input.Value(),
			m.hint("enter/q to quit", "enter/q quit"),
		}, sep)
		return m.frame(body)
	}

	switch m.wizard.Current() {
	case stepConfirm:
		body := strings.Join([]string{
			title.Render("Confirm"),
			m.wizard.View(m.confirm.Theme),
			"Name:     " + m.name.Value() + "\nEmail:    " + m.email.Value() + "\nLanguage: " + m.language.Input.Value(),
			m.confirm.View(),
			m.hint("esc back to editing", "esc back"),
		}, sep)
		return m.frame(body)

	default:
		body := strings.Join([]string{
			title.Render("Sign up"),
			m.wizard.View(m.confirm.Theme),
			m.name.View() + "\n" + m.email.View() + "\n" + m.language.View(),
			m.hint("tab next field  enter next/confirm  esc quit", "tab next  enter ok  esc quit"),
		}, sep)
		return m.frame(body)
	}
}

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
