// Command welcomescreen is the canonical reference for a static
// informational splash composed entirely from the existing widget catalog
// — widgets.Panel, widgets.BigText, widgets.Header and widgets.KeyValue
// side by side in a layout.Row. It introduces
// no new widget capability: every visual element here is an existing
// exported API called with its own arguments, the same "composition, not
// a new primitive" shape spec #25 calls for.
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

type model struct {
	t theme.Theme

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

// Terminal sizes the full panel needs: fullWidth columns for the logo and
// info panels side by side, fullHeight rows for them stacked under the header.
// Narrower, the logo goes; shorter, both panels go and the facts are listed
// plainly.
const (
	fullWidth  = 60
	fullHeight = 16
)

func initialModel() model {
	return model{t: theme.DarkTheme()}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width, m.height = rs.Width, rs.Height
		return m, nil
	}
	if key, ok := msg.(tui.Key); ok {
		switch key.Type {
		case tui.KeyCtrlC, tui.KeyEnter, tui.KeyEsc:
			return m, tui.Quit()
		}
	}
	return m, nil
}

var footerStyle = ansi.NewStyle().Faint()

// screen is the whole welcome panel as a layout.Node: the header, the logo and
// info panels side by side, and the footer, in a rounded padded box.
func (m model) screen() layout.Node {
	header := widgets.Header("Welcome to TUI", m.t)

	meta := widgets.KeyValue([]widgets.KV{
		{Key: "Version", Value: "v1.0.0"},
		{Key: "User", Value: "Ada Lovelace"},
		{Key: "Workspace", Value: "acline"},
	}, m.t)
	footer := footerStyle.Render("enter/esc continue  ctrl+c quit")
	child := func(n layout.Node) layout.FlexChild { return layout.FlexChild{Node: n} }
	frame := layout.NewBox().Border(layout.RoundedBorder()).PaddingAll(1)

	if m.height > 0 && m.height < fullHeight {
		// Too short for panels: the facts as plain rows, no padding.
		body := layout.Column(0, child(layout.Block(header)), child(layout.Block(meta)), child(layout.Block(footer)))
		return layout.BoxNode(frame.Padding(0, 1, 0, 1), body)
	}

	infoWidth := 30
	var split layout.Node
	if m.width > 0 && m.width < fullWidth+2 {
		// Too narrow for both panels: the info panel alone.
		infoWidth = max(12, min(infoWidth, m.width-4))
		split = layout.Block(widgets.Panel("Info", meta, m.t, infoWidth))
	} else {
		logo := widgets.BigText("TUI", widgets.FontBlock, m.t)
		logoPanel := widgets.Panel("", logo, m.t, 24)
		infoPanel := widgets.Panel("Info", meta, m.t, infoWidth)
		split = layout.Row(2,
			layout.FlexChild{Node: layout.Block(logoPanel), CrossAlign: layout.CrossStart},
			layout.FlexChild{Node: layout.Block(infoPanel), CrossAlign: layout.CrossStart},
		)
	}

	body := layout.Column(1, child(layout.Block(header)), child(split), child(layout.Block(footer)))
	return layout.BoxNode(frame, body)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
