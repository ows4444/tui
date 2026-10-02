package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/accordion"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/tabs"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// width is the content width of the panel: wide enough for the key hints,
// which the old string layout let spill past the box's right border. A
// terminal narrower than the panel shrinks it (see model.contentWidth).
const width = 52

var tabLabels = []string{"General", "Advanced"}

// tabName is the layout name of tab i, so its rectangle can be found again
// with layout.RectOf.
func tabName(i int) string { return "tab-" + strconv.Itoa(i) }

type model struct {
	tabs     tabs.Model
	general  accordion.Model
	advanced accordion.Model

	termWidth int // terminal width from the last ResizeMsg; 0 until known
}

// contentWidth is the panel's content width: width, or what is left of the
// terminal once the border and padding (4 columns) are paid for.
func (m model) contentWidth() int {
	if m.termWidth > 0 {
		return max(10, min(width, m.termWidth-4))
	}
	return width
}

func initialModel() model {
	return model{
		tabs: tabs.New(tabLabels...),
		general: accordion.New(
			accordion.Section{Title: "Display", Content: "Theme: Dark\nFont size: 14"},
			accordion.Section{Title: "Notifications", Content: "Sound: On\nDesktop alerts: Off"},
		),
		advanced: accordion.New(
			accordion.Section{Title: "Network", Content: "Proxy: none\nTimeout: 30s"},
			accordion.Section{Title: "Developer", Content: "Debug mode: Off\nVerbose logging: Off"},
		),
	}
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.termWidth = rs.Width
		return m, nil
	}
	key, isKey := msg.(tui.Key)
	if isKey {
		if key.Type == tui.KeyCtrlC || key.Type == tui.KeyEsc ||
			(key.Type == tui.KeyRunes && key.Text == "q") {
			return m, tui.Quit()
		}
		// Left/Right step the tab bar (clamped); everything except Tab goes
		// to the accordion of the active tab.
		switch key.Type {
		case tui.KeyLeft, tui.KeyRight:
			next, cmd := m.tabs.Update(msg)
			m.tabs = next
			return m, cmd
		case tui.KeyTab:
		default:
			return m.updateAccordion(msg)
		}
	} else if _, ok := msg.(tui.MouseEvent); !ok {
		return m, nil
	}

	// Tab, Shift+Tab and a left click on a tab are the automatic ring's: one
	// ring item per tab, focusing an item activates that tab.
	ring := focus.New(len(tabLabels)).Set(m.tabs.Active())
	ring, cmd := ring.RouteAuto(msg, m.tabZones(), m.tabFields(&m.tabs)...)
	m.tabs.SetActive(ring.Current())
	return m, cmd
}

func (m model) updateAccordion(msg tui.Msg) (tui.Model, tui.Cmd) {
	if m.tabs.Active() == 0 {
		next, cmd := m.general.Update(msg)
		m.general = next
		return m, cmd
	}
	next, cmd := m.advanced.Update(msg)
	m.advanced = next
	return m, cmd
}

// tabFields is the ring's view of the tab bar: one field per tab, which
// activates that tab when focused.
func (m model) tabFields(t *tabs.Model) []focus.Field {
	fs := make([]focus.Field, len(tabLabels))
	for i := range fs {
		i := i
		fs[i] = focus.Field{Focus: func() tui.Cmd { t.SetActive(i); return nil }}
	}
	return fs
}

// tabZones reads each tab's rectangle from the layout that View draws
// (layout.RectOf on the named tab nodes), not worked out by hand, so they
// cannot drift from what is on screen. The region ID is the tab index.
func (m model) tabZones() hittest.Map[int] {
	root := m.screen()
	size := root.Measure(layout.Unconstrained())
	var regions hittest.Map[int]
	for i := range tabLabels {
		if r, ok := layout.RectOf(root, size, tabName(i)); ok {
			regions = regions.Add(i, hittest.Rect(r))
		}
	}
	return regions
}

// tabAt reports which tab a mouse event landed on.
func (m model) tabAt(ev tui.MouseEvent) (int, bool) {
	h, ok := m.tabZones().AtEvent(ev)
	return h.ID, ok
}

// tabBar draws each tab as its own named node so layout.Rects can report
// where it is. tabs.Model still owns which tab is active and the keys; its
// LayoutNode is a single leaf, so the per-tab styling is repeated here, and
// TestTabBarMatchesTabsModelView keeps the two in step.
func (m model) tabBar() layout.Node {
	th := m.tabs.Theme
	active := th.ResolvedStates().Selected.Bold()
	inactive := ansi.NewStyle().Foreground(th.Muted)
	var kids []layout.FlexChild
	for i, l := range tabLabels {
		style, label := inactive, " "+l+" "
		if i == m.tabs.Active() {
			style, label = active, "["+l+"]"
		}
		kids = append(kids, layout.FlexChild{Node: layout.Named(tabName(i), layout.Block(style.Render(label)))})
	}
	return layout.Row(1, kids...)
}

// screen builds the settings panel as a layout.Node tree: the tab bar, the
// active tab's accordion and the key hints, in a padded box width cells wide.
func (m model) screen() layout.Node {
	t := theme.DarkTheme()

	acc := m.general
	if m.tabs.Active() == 1 {
		acc = m.advanced
	}

	hints := []widgets.Hint{
		{Key: "←→/click", Action: "tabs"},
		{Key: "↑↓", Action: "move"},
		{Key: "enter", Action: "toggle"},
		{Key: "q", Action: "quit"},
	}
	help := widgets.KeyHints("  ", hints...)
	if ansi.Width(help) > m.contentWidth() {
		// Too narrow for the words: keep the keys alone.
		for i := range hints {
			hints[i].Action = ""
		}
		help = ansi.Truncate(widgets.KeyHints(" ", hints...), m.contentWidth())
	}

	body := layout.Column(1,
		layout.FlexChild{Node: m.tabBar()},
		layout.FlexChild{Node: acc.LayoutNode()},
		layout.FlexChild{Node: layout.Block(help)},
	)
	sized := layout.Fixed(body, layout.Size{W: m.contentWidth()})
	return layout.BoxNode(layout.NewBox().Border(t.Border).PaddingAll(1), sized)
}

func (m model) View() string {
	return layout.Draw(m.screen(), layout.Unconstrained())
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithMouse(tui.MouseClick)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
