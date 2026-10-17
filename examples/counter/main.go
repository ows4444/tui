// Command counter is the smallest program that treats the mouse as a first
// class input. Click - and + (or scroll the wheel) to change the count, click
// a step chip to change how much each press adds, click Reset to start over.
// The keyboard does the same: + and - (or the arrows), s for the next step,
// 0 or r to reset, ? for help, q to quit. A button lights up when the pointer
// is over it and fires when the click is released on it, as in a GUI.
//
//	go run ./examples/counter
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/theme"
	"github.com/ows4444/tui/widgets"
)

// steps are the amounts a press can add.
var steps = [...]int{1, 5, 10}

// bound is how far the count may go either way; the gauge spans it.
const bound = 100

type model struct {
	count   int
	step    int    // index into steps
	hover   string // id of the target under the pointer
	pressed string // id of the target the left button went down on
	help    bool

	width, height int
}

func initialModel() model { return model{} }

func (m model) Init() tui.Cmd { return nil }

func (m model) add(n int) model {
	m.count = max(-bound, min(bound, m.count+n))
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tui.MouseEvent:
		return m.mouse(msg), nil
	case tui.Key:
		return m.key(msg)
	}
	return m, nil
}

func (m model) key(k tui.Key) (tui.Model, tui.Cmd) {
	switch k.Type {
	case tui.KeyCtrlC:
		return m, tui.Quit()
	case tui.KeyEsc:
		if m.help {
			m.help = false
			return m, nil
		}
		return m, tui.Quit()
	case tui.KeyUp, tui.KeyRight:
		return m.add(steps[m.step]), nil
	case tui.KeyDown, tui.KeyLeft:
		return m.add(-steps[m.step]), nil
	case tui.KeyRunes:
		switch k.Text {
		case "q":
			return m, tui.Quit()
		case "+", "=", "k", "l":
			return m.add(steps[m.step]), nil
		case "-", "j", "h":
			return m.add(-steps[m.step]), nil
		case "0", "r":
			m.count = 0
		case "s":
			m.step = (m.step + 1) % len(steps)
		case "?":
			m.help = !m.help
		}
	}
	return m, nil
}

// activate runs the action behind a click target.
func (m model) activate(id string) model {
	switch {
	case id == "dec":
		return m.add(-steps[m.step])
	case id == "inc":
		return m.add(steps[m.step])
	case id == "reset":
		m.count = 0
	case strings.HasPrefix(id, "step-"):
		m.step, _ = strconv.Atoi(strings.TrimPrefix(id, "step-"))
	}
	return m
}

func (m model) mouse(ev tui.MouseEvent) model {
	hit, ok := hittest.HitMap(m.screen(), m.size()).At(ev.X, ev.Y)
	id := ""
	if ok {
		id = hit.ID
	}
	m.hover = id
	switch ev.Button {
	case tui.MouseButtonWheelUp:
		return m.add(steps[m.step])
	case tui.MouseButtonWheelDown:
		return m.add(-steps[m.step])
	case tui.MouseButtonLeft:
	default:
		return m
	}
	switch ev.Action {
	case tui.MouseActionPress:
		m.pressed = id
	case tui.MouseActionRelease:
		if m.pressed != "" && m.pressed == id {
			m = m.activate(id)
		}
		m.pressed = ""
	}
	return m
}

func (m model) size() layout.Size {
	if m.width == 0 {
		return layout.Size{W: 80, H: 24}
	}
	return layout.Size{W: m.width, H: m.height}
}

// button draws a labelled click target. The label's width does not change with
// the state, so the layout does not move under the pointer.
func (m model) button(id, label string, th theme.Theme) layout.Node {
	st := ansi.NewStyle().Foreground(th.Text)
	switch {
	case m.pressed == id && m.hover == id:
		st = ansi.NewStyle().Bold().Reverse()
	case m.hover == id:
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	}
	return layout.Named(id, layout.Block(st.Render("[ "+label+" ]")))
}

