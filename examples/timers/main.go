// Command timers loads a report and shows the wait with the two time widgets:
// clockview in each of its three modes (the wall clock in the corner, a
// countdown to when the report arrives, and a stopwatch of how old the report
// is once it has) and a skeleton standing in for the report until it does.
// "r" loads it again and Space pauses the wait.
//
// Each widget schedules its own ticks. The model passes every message to all
// four, and each one acts only on the ticks that are its own.
package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/clockview"
	"github.com/ows4444/tui/skeleton"
	"github.com/ows4444/tui/widgets"
)

// loadTime is how long the report takes to arrive.
const loadTime = 3 * time.Second

var report = []string{
	"Deploys this week   14",
	"Failed               1",
	"Median build time    4m 12s",
}

type model struct {
	clock clockview.Model // the wall clock
	wait  clockview.Model // counts down to the report
	age   clockview.Model // counts up from when it arrived
	ghost skeleton.Model  // stands in for the report

	loading, paused bool
	width, height   int // terminal size from the last ResizeMsg; 0 until known

	// Init can only return a Cmd, so the widgets are started in newModel (on
	// the model that is kept) and their Cmds are held here.
	startCmd tui.Cmd
}

// newModel starts loading at once. now is the wall clock's source of time.
func newModel(now func() time.Time) model {
	m := model{clock: clockview.New(clockview.ModeClock), ghost: skeleton.New()}
	m.clock.Now = now
	m.ghost.Lines = len(report)
	m.ghost.Width = 30
	m.startCmd = tui.Batch(m.clock.Start(), m.load())
	return m
}

func (m model) Init() tui.Cmd { return m.startCmd }

// load starts a fresh countdown and the skeleton's shimmer, and puts the
// stopwatch back to zero for when the report arrives. The two clock widgets
// are new Models, so a tick still on its way from the ones they replace is
// not theirs and is ignored.
func (m *model) load() tui.Cmd {
	m.wait = clockview.New(clockview.ModeTimer)
	m.wait.Duration = loadTime
	m.age = clockview.New(clockview.ModeStopwatch)
	m.loading, m.paused = true, false
	return tui.Batch(m.wait.Start(), m.ghost.Start())
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.ghost.Width = max(8, min(30, msg.Width))
		return m, nil
	case tui.Key:
		switch {
		case msg.Type == tui.KeyCtrlC, msg.Type == tui.KeyRunes && msg.Text == "q":
			return m, tui.Quit()
		case msg.Type == tui.KeyRunes && msg.Text == "r":
			// The Cmd is taken first, on its own line: load changes m, and
			// "return m, m.load()" does not promise to read m afterwards.
			cmd := m.load()
			return m, cmd
		case msg.Type == tui.KeySpace && m.loading:
			cmd := m.togglePause()
			return m, cmd
		}
		return m, nil
	}
	// Anything else may be a tick. Each widget takes its own and ignores the
	// rest, so all four are offered every message.
	var a, b, c, d tui.Cmd
	m.clock, a = m.clock.Update(msg)
	m.wait, b = m.wait.Update(msg)
	m.age, c = m.age.Update(msg)
	m.ghost, d = m.ghost.Update(msg)
	if m.loading && !m.paused && !m.wait.Running() {
		// The countdown stops itself at zero: the report is here, and the
		// stopwatch starts counting its age.
		m.loading = false
		m.ghost.Stop()
		c = m.age.Start()
	}
	return m, tui.Batch(a, b, c, d)
}

// togglePause stops or restarts the countdown and the skeleton. The wall
// clock keeps going.
func (m *model) togglePause() tui.Cmd {
	m.paused = !m.paused
	if m.paused {
		m.wait.Stop()
		m.ghost.Stop()
		return nil
	}
	return tui.Batch(m.wait.Start(), m.ghost.Start())
}

var (
	title = ansi.NewStyle().Bold().Foreground(ansi.BrightCyan)
	muted = ansi.NewStyle().Faint()
	good  = ansi.NewStyle().Foreground(ansi.BrightGreen)
)

// contentWidth is the width every row is cut to: 50 columns, or the
// terminal's when that is narrower.
func (m model) contentWidth() int {
	if m.width > 0 {
		return min(50, m.width)
	}
	return 50
}

// header is the title with the wall clock at the right end of the row.
func (m model) header() string {
	w := m.contentWidth()
	name, now := "Weekly report", m.clock.View()
	pad := w - ansi.Width(name) - ansi.Width(now)
	if pad < 1 {
		return title.Render(ansi.Truncate(name, w))
	}
	return title.Render(name) + strings.Repeat(" ", pad) + muted.Render(now)
}

// status says where the load stands: from the countdown while it loads, and
// from the stopwatch once it has.
func (m model) status() string {
	var s string
	switch {
	case m.paused:
		s = "Paused with " + m.wait.View() + " left"
	case m.loading:
		s = "Loading, " + m.wait.View() + " left"
	default:
		s = good.Render("Loaded " + m.age.View() + " ago")
	}
	return ansi.Truncate(s, m.contentWidth())
}

// body is the report, or the skeleton while it loads.
func (m model) body() string {
	if m.loading {
		return m.ghost.View()
	}
	lines := make([]string, len(report))
	for i, l := range report {
		lines[i] = ansi.Truncate(l, m.contentWidth())
	}
	return strings.Join(lines, "\n")
}

// hints is the key legend: with words when they fit, else the keys alone.
func (m model) hints() string {
	hs := []widgets.Hint{{Key: "r", Action: "load again"}}
	if m.loading {
		action := "pause"
		if m.paused {
			action = "resume"
		}
		hs = append(hs, widgets.Hint{Key: "space", Action: action})
	}
	hs = append(hs, widgets.Hint{Key: "q", Action: "quit"})
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
	gap := "\n\n"
	if m.height > 0 && m.height < 12 {
		gap = "\n"
	}
	return strings.Join([]string{m.header(), m.status(), m.body(), m.hints()}, gap)
}

func main() {
	if _, err := tui.NewProgram(newModel(time.Now), tui.WithAltScreen(true)).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
