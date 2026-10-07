// Command menus puts the three menu widgets on one screen: a menubar across
// the top (F10, or Alt and a title's letter), a nested menu in the body (Enter
// drills into a group, Esc goes back up) and a contextmenu of actions for the
// current place ("a", Shift+F10 or a right click). The last line says what
// was chosen and from which menu.
//
// Both overlays (the bar's dropdown and the context menu) are drawn by Render
// over the finished screen, and whichever is open takes the keys until it
// closes.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/contextmenu"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/menu"
	"github.com/ows4444/tui/menubar"
	"github.com/ows4444/tui/widgets"
)

type model struct {
	bar menubar.Model
	nav menu.Model
	ctx contextmenu.Model

	last          string // the last choice, in words
	width, height int    // terminal size from the last ResizeMsg; 0 until known
}

func initialModel() model {
	bar := menubar.New(
		menubar.Menu{Title: "File", Accel: 'f', Items: []menubar.Item{
			{Label: "New", Value: "new", Shortcut: "ctrl+n", Accel: 'n'},
			{Label: "Open", Value: "open", Shortcut: "ctrl+o", Accel: 'o'},
			{Separator: true},
			{Label: "Quit", Value: "quit", Shortcut: "q", Accel: 'q'},
		}},
		menubar.Menu{Title: "View", Accel: 'v', Items: []menubar.Item{
			{Label: "Compact", Value: "compact"},
			{Label: "Wide", Value: "wide"},
		}},
		menubar.Menu{Title: "Help", Accel: 'h', Items: []menubar.Item{
			{Label: "About", Value: "about"},
		}},
	)
	bar.Mouse = true

	nav := menu.New([]menu.Item{
		{Label: "Projects", Children: []menu.Item{
			{Label: "alpha", Value: "project alpha"},
			{Label: "beta", Value: "project beta"},
			{Label: "gamma", Value: "project gamma"},
		}},
		{Label: "Environments", Children: []menu.Item{
			{Label: "staging", Value: "environment staging"},
			{Label: "production", Value: "environment production"},
		}},
		{Label: "Settings", Value: "settings"},
	})
	nav.Mouse = true

	ctx := contextmenu.New(
		contextmenu.Item{Label: "Open", Value: "open", Shortcut: "enter", Accel: 'o'},
		contextmenu.Item{Label: "Rename", Value: "rename", Accel: 'r'},
		contextmenu.Item{Label: "Duplicate", Value: "duplicate", Accel: 'd'},
		contextmenu.Item{Separator: true},
		contextmenu.Item{Label: "Delete", Value: "delete", Disabled: true},
	)
	ctx.Label = "Actions"
	ctx.Mouse = true
	// "a" opens it as well as the default Shift+F10, which not every terminal
	// reports.
	ctx.KeyMap.Open = keymap.NewBinding("actions", "a", "shift+f10")

	return model{bar: bar, nav: nav, ctx: ctx, last: "nothing chosen yet"}
}

func (m model) Init() tui.Cmd { return nil }

// compact reports whether the terminal is too short for blank rows between
// the sections.
func (m model) compact() bool { return m.height > 0 && m.height < 14 }

// navTop is the screen row the nested menu starts on: under the bar and the
// heading, and a blank row when there is room for one.
func (m model) navTop() int {
	if m.compact() {
		return 2
	}
	return 3
}

// place tells each widget where it is drawn and how big the screen is, which
// they need to hit-test the mouse and to keep an overlay on screen.
func (m model) place() model {
	m.bar.Bounds = hittest.Rect{X: 0, Y: 0, W: m.width, H: 1}
	m.bar.ScreenW, m.bar.ScreenH = m.width, m.height
	m.ctx.ScreenW, m.ctx.ScreenH = m.width, m.height
	m.nav.Bounds = hittest.Rect{X: 0, Y: m.navTop(), W: m.width, H: strings.Count(m.nav.View(), "\n") + 1}
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.place(), nil
	case menubar.SelectedMsg:
		m.last = "menu bar: " + m.bar.Menus[msg.Menu].Title + " > " + msg.Item.Label
		if msg.Item.Value == "quit" {
			return m, tui.Quit()
		}
		return m, nil
	case contextmenu.SelectedMsg:
		m.last = "actions: " + msg.Item.Label
		return m, nil
	case menu.SelectedMsg:
		m.last = "navigation: " + msg.Item.Value
		return m, nil
	case tui.Key:
		return m.key(msg)
	case tui.MouseEvent:
		return m.mouse(msg)
	}
	return m, nil
}

// key sends a key to the open overlay if there is one, then to whichever
// widget it opens, and otherwise to the nested menu.
func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	var cmd tui.Cmd
	switch {
	case m.ctx.Open():
		m.ctx, cmd = m.ctx.Update(k)
	case m.bar.Open():
		m.bar, cmd = m.bar.Update(k)
	case keymap.Matches(k, m.ctx.KeyMap.Open):
		// Opened from the keyboard it appears beside the nested menu.
		m.ctx.AnchorX, m.ctx.AnchorY = 4, m.navTop()+1
		m.ctx, cmd = m.ctx.Update(k)
	case keymap.Matches(k, m.bar.KeyMap.Focus), k.Mod&input.ModAlt != 0:
		m.bar, cmd = m.bar.Update(k)
	case k.Type == tui.KeyRunes && k.Text == "q":
		return m, tui.Quit()
	default:
		m.nav, cmd = m.nav.Update(k)
	}
	return m.place(), cmd
}

// mouse follows the same order as key: an open overlay first, then the bar's
// row and a right click, and otherwise the nested menu.
func (m model) mouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	var cmd tui.Cmd
	switch {
	case m.ctx.Open():
		m.ctx, cmd = m.ctx.Update(ev)
	case m.bar.Open(), ev.Y == 0:
		m.bar, cmd = m.bar.Update(ev)
	case ev.Button == tui.MouseButtonRight:
		m.ctx, cmd = m.ctx.Update(ev)
	default:
		m.nav, cmd = m.nav.Update(ev)
	}
	return m.place(), cmd
}

var heading = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)

// contentWidth is the width the body is cut to: the terminal's, or 60 before
// the first ResizeMsg.
func (m model) contentWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 60
}

// hints is the key legend: with words when they fit, else the keys alone.
func (m model) hints() string {
	hs := []widgets.Hint{
		{Key: "f10", Action: "menu bar"}, {Key: "enter", Action: "open"}, {Key: "esc", Action: "back"},
		{Key: "a", Action: "actions"}, {Key: "q", Action: "quit"},
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
	w := m.contentWidth()
	gap := "\n\n"
	if m.compact() {
		gap = "\n"
	}
	base := strings.Join([]string{
		m.bar.View(),
		heading.Render(ansi.Truncate("Workspace", w)) + "\n" + m.nav.View(),
		ansi.Truncate("Last choice: "+m.last, w),
		m.hints(),
	}, gap)
	if m.width > 0 && m.height > 0 {
		// Fill the terminal, so an overlay has the whole screen to open on.
		base = layout.DrawTight(layout.Block(base), layout.Size{W: m.width, H: m.height})
	}
	// Overlays go on last. A closed one returns the base unchanged.
	return m.ctx.Render(m.bar.Render(base))
}

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
