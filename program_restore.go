package tui

import (
	"io"
	"time"

	"github.com/ows4444/tui/ansi"
)

// restoreHandoffTimeout bounds how long a foreign goroutine (signal watcher,
// panicking Cmd) waits for the loop goroutine to restore the terminal before
// falling back to the fixed restore string.
const restoreHandoffTimeout = 500 * time.Millisecond

// restoreReqMsg asks the loop goroutine to restore the terminal; the loop
// closes done once it has.
type restoreReqMsg struct{ done chan struct{} }

// writeTo writes s to dest under the output mutex, so frames, prints and the
// restore sequence never interleave. Once the terminal has been restored
// (outClosed) the bytes are dropped: nothing may follow the restore sequence.
func (p *Program) writeTo(dest io.Writer, s string) {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	if p.outClosed {
		return
	}
	_, _ = io.WriteString(dest, s)
}

// writeBytes is write for a frame held in a buffer the caller reuses.
func (p *Program) writeBytes(b []byte) {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	if p.outClosed {
		return
	}
	_, _ = p.output.Write(b)
}

// closeOutput marks the output finished; see writeTo.
func (p *Program) closeOutput() {
	p.outMu.Lock()
	p.outClosed = true
	p.outMu.Unlock()
}

// restoreViaLoop restores the terminal from a goroutine other than the loop.
// It hands the work to the loop goroutine, which owns the cursor/frame state,
// and waits up to restoreHandoffTimeout; if the loop does not answer (stuck in
// Update, queue full, already gone) it writes a fixed restore string under the
// output mutex instead, touching no loop-owned state.
func (p *Program) restoreViaLoop() {
	req := restoreReqMsg{done: make(chan struct{})}
	timeout := time.NewTimer(restoreHandoffTimeout)
	defer timeout.Stop()
	select {
	case p.msgs <- req:
		select {
		case <-req.done:
			return
		case <-timeout.C:
		}
	case <-timeout.C:
	}
	p.restoreFixed()
}

// setMouseMode records mode as the active mouse mode and returns the previous
// one. Mode state is changed under outMu so the restore path, which may run on
// another goroutine (signal, panicking Cmd), reads a consistent snapshot.
func (p *Program) setMouseMode(mode MouseMode) (old MouseMode) {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	old, p.mouseMode = p.mouseMode, mode
	return old
}

// setAltScreen records whether the alternate screen is active and returns the
// previous value; see setMouseMode.
func (p *Program) setAltScreen(on bool) (old bool) {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	old, p.altScreen = p.altScreen, on
	return old
}

// restoreFixedString is the state-independent restore sequence: the mode
// state is snapshotted under outMu, the rest is immutable after NewProgram.
func (p *Program) restoreFixedString() string {
	p.outMu.Lock()
	defer p.outMu.Unlock()
	return p.restoreFixedStringLocked()
}

// restoreFixedStringLocked is restoreFixedString for a caller holding outMu.
func (p *Program) restoreFixedStringLocked() string {
	if p.accessible {
		return ""
	}
	s := ""
	if p.clusterMode {
		s += clusterModeDisable
	}
	if p.focusReporting {
		s += ansi.FocusReportingDisable
	}
	if p.kittyFlags() != 0 {
		s += ansi.KittyKeyboardDisable
	}
	if p.bracketedPaste {
		s += ansi.BracketedPasteDisable
	}
	if p.mouseMode != MouseOff {
		_, disable := mouseModeCodes(p.mouseMode)
		s += ansi.MouseSGRDisable + disable
	}
	s += ansi.CursorShow
	if p.altScreen {
		s += ansi.AltScreenDisable
	}
	return s + p.titleAndShapeRestoreLocked()
}

// restoreFixed is the timeout fallback of restoreViaLoop: it does what
// restoreTerminal does, but from any goroutine, with a fixed string written
// under the output mutex, after which further writes are dropped.
func (p *Program) restoreFixed() {
	p.restoreOnce.Do(func() {
		p.outMu.Lock()
		if !p.outClosed {
			_, _ = io.WriteString(p.output, p.restoreFixedStringLocked())
			p.outClosed = true
		}
		p.outMu.Unlock()
		if p.restoreOutputVT != nil {
			_ = p.restoreOutputVT()
		}
		p.termMu.Lock()
		p.termRestored = true
		if p.termRestore != nil {
			_ = p.termRestore()
		}
		p.termMu.Unlock()
	})
}
