// Command buildlog demonstrates tui.Println (see cmds.go): a Cmd that
// commits text permanently to the terminal's real scrollback, above the
// live-updating region, rather than being part of an ordinary View()
// repaint. This is the same idea as InkUI's "Static" component or
// Bubbletea's tea.Println — content that, once printed, is never redrawn
// or erased by later frames, even though the rest of the UI keeps
// animating beneath it.
//
// The scenario is a simulated build/install tool: a fixed list of steps
// ("Fetching dependencies", "Compiling", "Running tests", "Packaging")
// each complete on their own timer (motion.After). Every time a step
// completes, Update returns a tui.Println Cmd committing a permanent
// "✓ <step>" line to scrollback history — it happened, and stays visible
// even as the live region below it keeps repainting. That live region is
// an ordinary spinner.Model (see spinner/spinner.go for its
// self-rescheduling motion tick pattern) showing "Running: <next step>...",
// driven entirely by View()/Update() like any other widget, until every
// step has finished.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/motion"
	"github.com/ows4444/tui/spinner"
)

// steps is the fixed list of build steps this example simulates, in
// order.
var steps = []string{
	"Fetching dependencies",
	"Compiling",
	"Running tests",
	"Packaging",
}

// stepInterval is the spacing between step completions.
const stepInterval = 700 * time.Millisecond

// stepDoneMsg reports that steps[index] just finished.
type stepDoneMsg struct{ index int }

// stepDelayCmd returns a tui.Cmd that resolves into stepDoneMsg{index}
// after (index+1)*stepInterval. Scheduling every step's completion up
// front, at increasing delays, from Init (rather than chaining "when step
// N finishes, schedule step N+1") keeps each step-completion Update call
// self-contained: it only ever needs to return the tui.Println Cmd for
// the step that just finished, not also thread through the next tick —
// which is what keeps that return value directly testable (see
// main_test.go).
func stepDelayCmd(index int) tui.Cmd {
	delay := time.Duration(index+1) * stepInterval
	return tui.FromCtx(motion.After(delay, func(time.Time) tui.Msg { return stepDoneMsg{index: index} }))
}

type model struct {
	spinner   spinner.Model
	completed int // number of steps finished so far
	finished  bool
	// start is the spinner's first tick. Start has to run on the model the
	// Program keeps, and Init has a value receiver and gets a copy, so it
	// runs where the model is built and Init only hands the Cmd over.
	start tui.Cmd
}

func initialModel() model {
	s := spinner.New()
	s.Label = "Running: " + steps[0] + "..."
	m := model{spinner: s}
	m.start = m.spinner.Start()
	return m
}

// Init starts the spinner's own animation alongside every step's
// completion timer, all running concurrently via tui.Batch.
func (m model) Init() tui.Cmd {
	cmds := make([]tui.Cmd, 0, len(steps)+1)
	cmds = append(cmds, m.start)
	for i := range steps {
		cmds = append(cmds, stepDelayCmd(i))
	}
	return tui.Batch(cmds...)
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc {
			return m, tui.Quit()
		}
		if m.finished && (msg.Type == tui.KeyEnter ||
			(msg.Type == tui.KeyRunes && msg.Text == "q")) {
			return m, tui.Quit()
		}
		return m, nil

	case stepDoneMsg:
		// The step-completion event: commit a permanent scrollback line
		// for the step that just finished via tui.Println, while the
		// live region below (the spinner, updated separately by its own
		// tickMsg in the default case) keeps being repainted by the
		// ordinary View() below. This is the one Cmd returned here — no
		// batching with anything else — so each step-completion point
		// maps directly to one tui.Println Cmd.
		if msg.index+1 > m.completed {
			m.completed = msg.index + 1
		}
		if m.completed >= len(steps) {
			m.finished = true
			m.spinner.Stop()
		} else {
			m.spinner.Label = "Running: " + steps[m.completed] + "..."
		}
		return m, tui.Println("✓ " + steps[msg.index])

	default:
		if !m.finished {
			var cmd tui.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

// screen is the live region as a layout.Node: a status line, a blank row, and
// either the spinner or the quit hint.
func (m model) screen() layout.Node {
	if m.finished {
		return layout.Column(1,
			layout.FlexChild{Node: layout.Block(fmt.Sprintf("All %d steps complete.", len(steps)))},
			layout.FlexChild{Node: layout.Block("enter/q to quit")},
		)
	}
	return layout.Column(1,
		layout.FlexChild{Node: layout.Block(fmt.Sprintf("%d/%d steps complete", m.completed, len(steps)))},
		layout.FlexChild{Node: m.spinner.LayoutNode()},
	)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
