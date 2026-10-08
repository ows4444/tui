// Command asyncload demonstrates the async-Cmd-resolves-into-a-Msg pattern:
// a tui.Cmd is just a func() tui.Msg, and Program.dispatch (see program.go)
// runs each Cmd on its own goroutine and feeds whatever Msg it returns back
// into the event loop. That means a Cmd that blocks — here, time.Sleep to
// simulate a slow API call — does not block Update/View: the spinner keeps
// animating (via its own self-rescheduling motion tick Cmd) while the slow
// Cmd is still in flight on its goroutine. When the slow Cmd finally
// returns, its "done" Msg arrives in Update like any other message, and
// that's the only hook the model needs to swap the loading indicator for
// the resolved content — there is no callback, promise, or shared state,
// just Cmd-in / Msg-out.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/spinner"
)

// loadDelay is how long the simulated slow operation takes.
const loadDelay = 1500 * time.Millisecond

// loadedMsg is the Msg the simulated slow operation resolves into. Any
// payload the "API call" would have returned rides along on the Msg —
// here just a canned string.
type loadedMsg struct {
	result string
}

// fetchCmd returns a tui.Cmd that blocks for loadDelay (standing in for a
// slow network call, disk read, etc.) and then resolves into loadedMsg.
// Because Program runs Cmds on their own goroutine, this blocking sleep
// never freezes the UI: Update and View keep running against the model as
// usual for every Msg that arrives while this Cmd is still sleeping.
func fetchCmd() tui.Cmd {
	return func() tui.Msg {
		time.Sleep(loadDelay)
		return loadedMsg{result: "42 widgets loaded from the warehouse"}
	}
}

type model struct {
	spinner spinner.Model
	loading bool
	result  string
	// start is the spinner's first tick. Start has to run on the model the
	// Program keeps, and Init has a value receiver and gets a copy, so it
	// runs where the model is built and Init only hands the Cmd over.
	start tui.Cmd
}

func initialModel() model {
	s := spinner.New()
	s.Label = "fetching data..."
	m := model{spinner: s, loading: true}
	m.start = m.spinner.Start()
	return m
}

// Init kicks off the simulated slow operation immediately on startup,
// alongside starting the spinner's own animation Cmd. tui.Batch runs both
// concurrently; only fetchCmd's result (loadedMsg) ever reaches the
// "loading -> resolved" transition below, while the spinner's tickMsg just
// keeps the animation going in the meantime.
func (m model) Init() tui.Cmd {
	return tui.Batch(m.start, fetchCmd())
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.Key:
		if msg.Type == tui.KeyCtrlC || msg.Type == tui.KeyEsc {
			return m, tui.Quit()
		}
		if m.result != "" && (msg.Type == tui.KeyEnter ||
			(msg.Type == tui.KeyRunes && msg.Text == "q")) {
			return m, tui.Quit()
		}
		return m, nil

	case loadedMsg:
		// The slow Cmd resolved: stop the spinner explicitly (rather than
		// just ignoring further tickMsgs) so its internal Update no longer
		// self-reschedules a tick underneath the resolved content, and
		// switch the View over to that content.
		m.spinner.Stop()
		m.loading = false
		m.result = msg.result
		return m, nil

	default:
		if m.loading {
			var cmd tui.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}
}

// screen is the panel as a layout.Node: while loading, a heading over the
// spinner; once loaded, the result and the quit hint.
func (m model) screen() layout.Node {
	child := func(n layout.Node) layout.FlexChild { return layout.FlexChild{Node: n} }
	var body layout.Node
	if m.loading {
		body = layout.Column(1, child(layout.Block("Loading...")), child(m.spinner.LayoutNode()))
	} else {
		body = layout.Column(1,
			child(layout.Block("Done!")),
			child(layout.Block(m.result)),
			child(layout.Block("enter/q to quit")),
		)
	}
	return layout.BoxNode(layout.NewBox().Border(layout.RoundedBorder()).PaddingAll(1), body)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
