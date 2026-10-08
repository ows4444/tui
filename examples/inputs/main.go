// Command inputs is a small settings form built from the four single-purpose
// text fields: numberinput (digits only), emailinput (no whitespace, with a
// Valid check), maskedinput (the value is never drawn) and taginput (Enter
// adds a tag, Backspace on an empty field removes the last). Tab and Shift+Tab
// move between them and Esc quits. A status line under the form reads each
// field back, so what a field accepts and rejects shows as you type.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/emailinput"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/maskedinput"
	"github.com/ows4444/tui/numberinput"
	"github.com/ows4444/tui/taginput"
	"github.com/ows4444/tui/widgets"
)

// The fields, in Tab order.
const (
	fieldPort = iota
	fieldEmail
	fieldToken
	fieldTags
	fieldCount
)

var labels = [fieldCount]string{"Port", "Email", "Token", "Labels"}

// labelWidth is the width of the label column: the longest label and a gap.
const labelWidth = 8

type model struct {
	port  numberinput.Model
	email emailinput.Model
	token maskedinput.Model
	tags  taginput.Model

	focus         int
	width, height int // terminal size from the last ResizeMsg; 0 until known

	// Init can only return a Cmd, so the first Focus (which must happen on the
	// model that is kept) happens in initialModel and its Cmd is held here.
	focusCmd tui.Cmd
}

func initialModel() model {
	m := model{
		port:  numberinput.New(),
		email: emailinput.New(),
		token: maskedinput.New(),
		tags:  taginput.New(),
	}
	m.port.Placeholder = "8080"
	m.email.Placeholder = "ops@example.com"
	m.token.Placeholder = "paste a token"
	m.tags.Input.Placeholder = "type, then enter"
	m.tags.MaxTags = 5
	m.focusCmd = m.setFocus(fieldPort)
	return m
}

func (m model) Init() tui.Cmd { return m.focusCmd }

// setFocus blurs every field and focuses field i, returning its blink Cmd.
func (m *model) setFocus(i int) tui.Cmd {
	m.port.Blur()
	m.email.Blur()
	m.token.Blur()
	m.tags.Input.Blur()
	m.focus = (i + fieldCount) % fieldCount
	switch m.focus {
	case fieldPort:
		return m.port.Focus()
	case fieldEmail:
		return m.email.Focus()
	case fieldToken:
		return m.token.Focus()
	}
	return m.tags.Input.Focus()
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tui.Key:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tui.Quit()
		case "tab":
			// The Cmd is taken first, on its own line: setFocus changes m, and
			// "return m, m.setFocus(...)" does not promise to read m afterwards.
			cmd := m.setFocus(m.focus + 1)
			return m, cmd
		case "shift+tab":
			cmd := m.setFocus(m.focus - 1)
			return m, cmd
		case "enter":
			// Enter adds a tag in the tag field; anywhere else it moves on.
			if m.focus != fieldTags {
				cmd := m.setFocus(m.focus + 1)
				return m, cmd
			}
		}
	}
	// Everything else goes to the focused field only.
	var cmd tui.Cmd
	switch m.focus {
	case fieldPort:
		m.port, cmd = m.port.Update(msg)
	case fieldEmail:
		m.email, cmd = m.email.Update(msg)
	case fieldToken:
		m.token, cmd = m.token.Update(msg)
	case fieldTags:
		m.tags, cmd = m.tags.Update(msg)
	}
	return m, cmd
}

var (
	title  = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	active = ansi.NewStyle().Bold()
	muted  = ansi.NewStyle().Faint()
	good   = ansi.NewStyle().Foreground(ansi.BrightGreen)
	bad    = ansi.NewStyle().Foreground(ansi.BrightYellow)
)

// contentWidth is the width the form draws in: 60 columns, or the terminal's
// when that is narrower.
func (m model) contentWidth() int {
	if m.width > 0 {
		return max(20, min(60, m.width))
	}
	return 60
}

// compact reports whether the terminal is too short for blank rows between
// the sections.
func (m model) compact() bool { return m.height > 0 && m.height < 14 }

// row draws one labelled field at the content width. The field is given as a
// layout.Node so it is fitted to what is left of the row: a long value
// scrolls inside its field and never pushes the row past the terminal.
func (m model) row(i int, field layout.Node) string {
	label := labels[i]
	if i == m.focus {
		label = active.Render(label)
	} else {
		label = muted.Render(label)
	}
	label += strings.Repeat(" ", labelWidth-ansi.Width(labels[i]))
	return label + layout.DrawTight(field, layout.Size{W: m.contentWidth() - labelWidth, H: 1})
}

// status reads every field back in words, wrapped to the content width. The
// token's value is never shown, only its length.
func (m model) status() string {
	parts := make([]string, 0, fieldCount)
	if v := m.port.Value(); v == "" {
		parts = append(parts, muted.Render("no port"))
	} else {
		parts = append(parts, good.Render("port "+v))
	}
	switch {
	case m.email.Value() == "":
		parts = append(parts, muted.Render("no email"))
	case m.email.Valid():
		parts = append(parts, good.Render("email ok"))
	default:
		parts = append(parts, bad.Render("email incomplete"))
	}
	if n := len([]rune(m.token.Value())); n == 0 {
		parts = append(parts, muted.Render("no token"))
	} else {
		parts = append(parts, good.Render(fmt.Sprintf("token %d chars", n)))
	}
	parts = append(parts, fmt.Sprintf("%d of %d labels", len(m.tags.Tags), m.tags.MaxTags))
	return ansi.WrapStyled(strings.Join(parts, muted.Render(" · ")), m.contentWidth())
}

// hints is the key legend: with words when they fit, else the keys alone.
func (m model) hints() string {
	hs := []widgets.Hint{{Key: "tab", Action: "next"}, {Key: "shift+tab", Action: "back"}, {Key: "esc", Action: "quit"}}
	if m.focus == fieldTags {
		hs = append([]widgets.Hint{{Key: "enter", Action: "add label"}}, hs...)
	}
	full := widgets.KeyHints("  ", hs...)
	if ansi.Width(full) <= m.contentWidth() {
		return full
	}
	for i := range hs {
		hs[i].Action = ""
	}
	return ansi.Truncate(widgets.KeyHints(" ", hs...), m.contentWidth())
}

func (m model) View() string {
	gap := "\n\n"
	if m.compact() {
		gap = "\n"
	}
	rows := []string{
		m.row(fieldPort, m.port.LayoutNode()),
		m.row(fieldEmail, m.email.LayoutNode()),
		m.row(fieldToken, m.token.LayoutNode()),
		m.row(fieldTags, m.tags.LayoutNode()),
	}
	return strings.Join([]string{
		title.Render(ansi.Truncate("Service settings", m.contentWidth())),
		strings.Join(rows, "\n"),
		m.status(),
		m.hints(),
	}, gap)
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
