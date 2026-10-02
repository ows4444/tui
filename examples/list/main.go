package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/multiselect"
)

type model struct {
	list      multiselect.Model
	confirmed bool

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

// listTopOffset is how many rows sit above the first item in View's
// output when the terminal size is unknown or roomy: the box's top border,
// its top padding, the title, and the blank line after it. Used to map a
// mouse click's Y back to an item index; see model.topOffset for the compact
// layout.
const listTopOffset = 4

// Terminal sizes below which View sheds decoration: under wideLayout columns
// the "Done" box moves into the title row; under tallLayout rows the padding
// and the blank rows between sections go.
const (
	wideLayout = 74
	tallLayout = 14
)

// narrow reports whether the terminal is too narrow for the stats box beside
// the list.
func (m model) narrow() bool { return m.width > 0 && m.width < wideLayout }

// compact reports whether the terminal is too short for the airy layout.
func (m model) compact() bool { return m.height > 0 && m.height < tallLayout }

// topOffset is how many rows sit above the first item: the border, the top
// padding, the title and the blank row after it (the last two are gone when
// compact).
func (m model) topOffset() int {
	if m.compact() {
		return 2
	}
	return listTopOffset
}

func initialModel() model {
	return model{
		list: multiselect.NewStrings(
			"Design raw-mode terminal layer",
			"Build ANSI styling package",
			"Write escape-sequence key parser",
			"Implement Elm-architecture event loop",
			"Add diff-based renderer",
			"Ship zero-dependency TUI",
		),
	}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC, tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyRunes:
			if msg.Text == "q" {
				return m, tui.Quit()
			}
		}
	case tui.MouseEvent:
		if msg.Action == tui.MouseActionPress && msg.Button == tui.MouseButtonLeft {
			if i := msg.Y - m.topOffset(); i >= 0 && i < len(m.list.Items) {
				m.list.SetCursor(i)
				m.list.Toggle(i)
			}
		}
		return m, nil
	case multiselect.ConfirmedMsg:
		m.confirmed = true
		return m, nil
	}

	var cmd tui.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

var (
	title   = ansi.NewStyle().Bold().Underline()
	doneSty = ansi.NewStyle().Foreground(ansi.BrightGreen)
	dimSty  = ansi.NewStyle().Faint()
)

func (m model) View() string {
	pad, gap := 1, 1
	if m.compact() {
		pad, gap = 0, 0
	}
	done := fmt.Sprintf("%s / %d", doneSty.Render(fmt.Sprintf("%d", len(m.list.SelectedIndexes()))), len(m.list.Items))

	// Column's gap puts the blank row between sections. (Block("") is 0x0,
	// not a blank row, so it cannot be used as a spacer.)
	footer := dimSty.Render("up/down move  space/click toggle  enter confirm  q quit")
	heading := title.Render("Tasks")
	if m.narrow() {
		footer = dimSty.Render("up/down space enter q")
		heading += "  " + done + " done"
	}
	if m.confirmed {
		footer += "\n" + doneSty.Render("Confirmed!")
	}
	body := layout.Column(gap,
		layout.FlexChild{Node: layout.Block(heading)},
		layout.FlexChild{Node: m.list.LayoutNode()},
		layout.FlexChild{Node: layout.Block(footer)},
	)
	box := layout.NewBox().Border(layout.RoundedBorder()).Padding(pad, 1, pad, 1)
	if m.narrow() {
		// Clip the content to the window, then frame it.
		room := max(1, m.width-4)
		lines := strings.Split(layout.Draw(body, layout.Unconstrained()), "\n")
		for i, l := range lines {
			lines[i] = ansi.Truncate(l, room)
		}
		return layout.Draw(layout.BoxNode(box, layout.Block(strings.Join(lines, "\n"))), layout.Unconstrained())
	}
	listBox := layout.BoxNode(box, body)

	stats := "Done\n\n" + done
	statsBox := layout.BoxNode(layout.NewBox().Border(layout.NormalBorder()).PaddingAll(1), layout.Block(stats))

	// CrossStart keeps the stats box its own height instead of stretching it
	// to the list box's.
	ui := layout.Row(2,
		layout.FlexChild{Node: listBox},
		layout.FlexChild{Node: statsBox, CrossAlign: layout.CrossStart},
	)
	return layout.Draw(ui, layout.Unconstrained())
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithMouse(tui.MouseClick)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
