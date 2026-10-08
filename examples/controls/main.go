// Command controls is an export panel built from the pressable and
// adjustable controls: radiogroup (one format of three), buttongroup (any of
// three options, and a row of actions), slider (quality), rating, checkbox
// and toggle (a switch). Tab and Shift+Tab move between the rows, the arrow keys work
// inside one, and the mouse clicks and drags them. A status line reads every
// control back, so what a key or a click changed shows at once. Esc quits.
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/button"
	"github.com/ows4444/tui/buttongroup"
	"github.com/ows4444/tui/checkbox"
	"github.com/ows4444/tui/hittest"
	"github.com/ows4444/tui/radiogroup"
	"github.com/ows4444/tui/rating"
	"github.com/ows4444/tui/slider"
	"github.com/ows4444/tui/toggle"
	"github.com/ows4444/tui/widgets"
)

// The rows, in Tab order.
const (
	rowFormat = iota
	rowOptions
	rowQuality
	rowRating
	rowMetadata
	rowExisting
	rowActions
	rowCount
)

var labels = [rowCount]string{"Format", "Options", "Quality", "Rating", "Metadata", "Existing", ""}

// labelWidth is the width of the label column: the longest label and a gap.
const labelWidth = 9

type model struct {
	format  radiogroup.Model
	options buttongroup.Model
	quality slider.Model
	stars   rating.Model
	keep    checkbox.Model
	replace toggle.Model
	actions buttongroup.Model

	focus         int
	exported      string // what the last Export wrote, for the status line
	width, height int    // terminal size from the last ResizeMsg; 0 until known
}

func initialModel() model {
	m := model{
		format:  radiogroup.New("PNG", "JPEG", "WebP"),
		options: buttongroup.New(buttongroup.ModeMultiple, "Strip", "Resize", "Dither"),
		quality: slider.New(0, 100),
		stars:   rating.New(5),
		keep:    checkbox.New("Keep"),
		replace: toggle.New("Overwrite"),
		actions: buttongroup.New(buttongroup.ModeActions, "Export", "Reset"),
	}
	m.format.Horizontal = true
	m.quality.Step, m.quality.ShowValue = 5, true
	m.stars.ShowValue = true
	m.actions.Buttons[1].Variant = button.VariantSecondary
	for _, c := range []*bool{&m.format.Mouse, &m.options.Mouse, &m.quality.Mouse, &m.stars.Mouse, &m.keep.Mouse, &m.replace.Mouse, &m.actions.Mouse} {
		*c = true
	}
	m.reset()
	m.setFocus(rowFormat)
	return m
}

// reset puts every control back to its starting value.
func (m *model) reset() {
	m.format.Select(0)
	m.options.SetOn("Strip")
	m.quality.SetValue(80)
	m.stars.SetValue(3)
	m.keep.SetChecked(true)
	m.replace.SetOn(false)
	m.exported = ""
}

func (m model) Init() tui.Cmd { return nil }

// setFocus blurs every control and focuses row i. None of these controls
// returns a Cmd from Focus, so there is nothing to hand to Init.
func (m *model) setFocus(i int) {
	m.format.Blur()
	m.options.Blur()
	m.quality.Blur()
	m.stars.Blur()
	m.keep.Blur()
	m.replace.Blur()
	m.actions.Blur()
	m.focus = (i + rowCount) % rowCount
	switch m.focus {
	case rowFormat:
		m.format.Focus()
	case rowOptions:
		m.options.Focus()
	case rowQuality:
		m.quality.Focus()
	case rowRating:
		m.stars.Focus()
	case rowMetadata:
		m.keep.Focus()
	case rowExisting:
		m.replace.Focus()
	case rowActions:
		m.actions.Focus()
	}
}

// contentWidth is the width the panel draws in: 60 columns, or the
// terminal's when that is narrower.
func (m model) contentWidth() int {
	if m.width > 0 {
		return max(30, min(60, m.width))
	}
	return 60
}

// compact reports whether the terminal is too short for blank rows between
// the sections.
func (m model) compact() bool { return m.height > 0 && m.height < 15 }

// rowY is the screen row control i is drawn on: under the title, and under
// the blank row after it when there is room for one.
func (m model) rowY(i int) int {
	if m.compact() {
		return 1 + i
	}
	return 2 + i
}

