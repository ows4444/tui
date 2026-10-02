// Command procstream demonstrates streaming a child process's stdout into a
// scrolling widget (logview.Model) line by line, as it arrives, rather than
// buffering the whole thing and rendering it once the process exits.
//
// The command run here is:
//
//	sh -c 'for i in 1 2 3 4 5; do echo line $i; sleep 0.2; done'
//
// which is portable to any machine with /bin/sh (macOS, Linux, most CI
// images); it isn't portable to plain Windows cmd.exe, but that's a fine
// trade-off for an example whose point is the streaming pattern, not shell
// portability.
//
// # Why one tui.Cmd can't stream N messages
//
// A tui.Cmd has the signature `func() tui.Msg`: it is called once by the
// event loop's dispatch goroutine, and whatever single Msg it returns is
// the only thing that ever reaches Update from that call. There's no way
// for one Cmd invocation to push multiple Msgs over time — Cmd returns
// exactly once.
//
// So streaming N lines needs N (or N+1, counting completion) separate Cmd
// dispatches. The trick — the same one spinner.Model uses for its
// self-rescheduling animation tick, just reading from a channel instead of
// a timer — is: start the process and a goroutine that scans its stdout,
// pushing one streamEvent per line (and a final one on exit) into a
// channel. waitForEvent(ch) is a Cmd that blocks on a single receive from
// that channel and returns the event as a Msg. Update's job, every time it
// receives a streamEvent that isn't the final one, is to return
// waitForEvent(ch) again as its own Cmd — that reschedules "wait for the
// next line" exactly the way a tickMsg handler reschedules the next tick.
// Each dispatch only ever produces one Msg, but the chain of
// re-dispatches makes the whole channel's worth of output arrive at
// Update one message at a time, as it's produced.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/layout"
	"github.com/ows4444/tui/logview"
)

const (
	logWidth  = 60
	logHeight = 12
)

// streamEvent is one item read off the child process: either one line of
// stdout, or (once, as the last event before the channel is closed) the
// process's completion, carrying its error (nil on a clean exit).
type streamEvent struct {
	line string
	done bool
	err  error
}

// streamStartedMsg carries the channel the process's output is arriving
// on, once the process has been launched. Sent by startProcess's Cmd.
type streamStartedMsg struct {
	events <-chan streamEvent
	err    error // set if the process failed to start at all
}

// streamEventMsg wraps one streamEvent as it arrives from waitForEvent.
type streamEventMsg streamEvent

// startProcess returns a Cmd that launches the external command and wires
// up the channel it will stream events on. Starting the process
// (cmd.Start) does not wait for it to finish — it only forks/execs and
// returns — so even though this Cmd itself runs synchronously on its own
// dispatch goroutine (see program.go's dispatch), it returns almost
// immediately and never blocks the event loop's Update/View calls.
func startProcess() tui.Cmd {
	return func() tui.Msg {
		c := exec.Command("sh", "-c", "for i in 1 2 3 4 5; do echo line $i; sleep 0.2; done")
		stdout, err := c.StdoutPipe()
		if err != nil {
			return streamStartedMsg{err: err}
		}
		if err := c.Start(); err != nil {
			return streamStartedMsg{err: err}
		}

		events := make(chan streamEvent)
		go func() {
			scanner := bufio.NewScanner(stdout)
			for scanner.Scan() {
				events <- streamEvent{line: scanner.Text()}
			}
			waitErr := c.Wait()
			events <- streamEvent{done: true, err: waitErr}
			close(events)
		}()

		return streamStartedMsg{events: events}
	}
}

// waitForEvent returns a Cmd that receives exactly one streamEvent from
// ch. Update re-dispatches this Cmd after every non-final event so the
// channel keeps being drained, one Msg per line, until the process is
// done.
func waitForEvent(ch <-chan streamEvent) tui.Cmd {
	return func() tui.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return streamEventMsg(ev)
	}
}

type model struct {
	log      logview.Model
	events   <-chan streamEvent
	running  bool
	finished bool
	exitErr  error
	startErr error

	width, height int // terminal size from the last ResizeMsg; 0 until known
}

// gap is the blank row between the sections: one, or none on a terminal too
// short for the full-height log window (see resize).
func (m model) gap() int {
	if m.height > 0 && m.height < logHeight+5 {
		return 0
	}
	return 1
}

// resize fits the log window to the terminal: logWidth by logHeight, or the
// width of the window and what is left of its height after the title, the
// gaps and the two status rows.
func (m model) resize() model {
	if m.width > 0 {
		m.log.Viewport.Width = max(1, min(logWidth, m.width))
	}
	if m.height > 0 {
		m.log.Viewport.Height = max(1, min(logHeight, m.height-3-2*m.gap()))
	}
	return m
}

func initialModel() model {
	return model{log: logview.New(logWidth, logHeight)}
}

func (m model) Init() tui.Cmd {
	return startProcess()
}

func (m model) Update(msg tui.Msg) (tui.Model, tui.Cmd) {
	switch msg := msg.(type) {
	case tui.ResizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m.resize(), nil

	case tui.Key:
		switch msg.Type {
		case tui.KeyCtrlC, tui.KeyEsc:
			return m, tui.Quit()
		case tui.KeyRunes:
			if msg.Text == "q" {
				return m, tui.Quit()
			}
		}

	case streamStartedMsg:
		if msg.err != nil {
			m.startErr = msg.err
			m.finished = true
			return m, nil
		}
		m.events = msg.events
		m.running = true
		return m, waitForEvent(m.events)

	case streamEventMsg:
		if msg.done {
			m.running = false
			m.finished = true
			m.exitErr = msg.err
			return m, nil
		}
		m.log.Append(msg.line)
		return m, waitForEvent(m.events)
	}

	var cmd tui.Cmd
	m.log, cmd = m.log.Update(msg)
	return m, cmd
}

// screen is the whole view as a layout.Node: the title, the fixed-size log
// window, and the status and quit hint, one blank row apart.
func (m model) screen() layout.Node {
	status := "running..."
	switch {
	case m.startErr != nil:
		status = fmt.Sprintf("failed to start: %v", m.startErr)
	case m.finished && m.exitErr != nil:
		status = fmt.Sprintf("finished: %v", m.exitErr)
	case m.finished:
		status = "finished: exit status 0"
	}

	// The log scrolls, so it would measure to its whole content: give it a
	// fixed window, logWidth by logHeight unless the terminal is smaller.
	window := layout.Fixed(m.log.LayoutNode(), layout.Size{W: m.log.Viewport.Width, H: m.log.Viewport.Height})
	// Text rows are cut to the terminal width, status errors included.
	text := func(s string) layout.FlexChild {
		if m.width > 0 {
			lines := strings.Split(s, "\n")
			for i, l := range lines {
				lines[i] = ansi.Truncate(l, m.width)
			}
			s = strings.Join(lines, "\n")
		}
		return layout.FlexChild{Node: layout.Block(s)}
	}
	return layout.Column(m.gap(),
		text("procstream — streaming a child process's stdout"),
		layout.FlexChild{Node: window},
		text("status: "+status+"\n(q/esc to quit)"),
	)
}

func (m model) View() string { return layout.Draw(m.screen(), layout.Unconstrained()) }

func main() {
	if _, err := tui.NewProgram(initialModel()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
