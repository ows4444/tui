package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/datatable"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/treeview"
	"github.com/ows4444/tui/widgets"
)

// outerWidth is the widest the panel gets, border included; a narrower
// terminal shrinks it to the window.
const outerWidth = 60

// tallPanel is the terminal height the airy panel needs (padding and blank
// rows between sections); a shorter terminal gets a compact one.
const tallPanel = 14

const sampleJSON = `{
  "service": "inspector",
  "healthy": true,
  "replicas": 3,
  "region": null,
  "tags": ["edge", "prod"]
}`

type model struct {
	tabs  tabs.Model
	table datatable.Model
	tree  treeview.Model

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

func initialModel() model {
	tree, err := treeview.FromJSON([]byte(sampleJSON))
	if err != nil {
		panic(err)
	}
	return model{
		tabs: tabs.New("Table", "JSON"),
		table: datatable.New(
			[]string{"Name", "Status", "CPU"},
			[][]string{
				{"api-1", "running", "12%"},
				{"api-2", "running", "8%"},
				{"api-3", "degraded", "41%"},
			},
		),
		tree: tree,
	}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width, m.height = rs.Width, rs.Height
		return m, nil
	}
	key, ok := msg.(tui.Key)
	if !ok {
		return m, nil
	}

	if key.Type == tui.KeyCtrlC || key.Type == tui.KeyEsc {
		return m, tui.Quit()
	}
	if key.Type == tui.KeyRunes && key.Text == "q" {
		return m, tui.Quit()
	}

	if key.Type == tui.KeyLeft || key.Type == tui.KeyRight || key.Type == tui.KeyTab {
		next, cmd := m.tabs.Update(msg)
		m.tabs = next
		return m, cmd
	}

	if m.tabs.Active() == 0 {
		next, cmd := m.table.Update(msg)
		m.table = next
		return m, cmd
	}
	next, cmd := m.tree.Update(msg)
	m.tree = next
	return m, cmd
}

func (m model) View() string {
	t := theme.DarkTheme()

	body := m.table.LayoutNode()
	if m.tabs.Active() == 1 {
		body = m.tree.LayoutNode()
	}

	outer := outerWidth
	if m.width > 0 {
		outer = min(outerWidth, m.width)
	}
	compact := m.height > 0 && m.height < tallPanel
	pad, gap := 1, 1
	if compact {
		pad, gap = 0, 0
	}
	hints := []widgets.Hint{
		{Key: "←→", Action: "tabs"},
		{Key: "↑↓", Action: "move"},
		{Key: "enter", Action: "select/toggle"},
		{Key: "q", Action: "quit"},
	}
	help := widgets.KeyHints("  ", hints...)
	if inner := outer - 2 - 2*pad; ansi.Width(help) > inner {
		// Too narrow for the words: keep the keys alone.
		for i := range hints {
			hints[i].Action = ""
		}
		help = ansi.Truncate(widgets.KeyHints(" ", hints...), inner)
	}

	content := layout.Column(gap,
		layout.FlexChild{Node: m.tabs.LayoutNode()},
		layout.FlexChild{Node: body},
		layout.FlexChild{Node: layout.Block(help)},
	)
	ui := layout.BoxNode(layout.NewBox().Border(t.Border).PaddingAll(pad), content)
	// Pin the outer width; the height is whatever the content needs.
	return layout.Draw(ui, layout.Constraints{MinW: outer, MaxW: outer, MaxH: layout.Unbounded})
}

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
