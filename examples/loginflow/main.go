// Command loginflow is the canonical reference for a login-shaped screen
// composed entirely from the existing widget catalog — widgets.BigText for
// the title, widgets.Banner for an announcement, and picker.Model
// configured as a numbered-select account menu. There is deliberately
// no authentication logic (no credential validation or storage) here:
// like termcn's own reference, "login" here means picking an account
// from a menu, nothing more. It introduces no new widget capability.
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
	t      theme.Theme
	menu   picker.Model
	chosen string
	done   bool

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

func initialModel() model {
	return model{
		t:    theme.DarkTheme(),
		menu: picker.NewStrings("Continue as Ada Lovelace", "Continue as guest", "Quit"),
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
		if sel.Item.Value == "Quit" {
			return m, tui.Quit()
		}
		m.chosen = sel.Item.Value
		m.done = true
		return m, nil
	}

	return m, cmd
}

var titleStyle = ansi.NewStyle().Faint()

// screen stacks the title, the banner and then either the sign-in
// confirmation or the account menu and its footer, one blank row apart.
func (m model) screen() layout.Node {
	block := func(s string) layout.FlexChild { return layout.FlexChild{Node: layout.Block(s)} }
	title := widgets.BigText("LOGIN", widgets.FontBlock, m.t)
	// The banner and alert are 50 wide, or the terminal's width when that is
	// narrower; a narrow banner gets the short message. With under 13 rows
	// the blank rows between the sections go too.
	bw, text := 50, "Pick an account to continue — no password required in this example."
	if m.width > 0 && m.width < bw {
		bw, text = m.width, "Pick an account to continue."
	}
	gap := 1
	if m.height > 0 && m.height < 13 {
		gap = 0
	}
	banner := widgets.Banner(text, widgets.VariantInfo, m.t, bw)

	if m.done {
		return layout.Column(gap, block(title), block(banner),
			block(widgets.Alert("Signed in as: "+m.chosen, widgets.VariantSuccess, m.t, bw)))
	}

	footer := titleStyle.Render("enter select  ctrl+c quit")
	return layout.Column(gap, block(title), block(banner),
		layout.FlexChild{Node: m.menu.LayoutNode()}, block(footer))
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
