package tui

import (
	"io"
	"os"
	"runtime/debug"
	"time"

	"github.com/ows4444/tui/ansi"
	"github.com/ows4444/tui/internal/cancelreader"
)

// runLoop drives the event loop itself: spawning the input/resize watcher
// goroutines, seeding the model with its starting size, dispatching Cmds,
// and calling Update until QuitMsg. It only needs p.input/p.output to
// behave like a Reader/Writer, not an actual terminal, so it's split out
// from Run to be testable against an os.Pipe() in place of a tty.
func (p *Program) runLoop() (_ Model, err error) {
	if p.recoverPanics {
		// Init, Update and View run on this goroutine; Run's deferred
		// restoreTerminal still fires after this returns.
		defer func() {
			if r := recover(); r != nil {
				err = &PanicError{Value: r, Stack: debug.Stack()}
			}
		}()
	}
	p.loopStart = time.Now()
	// done coordinates shutdown of the background goroutines below: closing
	// it unblocks anything selecting on it, and cancelling the input source
	// unblocks the goroutine parked waiting for input on p.input. Without
	// this, those goroutines would leak past runLoop returning, and could
	// hang forever trying to send into msgs once nothing drains it.
	//
	// The input reader is waited for, briefly, before returning: on a real
	// terminal a reader left behind would compete with the next Run (or any
	// other reader of the terminal) for keystrokes.
	done := make(chan struct{})
	p.setLoopDone(done)
	rs := p.newReaderSet(done)
	rs.start()
	p.stopInput, p.startInput = rs.stop, rs.start
	defer func() {
		p.stopInput, p.startInput = nil, nil
		p.setLoopDone(nil)
		close(done)
		if p.chordTimer != nil {
			p.chordTimer.Stop()
		}
		rs.cancel()
		if p.inCloser != nil {
			p.inCloseOnce.Do(func() { _ = p.inCloser.Close() })
		}
		select {
		case <-p.rdDone:
		case <-time.After(readerStopTimeout):
		}
	}()

	// watchResize is implemented per-OS (resize_unix.go / resize_windows.go)
	// since there's no portable way to detect a terminal resize: unix has
	// SIGWINCH; windows reads window-buffer-size events from the console
	// input queue (see resizeKick), with a slow poll as a fallback.
	//
	// A Terminal that implements ResizeNotifier is the source instead, so a
	// remote session needs no signal to deliver a resize.
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		if rn, ok := p.terminal().(ResizeNotifier); ok {
			forwardResizeSignals(p, rn.Resizes(), p.termSize, done)
			return
		}
		watchResize(p, done)
	}()

	if p.themeSet && p.bgTimeout <= 0 {
		p.bgTimeout = defaultThemeDetectTimeout
	}
	p.startCapabilityProbe(done)
	if p.bgTimeout > 0 && !p.accessible {
		p.write(ansi.QueryBackgroundColor)
		p.write(ansi.QueryPalette) // OSC 4; replies reach Update as PaletteColorEvent
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			timer := time.NewTimer(p.bgTimeout)
			defer timer.Stop()
			select {
			case <-timer.C:
				if !p.bgSeen.Load() {
					select {
					case p.msgs <- BackgroundUnknownMsg{}:
					case <-done:
					}
				}
			case <-done:
			}
		}()
	}

	// The theme goes in before Init, so a model that builds state or Cmds
	// from its theme there sees the Program's theme, not its own default.
	if p.themeSet {
		p.applyTheme(p.themeAuto.Dark)
	}
	if cmd := p.model.Init(); cmd != nil {
		p.spawn(cmd, done)
	}

	// Seed the model with its starting size before the first paint.
	var cmd Cmd
	p.model, cmd = p.model.Update(p.resizeMsg(p.width, p.height))
	if cmd != nil {
		p.spawn(cmd, done)
	}
	p.render()
	p.announce.Done = done

	for msg := range p.msgs {
		if p.handleAnnounce(msg) {
			continue
		}
		if pm, ok := msg.(printMsg); ok {
			// Never forwarded to Update: printMsg is a renderer
			// instruction (like batchMsg/ResizeMsg/QuitMsg above), not
			// application data.
			if p.altScreen && !p.accessible {
				// The alternate screen has no scrollback and the next frame
				// repaints all of it, so a print made here would be erased.
				// Hold it and write it to the main screen once that is back.
				p.holdAltPrint(pm)
				continue
			}
			dest := p.output
			if pm.stderr {
				dest = p.errOutput
			}
			p.printLines(pm.text, dest)
			p.render()
			continue
		}
		var handled bool
		if msg, handled = p.interceptCmdMsg(msg, done); handled {
			continue
		}
		if bm, ok := msg.(batchMsg); ok {
			for _, c := range bm {
				p.spawn(c, done)
			}
			continue
		}
		if sm, ok := msg.(suspendMsg); ok {
			// Never forwarded to Update directly, the same as printMsg:
			// suspend's result is delivered as a SuspendMsg below instead,
			// once fn has actually run.
			err := p.suspend(sm.fn)
			var cmd Cmd
			p.model, cmd = p.model.Update(SuspendMsg{Err: err})
			if cmd != nil {
				p.spawn(cmd, done)
			}
			// Bypass the cap: the frame right after a suspend must be a
			// full repaint against the just-reset lastFrame/liveLines,
			// not skipped and left showing whatever the resumed terminal
			// happens to still have on screen from the suspended fn.
			p.render()
			continue
		}
		if p.isSuspendKey(msg) {
			// Never forwarded to Update; see WithSuspendOnCtrlZ.
			rm, resized := p.suspendToBackground()
			if resized {
				var cmd Cmd
				p.model, cmd = p.model.Update(rm)
				if cmd != nil {
					p.spawn(cmd, done)
				}
			}
			// Bypass the cap: the screen is unknown after a stop.
			if p.accessible {
				p.render()
			} else {
				p.repaintFresh()
			}
			continue
		}
		msg = splitKeyAction(msg)
		if p.inspectorKeyPressed(msg) {
			p.render() // show or hide the overlay
			continue
		}
		if p.chords != nil {
			if handled := p.handleChordMsg(msg, done); handled {
				continue
			}
		}
		if ev, ok := msg.(BackgroundColorEvent); ok && p.themeSet {
			p.applyTheme(p.themeAuto.For(ansi.RGB{R: ev.R, G: ev.G, B: ev.B}))
		}
		rm, isResize := msg.(ResizeMsg)
		if isResize {
			// The size that triggered this message may already be stale
			// (a drag sends many SIGWINCHes, and watchResize coalesces them
			// while the queue is full), so trust the terminal's current size.
			if w, h, ok := p.termSize(); ok {
				rm = ResizeMsg{Width: w, Height: h}
			}
			rm.Measurer = p.Measurer()
			p.sentMeasurer = rm.Measurer
			msg = rm
			p.width, p.height = rm.Width, rm.Height
			p.rec.size(rm.Width, rm.Height)
		}
		if _, ok := msg.(flushRenderMsg); ok {
			// Never forwarded to Update: the timer armed by allowRender
			// asks for the frame it throttled away. A render that
			// happened since already cleared the flag.
			if p.renderPending {
				p.lastRenderAt = time.Now()
				p.renderIfChanged()
			}
			continue
		}
		if pm, ok := msg.(panicMsg); ok {
			p.stopFlushTimer()
			return p.model, pm.err
		}
		if _, ok := msg.(interruptMsg); ok {
			p.stopFlushTimer()
			return p.model, ErrInterrupted
		}
		if _, ok := msg.(QuitMsg); ok {
			p.stopFlushTimer()
			if p.maxFPS > 0 || p.renderPending {
				// Bypass the cap: the final frame at quit must reflect
				// the latest model state, never a stale one left behind
				// by a throttled-away render.
				p.render()
			}
			return p.model, nil
		}

		var cmd Cmd
		if p.msgKey != "" {
			began := time.Now()
			p.model, cmd = p.model.Update(msg)
			p.msgLog.add(msg, time.Since(began))
		} else {
			p.model, cmd = p.model.Update(msg)
		}
		p.rec.upd()
		if cmd != nil {
			p.spawn(cmd, done)
		}
		if isResize {
			// Never throttled, and never a diff: see repaintFresh.
			p.repaintFresh()
		} else if _, ok := msg.(CapabilitiesMsg); ok && p.Measurer() != p.sentMeasurer {
			// The probe changed how this terminal measures text: tell the
			// model with the size it already has, and redraw from scratch
			// since every cell width may differ.
			p.model, cmd = p.model.Update(p.resizeMsg(p.width, p.height))
			if cmd != nil {
				p.spawn(cmd, done)
			}
			p.repaintFresh()
		} else if p.allowRenderFor(done, isInputMsg(msg)) {
			p.renderIfChanged()
		}
	}

	p.stopFlushTimer()
	return p.model, nil
}

