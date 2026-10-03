package tui

import (
	"errors"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ows4444/tui/input"
	"github.com/ows4444/tui/internal/cancelreader"
)

// swapReader is the io.Reader one input.Reader keeps for the life of runLoop
// while the source underneath it is replaced each time Suspend restarts the
// reader. Keeping the single input.Reader (and its bufio buffer) means bytes
// already buffered survive a restart, and a new goroutine using it only ever
// starts after the old one has exited (readerSet.start does nothing while one
// is still running), so the buffer is never shared.
type swapReader struct {
	mu  sync.Mutex
	cur io.Reader
}

func (s *swapReader) set(r io.Reader) {
	s.mu.Lock()
	s.cur = r
	s.mu.Unlock()
}

func (s *swapReader) Read(b []byte) (int, error) {
	s.mu.Lock()
	r := s.cur
	s.mu.Unlock()
	return r.Read(b)
}

// readerSet owns the input reader goroutine of one runLoop: it can be
// stopped (Suspend) and started again. All methods run on the loop
// goroutine.
type readerSet struct {
	p    *Program
	done <-chan struct{} // runLoop's shutdown
	sw   *swapReader
	rd   *input.Reader

	running  bool          // a generation was started and has not been stopped
	cancelFn func()        // cancels the current generation's source
	stopCh   chan struct{} // closed to ask the current generation to leave
	focus    focusDedupe   // survives restarts, so a repeat across one is still dropped
	carry    Msg           // event read but undelivered when stopped; next generation sends it first
}

func (p *Program) newReaderSet(done <-chan struct{}) *readerSet {
	sw := &swapReader{}
	rd := input.NewReader(sw)
	if p.escTimeout > 0 {
		rd.SetEscTimeout(p.escTimeout)
	}
	rd.SetReportEvents(p.kittyFlags()&KeyboardReportEvents != 0)
	rd.SetReportCSIReplies(p.capProbe.active())
	return &readerSet{p: p, done: done, sw: sw, rd: rd, cancelFn: func() {}}
}

func (rs *readerSet) cancel() { rs.cancelFn() }

// start begins a new reader goroutine on a fresh source. It does nothing
// while the previous generation is still running, which is the case after
// stop on a WithInput source.
func (rs *readerSet) start() {
	if rs.running {
		return
	}
	rs.running = true
	p := rs.p
	var src io.Reader
	var closeInput func()
	if p.inReader != nil {
		src, rs.cancelFn, closeInput = p.inReader, func() {}, func() {}
	} else {
		// A previous generation on the deadline fallback left one in the past.
		_ = p.input.SetReadDeadline(time.Time{})
		src, rs.cancelFn, closeInput = inputSource(p.input, p.kickResize)
	}
	if p.rec != nil {
		src = recReader{r: src, rec: p.rec}
	}
	rs.sw.set(src)
	stop := make(chan struct{})
	rdDone := make(chan struct{})
	rs.stopCh, p.rdDone = stop, rdDone
	carry := rs.carry
	rs.carry = nil

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		defer close(rdDone)
		defer closeInput()
		rs.read(stop, carry)
	}()
}

// stop cancels the current reader and waits for it to exit. A reader
// cancelled mid-read leaves unread bytes in the file (cancelreader never
// consumes on cancel); one cancelled while handing over an event keeps it
// as carry. A WithInput source cannot be cancelled, so it is left
// running.
func (rs *readerSet) stop() {
	if rs.p.inReader != nil {
		return
	}
	close(rs.stopCh)
	rs.cancelFn()
	<-rs.p.rdDone
	rs.running = false
}

// read is the reader goroutine body. It returns when its source is
// cancelled, on stop or shutdown, or on a terminal error.
func (rs *readerSet) read(stop <-chan struct{}, carry Msg) {
	p, done := rs.p, rs.done
	deliver := func(ev Msg) bool {
		select {
		case p.msgs <- ev:
			return true
		case <-stop:
			rs.carry = ev
			return false
		case <-done:
			return false
		}
	}
	if carry != nil && !deliver(carry) {
		return
	}
	for {
		ev, err := rs.rd.ReadEvent()
		if err != nil {
			// Cancelled by stop or runLoop's exit: not a failure to report.
			if errors.Is(err, cancelreader.ErrCanceled) || errors.Is(err, os.ErrDeadlineExceeded) {
				return
			}
			// A WithInput source ends at EOF: quit, after the events
			// already queued ahead of this message.
			if p.inReader != nil && errors.Is(err, io.EOF) {
				select {
				case p.msgs <- QuitMsg{}:
				case <-done:
				}
				return
			}
			select {
			case <-done:
			case <-stop:
			default:
				select {
				case p.msgs <- InputErrorMsg{Err: err}:
				case <-done:
				case <-stop:
				}
			}
			return
		}
		if _, ok := ev.(BackgroundColorEvent); ok {
			p.bgSeen.Store(true)
		}
		if cm, ok := p.clipboardReply(ev); ok {
			if !deliver(cm) {
				return
			}
			continue
		}
		if extra, consumed := p.capProbe.observe(ev); extra != nil || consumed {
			if cm, ok := extra.(CapabilitiesMsg); ok {
				p.applyCapabilities(cm.Capabilities)
			}
			if extra != nil && !deliver(extra) {
				return
			}
			continue
		}
		if fe, ok := ev.(FocusEvent); ok && !rs.focus.deliver(fe) {
			continue
		}
		if !deliver(ev) {
			return
		}
	}
}
