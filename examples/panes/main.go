// Command panes is a log browser built from the five pane widgets. A
// splitpane holds a virtuallist of 10,000 entries on the left, with a
// scrollbar beside it that follows the list, and the selected entry's detail
// on the right. "[" and "]" (or Ctrl and an arrow) move the divider. Two
// overlays open over the whole screen: a popover beside the selected row
// ("i") and a drawer on the right edge with the key ("l").
//
// Only the rows on screen are ever rendered: the list asks for an entry by
// index, and an entry is computed from its index alone.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/drawer"
	"github.com/ows4444/tui/keymap"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/popover"
	"github.com/ows4444/tui/scrollbar"
	"github.com/ows4444/tui/splitpane"
	"github.com/ows4444/tui/virtuallist"
	"github.com/ows4444/tui/widgets"
)

// entryCount is how many log entries the list holds.
const entryCount = 10_000

// entry is one log line.
type entry struct {
	at      time.Time
	level   string
	source  string
	message string
}

var (
	sources  = []string{"api", "worker", "scheduler", "gateway"}
	messages = []string{
		"request served",
		"cache refreshed from the primary",
		"retrying after a timeout from the upstream service",
		"connection pool is at its limit, new requests are queued",
		"job finished",
		"configuration reloaded",
		"upstream returned an error and the request was not retried",
	}
	start = time.Date(2026, time.August, 14, 9, 0, 0, 0, time.UTC)
)

// entryAt computes entry i. Nothing is stored, so the list costs the same
// whether it holds ten entries or ten million.
func entryAt(i int) entry {
	level := "INFO"
	switch {
	case i%17 == 6:
		level = "ERROR"
	case i%5 == 3:
		level = "WARN"
	}
	msg := messages[i%len(messages)]
	switch level {
	case "ERROR":
		msg = messages[6]
	case "WARN":
		msg = messages[2+i%2]
	}
	return entry{at: start.Add(time.Duration(i) * 7 * time.Second), level: level, source: sources[i%len(sources)], message: msg}
}

// row is entry i as one line of the list.
func row(i int) string {
	e := entryAt(i)
	return fmt.Sprintf("%s %-5s %s", e.at.Format("15:04:05"), e.level, e.message)
}

type model struct {
	list  virtuallist.Model
	bar   scrollbar.Model
	split splitpane.Model
	info  popover.Model
	key   drawer.Model

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

const legend = "Levels\n\n" +
	"INFO   routine work\n" +
	"WARN   retried or queued\n" +
	"ERROR  gave up\n\n" +
	"enter or esc closes"

func initialModel() model {
	m := model{
		list:  virtuallist.New(entryCount, 10, row),
		bar:   scrollbar.New(entryCount, 10),
		split: splitpane.New(nil, nil),
		info:  popover.New("", 0, 0),
		key:   drawer.New(legend),
	}
	// Both overlays start closed; New returns them open.
	m.info.Hide()
	m.key.Hide()
	m.key.Width = 26
	m.split.Min1, m.split.Min2 = 16, 12
	// The defaults are Ctrl and an arrow, which some terminals keep for
	// themselves, so the brackets move the divider too.
	m.split.KeyMap.Shrink = keymap.NewBinding("narrow the list", "[", "ctrl+left")
	m.split.KeyMap.Grow = keymap.NewBinding("widen the list", "]", "ctrl+right")
	return m.resized(80, 24)
}

func (m model) Init() tui.Cmd { return nil }

// bodyRows is the height of the split: the terminal's, less the title row and
// the hints row.
func (m model) bodyRows() int { return max(1, m.height-2) }

// resized fits every widget to a w x h terminal.
func (m model) resized(w, h int) model {
	m.width, m.height = w, h
	m.list.Height = m.bodyRows()
	m.split.SetTotal(w)
	return m.synced()
}

// synced makes the scrollbar show where the list is. The bar owns no content:
// it draws the three numbers it is given.
func (m model) synced() model {
	m.bar.SetContent(entryCount, m.list.Height)
	m.bar.SetOffset(m.list.Offset())
	m.bar.Length = m.list.Height
	return m
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		return m.resized(msg.Width, msg.Height), nil
	case tui.Key:
		return m.onKey(msg)
	}
	return m, nil
}