// place tells every control where it is drawn, so each can read the pointer.
// It runs before a mouse event is handed on, since the layout follows the
// terminal's size.
func (m *model) place() {
	at := func(i, w int) hittest.Rect { return hittest.Rect{X: labelWidth, Y: m.rowY(i), W: w, H: 1} }
	m.quality.Width = max(5, min(30, m.contentWidth()-labelWidth-6))
	m.format.Bounds = at(rowFormat, m.contentWidth())
	m.options.Bounds = at(rowOptions, m.options.Width())
	m.quality.Bounds = at(rowQuality, m.quality.Width+2)
	m.stars.Bounds = at(rowRating, m.stars.Max+2)
	m.keep.Bounds = at(rowMetadata, m.keep.Width())
	m.replace.Bounds = at(rowExisting, m.replace.Width())
	m.actions.Bounds = at(rowActions, m.actions.Width())
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.place()
		return m, nil
	case tui.Key:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tui.Quit()
		case "tab":
			m.setFocus(m.focus + 1)
			return m, nil
		case "shift+tab":
			m.setFocus(m.focus - 1)
			return m, nil
		}
	case tui.MouseEvent:
		return m.updateMouse(msg)
	case button.PressedMsg:
		switch msg.ID {
		case "Export":
			m.exported = m.summary()
		case "Reset":
			m.reset()
		}
		return m, nil
	}
	// A key goes to the focused row only.
	var cmd tui.Cmd
	switch m.focus {
	case rowFormat:
		m.format, cmd = m.format.Update(msg)
	case rowOptions:
		m.options, cmd = m.options.Update(msg)
	case rowQuality:
		m.quality, cmd = m.quality.Update(msg)
	case rowRating:
		m.stars, cmd = m.stars.Update(msg)
	case rowMetadata:
		m.keep, cmd = m.keep.Update(msg)
	case rowExisting:
		m.replace, cmd = m.replace.Update(msg)
	case rowActions:
		m.actions, cmd = m.actions.Update(msg)
	}
	return m, cmd
}

// updateMouse moves focus to the row a press lands on and then gives the
// event to every control: each ignores what is outside its own rectangle,
// and a drag or a held-down button must see the release wherever it happens.
func (m model) updateMouse(ev tui.MouseEvent) (tui.Model, tui.Cmd) {
	m.place()
	if ev.Action == tui.MouseActionPress && ev.Button == tui.MouseButtonLeft {
		for i := 0; i < rowCount; i++ {
			if ev.Y == m.rowY(i) && ev.X >= labelWidth {
				m.setFocus(i)
			}
		}
	}
	var cmds [rowCount]tui.Cmd
	m.format, cmds[rowFormat] = m.format.Update(ev)
	m.options, cmds[rowOptions] = m.options.Update(ev)
	m.quality, cmds[rowQuality] = m.quality.Update(ev)
	m.stars, cmds[rowRating] = m.stars.Update(ev)
	m.keep, cmds[rowMetadata] = m.keep.Update(ev)
	m.replace, cmds[rowExisting] = m.replace.Update(ev)
	m.actions, cmds[rowActions] = m.actions.Update(ev)
	return m, tui.Batch(cmds[:]...)
}

var (
	title  = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	active = ansi.NewStyle().Bold()
	muted  = ansi.NewStyle().Faint()
	good   = ansi.NewStyle().Foreground(ansi.BrightGreen)
)

// row draws one labelled control, cut to the content width.
func (m model) row(i int, control string) string {
	label := labels[i]
	pad := strings.Repeat(" ", labelWidth-ansi.Width(label))
	if i == m.focus {
		label = active.Render(label)
	} else {
		label = muted.Render(label)
	}
	return ansi.Truncate(label+pad+control, m.contentWidth())
}

// summary reads every control back in words.
func (m model) summary() string {
	opts := "no options"
	if on := m.options.On(); len(on) > 0 {
		opts = strings.ToLower(strings.Join(on, ", "))
	}
	s := fmt.Sprintf("%s · %s · quality %s · %d/%d",
		m.format.Value(), opts, strconv.FormatFloat(m.quality.Value(), 'f', -1, 64), m.stars.Value(), m.stars.Max)
	if m.keep.Checked() {
		s += " · metadata"
	}
	if m.replace.On() {
		s += " · overwrite"
	}
	return s
}

// status is the summary, or what the last Export wrote. It wraps when there
// is room for a second line and is cut when there is not.
func (m model) status() string {
	line := muted.Render(m.summary())
	if m.exported != "" {
		line = good.Render("Exported: " + m.exported)
	}
	if m.compact() {
		return ansi.Truncate(line, m.contentWidth())
	}
	return ansi.WrapStyled(line, m.contentWidth())
}

// hints is the key legend: with words when they fit, else the keys alone.
func (m model) hints() string {
	hs := []widgets.Hint{{Key: "tab", Action: "next row"}, {Key: "arrows", Action: "change"}, {Key: "space", Action: "press"}, {Key: "esc", Action: "quit"}}
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
	m.place() // on the copy View has: the slider's track follows the width
	gap := "\n\n"
	if m.compact() {
		gap = "\n"
	}
	rows := []string{
		m.row(rowFormat, m.format.View()),
		m.row(rowOptions, m.options.View()),
		m.row(rowQuality, m.quality.View()),
		m.row(rowRating, m.stars.View()),
		m.row(rowMetadata, m.keep.View()),
		m.row(rowExisting, m.replace.View()),
		m.row(rowActions, m.actions.View()),
	}
	return strings.Join([]string{
		title.Render(ansi.Truncate("Export image", m.contentWidth())),
		strings.Join(rows, "\n"),
		m.status(),
		m.hints(),
	}, gap)
}

func main() {
	p := tui.NewProgram(initialModel(), tui.WithAltScreen(true), tui.WithMouse(tui.MouseCellMotion))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
