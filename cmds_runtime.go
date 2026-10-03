package tui

import (
	"context"
	"sync"
	"time"

	"github.com/ows4444/tui/ansi"
)

// sequenceMsg carries an ordered list of Cmds; the event loop runs them one
// after another instead of handing it to Update.
type sequenceMsg []Cmd

// Sequence runs cmds one at a time, in order: each Cmd's message is
// delivered to the event loop before the next Cmd starts. A nil Cmd is
// skipped, and a Cmd that returns nil produces no message. If a Cmd
// produces QuitMsg the remaining Cmds are not run.
func Sequence(cmds ...Cmd) Cmd {
	var filtered []Cmd
	for _, c := range cmds {
		if c != nil {
			filtered = append(filtered, c)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return func() Msg { return sequenceMsg(filtered) }
}

// everyMsg asks the event loop to start a repeating ticker.
type everyMsg struct {
	d    time.Duration
	fn   func(time.Time) Msg
	stop chan struct{}

	id         int // registration order among running tickers, for the recorder
	rec        *recorder
	tickC      <-chan time.Time // set by the loop when a Clock is in use
	stopTicker func()
}

// everyTickMsg wraps one tick so the loop can drop it if its Every was
// cancelled after the tick was produced but before it was delivered.
type everyTickMsg struct {
	msg  Msg
	stop <-chan struct{}

	id  int // the ticker's registration order and period, for the recorder
	d   time.Duration
	rec *recorder
}

// Every returns a Cmd that produces fn(t) every d until the returned cancel
// func is called, plus that cancel func. After cancel returns, no further
// tick is delivered to Update, including one already in flight. cancel is
// safe to call more than once and from any goroutine. A non-positive d
// produces no ticks.
func Every(d time.Duration, fn func(time.Time) Msg) (Cmd, func()) {
	stop := make(chan struct{})
	var once sync.Once
	cancel := func() { once.Do(func() { close(stop) }) }
	cmd := func() Msg { return everyMsg{d: d, fn: fn, stop: stop} }
	return cmd, cancel
}

// Go returns a Cmd that runs fn with Program.Context and delivers its
// result. The context is cancelled when Run returns or the context given to
// WithContext is cancelled, so fn can stop early. It is FromCtx for a plain
// function: Sequence waits for it and RunCmd runs it the same way.
func Go(fn func(ctx context.Context) Msg) Cmd {
	if fn == nil {
		return nil
	}
	return FromCtx(fn)
}

// modeMsg asks the event loop to change a terminal mode.
type modeMsg struct{ apply func(p *Program) }

// EnableMouse returns a Cmd that switches mouse reporting to mode at run
// time (MouseOff turns it off).
func EnableMouse(mode MouseMode) Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			if old := p.setMouseMode(mode); old != MouseOff {
				_, disable := mouseModeCodes(old)
				p.write(ansi.MouseSGRDisable + disable)
			}
			if mode != MouseOff {
				enable, _ := mouseModeCodes(mode)
				p.write(enable + ansi.MouseSGREnable)
			}
		}}
	}
}

// EnterAltScreen returns a Cmd that switches to the alternate screen. It is
// a no-op in accessible mode or if already there.
func EnterAltScreen() Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			if p.accessible || p.setAltScreen(true) {
				return
			}
			p.write(ansi.AltScreenEnable)
			p.resetFrame()
		}}
	}
}

// ExitAltScreen returns a Cmd that leaves the alternate screen.
func ExitAltScreen() Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			if !p.setAltScreen(false) {
				return
			}
			p.write(ansi.AltScreenDisable)
			p.resetFrame()
			p.flushAltPrints()
		}}
	}
}

// SetWindowTitle returns a Cmd that sets the terminal window title. The first
// use pushes the terminal's current title (CSI 22;0t) so the Program can pop
// it (CSI 23;0t) on the way out, on every exit path: the window keeps the
// title it had before the program ran. Terminals without a title stack ignore
// both sequences and keep the last title set.
func SetWindowTitle(title string) Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			p.write(p.saveTitleOnce() + "\x1b]0;" + sanitizeTitle(title) + "\x07")
		}}
	}
}

// ClearScreen returns a Cmd that erases the screen and repaints the view.
func ClearScreen() Cmd {
	return func() Msg {
		return modeMsg{apply: func(p *Program) {
			p.write(ansi.ClearScreen + ansi.CSI + "H")
			p.resetFrame()
		}}
	}
}

// sanitizeTitle drops control characters so a title can't end the OSC
// sequence early or inject other escapes.
func sanitizeTitle(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 0x20 && r != 0x7f && !(r >= 0x80 && r < 0xa0) {
			out = append(out, r)
		}
	}
	return string(out)
}