// onKey sends a key to the open overlay if there is one, and otherwise to
// whichever widget it belongs to.
func (m model) onKey(k tui.Key) (tui.Model, tui.Cmd) {
	if k.Type == tui.KeyCtrlC {
		return m, tui.Quit()
	}
	var cmd tui.Cmd
	switch {
	case m.info.Open():
		m.info, cmd = m.info.Update(k)
	case m.key.Open():
		m.key, cmd = m.key.Update(k)
	case k.Type == tui.KeyRunes && k.Text == "q":
		return m, tui.Quit()
	case k.Type == tui.KeyRunes && k.Text == "l":
		m.key.Show()
	case k.Type == tui.KeyRunes && k.Text == "i", k.Type == tui.KeyEnter:
		e := entryAt(m.list.Cursor())
		m.info.Content = fmt.Sprintf("Entry %d\n%s from %s\n%s", m.list.Cursor()+1, e.level, e.source, e.at.Format("15:04:05 UTC"))
		// Beside the selected row: one row down for the title, then the
		// row's place in the window.
		m.info.AnchorX, m.info.AnchorY = 4, 1+m.list.Cursor()-m.list.Offset()
		m.info.Show()
	case keymap.Matches(k, m.split.KeyMap.Shrink), keymap.Matches(k, m.split.KeyMap.Grow), keymap.Matches(k, m.split.KeyMap.Reset):
		m.split, cmd = m.split.Update(k)
	default:
		m.list, cmd = m.list.Update(k)
	}
	return m.synced(), cmd
}

var title = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)

// detail is the right pane: the selected entry in full. Its message wraps to
// whatever width the divider leaves.
func (m model) detail() layout.Node {
	e := entryAt(m.list.Cursor())
	text := fmt.Sprintf("Entry %d of %d\n\nTime: %s\nLevel: %s\nSource: %s\n\n%s",
		m.list.Cursor()+1, entryCount, e.at.Format("15:04:05 UTC"), e.level, e.source, e.message)
	return layout.StyledText(text, ansi.Style{}, true)
}

// hints is the key legend: with words when they fit, else the keys alone.
func (m model) hints() string {
	hs := []widgets.Hint{
		{Key: "up/down", Action: "move"}, {Key: "brackets", Action: "divider"}, {Key: "i", Action: "info"},
		{Key: "l", Action: "levels"}, {Key: "q", Action: "quit"},
	}
	full := widgets.KeyHints("  ", hs...)
	if ansi.Width(full) <= m.width {
		return full
	}
	for i := range hs {
		hs[i].Action = ""
	}
	return ansi.Truncate(widgets.KeyHints(" ", hs...), m.width)
}

func (m model) View() string {
	// The panes are handed to the split each frame: it owns the divider's
	// position and nothing else.
	split := m.split
	// The list draws rows by calling a function, so marking the selected one
	// is a matter of handing it a function that knows which that is.
	list, cursor := m.list, m.list.Cursor()
	list.RenderItem = func(i int) string {
		if i == cursor {
			return "> " + row(i)
		}
		return "  " + row(i)
	}
	split.First = layout.Row(0, layout.Fill(list.LayoutNode()), layout.FlexChild{Node: m.bar.LayoutNode()})
	split.Second = m.detail()

	first, _ := m.split.Sizes()
	head := fmt.Sprintf("Log: %d entries, list pane %d wide", entryCount, first)
	base := strings.Join([]string{
		title.Render(ansi.Truncate(head, m.width)),
		layout.DrawTight(split.LayoutNode(), layout.Size{W: m.width, H: m.bodyRows()}),
		m.hints(),
	}, "\n")
	// Overlays go on last. A closed one returns the base unchanged.
	return m.key.Render(m.info.Render(base))
}

func main() {
	if _, err := tui.NewProgram(initialModel(), tui.WithAltScreen(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
