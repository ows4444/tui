// Command focus is the canonical, minimal reference for coordinating
// Focus()/Blur() across mixed widget types in this repo.
//
// examples/form also does Blur-advance-Focus for its fields, but there it's
// incidental — one piece of a larger multi-step sign-up scenario. This
// example exists solely to demonstrate the pattern in isolation: a Model
// composing three different widget types (textinput.Model, passwordinput.Model,
// textarea.Model), a single `focused` field selecting which one is active,
// and Tab/Shift+Tab cycling focus with wraparound in both directions. The
// View also visibly marks whichever widget currently has focus, so the
// coordination is observable, not just internal bookkeeping.
//
// Which field is focused is tracked by a focus.Ring, which also does the
// Tab/Shift+Tab wraparound; this file only decides what "focus" and "blur"
// mean for each widget type.
package main

import (
	"fmt"
	"os"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/focus"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/passwordinput"
	"github.com/ows4444/tui/textarea"
	"github.com/ows4444/tui/textinput"
)

// field enumerates the widgets that participate in the tab order; a field
// is an index into the focus.Ring. Every other piece of coordination (which
// widget to Blur, which to Focus, which to forward keys to, which to draw
// highlighted) is derived from the ring's current index rather than tracked
// redundantly per widget.
type field int

const (
	fieldName field = iota
	fieldPassword
	fieldBio
	fieldCount // sentinel: number of focusable widgets, the ring's size
)

// model composes three genuinely different widget types on purpose, so the
// pattern reads clearly as "coordinate focus across mixed widgets" rather
// than "coordinate focus across N copies of the same widget".
type model struct {
	name     textinput.Model
	password passwordinput.Model
	bio      textarea.Model

	ring focus.Ring

	width, height int // terminal size from the last ResizeMsg; 0 until known

	// focusCmd carries the initial Focus() cmd from initialModel to Init:
	// Init can only return a Cmd, not a mutated Model, so the first Focus()
	// call has to happen on the persisted model in initialModel instead of
	// being reconstructed here.
	focusCmd tui.Cmd
}

func initialModel() model {
	var m model

	m.name = textinput.New()
	m.name.Prompt = "Name:     "
	m.name.Placeholder = "Ada Lovelace"
	m.name.Width = 30
	m.name.CharLimit = 60

	m.password = passwordinput.New()
	m.password.Prompt = "Password: "
	m.password.Width = 30
	m.password.CharLimit = 60

	m.bio = textarea.New()
	m.bio.Width = 30
	m.bio.Placeholder = "a few words about you..."

	m.ring = focus.New(int(fieldCount))
	m.focusCmd = m.ring.Sync(m.fields()...)
	return m
}

// Sizes of the framed layout: its chrome (outer border and padding plus a
// field's own border) takes frameChrome columns and fullHeight rows; below
// that the example drops the frames (see compact).
const (
	frameChrome = 6
	fullHeight  = 16
	fieldWidth  = 30 // widest a field gets
)

// compact reports whether the terminal is too short for the framed layout.
func (m model) compact() bool { return m.height > 0 && m.height < fullHeight }

// layout sizes the three widgets to the terminal width: fieldWidth, or what
// is left once the frames and the prompt are paid for.
func (m *model) layout() {
	chrome := frameChrome
	if m.compact() {
		chrome = 2 // just the focus marker
	}
	room := func(prompt string) int {
		if m.width <= 0 {
			return fieldWidth
		}
		return max(3, min(fieldWidth, m.width-chrome-ansi.Width(prompt)))
	}
	m.name.Width = room(m.name.Prompt)
	m.password.Width = room(m.password.Prompt)
	m.bio.Width = max(3, min(fieldWidth, m.width-chrome))
	if m.width <= 0 {
		m.bio.Width = fieldWidth
	}
}

// current is the field that has focus.
func (m model) current() field { return field(m.ring.Current()) }

func (m model) Init() tui.Cmd { return m.focusCmd }

// fields binds each widget, in ring order, to the focus package. The pointers
// point into this copy of the model, so build them in the method that goes on
// to return m.
func (m *model) fields() []focus.Field {
	return []focus.Field{
		focus.Bind(&m.name),
		focus.Bind(&m.password.Model), // passwordinput embeds a textinput.Model
		focus.Bind(&m.bio),
	}
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	if rs, ok := msg.(tui.ResizeMsg); ok {
		m.width, m.height = rs.Width, rs.Height
		m.layout()
		return m, nil
	}
	if key, ok := msg.(tui.Key); ok && (key.Type == tui.KeyCtrlC || key.Type == tui.KeyEsc) {
		return m, tui.Quit()
	}

	// Route does the whole dispatch: Tab and Shift+Tab (the input reader
	// decodes both ESC [ Z and the kitty form to a Tab key with the Shift
	// modifier) blur the old widget and focus the next one, wrapping in both
	// directions; every other message goes to the focused widget only.
	var cmd tui.Cmd
	m.ring, cmd = m.ring.Route(msg, m.fields()...)
	return m, cmd
}

var (
	title           = ansi.NewStyle().Bold().Underline()
	dimSty          = ansi.NewStyle().Faint()
	focusedBorder   = ansi.BrightCyan
	unfocusedBorder = ansi.White
)

// box wraps a widget's View in a bordered box, using a highlighted border
// color when it's the focused widget and a plain one otherwise — the
// visible half of the Focus/Blur coordination: not just internal state, but
// a rendered difference driven directly by m.focused.
func box(content string, focused bool) layout.Node {
	b := layout.NewBox().Border(layout.RoundedBorder()).PaddingAll(0)
	if focused {
		b = b.BorderColor(focusedBorder)
	} else {
		b = b.BorderColor(unfocusedBorder)
	}
	return layout.BoxNode(b, layout.Block(content))
}

// screen is the whole reference as a layout.Node: the title, the three boxed
// widgets, and the key hints, in a rounded padded box.
func (m model) screen() layout.Node {
	child := func(n layout.Node) layout.FlexChild { return layout.FlexChild{Node: n} }
	if m.compact() {
		// Too short for frames: one row per widget, the focused one marked.
		row := func(content string, f field) layout.FlexChild {
			mark := "  "
			if m.current() == f {
				mark = "> "
			}
			return child(layout.Block(mark + content))
		}
		return layout.Column(0,
			child(layout.Block(title.Render("Focus/Blur reference"))),
			row(m.name.View(), fieldName),
			row(m.password.View(), fieldPassword),
			row(m.bio.View(), fieldBio),
			child(layout.Block(dimSty.Render(m.hint()))),
		)
	}
	body := layout.Column(1,
		child(layout.Block(title.Render("Focus/Blur reference"))),
		child(layout.Column(0,
			child(box(m.name.View(), m.current() == fieldName)),
			child(box(m.password.View(), m.current() == fieldPassword)),
			child(box(m.bio.View(), m.current() == fieldBio)),
		)),
		child(layout.Block(dimSty.Render(m.hint()))),
	)
	return layout.BoxNode(layout.NewBox().Border(layout.RoundedBorder()).PaddingAll(1), body)
}

// hint is the key legend, shortened to fit a narrow terminal.
func (m model) hint() string {
	full := "tab next  shift+tab prev  esc/ctrl+c quit"
	if m.width > 0 && ansi.Width(full)+frameChrome > m.width {
		return ansi.Truncate("tab next  S-tab prev  esc quit", max(0, m.width-2))
	}
	return full
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
