package input

import (
	"time"
	"unicode/utf8"
)

// PasteEvent carries the full text of a bracketed paste
// (ESC[200~...ESC[201~), delivered as one event rather than as individual
// key presses.
type PasteEvent struct {
	Text string
	// Incomplete is true when the end marker never arrived: the stream went
	// idle for PasteIdleTimeout or ended. Text is what was received.
	Incomplete bool
	// Truncated is true when the paste was longer than MaxPasteBytes. Text
	// holds the first MaxPasteBytes bytes (never a partial character) and the
	// rest, up to the end marker, was discarded.
	Truncated bool
}

const (
	pasteEndMarker = "\x1b[201~"

	// PasteIdleTimeout is how long a paste may go without a new byte before
	// the Reader gives up on its end marker and delivers what it has.
	PasteIdleTimeout = 250 * time.Millisecond

	// MaxPasteBytes is the most paste text one PasteEvent holds.
	MaxPasteBytes = 16 << 20
)

// readPaste collects bytes until it sees the bracketed-paste end marker. The
// marker's bytes are never consumed as paste content. If the stream ends
// first, or goes idle for PasteIdleTimeout, whatever was collected is
// returned as an Incomplete paste (alongside the read error, for the end of
// the stream). Text beyond MaxPasteBytes is discarded and marks the paste
// Truncated.
func (rd *Reader) readPaste() (Event, error) {
	var buf []byte
	var truncated bool
	done := func(incomplete bool) PasteEvent {
		return PasteEvent{Text: string(buf), Incomplete: incomplete, Truncated: truncated}
	}
	for {
		if ok, err := rd.waitBytesErr(1, PasteIdleTimeout); !ok {
			return done(true), err
		}
		b, err := rd.r.ReadByte()
		if err != nil {
			return done(true), err
		}
		if b == pasteEndMarker[0] {
			// Peek for the rest of the marker without consuming it, but only
			// as long as the stream keeps up: a short read is a lost marker.
			if ok, err := rd.waitBytesErr(len(pasteEndMarker)-1, PasteIdleTimeout); !ok {
				return done(true), err
			}
			if peek, err := rd.r.Peek(len(pasteEndMarker) - 1); err == nil && string(peek) == pasteEndMarker[1:] {
				_, _ = rd.r.Discard(len(pasteEndMarker) - 1)
				return done(false), nil
			}
		}
		r, err := rd.decodeRune(b)
		if err != nil {
			return done(true), err
		}
		if len(buf)+utf8.RuneLen(r) > MaxPasteBytes {
			truncated = true
			continue
		}
		buf = utf8.AppendRune(buf, r)
	}
}