// chip draws one step choice; the chosen one is filled.
func (m model) chip(i int, th theme.Theme) layout.Node {
	id := "step-" + strconv.Itoa(i)
	label := " " + strconv.Itoa(steps[i]) + " "
	st := ansi.NewStyle().Foreground(th.Muted)
	switch {
	case i == m.step:
		st = ansi.NewStyle().Bold().Foreground(th.TextInverse).Background(th.Primary)
	case m.hover == id:
		st = ansi.NewStyle().Foreground(th.Primary).Underline()
	}
	return layout.Named(id, layout.Block(st.Render(label)))
}

// gauge is a bar from -bound to +bound with a marker at the count.
func (m model) gauge(w int, th theme.Theme) string {
	pos := (m.count + bound) * (w - 1) / (2 * bound)
	track := ansi.NewStyle().Foreground(th.Muted)
	mark := ansi.NewStyle().Bold().Foreground(th.Primary)
	var b strings.Builder
	for i := 0; i < w; i++ {
		switch {
		case i == pos:
			b.WriteString(mark.Render("◆"))
		case i == (w-1)/2:
			b.WriteString(track.Render("┼"))
		default:
			b.WriteString(track.Render("─"))
		}
	}
	return b.String()
}

func (m model) screen() layout.Node {
	th := theme.DarkTheme()
	inner := 44
	if m.width > 0 {
		inner = max(20, min(inner, m.width-4))
	}
	compact := m.height > 0 && m.height < 17 // drop the gaps and the footer
	gap := 1
	if compact {
		gap = 0
	}

	value := ansi.NewStyle().Bold().Foreground(th.Success)
	switch {
	case m.count < 0:
		value = ansi.NewStyle().Bold().Foreground(th.Error)
	case m.count == 0:
		value = ansi.NewStyle().Bold().Foreground(th.Muted)
	}
	number := widgets.Center(value.Render(fmt.Sprintf("%+d", m.count)), inner, 1)

	var chips []layout.FlexChild
	chips = append(chips, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render("Step"))})
	for i := range steps {
		chips = append(chips, layout.FlexChild{Node: m.chip(i, th)})
	}

	hints := []widgets.Hint{{Key: "+ -", Action: "change"}, {Key: "s", Action: "step"}, {Key: "?", Action: "help"}, {Key: "q", Action: "quit"}}
	foot := widgets.Footer(inner, hints...)
	if m.help && !compact {
		foot = strings.Join([]string{
			"+ - ↑ ↓ ← →  change by the step",
			"s            next step size",
			"0 r          reset",
			"wheel        change by the step",
			"?            close this help",
			"q esc        quit",
		}, "\n")
	}

	kids := []layout.FlexChild{
		{Node: layout.Block(ansi.NewStyle().Bold().Foreground(th.Primary).Render("Counter"))},
		{Node: layout.Block(number)},
		{Node: layout.Block(m.gauge(inner, th))},
		{Node: layout.Row(2,
			layout.FlexChild{Node: m.button("dec", "−", th)},
			layout.FlexChild{Node: m.button("reset", "Reset", th)},
			layout.FlexChild{Node: m.button("inc", "+", th)},
		)},
		{Node: layout.Row(1, chips...)},
	}
	if !compact {
		kids = append(kids, layout.FlexChild{Node: layout.Block(ansi.NewStyle().Faint().Render(foot))})
	}
	body := layout.Column(gap, kids...)
	box := layout.NewBox().Border(th.Border).PaddingAll(1)
	if compact {
		box = layout.NewBox().Border(th.Border)
	}
	panel := layout.BoxNode(box, layout.Fixed(body, layout.Size{W: inner}))
	return layout.Stack(layout.Anchor(layout.Center, 0, 0, panel))
}

func (m model) View() string {
	return layout.DrawTight(m.screen(), m.size())
}

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseAllMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