// focusDedupe remembers the last FocusEvent state handed to Update. Its
// zero value is "unknown", so the first event is always delivered; a later
// event with the same state (some terminals repeat one) is dropped.
type focusDedupe struct {
	known, focused bool
}

// deliver reports whether ev should reach Update, and records its state.
func (d *focusDedupe) deliver(ev FocusEvent) bool {
	if d.known && d.focused == ev.Focused {
		return false
	}
	d.known, d.focused = true, ev.Focused
	return true
}

// readerStopTimeout bounds how long runLoop waits for the input reader to
// notice it was cancelled. It normally stops in microseconds; the bound only
// keeps a misbehaving input from hanging Run.
const readerStopTimeout = 250 * time.Millisecond

// inputSource returns what the input reader should read from, and how to
// cancel and release it. On darwin, linux and windows that is a cancellable
// reader that works on terminals and pipes; if one can't be made (or
// elsewhere) it falls back to the file itself and a read deadline, the
// original behaviour.
func inputSource(f *os.File, onResize func()) (src io.Reader, cancel, closeFn func()) {
	if cr, err := cancelreader.New(f); err == nil {
		// Only the Windows reader sees console resize events.
		if n, ok := any(cr).(interface{ SetResizeNotify(func()) }); ok {
			n.SetResizeNotify(onResize)
		}
		return cr, cr.Cancel, cr.Close
	}
	return f, func() { _ = f.SetReadDeadline(time.Now()) }, func() {}
}

