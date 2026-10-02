package tui

// running reports whether the event loop is currently accepting messages.
func (p *Program) running() bool {
	p.loopMu.RLock()
	defer p.loopMu.RUnlock()
	return p.loopDone != nil
}

// setLoopDone publishes (or, with nil, withdraws) the channel that is closed
// when the event loop stops draining p.msgs. Withdrawing marks Run as having
// returned, after which Send and TrySend drop messages.
func (p *Program) setLoopDone(done chan struct{}) {
	p.loopMu.Lock()
	p.loopDone = done
	p.loopEnded = done == nil
	p.loopMu.Unlock()
	// Wake every Send waiting for the loop: it has started, or has ended.
	if p.loopStarted != nil { // nil on a Program not built by NewProgram
		p.loopStartedOnce.Do(func() { close(p.loopStarted) })
	}
}

// loopState returns the current done channel (nil if the loop is not
// running) and whether Run has already returned.
func (p *Program) loopState() (done chan struct{}, ended bool) {
	p.loopMu.RLock()
	defer p.loopMu.RUnlock()
	return p.loopDone, p.loopEnded
}

// Send delivers msg to the model's Update, as if it had come from the
// terminal. It is safe to call from any goroutine; messages from one
// goroutine arrive in call order. Before Run starts, msg is queued and
// delivered once Run begins (if the queue fills first, Send waits for Run to
// start). While the queue is full during Run, Send waits for room. After Run
// has returned, Send drops msg and returns immediately. It never panics.
//
// Send must not be called from Update or View: the event loop is the only
// drainer, so a full queue would block it forever. Use TrySend there, or
// return a Cmd.
func (p *Program) Send(msg Msg) {
	if msg == nil {
		return
	}
	for {
		done, ended := p.loopState()
		if ended {
			return
		}
		if done == nil {
			// Not started yet: enqueue; if the queue is full, park until Run
			// begins draining it (or ends), then look at the state again.
			select {
			case p.msgs <- msg:
				return
			case <-p.loopStarted:
				continue
			}
		}
		select {
		case p.msgs <- msg:
		case <-done:
		}
		return
	}
}

// TrySend is like Send but never blocks: it reports whether msg was queued.
// It returns false if the queue is full or Run has already returned. Before
// Run starts it queues msg like Send. It is safe to call from any goroutine,
// including from Update.
func (p *Program) TrySend(msg Msg) bool {
	if msg == nil {
		return false
	}
	if _, ended := p.loopState(); ended {
		return false
	}
	select {
	case p.msgs <- msg:
		return true
	default:
		return false
	}
}

// Quit asks Run to return, as if the model had returned the Quit command.
// Messages already sent are handled first. It is safe to call from any
// goroutine and a no-op if Run is not active.
func (p *Program) Quit() { p.Send(QuitMsg{}) }
