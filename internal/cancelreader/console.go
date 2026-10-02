package cancelreader

import "time"

// consoleWait is the longest one wait on a console lasts, so a Cancel lands
// within that.
const consoleWait = 50 * time.Millisecond

// consoleRecord is one queued console input event, reduced to what the
// reader decides on.
type consoleRecord struct {
	// key is true for a key-down event, which ReadFile turns into bytes. Every
	// other event (window-buffer-size, focus, menu, mouse, key-up) makes the
	// console signal input but gives ReadFile nothing to return.
	key bool
	// resize is true for a WINDOW_BUFFER_SIZE_EVENT; w and h are the new
	// screen-buffer size it carries (not necessarily the visible window's).
	resize bool
	w, h   int
}

// Win32 INPUT_RECORD event types the reader looks at.
const (
	eventKey              = 0x0001
	eventWindowBufferSize = 0x0004
)

// inputRecord is the Win32 INPUT_RECORD: an event type, padding, and a
// 16-byte union. For a KEY_EVENT_RECORD the first four bytes are bKeyDown; for
// a WINDOW_BUFFER_SIZE_RECORD they are the COORD dwSize (X then Y, int16).
type inputRecord struct {
	eventType uint16
	_         uint16
	event     [4]uint32
}

// decodeRecord reduces one INPUT_RECORD to a consoleRecord.
func decodeRecord(r inputRecord) consoleRecord {
	switch r.eventType {
	case eventKey:
		return consoleRecord{key: r.event[0] != 0}
	case eventWindowBufferSize:
		return consoleRecord{
			resize: true,
			w:      int(int16(r.event[0] & 0xFFFF)), // #nosec G115 -- a COORD holds two signed 16-bit values
			h:      int(int16(r.event[0] >> 16)),    // #nosec G115 -- a COORD holds two signed 16-bit values
		}
	}
	return consoleRecord{}
}

// decodeRecords decodes the first n records of buf.
func decodeRecords(buf []inputRecord, n int) []consoleRecord {
	n = max(0, min(n, len(buf)))
	out := make([]consoleRecord, n)
	for i := range out {
		out[i] = decodeRecord(buf[i])
	}
	return out
}

// hasResize reports whether any record is a window-buffer-size event, and the
// size the last one carries.
func hasResize(recs []consoleRecord) (w, h int, ok bool) {
	for _, r := range recs {
		if r.resize {
			w, h, ok = r.w, r.h, true
		}
	}
	return w, h, ok
}

// console is the small part of the Windows console API the reader needs. It
// is an interface so the decision logic below runs, and is tested, on every
// platform; only the Win32 binding is Windows-only.
type console interface {
	// wait blocks for at most d until the console has queued input, and
	// reports whether it has (WaitForSingleObject).
	wait(d time.Duration) (bool, error)
	// peek returns the queued events without removing them
	// (PeekConsoleInputW).
	peek() ([]consoleRecord, error)
	// discard removes the first n queued events without waiting
	// (ReadConsoleInputW).
	discard(n int) error
}

// consoleKeyReady runs one wait slice. It reports true only when a key event
// is queued, after removing the non-key events in front of it; when the queue
// holds only non-key events it removes them and reports false, so the caller
// keeps waiting instead of calling a ReadFile that would block on the next key.
//
// onResize, when non-nil, is called once per pass that saw a window-buffer-size
// event, before those events are discarded: the reader owns the console input
// queue, so it is the only place a resize event can be observed.
func consoleKeyReady(c console, onResize func()) (bool, error) {
	signalled, err := c.wait(consoleWait)
	if err != nil || !signalled {
		return false, err
	}
	recs, err := c.peek()
	if err != nil {
		return false, err
	}
	if onResize != nil {
		if _, _, ok := hasResize(recs); ok {
			onResize()
		}
	}
	lead := 0
	for lead < len(recs) && !recs[lead].key {
		lead++
	}
	if lead > 0 {
		if err := c.discard(lead); err != nil {
			return false, err
		}
	}
	return lead < len(recs), nil
}

// waitForKey waits until the console has a key to read, discarding the events
// that are not keys. It returns ErrCanceled once canceled reports true, and
// any error from the console.
func waitForKey(c console, canceled func() bool) error {
	return waitForKeyNotify(c, canceled, nil)
}

// waitForKeyNotify is waitForKey that calls onResize for window-buffer-size
// events it sees in the queue.
func waitForKeyNotify(c console, canceled func() bool, onResize func()) error {
	for {
		if canceled() {
			return ErrCanceled
		}
		ready, err := consoleKeyReady(c, onResize)
		if err != nil {
			return err
		}
		if ready {
			return nil
		}
	}
}