// kickResize tells watchResize the console reported a resize. It never
// blocks: one pending kick already means "re-read the size".
func (p *Program) kickResize() {
	select {
	case p.resizeKick <- struct{}{}:
	default:
	}
}

// dispatch runs cmd and feeds its result back into the event loop. It's
// launched on its own goroutine, so a long-running Cmd (a slow request, a
// Tick) that finishes after Run has already returned must not block forever
// trying to send into a channel nobody drains anymore — done, closed when
// Run returns, guards against exactly that.
func (p *Program) dispatch(cmd Cmd, done <-chan struct{}) {
	// A panic on a Cmd goroutine would end the process without Run's
	// deferred restoreTerminal; restore the terminal, then re-panic with the
	// same value so the trace is still reported.
	defer func() {
		if r := recover(); r != nil {
			if p.recoverPanics {
				select {
				case p.msgs <- panicMsg{&PanicError{Value: r, Stack: debug.Stack()}}:
				case <-done:
				}
				return
			}
			p.restoreViaLoop()
			panic(r)
		}
	}()
	msg := cmd()
	if msg == nil {
		return
	}
	select {
	case p.msgs <- msg:
	case <-done:
	}
}

// allowRender reports whether render() should run now for an ordinary
// Msg, given WithMaxFPS's cap, and records the time as the latest render
// if so. maxFPS <= 0 always allows, matching the behavior from before
// WithMaxFPS existed exactly. printMsg/QuitMsg bypass this entirely (see
// runLoop) — it only gates the per-Msg repaint after Update.
//
// A denied render is remembered: one timer is armed for the rest of the
// interval and sends a flushRenderMsg, so the latest state is drawn even if
// no other Msg follows. Nothing is armed while nothing is pending.
func (p *Program) allowRender(done <-chan struct{}) bool { return p.allowRenderFor(done, false) }

