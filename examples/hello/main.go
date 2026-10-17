// Command hello is the first program to read: a text field and two buttons,
// all of which work with the keyboard and the mouse. Type a name and press
// Enter or click Greet; Tab or a click moves focus; click Quit or press Ctrl+C
// to leave. The three widgets are named layout nodes, and each frame their
// rectangles are handed back to the widgets as Bounds, which is all the mouse
// needs.
//
//	go run ./examples/hello
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/textinput"
	"github.com/ows4444/tui/theme"
)

var names = []string{"name", "greet", "quit"}

type model struct {
	name     textinput.Model
	greet    button.Model
	quit     button.Model
	ring     focus.Ring
	greeting string

	width, height int
}

func initialModel() model {
	in := textinput.New()
	in.Prompt = "Your name: "
	in.Placeholder = "Ada"
	in.Width = 24
	in.Mouse = true
	in.Focus() // the blink Cmd comes back from Init
	greet, quit := button.New("Greet"), button.New("Quit")
	greet.ID, quit.ID = "greet", "quit"
	greet.Mouse, quit.Mouse = true, true
	return model{name: in, greet: greet, quit: quit, ring: focus.New(len(names))}
}

func (m model) Init() tui.Cmd { return m.name.Focus() }

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

// bounds copies each widget's drawn rectangle into its Bounds.
func (m model) bounds() model {
	root, size := m.screen(), m.size()
	if r, ok := layout.RectOf(root, size, "name"); ok {
		m.name.Bounds = r
	}
	if r, ok := layout.RectOf(root, size, "greet"); ok {
		m.greet.Bounds = r
	}
	if r, ok := layout.RectOf(root, size, "quit"); ok {
		m.quit.Bounds = r
	}
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.bounds(), nil
	case button.PressedMsg:
		if msg.ID == "quit" {
			return m, tui.Quit()
		}
		return m.sayHello(), nil
	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC:
			return m, tui.Quit()
		case msg.Type == tui.KeyEnter && m.ring.Current() == 0:
			return m.sayHello(), nil
		}
	}
	m = m.bounds()
	fields := []focus.Field{focus.Bind(&m.name), focus.Bind(&m.greet), focus.Bind(&m.quit)}
	ring, cmd := m.ring.RouteAuto(msg, focus.Zones(m.screen(), m.size(), names...), fields...)
	m.ring = ring
	return m, cmd
}

func (m model) sayHello() model {
	if n := strings.TrimSpace(m.name.Value()); n != "" {
		m.greeting = "Hello, " + n + "!"
	} else {
		m.greeting = "Please type your name first."
	}
	return m
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	greeting := ansi.NewStyle().Faint().Render("Press Enter, or click Greet.")
	if m.greeting != "" {
		greeting = ansi.NewStyle().Bold().Foreground(th.Success).Render(m.greeting)
	}
	kids := []layout.FlexChild{
		{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Hello, tui"))},
		{Node: layout.Named("name", layout.Block(m.name.View()))},
		{Node: layout.Row(2,
			layout.FlexChild{Node: layout.Named("greet", layout.Block(m.greet.View()))},
			layout.FlexChild{Node: layout.Named("quit", layout.Block(m.quit.View()))},
		)},
		{Node: layout.Block(greeting)},
	}
	// On a short terminal the padding, the gaps and the title go.
	gap, box := 1, layout.NewBox().Border(th.Border).PaddingAll(1)
	if m.height > 0 && m.height < 14 {
		gap, box, kids = 0, layout.NewBox().Border(th.Border), kids[1:]
	}
	panel := layout.BoxNode(box, layout.Fixed(layout.Column(gap, kids...), layout.Size{W: 36}))
	return layout.Stack(layout.Anchor(layout.Center, 0, 0, panel))
}

func (m model) View() string { return layout.DrawTight(m.screen(), m.size()) }

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
