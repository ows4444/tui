package tui

import "github.com/ows4444/tui/input"

// WithSuspendOnCtrlZ makes Ctrl+Z suspend the process to the background, as
// a shell job would, instead of reaching Update. Off by default: without this
// option Ctrl+Z is an ordinary Key event (Type KeyCtrl, Code 'z').
//
// On Ctrl+Z (a press; a kitty release or repeat is not one) Run restores the
// terminal exactly as Suspend does (modes off, cooked mode back), stops the
// process, and blocks until the shell continues it (SIGCONT, `fg`). It then
// re-enters raw mode and every mode the options asked for (including
// WithKeyboard flags), restarts the input reader, re-reads the terminal
// size (Update receives a ResizeMsg if it changed) and repaints the whole
// frame. Update sees neither the Ctrl+Z nor a SuspendMsg. Messages sent with
// Send while stopped are handled after the resume.
//
// The process stops itself with SIGSTOP, which cannot be caught or ignored,
// so it works even in an orphaned process group where SIGTSTP is discarded;
// job control reports it as stopped either way. Only the process stops, not
// its whole process group.
//
// On Windows, which has no job control, and on other platforms without
// SIGSTOP, this option is a no-op and Ctrl+Z stays an ordinary Key event.
func WithSuspendOnCtrlZ(enabled bool) ProgramOption {
	return func(p *Program) { p.suspendOnCtrlZ = enabled }
}

// stopFn is what stops the process and returns once it was continued: the
// test hook if set, else the platform's (nil where unsupported).
func (p *Program) stopFn() func() error {
	if p.stopProcess != nil {
		return p.stopProcess
	}
	return stopSelf
}

// isSuspendKey reports whether msg is a Ctrl+Z press that WithSuspendOnCtrlZ
// turns into a suspend.
func (p *Program) isSuspendKey(msg Msg) bool {
	if !p.suspendOnCtrlZ || p.stopFn() == nil {
		return false
	}
	k, ok := msg.(Key)
	return ok && k.Type == input.KeyCtrl && k.Action == input.KeyPress &&
		k.Code == 'z'
}

// suspendToBackground runs on the loop goroutine. It reuses suspend (and so
// termMu, the reader stop/restart and enterModes/leaveModes) with the
// process stop as fn, then resyncs the size and repaints in full, since the
// screen is unknown after a stop. It reports the new size when it changed.
func (p *Program) suspendToBackground() (ResizeMsg, bool) {
	_ = p.suspend(p.stopFn())
	var rm ResizeMsg
	changed := false
	if w, h, ok := p.termSize(); ok && (w != p.width || h != p.height) {
		p.width, p.height = w, h
		rm, changed = p.resizeMsg(w, h), true
	}
	return rm, changed
}
