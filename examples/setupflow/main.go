// Command setupflow follows the same composition-only shape as
// examples/welcomescreen and examples/loginflow: widgets.BigText for the
// title, widgets.Alert for a status message, and picker.Model as a
// numbered-select menu of setup steps. It introduces no new pattern
// beyond what those two examples already demonstrate — picking a setup
// step from a menu, not a real multi-step wizard (see wizard.Model, in
// examples/form, for that shape instead).
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

type model struct {
	t       theme.Theme
	menu    picker.Model
	applied string

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

func initialModel() model {
	return model{
		t:    theme.DarkTheme(),
		menu: picker.NewStrings("Configure workspace", "Configure notifications", "Finish"),
	}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width, m.height = rs.Width, rs.Height
		return m, nil
	}
	if key, ok := msg.(tui.Key); ok && key.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}

	var cmd tui.Cmd
	m.menu, cmd = m.menu.Update(msg)

	if sel, ok := msg.(picker.SelectedMsg); ok {
		if sel.Item.Value == "Finish" {
			return m, tui.Quit()
		}
		m.applied = sel.Item.Value
		return m, nil
	}

	return m, cmd
}

var setupFooter = ansi.NewStyle().Faint()

// screen stacks the title, the status alert, the step menu and the footer, one
// blank row apart, as a layout.Node.
func (m model) screen() layout.Node {
	// Below 12 rows the five-row banner title does not leave room for the
	// rest, so a bold word stands in; below 15 the blank rows go too.
	title := widgets.BigText("SETUP", widgets.FontBlock, m.t)
	gap := 1
	if m.height > 0 && m.height < 12 {
		title = ansi.NewStyle().Bold().Render("SETUP")
	}
	if m.height > 0 && m.height < 15 {
		gap = 0
	}
	alertWidth := 50
	if m.width > 0 {
		alertWidth = min(alertWidth, m.width)
	}

	status := "No step applied yet."
	if m.applied != "" {
		status = "Applied: " + m.applied
	}
	alert := widgets.Alert(status, widgets.VariantInfo, m.t, alertWidth)

	footer := setupFooter.Render("enter apply  ctrl+c quit")
	return layout.Column(gap,
		layout.FlexChild{Node: layout.Block(title)},
		layout.FlexChild{Node: layout.Block(alert)},
		layout.FlexChild{Node: m.menu.LayoutNode()},
		layout.FlexChild{Node: layout.Block(footer)},
	)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
