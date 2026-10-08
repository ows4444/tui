// Command pickers fills in a calendar event with the three picker widgets: a
// datepicker for the day, a colorpicker for its colour and a filepicker for a
// file to attach. Tab and Shift+Tab move between the three, the arrow keys move
// inside the one that has focus, and Enter confirms its value and moves on.
// Each picker reports its choice as a SelectedMsg; the summary rows at the top
// are built from those messages alone.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/colorpicker"
	"github.com/ows4444/tui/datepicker"
	"github.com/ows4444/tui/filepicker"
	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/widgets"
)

// The sections, in Tab order.
const (
	sectionDate = iota
	sectionColour
	sectionFile
	sectionCount
)

var sectionNames = [sectionCount]string{"Date", "Colour", "File"}

// palette is the colours offered, with the name shown for each.
var palette = []struct {
	name  string
	color ansi.Color
}{
	{"red", ansi.Red}, {"yellow", ansi.Yellow}, {"green", ansi.Green},
	{"cyan", ansi.Cyan}, {"blue", ansi.Blue}, {"magenta", ansi.Magenta},
}

type model struct {
	date   datepicker.Model
	colour colorpicker.Model
	file   filepicker.Model

	focus  int
	chosen [sectionCount]string // what each section's SelectedMsg said; "" until confirmed

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

// newModel starts the date picker on start and the file picker in dir.
func newModel(start time.Time, dir string) model {
	colours := make([]ansi.Color, len(palette))
	for i, p := range palette {
		colours[i] = p.color
	}
	m := model{
		date:   datepicker.New(start),
		colour: colorpicker.New(colours...),
		file:   filepicker.New(dir),
	}
	// Tab moves between the sections here, so the colour picker's own switch
	// between its swatches and its hex field moves to "/". It cannot be "#":
	// a hex value is typed as "#RRGGBB".
	m.colour.KeyMap.SwitchPane = keymap.NewBinding("swatches or hex", "/")
	return m
}

func (m model) Init() tui.Cmd { return nil }

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.file.Height = m.fileRows()
		return m, nil
	case datepicker.SelectedMsg:
		m.chosen[sectionDate] = msg.Date.Format("Mon 2 Jan 2006")
		m.focus = sectionColour
		return m, nil
	case colorpicker.SelectedMsg:
		m.chosen[sectionColour] = colourName(msg.Color)
		m.focus = sectionFile
		return m, nil
	case filepicker.SelectedMsg:
		m.chosen[sectionFile] = filepath.Base(msg.Path)
		return m, nil
	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC, msg.Type == tui.KeyEsc:
			return m, tui.Quit()
		case msg.Type == tui.KeyTab && msg.Mod&input.ModShift != 0:
			m.focus = (m.focus + sectionCount - 1) % sectionCount
			return m, nil
		case msg.Type == tui.KeyTab:
			m.focus = (m.focus + 1) % sectionCount
			return m, nil
		}
	}
	// Everything else goes to the section that has focus.
	var cmd tui.Cmd
	switch m.focus {
	case sectionDate:
		m.date, cmd = m.date.Update(msg)
	case sectionColour:
		m.colour, cmd = m.colour.Update(msg)
	case sectionFile:
		m.file, cmd = m.file.Update(msg)
	}
	return m, cmd
}

// colourName is the palette's name for c, or its value written out when c
// came from the hex field.
func colourName(c ansi.Color) string {
	for _, p := range palette {
		if p.color == c {
			return p.name
		}
	}
	if rgb, ok := c.(ansi.RGB); ok {
		return fmt.Sprintf("#%02x%02x%02x", rgb.R, rgb.G, rgb.B)
	}
	return fmt.Sprint(c)
}

var (
	title  = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	active = ansi.NewStyle().Bold()
	muted  = ansi.NewStyle().Faint()
)

// contentWidth is the width every row is cut to: the terminal's, or 60 before
// the first ResizeMsg.
func (m model) contentWidth() int {
	if m.width > 0 {
		return m.width
	}
	return 60
}

// compact reports whether the terminal is too short to show the title, the
// rows of the sections that do not have focus, and blank rows between parts.
func (m model) compact() bool { return m.height > 0 && m.height < 16 }

// fileRows is how many entries the file picker shows: six, or fewer in a
// short terminal.
func (m model) fileRows() int {
	if m.compact() {
		return max(2, min(6, m.height-4))
	}
	return 6
}

// summary is one section's row: its name, and the value it confirmed.
func (m model) summary(i int) string {
	name := fmt.Sprintf("%-8s", sectionNames[i])
	value := m.chosen[i]
	if value == "" {
		value = muted.Render("not chosen")
	}
	if i == m.focus {
		return ansi.Truncate(active.Render("> "+name)+value, m.contentWidth())
	}
	return ansi.Truncate(muted.Render("  "+name)+value, m.contentWidth())
}

// body is the picker that has focus, with a line above it that says what the
// cursor is on, since a date grid or a row of swatches does not say it itself.
func (m model) body() string {
	w := m.contentWidth()
	switch m.focus {
	case sectionDate:
		return ansi.Truncate(m.date.Cursor().Format("January 2006"), w) + "\n" + m.date.View()
	case sectionColour:
		on := "hex value, typed as #RRGGBB"
		if !m.colour.HexFocused() {
			on = palette[m.colour.Cursor()].name
		}
		return ansi.Truncate("Cursor on: "+on, w) + "\n" + m.colour.View()
	}
	lines := strings.Split(m.file.View(), "\n")
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, w)
	}
	head := "In: " + filepath.Base(m.file.Dir)
	if err := m.file.Err(); err != nil {
		head = "Cannot read the directory: " + err.Error()
	}
	return ansi.Truncate(head, w) + "\n" + strings.Join(lines, "\n")
}

// hints is the key legend for the section that has focus: with words when
// they fit, else the keys alone.
func (m model) hints() string {
	hs := []widgets.Hint{{Key: "arrows", Action: "move"}, {Key: "enter", Action: "confirm"}}
	if m.focus == sectionColour {
		hs = append(hs, widgets.Hint{Key: "/", Action: "hex"})
	}
	hs = append(hs, widgets.Hint{Key: "tab", Action: "next"}, widgets.Hint{Key: "esc", Action: "quit"})
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
	if m.compact() {
		return strings.Join([]string{m.summary(m.focus), m.body(), m.hints()}, "\n")
	}
	rows := make([]string, sectionCount)
	for i := range rows {
		rows[i] = m.summary(i)
	}
	return strings.Join([]string{
		title.Render(ansi.Truncate("New event", m.contentWidth())),
		strings.Join(rows, "\n"),
		m.body(),
		m.hints(),
	}, "\n\n")
}

func main() {
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	if _, err := tui.NewProgram(newModel(time.Now(), dir), tui.WithAltScreen(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
