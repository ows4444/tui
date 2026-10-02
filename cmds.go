package tui

import (
	"context"
	"time"
)

// Quit returns a Cmd that ends the program's event loop.
func Quit() Cmd {
	return func() Msg { return QuitMsg{} }
}

// batchMsg carries a set of commands to be dispatched concurrently; the
// event loop unpacks it instead of ever handing it to Update.
type batchMsg []Cmd

// Batch runs several commands concurrently. Their resulting messages are
// each delivered to Update as they complete, in no particular order.
func Batch(cmds ...Cmd) Cmd {
	var filtered []Cmd
	for _, c := range cmds {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func() Msg { return batchMsg(filtered) }
}

// printMsg carries text to be committed permanently to the terminal's real
// scrollback; the event loop intercepts it the same way it does batchMsg,
// writing it to output (or errOutput, for Eprintln) directly instead of
// ever handing it to Update.
type printMsg struct {
	text   string
	stderr bool
}

// Println returns a Cmd that commits text to the terminal's real,
// permanent scrollback — above the live region that the Model keeps
// redrawing — the same way Bubbletea's tea.Println or InkUI's Static
// works. text may contain embedded newlines; each resulting line is
// written terminated with a real "\r\n" so the terminal's own scrolling
// absorbs it into history, distinct from a live-region repaint.
func Println(text string) Cmd {
	return func() Msg { return printMsg{text: text} }
}

// Eprintln is Println for the error stream (os.Stderr by default, or
// WithErrOutput's override): the same permanent-scrollback commit, using
// the same cursor-up/erase-then-write sequence anchored to the live
// region's position, just written to a different destination. Terminal
// cursor position belongs to the terminal, not to whichever file
// descriptor a write goes through, so a raw, uncoordinated write to
// stderr while the live region is up can land mid-repaint the same way an
// uncoordinated stdout write would — Eprintln avoids that the same way
// Println does.
func Eprintln(text string) Cmd {
	return func() Msg { return printMsg{text: text, stderr: true} }
}

// suspendMsg carries the caller-provided function to run with the
// terminal released; the event loop intercepts it the same way it does
// printMsg, calling it directly instead of ever handing it to Update —
// its result reaches Update as a SuspendMsg once fn has actually run.
type suspendMsg struct{ fn func() error }

// Suspend returns a Cmd that temporarily hands the real terminal back —
// undoing Run's raw-mode/alt-screen/mouse/bracketed-paste/cursor setup,
// running fn with the terminal in its normal (non-raw) state, then
// restoring everything and forcing a fresh repaint — for handing off to
// an external interactive program (an editor, a shell prompt) that needs
// the terminal to itself, the way InkUI's suspendTerminal does. fn is
// responsible for the external program itself (spawning it with
// inherited stdio and waiting for it to exit); its returned error, if
// any, is delivered to Update via SuspendMsg once resumed — Suspend never
// quits the Program on fn's behalf.
func Suspend(fn func() error) Cmd {
	return func() Msg { return suspendMsg{fn: fn} }
}

// SuspendMsg is delivered to Update once a Suspend's fn has returned and
// the terminal has been restored, carrying fn's error, if any.
type SuspendMsg struct{ Err error }

// TickCtx is Tick as a CtxCmd, for composing inside a FromCtx Cmd of your own. The
// wait ends when the Program's context is cancelled (Run returned).
func TickCtx(d time.Duration, fn func(time.Time) Msg) CtxCmd {
	return func(ctx context.Context) Msg {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case t := <-timer.C:
			return fn(t)
		case <-ctx.Done():
			return nil
		}
	}
}

// Tick returns a Cmd that waits for d and then produces a Msg from fn,
// useful for animations, spinners, or polling. It is TickCtx wrapped with
// FromCtx: the Program runs it with its context, so Run returning ends the wait,
// fn is not called and no Msg is sent, and no goroutine outlives Run waiting on
// a long timer. The Cmd does nothing useful when called directly; the Program
// runs it.
func Tick(d time.Duration, fn func(time.Time) Msg) Cmd { return FromCtx(TickCtx(d, fn)) }
