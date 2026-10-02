// Command table is a dedicated, standalone example for datatable.Model's
// interactive row selection: Up/Down move the cursor and Enter confirms
// the row under it via SelectedMsg. This is unlike examples/dashboard,
// where a table shows up only incidentally alongside several other
// widgets — here the table is the whole point, so every key handled and
// every line rendered is about demonstrating that one interaction.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/widgets"
)

var headers = []string{"Name", "Role"}

var rows = [][]string{
	{"Ada Lovelace", "Engineer"},
	{"Grace Hopper", "Rear Admiral"},
	{"Margaret Hamilton", "Director"},
	{"Katherine Johnson", "Mathematician"},
}

type model struct {
	table    datatable.Model
	selected *datatable.SelectedMsg
}

func initialModel() model {
	return model{table: datatable.New(headers, rows)}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if key, ok := msg.(tui.Key); ok {
		switch key.Type {
		case tui.KeyCtrlC, tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyRunes:
			if key.Text == "q" {
				return m, tui.Quit()
			}
		}
	}

	if sel, ok := msg.(datatable.SelectedMsg); ok {
		m.selected = &sel
		return m, nil
	}

	// Forward the raw Msg — Up/Down, Enter and anything else — straight
	// to the table; it owns cursor movement and emits SelectedMsg via
	// the returned Cmd when Enter confirms the row under the cursor.
	var cmd tui.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

// screen stacks the table, the selection line and the key hints with one blank
// row between them, as a layout.Node. The table is the datatable's own node,
// so it takes the width the layout gives it.
func (m model) screen() layout.Node {
	selection := "Selected: (none yet — press Enter)"
	if m.selected != nil {
		selection = fmt.Sprintf("Selected: %s", strings.Join(m.selected.Cells, " | "))
	}
	hints := widgets.KeyHints("  ",
		widgets.Hint{Key: "↑/↓", Action: "move"},
		widgets.Hint{Key: "enter", Action: "select"},
		widgets.Hint{Key: "q", Action: "quit"})

	return layout.Column(1,
		layout.FlexChild{Node: m.table.LayoutNode()},
		layout.FlexChild{Node: layout.Block(selection)},
		layout.FlexChild{Node: layout.Block(hints)},
	)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