// allowRenderFor is allowRender told whether the Msg is input-driven, which
// the default cap (WithMaxFPS never called) lets through immediately.
func (p *Program) allowRenderFor(done <-chan struct{}, input bool) bool {
	fps := p.maxFPS
	if !p.fpsSet {
		// Default cap: only non-input Msgs are paced.
		fps = defaultMaxFPS
		if input {
			p.lastRenderAt = time.Now()
			return true
		}
	}
	if fps <= 0 {
		return true
	}
	interval := time.Second / time.Duration(fps)
	now := time.Now()
	if since := now.Sub(p.lastRenderAt); since < interval {
		p.renderPending = true
		if p.flushTimer == nil {
			p.flushTimer = time.AfterFunc(interval-since, func() {
				select {
				case p.msgs <- flushRenderMsg{}:
				case <-done:
				}
			})
		}
		return false
	}
	p.lastRenderAt = now
	return true
}

// isInputMsg reports whether msg comes from the user's input device, which
// the default frame cap never delays.
func isInputMsg(msg Msg) bool {
	switch msg.(type) {
	case Key, MouseEvent, PasteEvent, FocusEvent, ChordMsg:
		return true
	}
	return false
}

// renderIfChanged is render for the per-Msg path: when the View string is
// identical to the last painted frame's (and that frame is still what the
// terminal shows), nothing is written. The View result is handed to
// renderBody so View runs once per frame.
func (p *Program) renderIfChanged() {
	if p.accessible {
		p.render()
		return
	}
	if _, ok := p.directDraw(); ok {
		p.render() // a direct frame has no View string to compare with the last
		return
	}
	v := p.model.View()
	if p.haveView && len(p.lastFrame) > 0 && v == p.lastView {
		p.stopFlushTimer() // the pending frame, if any, is already on screen
		p.followCursor()
		return
	}
	p.peekedView, p.hasPeeked = v, true
	p.render()
}

// flushRenderMsg asks the loop to draw a frame that WithMaxFPS throttled away.
type flushRenderMsg struct{}

// stopFlushTimer drops any pending throttled render and its timer.
func (p *Program) stopFlushTimer() {
	p.renderPending = false
	if p.flushTimer != nil {
		p.flushTimer.Stop()
		p.flushTimer = nil
	}
}

// suspend implements tui.Suspend's runtime behavior: undo Run's terminal
// setup (in the reverse order Run enabled it), release raw mode, call fn
// with the terminal returned to its normal state, then redo the setup and
// reset frame-tracking state so the following render() repaints from
// scratch rather than diffing against a frame the relinquished terminal
// no longer necessarily still shows. Before any of that the input reader is
// stopped (and waited for), so fn owns stdin; a new reader starts once raw
// mode and the modes are back. Each undo/redo step is conditional
// on the corresponding feature having been enabled, mirroring Run's own
// conditionals exactly. If termState is nil (Run never ran against a real
// terminal — the case for a runLoop test driven directly), the
// raw-mode-specific term.Restore/term.MakeRaw calls are skipped, but
// everything else — the ansi escape writes, fn, and the frame-state
// reset — still runs, which is what makes Suspend testable without one.
func (p *Program) suspend(fn func() error) error {
	if p.stopInput != nil {
		p.stopInput()
	}
	p.leaveModes()

	p.termMu.Lock()
	if p.termRestore != nil && !p.termRestored {
		_ = p.termRestore()
	}
	p.termMu.Unlock()

	err := fn()

	p.termMu.Lock()
	gone := p.termRestored
	if p.termRestore != nil && !gone {
		if restore, rawErr := p.terminal().MakeRaw(p.mouseMode != MouseOff); rawErr == nil {
			p.termRestore = restore
		}
	}
	p.termMu.Unlock()

	if !gone {
		p.enterModes()
	}
	if p.startInput != nil {
		p.startInput()
	}

	p.lastFrame = nil
	p.liveLines = 0
	return err
}