// resetFrame forgets the last painted frame so the next render is a full
// repaint from the top of the screen.
func (p *Program) resetFrame() {
	p.lastFrame = nil
	p.liveLines = 0
}

// interceptCmdMsg handles the messages produced by the Cmds in this file.
// handled reports the loop should skip the message; otherwise the returned
// Msg (possibly unwrapped) continues through the normal path.
func (p *Program) interceptCmdMsg(msg Msg, done <-chan struct{}) (out Msg, handled bool) {
	switch m := msg.(type) {
	case sequenceMsg:
		go p.runSequence(m, done)
		return nil, true
	case everyMsg:
		if m.d > 0 {
			m.id, m.rec = p.tickSeq, p.rec
			p.tickSeq++
		}
		if p.clock != nil && m.d > 0 {
			// Register with the clock here, on the loop, so a fake clock
			// knows the ticker exists before the loop handles anything else.
			m.tickC, m.stopTicker = p.clock.NewTicker(m.d)
		}
		go p.runEvery(m, done)
		return nil, true
	case everyTickMsg:
		select {
		case <-m.stop:
			return nil, true
		default:
		}
		m.rec.tick(m.id, m.d) // at handling, so the log has the loop's order
		return m.msg, false
	case ctxMsg:
		p.spawnFn(func() { p.dispatchCtx(m.fn, done) })
		return nil, true
	case restoreReqMsg:
		p.restoreTerminal()
		close(m.done)
		return nil, true
	case modeMsg:
		m.apply(p)
		p.render()
		return nil, true
	}
	return msg, false
}

func (p *Program) runSequence(cmds []Cmd, done <-chan struct{}) {
	defer p.recoverCmdPanic(done)
	for _, c := range cmds {
		msg := c()
		if cm, ok := msg.(ctxMsg); ok {
			// A context-carrying Cmd (FromCtx, Go, Tick) runs here, in order, with the
			// Program's context, so the next Cmd waits for it as it would for any other.
			msg = cm.fn(p.ctx)
		}
		if msg == nil {
			continue
		}
		select {
		case p.msgs <- msg:
		case <-done:
			return
		}
		if _, quit := msg.(QuitMsg); quit {
			return
		}
	}
}

// cmdPool limits how many Cmds run at once (WithMaxConcurrentCmds).
type cmdPool struct {
	mu      sync.Mutex
	max     int // <= 0: unlimited
	running int
	queue   []func()
}

// spawn starts cmd on its own goroutine or, when WithMaxConcurrentCmds is set
// and the limit is reached, queues it. Queued Cmds start in the order spawn
// was called, which is the order the loop received them.
func (p *Program) spawn(cmd Cmd, done <-chan struct{}) {
	p.spawnFn(func() { p.dispatch(cmd, done) })
}

// spawnFn is spawn for an arbitrary run function.
func (p *Program) spawnFn(run func()) {
	pool := &p.cmdPool
	if pool.max <= 0 {
		go run()
		return
	}
	pool.mu.Lock()
	if pool.running >= pool.max {
		pool.queue = append(pool.queue, run)
		pool.mu.Unlock()
		return
	}
	pool.running++
	pool.mu.Unlock()
	go pool.work(run)
}

// work runs f, then keeps draining the queue while it holds the slot.
func (pool *cmdPool) work(f func()) {
	for f != nil {
		func() {
			defer func() {
				pool.mu.Lock()
				if len(pool.queue) > 0 {
					f, pool.queue = pool.queue[0], pool.queue[1:]
				} else {
					f = nil
					pool.running--
				}
				pool.mu.Unlock()
			}()
			f()
		}()
	}
}

func (p *Program) runEvery(e everyMsg, done <-chan struct{}) {
	defer p.recoverCmdPanic(done)
	if e.d <= 0 {
		return
	}
	tc, stopTicker := e.tickC, e.stopTicker
	if tc == nil {
		t := time.NewTicker(e.d)
		tc, stopTicker = t.C, t.Stop
	}
	defer stopTicker()
	for {
		select {
		case <-e.stop:
			return
		case <-done:
			return
		case now := <-tc:
			select {
			case <-e.stop:
				return
			default:
			}
			msg := e.fn(now)
			if msg == nil {
				continue
			}
			select {
			case p.msgs <- everyTickMsg{msg: msg, stop: e.stop, id: e.id, d: e.d, rec: e.rec}:
			case <-e.stop:
				return
			case <-done:
				return
			}
		}
	}
}
