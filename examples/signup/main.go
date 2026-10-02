// Command signup is a small account form built on package form: text, secret,
// select and checkbox fields with validation, Tab and Shift+Tab to move between
// them, the arrow keys and Space to change a choice, Enter to submit (Esc quits).
// The screen is a layout.Node tree, so the form's LayoutNode sits in a padded
// box like any other widget. A failed submit shows every field's error and
// focuses the first invalid field; a valid one shows a summary that never
// includes the password.
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/form"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// width is the content width of the panel, or what is left of a narrower
// terminal (see model.contentWidth).
const width = 50

type model struct {
	form   form.Model
	done   bool
	values map[string]string

	termWidth, termHeight int // terminal size from the last ResizeMsg; 0 until known

	// Captured once at construction so Init can hand it back: Init can only
	// return a Cmd, not a mutated Model, so the initial Focus() (which must
	// happen on the model that is kept) happens in initialModel.
	focusCmd tui.Cmd
}

// mustBeTicked rejects a checkbox that is not ticked: a checkbox's value is the
// string "true" or "false".
func mustBeTicked(v string) error {
	if v != "true" {
		return errors.New("you must accept the terms")
	}
	return nil
}

func initialModel() model {
	f := form.New(
		form.Field{Name: "name", Label: "Name", Placeholder: "Ada Lovelace",
			Validators: []form.Validator{form.Required()}},
		form.Field{Name: "email", Label: "Email", Placeholder: "ada@example.com",
			Validators: []form.Validator{form.Required(), form.Email()}},
		form.Field{Name: "password", Label: "Password", Secret: true,
			Validators: []form.Validator{form.Required(), form.MinLen(8)}},
		form.Field{Name: "plan", Label: "Plan", Kind: form.FieldSelect, Options: []string{"Free", "Pro", "Team"}},
		form.Field{Name: "terms", Label: "Accept terms", Kind: form.FieldCheckbox,
			Validators: []form.Validator{mustBeTicked}},
	)
	cmd := f.Focus()
	return model{form: f, focusCmd: cmd}
}

func (m model) Init() tui.Cmd { return m.focusCmd }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.termWidth, m.termHeight = msg.Width, msg.Height
		return m, nil
	case form.SubmittedMsg:
		m.done, m.values = true, msg.Values
		return m, nil
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc {
			return m, tui.Quit()
		}
		if m.done {
			if msg.Type == tui.KeyEnter || (msg.Type == tui.KeyRunes && msg.Text == "q") {
				return m, tui.Quit()
			}
			return m, nil
		}
	}
	if m.done {
		return m, nil
	}
	next, cmd := m.form.Update(msg)
	m.form = next
	return m, cmd
}

var (
	title = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	good  = ansi.NewStyle().Bold().Foreground(ansi.BrightGreen)
)

// contentWidth is the panel's content width: width, or what is left of the
// terminal once the border and padding (4 columns) are paid for.
func (m model) contentWidth() int {
	if m.termWidth > 0 {
		return max(10, min(width, m.termWidth-4))
	}
	return width
}

// compact reports whether the terminal is too short for the airy panel (blank
// rows between sections, padding inside the box).
func (m model) compact() bool { return m.termHeight > 0 && m.termHeight < 14 }

// hints is the key legend: with words when they fit, else the keys alone.
func (m model) hints(hs ...widgets.Hint) string {
	full := widgets.KeyHints("  ", hs...)
	if ansi.Width(full) <= m.contentWidth() {
		return full
	}
	for i := range hs {
		hs[i].Action = ""
	}
	return ansi.Truncate(widgets.KeyHints(" ", hs...), m.contentWidth())
}

// summary is the screen shown after a valid submit. It names the account by
// name, email and plan and says only how long the password is.
func (m model) summary() layout.Node {
	pw := len([]rune(m.values["password"]))
	return layout.Column(m.gap(),
		layout.FlexChild{Node: layout.Block(good.Render("Account created"))},
		layout.FlexChild{Node: layout.Block(strings.Join([]string{
			"Name:     " + m.values["name"],
			"Email:    " + m.values["email"],
			"Plan:     " + m.values["plan"],
			fmt.Sprintf("Password: %d characters, not shown", pw),
		}, "\n"))},
		layout.FlexChild{Node: layout.Block(m.hints(widgets.Hint{Key: "enter", Action: "quit"}))},
	)
}

// gap is the blank row between sections: one, or none when compact.
func (m model) gap() int {
	if m.compact() {
		return 0
	}
	return 1
}

// screen builds the panel as a layout.Node tree.
func (m model) screen() layout.Node {
	t := theme.DarkTheme()
	var body layout.Node
	if m.done {
		body = m.summary()
	} else {
		body = layout.Column(m.gap(),
			layout.FlexChild{Node: layout.Block(title.Render("Create an account"))},
			layout.FlexChild{Node: m.form.LayoutNode()},
			layout.FlexChild{Node: layout.Block(m.hints(
				widgets.Hint{Key: "tab", Action: "next"},
				widgets.Hint{Key: "←→/space", Action: "change"},
				widgets.Hint{Key: "enter", Action: "submit"}))},
		)
	}
	sized := layout.Fixed(body, layout.Size{W: m.contentWidth()})
	pad := m.gap()
	return layout.BoxNode(layout.NewBox().Border(t.Border).Padding(pad, 1, pad, 1), sized)
}

func (m model) View() string {
	return layout.Draw(m.screen(), layout.Unconstrained())
}

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
