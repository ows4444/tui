// Command router is the canonical reference for switching between
// independent top-level screens in this repo, the way examples/focus is the
// canonical reference for Focus/Blur coordination.
//
// This is a genuinely different pattern from wizard.Model: wizard.Model's
// Next/Back move linearly through an ordered sequence of steps that all
// belong to one flow (a single form spread over several pages, sharing one
// notion of "progress" via widgets.Stepper). Router, by contrast, has no
// ordering and no shared progress — screenMenu, screenSettings and
// screenAbout are unrelated destinations you can jump directly between (menu
// selection, or Esc to go back), each with its own independent state, not
// steps 1/2/3 of a single wizard. If your screens have a natural order and
// collectively represent one multi-step task, reach for wizard.Model
// instead; if they're independent top-level destinations a user picks
// between, this Router shape is the pattern to copy.
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/picker"
	"github.com/ows4444/tui/textinput"
)

// screen enumerates the top-level destinations. It's the single field that
// Update and View both branch on — nothing else decides what's on screen.
type screen int

const (
	screenMenu screen = iota
	screenSettings
	screenAbout
)

// model composes one independent sub-model per screen. Each sub-model is a
// field here, not something reconstructed on every screen switch, which is
// exactly what lets a screen's state (e.g. Settings' typed text) survive a
// round trip through other screens.
type model struct {
	screen screen

	menu     picker.Model
	settings textinput.Model

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

func initialModel() model {
	m := model{
		screen: screenMenu,
		menu:   picker.NewStrings("Settings", "About", "Quit"),
	}
	m.settings = textinput.New()
	m.settings.Prompt = "Display name: "
	m.settings.Placeholder = "Ada Lovelace"
	m.settings.Width = 30
	m.settings.CharLimit = 60
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width, m.height = rs.Width, rs.Height
		// Leave the border, padding and prompt room: the input is 30 wide, or less.
		m.settings.Width = 30
		if m.width > 0 {
			m.settings.Width = max(3, min(30, m.width-4-ansi.Width(m.settings.Prompt)))
		}
		return m, nil
	}
	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenSettings:
		return m.updateSettings(msg)
	case screenAbout:
		return m.updateAbout(msg)
	}
	return m, nil
}

// updateMenu drives the menu screen's own picker.Model and switches m.screen
// when a menu item is confirmed. Quitting from the menu is the one global
// action that doesn't map to another screen.
func (m model) updateMenu(msg tui.Msg) (tui.Model, tui.Cmd) {
	if key, ok := msg.(tui.Key); ok {
		if key.Type == tui.KeyCtrlC {
			return m, tui.Quit()
		}
	}

	var cmd tui.Cmd
	m.menu, cmd = m.menu.Update(msg)

	if sel, ok := msg.(picker.SelectedMsg); ok {
		switch sel.Item.Value {
		case "Settings":
			m.screen = screenSettings
			focusCmd := m.settings.Focus()
			return m, focusCmd
		case "About":
			m.screen = screenAbout
			return m, nil
		case "Quit":
			return m, tui.Quit()
		}
	}

	return m, cmd
}

// updateSettings forwards keys to the Settings screen's own textinput.Model.
// Esc returns to the menu without discarding whatever's been typed — the
// textinput.Model stays exactly as it is on the model, so switching back to
// this screen later shows the same value.
func (m model) updateSettings(msg tui.Msg) (tui.Model, tui.Cmd) {
	if key, ok := msg.(tui.Key); ok {
		switch key.Type {
		case tui.KeyCtrlC:
			return m, tui.Quit()
		case tui.KeyEsc:
			m.settings.Blur()
			m.screen = screenMenu
			return m, nil
		}
	}

	var cmd tui.Cmd
	m.settings, cmd = m.settings.Update(msg)
	return m, cmd
}

// updateAbout is mostly static; only Esc/Ctrl+C do anything.
func (m model) updateAbout(msg tui.Msg) (tui.Model, tui.Cmd) {
	if key, ok := msg.(tui.Key); ok {
		switch key.Type {
		case tui.KeyCtrlC:
			return m, tui.Quit()
		case tui.KeyEsc:
			m.screen = screenMenu
			return m, nil
		}
	}
	return m, nil
}

var title = ansi.NewStyle().Bold().Underline()
var dimSty = ansi.NewStyle().Faint()

// View draws the current screen; see layout.
func (m model) View() string { return layout.Draw(m.layout(), layout.Unconstrained()) }

// layout is the current screen as a layout.Node: a title, the screen's own
// content and a hint, one blank row apart, in a rounded padded box. Exactly one
// screen's content is in the tree, never a blend of two.
func (m model) layout() layout.Node {
	var heading, hint string
	var content layout.Node
	switch m.screen {
	case screenMenu:
		heading, hint = "Router reference — Menu", "enter select  ctrl+c quit"
		content = m.menu.LayoutNode()
	case screenSettings:
		heading, hint = "Router reference — Settings", "esc back  ctrl+c quit"
		content = layout.Block(m.settings.View())
	case screenAbout:
		heading, hint = "Router reference — About", "esc back  ctrl+c quit"
		about := "This example demonstrates switching between independent\n" +
			"top-level screens via an internal `screen` field, distinct\n" +
			"from wizard.Model's linear within-one-flow step navigation."
		if m.width > 0 && m.width-4 < 58 {
			about = ansi.Wrap(about, max(1, m.width-4)) // reflow to the window
		}
		content = layout.Block(about)
	}
	if m.width > 0 {
		heading = ansi.Truncate(heading, max(1, m.width-4))
	}
	// With under 12 rows the padding and the blank rows between the sections go.
	gap := 1
	if m.height > 0 && m.height < 12 {
		gap = 0
	}
	body := layout.Column(gap,
		layout.FlexChild{Node: layout.Block(title.Render(heading))},
		layout.FlexChild{Node: content},
		layout.FlexChild{Node: layout.Block(dimSty.Render(hint))},
	)
	return layout.BoxNode(layout.NewBox().Border(layout.RoundedBorder()).Padding(gap, 1, gap, 1), body)
}

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
