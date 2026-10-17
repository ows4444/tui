package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/ows4444/tui"
	"github.com/ows4444/tui/ansi"
)

// A run streams its output through a channel: the goroutine doing the work
// sends events, and a Cmd turns one event at a time into a Msg.
type event struct {
	line string
	done bool
	code int
	err  error
}

// outMsg carries a line and what is needed to ask for the next one.
type outMsg struct {
	seq   int
	line  string
	ch    <-chan event
	began time.Time
}

type doneMsg struct {
	seq     int
	code    int
	err     error
	elapsed time.Duration
}

// runner does a command's work, sending each output line to emit, and returns
// the exit code. A returned error means it could not run at all.
type runner func(ctx context.Context, c command, emit func(string)) (int, error)

const (
	maxLineBytes = 1 << 20
	maxLines     = 5000
)

// shellRunner runs c.shell with sh -c, standard output and error together.
func shellRunner(ctx context.Context, c command, emit func(string)) (int, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", c.shell) // #nosec G204 -- the user's own command file names what to run
	ownGroup(cmd)
	cmd.WaitDelay = 2 * time.Second // do not wait forever on a child that keeps the pipe open
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	sc := bufio.NewScanner(pipe)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	for sc.Scan() {
		emit(sc.Text())
	}
	scanErr := sc.Err()
	if scanErr != nil {
		cmd.Process.Kill() // #nosec G104 -- already failing; Wait reports the outcome
	}
	waitErr := cmd.Wait()
	var exit *exec.ExitError
	switch {
	case ctx.Err() != nil:
		return 0, ctx.Err() // cancelled: the exit status of the killed process is not news
	case errors.As(waitErr, &exit):
		return exit.ExitCode(), scanErr
	case waitErr != nil:
		return 0, waitErr
	}
	return 0, scanErr
}

// scriptRunner plays c.script, one line every gap.
func scriptRunner(gap time.Duration) runner {
	return func(ctx context.Context, c command, emit func(string)) (int, error) {
		for _, l := range c.script {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(gap):
			}
			emit(l)
		}
		return c.exit, nil
	}
}

// start launches c and returns the channel its events arrive on. cancel stops
// it; the channel is closed after the done event.
func start(parent context.Context, r runner, c command) (<-chan event, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	ch := make(chan event, 64)
	go func() {
		defer close(ch)
		code, err := r(ctx, c, func(l string) {
			select {
			case ch <- event{line: ansi.Sanitize(l)}:
			case <-ctx.Done():
			}
		})
		ch <- event{done: true, code: code, err: err}
	}()
	return ch, cancel
}

// next returns a Cmd that waits for the channel's next event.
func next(seq int, ch <-chan event, began time.Time) tui.Cmd {
	return func() tui.Msg {
		ev, ok := <-ch
		if !ok {
			return doneMsg{seq: seq, err: errors.New("the run ended without a result"), elapsed: time.Since(began)}
		}
		if ev.done {
			return doneMsg{seq: seq, code: ev.code, err: ev.err, elapsed: time.Since(began)}
		}
		return outMsg{seq: seq, line: ev.line, ch: ch, began: began}
	}
}

func describe(code int, err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "cancelled"
	case err != nil:
		return fmt.Sprintf("could not run: %v", err)
	case code != 0:
		return fmt.Sprintf("exit %d", code)
	}
	return "exit 0"
}
